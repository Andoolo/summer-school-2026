package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeDemoRepo struct {
	active        int
	takenPhones   map[string]bool
	created       []string
	accessExpiry  time.Time
	refreshExpiry time.Time
	sessions      int
}

func (r *fakeDemoRepo) CountActiveDemoClients(context.Context, time.Time) (int, error) {
	return r.active, nil
}

func (r *fakeDemoRepo) CreateDemoClient(_ context.Context, phone, name string, now, _ time.Time) (Client, error) {
	if r.takenPhones[phone] {
		return Client{}, ErrDemoPhoneTaken
	}
	r.created = append(r.created, phone)
	return Client{ID: "11111111-1111-1111-1111-111111111111", Name: &name, Phone: phone, CreatedAt: now}, nil
}

func (r *fakeDemoRepo) IssueSession(_ context.Context, _, _, _ string, accessExpiresAt, refreshExpiresAt time.Time) error {
	r.sessions++
	r.accessExpiry = accessExpiresAt
	r.refreshExpiry = refreshExpiresAt
	return nil
}

func TestIsDemoPhone(t *testing.T) {
	if !IsDemoPhone("+70001234567") {
		t.Fatal("+7000… must be a demo phone")
	}
	if IsDemoPhone("+79991234567") || IsDemoPhone("+17000123456") {
		t.Fatal("regular numbers must not be demo phones")
	}
}

func TestRandomDemoPhoneMatchesContract(t *testing.T) {
	for i := 0; i < 50; i++ {
		phone, err := randomDemoPhone()
		if err != nil {
			t.Fatalf("randomDemoPhone() error = %v", err)
		}
		if !IsDemoPhone(phone) || !phonePattern.MatchString(phone) || len(phone) != 12 {
			t.Fatalf("phone %q must be a valid E.164 demo number of 11 digits", phone)
		}
	}
}

func TestDemoLoginCreatesGuestWithSessionCappedByGuestLifetime(t *testing.T) {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	repo := &fakeDemoRepo{}
	service := NewDemoService(repo, DemoConfig{TTL: 6 * time.Hour, MaxActive: 10})
	service.now = func() time.Time { return now }

	result, err := service.Login(context.Background())
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.Token == "" || result.RefreshToken == "" || !result.IsNew {
		t.Fatalf("result = %+v, want tokens and IsNew", result)
	}
	if result.Client.Name == nil || *result.Client.Name != "Гость" {
		t.Fatalf("guest name = %v, want Гость", result.Client.Name)
	}
	wantExpiry := now.Add(6 * time.Hour)
	if !repo.accessExpiry.Equal(wantExpiry) || !repo.refreshExpiry.Equal(wantExpiry) {
		t.Fatalf("session expiry access=%v refresh=%v, want both %v", repo.accessExpiry, repo.refreshExpiry, wantExpiry)
	}
	if result.AccessTTLSeconds != int((6 * time.Hour).Seconds()) {
		t.Fatalf("AccessTTLSeconds = %d", result.AccessTTLSeconds)
	}
}

func TestDemoLoginRejectsWhenCapacityReached(t *testing.T) {
	repo := &fakeDemoRepo{active: 10}
	service := NewDemoService(repo, DemoConfig{TTL: time.Hour, MaxActive: 10})

	if _, err := service.Login(context.Background()); !errors.Is(err, ErrDemoUnavailable) {
		t.Fatalf("Login() error = %v, want %v", err, ErrDemoUnavailable)
	}
	if len(repo.created) != 0 || repo.sessions != 0 {
		t.Fatal("nothing must be created when capacity is reached")
	}
}

func TestDemoLoginRetriesPhoneCollision(t *testing.T) {
	repo := &fakeDemoRepo{takenPhones: map[string]bool{"+70000000001": true}}
	service := NewDemoService(repo, DefaultDemoConfig())
	phones := []string{"+70000000001", "+70000000002"}
	service.randomPhone = func() (string, error) {
		phone := phones[0]
		phones = phones[1:]
		return phone, nil
	}

	result, err := service.Login(context.Background())
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.Client.Phone != "+70000000002" {
		t.Fatalf("phone = %q, want retry with the next number", result.Client.Phone)
	}
}

func TestDemoPhonesCannotLoginWithCode(t *testing.T) {
	service := NewService(&fakeRepo{}, nil)
	if _, err := service.RequestCode(context.Background(), "+70001234567"); !errors.Is(err, ErrInvalidPhone) {
		t.Fatalf("RequestCode(demo) error = %v, want %v", err, ErrInvalidPhone)
	}
	if _, err := service.VerifyCode(context.Background(), "+70001234567", "1234"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("VerifyCode(demo) error = %v, want %v", err, ErrInvalidCode)
	}
}
