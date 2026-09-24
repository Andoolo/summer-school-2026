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
		EntryID: entryID, SlotID: "99999999-9999-9999-9999-999999999999", ChatID: chat, SeatsWanted: 2, FreeSeats: 3, RouteName: "Спортивная трасса",
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
	d := newTestDispatcherWith(Config{Repo: repo, Bot: bot, Waitlist: wl, OfferTTL: 15 * time.Minute, AppURL: "https://apex.example/"})

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
	if len(bot.sent) != 1 || !strings.Contains(bot.sent[0], "Записаться: https://apex.example/#slot/99999999-9999-9999-9999-999999999999\n") {
		t.Fatalf("sent = %q, want link straight to the race", bot.sent)
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
		"в течение 15 минут", "до 15:15 (мск)", "не закреплено", "Записаться: https://apex.example/#slot/99999999-9999-9999-9999-999999999999", "/stop",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("offer text lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(OfferText(offer("e", 1), 15*time.Minute, ""), "Записаться:") {
		t.Error("no link expected without app URL")
	}
}

type countingObserver struct {
	counts map[string]int
	alerts []string
	window map[string]int
}

func (o *countingObserver) Inc(name string) { o.counts[name]++ }

func (o *countingObserver) Alert(key, text string) { o.alerts = append(o.alerts, key+"|"+text) }

func (o *countingObserver) CountAndAlert(key string, threshold int, _ time.Duration, text func(int) string) {
	o.window[key]++
	if o.window[key] >= threshold {
		o.Alert(key, text(o.window[key]))
	}
}

func TestDispatcherReportsToObserver(t *testing.T) {
	observer := &countingObserver{counts: map[string]int{}, window: map[string]int{}}
	repo := &fakeRepo{due: map[Kind][]Notice{KindReminder: {
		notice(KindReminder, "ok", 1), notice(KindReminder, "blocked", 2),
		notice(KindReminder, "f1", 3), notice(KindReminder, "f2", 4), notice(KindReminder, "f3", 5),
	}}}
	wl := &fakeWaitlist{offers: []Offer{offer("offer", 6)}}
	bot := &fakeBot{errs: map[int64]error{
		2: &telegram.APIError{Code: 403, Description: "Forbidden: bot was blocked by the user"},
		3: errors.New("network"), 4: errors.New("network"), 5: errors.New("network: last"),
	}}
	newTestDispatcherWith(Config{Repo: repo, Bot: bot, Waitlist: wl, OfferTTL: 15 * time.Minute, Observer: observer}).RunOnce(context.Background())

	if observer.counts["tg_sent"] != 2 || observer.counts["waitlist_offers"] != 1 || observer.counts["tg_blocked"] != 1 || observer.counts["tg_failed"] != 3 {
		t.Fatalf("counts = %v", observer.counts)
	}
	// Блокировка бота считается одинаково для уведомлений и предложений: как «заблокировали»,
	// но не как ошибка отправки.
	blockedOffer := &countingObserver{counts: map[string]int{}, window: map[string]int{}}
	wl2 := &fakeWaitlist{offers: []Offer{offer("blocked-offer", 2)}}
	newTestDispatcherWith(Config{Repo: &fakeRepo{due: map[Kind][]Notice{}}, Bot: bot, Waitlist: wl2, OfferTTL: 15 * time.Minute, Observer: blockedOffer}).RunOnce(context.Background())
	if blockedOffer.counts["tg_blocked"] != 1 || blockedOffer.counts["tg_failed"] != 0 {
		t.Fatalf("blocked offer counts = %v, want tg_blocked 1 and no tg_failed", blockedOffer.counts)
	}
	if len(observer.alerts) != 1 || !strings.Contains(observer.alerts[0], "Рассылка в Telegram сбоит: 3 ошибок") || strings.Contains(observer.alerts[0], "last") {
		t.Fatalf("alerts = %v", observer.alerts)
	}
}

func TestSlotLink(t *testing.T) {
	cases := []struct{ app, slot, want string }{
		{"https://apex.example", "99999999-9999-9999-9999-999999999999", "https://apex.example/#slot/99999999-9999-9999-9999-999999999999"},
		{"https://apex.example", "", "https://apex.example"},
		{"", "99999999-9999-9999-9999-999999999999", ""},
		// id всегда uuid, но экранируем на случай мусора: ссылка не должна ломаться.
		{"https://apex.example", "a b/c", "https://apex.example/#slot/a%20b%2Fc"},
	}
	for _, c := range cases {
		if got := SlotLink(c.app, c.slot); got != c.want {
			t.Errorf("SlotLink(%q, %q) = %q, want %q", c.app, c.slot, got, c.want)
		}
	}
}
