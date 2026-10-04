package notification

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"temu.in/backend/internal/models"
	"temu.in/backend/internal/testutil"
)

func TestMain(m *testing.M) { os.Exit(testutil.Main(m)) }

func TestNewServiceWithBaseURLNormalisesBaseURL(t *testing.T) {
	cases := map[string]string{
		"https://app.example.test":   "https://app.example.test",
		"https://app.example.test/":  "https://app.example.test",
		"https://app.example.test//": "https://app.example.test",
		"":                           DefaultFrontendBaseURL,
	}
	for in, want := range cases {
		svc := NewServiceWithBaseURL(nil, in).(*service)
		if svc.baseURL != want {
			t.Errorf("NewServiceWithBaseURL(%q).baseURL = %q, want %q", in, svc.baseURL, want)
		}
	}
	if got := NewService(nil).(*service).baseURL; got != DefaultFrontendBaseURL {
		t.Errorf("NewService baseURL = %q, want default", got)
	}
}

func TestFrontendLinkEscapesToken(t *testing.T) {
	svc := NewServiceWithBaseURL(nil, "https://app.example.test/").(*service)
	got := svc.frontendLink("/reset-password", "a b&c")
	want := "https://app.example.test/reset-password?token=a+b%26c"
	if got != want {
		t.Fatalf("frontendLink = %q, want %q", got, want)
	}
}

func TestAuthEmailsUseConfiguredBaseURL(t *testing.T) {
	db := testutil.DB(t)
	user := testutil.CreateUser(t, db, "user")
	svc := NewServiceWithBaseURL(NewRepository(db), "https://app.example.test/")
	ctx := context.Background()

	if err := svc.SendPasswordReset(ctx, user, "reset-tok"); err != nil {
		t.Fatalf("SendPasswordReset: %v", err)
	}
	if err := svc.SendEmailVerification(ctx, user, "verify-tok"); err != nil {
		t.Fatalf("SendEmailVerification: %v", err)
	}

	assertPayloadLink := func(notificationType, key, want string) {
		t.Helper()
		var n models.Notification
		if err := db.Where("type = ?", notificationType).First(&n).Error; err != nil {
			t.Fatalf("load %s notification: %v", notificationType, err)
		}
		var data map[string]string
		if err := json.Unmarshal([]byte(n.Payload), &data); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if data[key] != want {
			t.Fatalf("%s = %q, want %q", key, data[key], want)
		}
	}
	assertPayloadLink(models.NotificationTypePasswordReset, "reset_link", "https://app.example.test/reset-password?token=reset-tok")
	assertPayloadLink(models.NotificationTypeEmailVerification, "verify_link", "https://app.example.test/verify-email?token=verify-tok")
}
