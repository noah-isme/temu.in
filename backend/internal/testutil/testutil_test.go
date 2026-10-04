package testutil

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"temu.in/backend/internal/models"
)

func TestMain(m *testing.M) { os.Exit(Main(m)) }

// TestHelperDB is executed in a subprocess by TestDBWithoutURL.
func TestHelperDB(t *testing.T) {
	if os.Getenv("TESTUTIL_HELPER") != "1" {
		t.Skip("helper process only")
	}
	DB(t)
	t.Log("reached after DB")
}

func runHelper(t *testing.T, require string) (string, error) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperDB$", "-test.v")
	env := []string{"TESTUTIL_HELPER=1", "TEST_DATABASE_URL=", "REQUIRE_INTEGRATION=" + require}
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "TEST_DATABASE_URL=") || strings.HasPrefix(kv, "REQUIRE_INTEGRATION=") || strings.HasPrefix(kv, "TESTUTIL_HELPER=") {
			continue
		}
		env = append(env, kv)
	}
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestDBWithoutURL(t *testing.T) {
	t.Run("skips by default", func(t *testing.T) {
		out, err := runHelper(t, "")
		if err != nil {
			t.Fatalf("helper failed: %v\n%s", err, out)
		}
		if !strings.Contains(out, "--- SKIP: TestHelperDB") || !strings.Contains(out, "TEST_DATABASE_URL not set") {
			t.Fatalf("expected skip, got:\n%s", out)
		}
	})
	t.Run("fails when integration required", func(t *testing.T) {
		out, err := runHelper(t, "1")
		if err == nil {
			t.Fatalf("expected non-zero exit, got:\n%s", out)
		}
		if !strings.Contains(out, "--- FAIL: TestHelperDB") || !strings.Contains(out, "integration tests required") {
			t.Fatalf("expected failure, got:\n%s", out)
		}
		if strings.Contains(out, "reached after DB") {
			t.Fatal("test continued after required-integration failure")
		}
	})
}

func TestPackageDBName(t *testing.T) {
	valid := regexp.MustCompile(`^temu_in_test_[a-z0-9_]+_[0-9a-f]{8}$`)
	cases := map[string]string{
		"/tmp/go-build123/b001/booking.test": "temu_in_test_booking_",
		"Payment-Svc.test":                   "temu_in_test_payment_svc_",
		"...":                                "temu_in_test_pkg_",
	}
	for in, prefix := range cases {
		got, err := packageDBName(in)
		if err != nil {
			t.Fatalf("packageDBName(%q): %v", in, err)
		}
		if !strings.HasPrefix(got, prefix) || !valid.MatchString(got) {
			t.Errorf("packageDBName(%q) = %q, want prefix %q", in, got, prefix)
		}
	}
	long, err := packageDBName(strings.Repeat("a", 100) + ".test")
	if err != nil || len(long) > 63 {
		t.Errorf("long name = %q (%d bytes), err %v", long, len(long), err)
	}
	a, _ := packageDBName("x.test")
	b, _ := packageDBName("x.test")
	if a == b {
		t.Error("expected random suffixes to differ")
	}
}

func TestWithDatabase(t *testing.T) {
	got, err := withDatabase("postgres://u:p@localhost:5432/postgres?sslmode=disable", "temu_in_test_x_00")
	if err != nil {
		t.Fatal(err)
	}
	want := "postgres://u:p@localhost:5432/temu_in_test_x_00?sslmode=disable"
	if got != want {
		t.Errorf("withDatabase = %q, want %q", got, want)
	}
	if _, err := withDatabase("://bad", "x"); err == nil {
		t.Error("expected parse error")
	}
}

func TestDBIsolatedAndTruncated(t *testing.T) {
	db := DB(t)

	var current string
	if err := db.Raw("SELECT current_database()").Scan(&current).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(current, dbNamePrefix+"testutil_") {
		t.Fatalf("connected to %q, want a per-package test database", current)
	}

	u := CreateUser(t, db, "provider")
	p := CreateProvider(t, db, u.ID)
	s := CreateService(t, db, p.ID, 60, 0)
	CreateSchedule(t, db, p.ID, 1, "09:00", "17:00")
	b := CreateBooking(t, db, u.ID, p.ID, &s.ID, "2030-01-07", "09:00", "10:00", models.BookingStatusConfirmed)
	if b.ID == 0 {
		t.Fatal("booking not inserted")
	}

	db = DB(t) // a second call truncates
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("users after truncate = %d, want 0", count)
	}
	var version int
	if err := db.Raw("SELECT version FROM schema_migrations").Scan(&version).Error; err != nil || version == 0 {
		t.Fatalf("schema_migrations should survive truncation: version=%d err=%v", version, err)
	}
}

func TestRedis(t *testing.T) {
	client, srv := Redis(t)
	ctx := context.Background()
	if err := client.Set(ctx, "k", "v", 0).Err(); err != nil {
		t.Fatal(err)
	}
	if got, _ := srv.Get("k"); got != "v" {
		t.Fatalf("miniredis value = %q", got)
	}
	if !client.Options().ContextTimeoutEnabled {
		t.Fatal("test client must use the production constructor settings")
	}
}
