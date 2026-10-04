package audit

import (
	"os"
	"testing"
	"time"

	"temu.in/backend/internal/models"
	"temu.in/backend/internal/testutil"
)

// Integration tests run against a per-package Postgres database created by
// testutil (TEST_DATABASE_URL). They must not call t.Parallel().
func TestMain(m *testing.M) { os.Exit(testutil.Main(m)) }

func TestRecordAndListNewestFirst(t *testing.T) {
	db := testutil.DB(t)
	repo := NewRepository(db)
	actor := testutil.CreateUser(t, db, "admin")

	base := time.Now().Add(-time.Hour)
	for i, action := range []string{"first", "second", "third"} {
		a := &models.AdminAudit{CreatedAt: base.Add(time.Duration(i) * time.Minute), ActorID: actor.ID, Action: action, Target: "user:x", Details: "d"}
		if err := repo.Create(a); err != nil {
			t.Fatalf("create %s: %v", action, err)
		}
		if a.ID == 0 {
			t.Fatalf("create %s: id not set", action)
		}
	}

	got, err := repo.ListAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	for i, want := range []string{"third", "second", "first"} {
		if got[i].Action != want {
			t.Fatalf("order[%d] = %q, want %q (newest first)", i, got[i].Action, want)
		}
	}
}
