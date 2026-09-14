package notify

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"summer-school-2026/backend/internal/telegram"
)

type fakeWaitlist struct {
	offers   []Offer
	released []string
	expired  []string
}

func (w *fakeWaitlist) ClaimOffers(context.Context, time.Time, time.Duration) ([]Offer, error) {
	offers := w.offers
	w.offers = nil
	return offers, nil
}

func (w *fakeWaitlist) ReleaseOffer(_ context.Context, entryID string) error {
	w.released = append(w.released, entryID)
	return nil
}

func (w *fakeWaitlist) ExpireOffer(_ context.Context, entryID string, _ time.Time) error {
	w.expired = append(w.expired, entryID)
	return nil
}

func offer(entryID string, chat int64) Offer {
	return Offer{
		EntryID: entryID, ChatID: chat, SeatsWanted: 2, FreeSeats: 3, RouteName: "Спортивная трасса",
		InstructorName: "Виктор", StartAt: testNow.Add(5 * time.Hour), MeetingPoint: "Боксы",
		ExpiresAt: testNow.Add(15 * time.Minute),
	}
}

func TestWaitlistOffersAreSentAndFailuresHandled(t *testing.T) {
	repo := &fakeRepo{due: map[Kind][]Notice{}}
	wl := &fakeWaitlist{offers: []Offer{offer("ok", 1), offer("blocked", 2), offer("gone", 3), offer("flaky", 4)}}
	bot := &fakeBot{errs: map[int64]error{
		2: &telegram.APIError{Code: 403, Description: "Forbidden: bot was blocked by the user"},
		3: &telegram.APIError{Code: 400, Description: "Bad Request: chat not found"},
		4: errors.New("network"),
	}}
	d := newTestDispatcher(repo, bot).WithWaitlist(wl, 15*time.Minute, "https://apex.example/")

	if sent := d.RunOnce(context.Background()); sent != 1 {
		t.Fatalf("sent = %d, want 1", sent)
	}
	if len(repo.disabled) != 1 || repo.disabled[0] != 2 {
		t.Fatalf("disabled chats = %v, want [2]", repo.disabled)
	}
	// Недоставляемые предложения закрываются — место уйдёт следующему.
	if strings.Join(wl.expired, ",") != "blocked,gone" {
		t.Fatalf("expired = %v", wl.expired)
	}
	// Временный сбой — запись возвращается в очередь.
	if strings.Join(wl.released, ",") != "flaky" {
		t.Fatalf("released = %v", wl.released)
	}
	if len(bot.sent) != 1 || !strings.Contains(bot.sent[0], "https://apex.example\n") {
		t.Fatalf("sent = %q, want app link without trailing slash", bot.sent)
	}
}

func TestDispatcherWithoutWaitlistSkipsOffers(t *testing.T) {
	d := newTestDispatcher(&fakeRepo{due: map[Kind][]Notice{}}, &fakeBot{})
	if sent := d.RunOnce(context.Background()); sent != 0 {
		t.Fatalf("sent = %d", sent)
	}
}

func TestOfferText(t *testing.T) {
	text := OfferText(offer("e", 1), 15*time.Minute, "https://apex.example")
	for _, want := range []string{
		"Освободилось место", "Спортивная трасса", "Свободно мест: 3, вы ждали: 2",
		"в течение 15 минут", "до 15:15 (мск)", "не закреплено", "Записаться: https://apex.example", "/stop",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("offer text lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(OfferText(offer("e", 1), 15*time.Minute, ""), "Записаться:") {
		t.Error("no link expected without app URL")
	}
}
