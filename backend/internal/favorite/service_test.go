package favorite

import (
	"errors"
	"testing"

	"temu.in/backend/internal/models"
)

type fakeRepo struct {
	favs      []models.Favorite
	existsErr error
	createErr error
}

func matches(f models.Favorite, userID uint, serviceID, providerID *string) bool {
	if f.UserID != userID {
		return false
	}
	if serviceID != nil && (f.ServiceID == nil || *f.ServiceID != *serviceID) {
		return false
	}
	if providerID != nil && (f.ProviderID == nil || *f.ProviderID != *providerID) {
		return false
	}
	return true
}

func (r *fakeRepo) Create(fav *models.Favorite) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.favs = append(r.favs, *fav)
	return nil
}

func (r *fakeRepo) Delete(userID uint, favID string) error { return nil }

func (r *fakeRepo) DeleteByTarget(userID uint, serviceID, providerID *string) error {
	kept := r.favs[:0]
	for _, f := range r.favs {
		if !matches(f, userID, serviceID, providerID) {
			kept = append(kept, f)
		}
	}
	r.favs = kept
	return nil
}

func (r *fakeRepo) ListByUser(userID uint) ([]models.Favorite, error) {
	var out []models.Favorite
	for _, f := range r.favs {
		if f.UserID == userID {
			out = append(out, f)
		}
	}
	return out, nil
}

func (r *fakeRepo) Exists(userID uint, serviceID, providerID *string) (bool, error) {
	if r.existsErr != nil {
		return false, r.existsErr
	}
	for _, f := range r.favs {
		if matches(f, userID, serviceID, providerID) {
			return true, nil
		}
	}
	return false, nil
}

func ptr(s string) *string { return &s }

func TestToggleFavoriteAddsThenRemoves(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo)

	added, err := svc.ToggleFavorite(1, ptr("svc-1"), nil)
	if err != nil || !added {
		t.Fatalf("first toggle = %v %v, want added", added, err)
	}
	if _, err := svc.ToggleFavorite(1, nil, ptr("prov-1")); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ToggleFavorite(2, ptr("svc-1"), nil); err != nil {
		t.Fatal(err)
	}

	list, err := svc.ListFavorites(1)
	if err != nil || len(list) != 2 {
		t.Fatalf("ListFavorites(1) = %d %v, want 2", len(list), err)
	}

	added, err = svc.ToggleFavorite(1, ptr("svc-1"), nil)
	if err != nil || added {
		t.Fatalf("second toggle = %v %v, want removed", added, err)
	}
	list, _ = svc.ListFavorites(1)
	if len(list) != 1 || list[0].ProviderID == nil || *list[0].ProviderID != "prov-1" {
		t.Fatalf("after removal = %+v", list)
	}
	if other, _ := svc.ListFavorites(2); len(other) != 1 {
		t.Fatalf("other user's favourite affected: %+v", other)
	}
}

func TestToggleFavoriteErrors(t *testing.T) {
	if _, err := NewService(&fakeRepo{}).ToggleFavorite(1, nil, nil); err == nil {
		t.Fatal("expected error when neither target is given")
	}
	boom := errors.New("boom")
	if _, err := NewService(&fakeRepo{existsErr: boom}).ToggleFavorite(1, ptr("s"), nil); !errors.Is(err, boom) {
		t.Fatalf("exists err = %v", err)
	}
	if _, err := NewService(&fakeRepo{createErr: boom}).ToggleFavorite(1, ptr("s"), nil); !errors.Is(err, boom) {
		t.Fatalf("create err = %v", err)
	}
}
