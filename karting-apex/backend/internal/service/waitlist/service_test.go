package waitlist

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRepo struct {
	client  Client
	found   bool
	joins   int
	mineFor string
}

func (r *fakeRepo) ClientBySessionTokenHash(context.Context, string) (Client, bool, error) {
	return r.client, r.found, nil
}

func (r *fakeRepo) Join(_ context.Context, _, slotID string, seats int, now time.Time) (Entry, bool, error) {
	r.joins++
	return Entry{ID: "e", SlotID: slotID, SeatsCount: seats, Status: EntryWaiting, Position: 1, CreatedAt: now}, true, nil
}

func (r *fakeRepo) Leave(context.Context, string, string, time.Time) error { return nil }

func (r *fakeRepo) ActiveEntry(context.Context, string, string) (Entry, bool, error) {
	return Entry{}, false, nil
}

func (r *fakeRepo) ActiveEntriesForClient(_ context.Context, clientID string, _ time.Time) ([]MyEntry, error) {
	r.mineFor = clientID
	return []MyEntry{{RouteName: "Городское кольцо"}}, nil
}

func TestJoinChecksRequestAndTelegram(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name   string
		repo   *fakeRepo
		token  string
		seats  int
		want   error
		joined bool
	}{
		{"no token", &fakeRepo{found: true}, "", 1, ErrUnauthorized, false},
		{"unknown session", &fakeRepo{found: false}, "t", 1, ErrUnauthorized, false},
		{"zero seats", &fakeRepo{found: true}, "t", 0, ErrInvalidRequest, false},
		{"four seats", &fakeRepo{found: true}, "t", 4, ErrInvalidRequest, false},
		{"no telegram", &fakeRepo{found: true, client: Client{ID: "c"}}, "t", 1, ErrTelegramRequired, false},
		{"muted", &fakeRepo{found: true, client: Client{ID: "c", TelegramLinked: true}}, "t", 1, ErrNotificationsDisabled, false},
		{"ok", &fakeRepo{found: true, client: Client{ID: "c", TelegramLinked: true, NotificationsEnabled: true}}, "t", 3, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			signals := 0
			_, _, err := NewService(tc.repo, func() { signals++ }).Join(ctx, tc.token, "slot", tc.seats)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Join() error = %v, want %v", err, tc.want)
			}
			if (tc.repo.joins == 1) != tc.joined {
				t.Fatalf("repository joins = %d, joined want %v", tc.repo.joins, tc.joined)
			}
			// Сигнал рассыльщику — только после новой записи в очередь.
			if (signals == 1) != tc.joined {
				t.Fatalf("dispatcher signals = %d, want %v", signals, tc.joined)
			}
		})
	}
}

func TestOfferExpiresAt(t *testing.T) {
	notifiedAt := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	offered := Entry{Status: EntryNotified, NotifiedAt: &notifiedAt}
	if got := offered.OfferExpiresAt(); got == nil || !got.Equal(notifiedAt.Add(OfferTTL)) {
		t.Fatalf("OfferExpiresAt() = %v", got)
	}
	if (Entry{Status: EntryWaiting}).OfferExpiresAt() != nil {
		t.Fatal("waiting entry has no offer")
	}
}

func TestMineRequiresSessionAndUsesClientID(t *testing.T) {
	ctx := context.Background()
	if _, err := NewService(&fakeRepo{found: false}, nil).Mine(ctx, "t"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Mine() without session error = %v", err)
	}
	if _, err := NewService(&fakeRepo{found: true}, nil).Mine(ctx, ""); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Mine() without token error = %v", err)
	}
	// Не нужен Telegram: посмотреть и покинуть очереди можно, даже если уведомления выключены.
	repo := &fakeRepo{found: true, client: Client{ID: "client-7"}}
	entries, err := NewService(repo, nil).Mine(ctx, "t")
	if err != nil || len(entries) != 1 || repo.mineFor != "client-7" {
		t.Fatalf("Mine() = %v, %v; repo asked for %q", entries, err, repo.mineFor)
	}
}
