package token

import (
	"errors"
	"os"
	"testing"
	"time"

	"gorm.io/gorm"

	"temu.in/backend/internal/models"
	"temu.in/backend/internal/testutil"
)

// Integration tests run against a per-package Postgres database created by
// testutil (TEST_DATABASE_URL). They must not call t.Parallel().
func TestMain(m *testing.M) { os.Exit(testutil.Main(m)) }

// The Redis blacklist is checked before Postgres, so a revoked hash is
// rejected without touching the database (db is nil here and would panic).
func TestFindByHashRedisBlacklistShortCircuits(t *testing.T) {
	client, mr := testutil.Redis(t)
	if err := mr.Set("revoked_rt:abc", "1"); err != nil {
		t.Fatal(err)
	}
	repo := NewRepositoryWithCache(nil, client)
	if _, err := repo.FindByHash("abc"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("err = %v, want gorm.ErrRecordNotFound", err)
	}
}

func TestRepositoryIntegration(t *testing.T) {
	db := testutil.DB(t)
	client, mr := testutil.Redis(t)
	repo := NewRepositoryWithCache(db, client)
	u := testutil.CreateUser(t, db, "user")

	rt := &models.RefreshToken{TokenHash: "hash-1", UserID: u.ID, ExpiresAt: time.Now().Add(time.Hour)}
	if err := repo.Create(rt); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.FindByHash("hash-1")
	if err != nil || got.UserID != u.ID || got.Revoked {
		t.Fatalf("find: %+v %v", got, err)
	}

	if err := repo.RevokeByHash("hash-1"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if !mr.Exists("revoked_rt:hash-1") {
		t.Fatal("revoked hash not blacklisted in Redis")
	}
	if ttl := mr.TTL("revoked_rt:hash-1"); ttl <= 0 {
		t.Fatalf("blacklist key has no TTL: %v", ttl)
	}
	if _, err := repo.FindByHash("hash-1"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("revoked find err = %v", err)
	}
	// Without Redis the DB row still records the revocation.
	plain := NewRepository(db)
	stored, err := plain.FindByHash("hash-1")
	if err != nil || !stored.Revoked {
		t.Fatalf("db revoked flag: %+v %v", stored, err)
	}

	for _, h := range []string{"hash-2", "hash-3"} {
		if err := plain.Create(&models.RefreshToken{TokenHash: h, UserID: u.ID, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.RevokeAllForUser(u.ID); err != nil {
		t.Fatalf("revoke all: %v", err)
	}
	var active int64
	db.Model(&models.RefreshToken{}).Where("user_id = ? AND revoked = false", u.ID).Count(&active)
	if active != 0 {
		t.Fatalf("active tokens after RevokeAllForUser = %d", active)
	}
	if err := plain.RevokeAllForUser(u.ID); err != nil {
		t.Fatalf("revoke all without cache: %v", err)
	}
	if _, err := plain.FindByHash("missing"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("missing err = %v", err)
	}
}
