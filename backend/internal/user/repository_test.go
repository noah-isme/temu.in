package user

import (
	"os"
	"testing"

	"temu.in/backend/internal/models"
	"temu.in/backend/internal/testutil"
)

// Integration tests run against a per-package Postgres database created by
// testutil (TEST_DATABASE_URL). They must not call t.Parallel().
func TestMain(m *testing.M) { os.Exit(testutil.Main(m)) }

func TestRepositoryIntegration(t *testing.T) {
	db := testutil.DB(t)
	repo := NewRepository(db)

	u := &models.User{Email: "repo@example.test", Password: "hash", Name: "Repo", Role: "user"}
	if err := repo.Create(u); err != nil {
		t.Fatalf("create: %v", err)
	}
	if got, err := repo.FindByEmail("repo@example.test"); err != nil || got == nil || got.ID != u.ID {
		t.Fatalf("FindByEmail: %+v %v", got, err)
	}
	if got, err := repo.FindByEmail("missing@example.test"); err != nil || got != nil {
		t.Fatalf("FindByEmail missing = %+v %v, want nil, nil", got, err)
	}
	if got, err := repo.FindByID(u.ID); err != nil || got.Email != u.Email {
		t.Fatalf("FindByID: %+v %v", got, err)
	}
	if _, err := repo.FindByID(9999); err == nil {
		t.Fatal("FindByID unknown: expected error")
	}

	t.Run("password reset", func(t *testing.T) {
		if err := repo.SavePasswordReset(u.ID, "reset-token"); err != nil {
			t.Fatal(err)
		}
		id, err := repo.VerifyPasswordResetToken("reset-token")
		if err != nil || id != u.ID {
			t.Fatalf("verify: %d %v", id, err)
		}
		if _, err := repo.VerifyPasswordResetToken("other"); err == nil {
			t.Fatal("unknown reset token verified")
		}
		if err := repo.UpdatePassword(u.ID, "new-hash"); err != nil {
			t.Fatal(err)
		}
		if got, _ := repo.FindByID(u.ID); got.Password != "new-hash" {
			t.Fatalf("password = %q", got.Password)
		}
		if err := repo.DeletePasswordResetToken("reset-token"); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.VerifyPasswordResetToken("reset-token"); err == nil {
			t.Fatal("deleted reset token still verifies")
		}
		// Expired tokens are rejected.
		if err := db.Exec(`INSERT INTO password_resets (user_id, token_hash, expires_at) VALUES (?, 'expired', NOW() - INTERVAL '1 minute')`, u.ID).Error; err != nil {
			t.Fatal(err)
		}
		if _, err := repo.VerifyPasswordResetToken("expired"); err == nil {
			t.Fatal("expired reset token verified")
		}
	})

	t.Run("email verification", func(t *testing.T) {
		if err := repo.SaveVerificationToken(u.ID, "verify-token"); err != nil {
			t.Fatal(err)
		}
		got, err := repo.FindByVerificationToken("verify-token")
		if err != nil || got == nil || got.ID != u.ID {
			t.Fatalf("FindByVerificationToken: %+v %v", got, err)
		}
		if err := repo.MarkVerified(u.ID); err != nil {
			t.Fatal(err)
		}
		after, _ := repo.FindByID(u.ID)
		if !after.IsVerified || after.VerificationToken != nil {
			t.Fatalf("after MarkVerified: verified=%v token=%v", after.IsVerified, after.VerificationToken)
		}
		if got, err := repo.FindByVerificationToken("verify-token"); err != nil || got != nil {
			t.Fatalf("consumed token = %+v %v, want nil, nil", got, err)
		}
	})

	t.Run("roles and listing", func(t *testing.T) {
		if err := repo.SetRole(u.ID, "admin"); err != nil {
			t.Fatal(err)
		}
		other := testutil.CreateUser(t, db, "user")
		users, err := repo.ListAll()
		if err != nil || len(users) != 2 {
			t.Fatalf("ListAll: %d %v", len(users), err)
		}
		roles := map[uint]string{}
		for _, lu := range users {
			roles[lu.ID] = lu.Role
			if lu.Password != "" {
				t.Fatal("ListAll must not load password hashes")
			}
		}
		if roles[u.ID] != "admin" || roles[other.ID] != "user" {
			t.Fatalf("roles = %v", roles)
		}
	})
}
