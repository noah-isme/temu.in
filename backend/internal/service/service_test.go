package service

import (
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"

	"temu.in/backend/internal/models"
)

type fakeRepo struct {
	services   map[string]*models.Service
	created    []*models.Service
	updated    []*models.Service
	deleted    []string
	lastFilter ServiceFilter
	createErr  error
}

func newFakeRepo(seed ...*models.Service) *fakeRepo {
	r := &fakeRepo{services: map[string]*models.Service{}}
	for _, s := range seed {
		r.services[s.ID] = s
	}
	return r
}

func (r *fakeRepo) Create(s *models.Service) error {
	if r.createErr != nil {
		return r.createErr
	}
	s.ID = "new-id"
	r.created = append(r.created, s)
	return nil
}

func (r *fakeRepo) FindByID(id string) (*models.Service, error) {
	if s, ok := r.services[id]; ok {
		cp := *s
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeRepo) FindByProviderID(providerID string) ([]models.Service, error) {
	return r.ListByProvider(providerID, false)
}

func (r *fakeRepo) Update(s *models.Service) error {
	r.updated = append(r.updated, s)
	return nil
}

func (r *fakeRepo) Delete(id string) error {
	r.deleted = append(r.deleted, id)
	return nil
}

func (r *fakeRepo) ListByProvider(providerID string, activeOnly bool) ([]models.Service, error) {
	var out []models.Service
	for _, s := range r.services {
		if s.ProviderID == providerID && (!activeOnly || s.IsActive) {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (r *fakeRepo) Search(filter ServiceFilter) ([]models.Service, error) {
	r.lastFilter = filter
	return nil, nil
}

func TestCreateService(t *testing.T) {
	repo := newFakeRepo()
	got, err := NewService(repo).CreateService("prov-1", &CreateServiceRequest{Name: "Cut", Description: "d", Duration: 30, Price: 50000})
	if err != nil {
		t.Fatal(err)
	}
	if got.ProviderID != "prov-1" || !got.IsActive || got.Duration != 30 || got.Price != 50000 || got.ID != "new-id" {
		t.Fatalf("created = %+v", got)
	}

	repo.createErr = errors.New("db down")
	if _, err := NewService(repo).CreateService("prov-1", &CreateServiceRequest{Name: "x", Duration: 1}); err == nil || !strings.Contains(err.Error(), "failed to create service") {
		t.Fatalf("create err = %v", err)
	}
}

func TestUpdateAndDeleteOwnership(t *testing.T) {
	repo := newFakeRepo(&models.Service{ID: "s1", ProviderID: "prov-1", Name: "Old", Duration: 30, Price: 10, IsActive: true})
	svc := NewService(repo)

	name, dur, price, active := "New", 45, 20.0, false
	desc := "updated"
	got, err := svc.UpdateService("s1", "prov-1", &UpdateServiceRequest{Name: &name, Description: &desc, Duration: &dur, Price: &price, IsActive: &active})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "New" || got.Duration != 45 || got.Price != 20 || got.IsActive || got.Description != "updated" {
		t.Fatalf("updated = %+v", got)
	}
	if _, err := svc.UpdateService("s1", "prov-2", &UpdateServiceRequest{Name: &name}); err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("foreign update err = %v", err)
	}
	if _, err := svc.UpdateService("missing", "prov-1", &UpdateServiceRequest{}); err == nil {
		t.Fatal("update of missing service succeeded")
	}

	if err := svc.DeleteService("s1", "prov-2"); err == nil {
		t.Fatal("foreign delete succeeded")
	}
	if err := svc.DeleteService("missing", "prov-1"); err == nil {
		t.Fatal("delete of missing service succeeded")
	}
	if err := svc.DeleteService("s1", "prov-1"); err != nil || len(repo.deleted) != 1 {
		t.Fatalf("delete: %v %v", err, repo.deleted)
	}
}

func TestGetAndListAndSearch(t *testing.T) {
	repo := newFakeRepo(
		&models.Service{ID: "a", ProviderID: "p", IsActive: true},
		&models.Service{ID: "b", ProviderID: "p", IsActive: false},
	)
	svc := NewService(repo)
	if s, err := svc.GetService("a"); err != nil || s.ID != "a" {
		t.Fatalf("GetService: %+v %v", s, err)
	}
	if _, err := svc.GetService("zzz"); err == nil {
		t.Fatal("GetService missing: expected error")
	}
	if all, _ := svc.GetServicesByProvider("p", false); len(all) != 2 {
		t.Fatalf("all = %d", len(all))
	}
	if active, _ := svc.GetServicesByProvider("p", true); len(active) != 1 {
		t.Fatalf("active = %d", len(active))
	}

	minPrice := 10.0
	filter := ServiceFilter{Name: "cut", MinPrice: &minPrice, ProviderID: "p", Category: "salon"}
	if _, err := svc.SearchServices(filter); err != nil {
		t.Fatal(err)
	}
	if repo.lastFilter.Name != "cut" || repo.lastFilter.MinPrice != &minPrice || repo.lastFilter.ProviderID != "p" || repo.lastFilter.Category != "salon" {
		t.Fatalf("filter not passed through: %+v", repo.lastFilter)
	}
}
