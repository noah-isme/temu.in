package provider

import (
	"errors"
	"testing"

	"temu.in/backend/internal/models"
)

type fakeRepository struct {
	provider *models.ServiceProvider
}

func (f *fakeRepository) Create(provider *models.ServiceProvider) error {
	f.provider = provider
	return nil
}
func (f *fakeRepository) FindByID(string) (*models.ServiceProvider, error) {
	if f.provider == nil {
		return nil, errors.New("not found")
	}
	return f.provider, nil
}
func (f *fakeRepository) FindByUserID(userID uint) (*models.ServiceProvider, error) {
	if f.provider != nil && f.provider.UserID == userID {
		return f.provider, nil
	}
	return nil, errors.New("not found")
}
func (f *fakeRepository) Update(provider *models.ServiceProvider) error {
	f.provider = provider
	return nil
}
func (f *fakeRepository) Delete(string) error { return nil }
func (f *fakeRepository) List(ProviderFilters) ([]models.ServiceProvider, int64, error) {
	return nil, 0, nil
}

func TestUpdateProviderRequiresOwnerUnlessAdmin(t *testing.T) {
	repo := &fakeRepository{provider: &models.ServiceProvider{ID: "provider-1", UserID: 7, BusinessName: "Old"}}
	svc := NewService(repo)
	newName := "New"

	if _, err := svc.UpdateProvider("provider-1", 8, false, &UpdateProviderRequest{BusinessName: &newName}); err == nil {
		t.Fatal("non-owner update must be rejected")
	}
	updated, err := svc.UpdateProvider("provider-1", 8, true, &UpdateProviderRequest{BusinessName: &newName})
	if err != nil {
		t.Fatalf("admin update failed: %v", err)
	}
	if updated.BusinessName != newName {
		t.Fatalf("business name = %q, want %q", updated.BusinessName, newName)
	}
}
