package notify

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"summer-school-2026/backend/internal/telegram"
)

type fakeRepo struct {
	due       map[Kind][]Notice
	onClaim   func()
	claims    int
	unclaimed []string
	disabled  []int64
}

func (r *fakeRepo) ClaimDue(_ context.Context, kind Kind, _ time.Time, limit int) ([]Notice, error) {
	r.claims++
	if r.onClaim != nil {
		r.onClaim()
	}
	batch := r.due[kind]
	if len(batch) > limit {
		batch = batch[:limit]
	}
	r.due[kind] = r.due[kind][len(batch):]
	return batch, nil
}

func (r *fakeRepo) Unclaim(_ context.Context, kind Kind, bookingID string) error {
	r.unclaimed = append(r.unclaimed, string(kind)+":"+bookingID)
	return nil
}

func (r *fakeRepo) DisableChat(_ context.Context, chatID int64) error {
	r.disabled = append(r.disabled, chatID)
	return nil
}

type fakeBot struct {
	errs    map[int64]error
	sent    []string
	markups []any
}

func (b *fakeBot) SendMessage(_ context.Context, chatID int64, text string, markup any) error {
	if err := b.errs[chatID]; err != nil {
		return err
	}
	b.sent = append(b.sent, text)
	b.markups = append(b.markups, markup)
	return nil
}

var testNow = time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)

func notice(kind Kind, id string, chat int64) Notice {
	return Notice{
		Kind: kind, BookingID: id, ChatID: chat, Status: "active", SeatsCount: 2, RentalCount: 1, PriceTotal: 5800,
		CreatedAt: testNow.Add(-24 * time.Hour), RouteName: "Городское кольцо", InstructorName: "Марат",
		StartAt: testNow.Add(90 * time.Minute), MeetingPoint: "Паддок, вход 2",
	}
}

func newTestDispatcher(repo *fakeRepo, bot *fakeBot) *Dispatcher {
	d := NewDispatcher(repo, bot, nil)
	d.now = func() time.Time { return testNow }
	return d
}

func TestRunOnceSendsAllKinds(t *testing.T) {
	repo := &fakeRepo{due: map[Kind][]Notice{
		KindConfirm:  {notice(KindConfirm, "b1", 1)},
		KindCancel:   {notice(KindCancel, "b2", 2)},
		KindReminder: {notice(KindReminder, "b3", 3)},
	}}
	bot := &fakeBot{}
	if sent := newTestDispatcher(repo, bot).RunOnce(context.Background()); sent != 3 {
		t.Fatalf("sent = %d, want 3", sent)
	}
	if len(repo.unclaimed) != 0 || len(repo.disabled) != 0 {
		t.Fatalf("unexpected unclaim %v / disable %v", repo.unclaimed, repo.disabled)
	}
}

func TestRunOnceDrainsFullBatches(t *testing.T) {
	var many []Notice
	for i := 0; i < batchLimit+7; i++ {
		many = append(many, notice(KindConfirm, "b", int64(i+1)))
	}
	repo := &fakeRepo{due: map[Kind][]Notice{KindConfirm: many}}
	bot := &fakeBot{}
	if sent := newTestDispatcher(repo, bot).RunOnce(context.Background()); sent != batchLimit+7 {
		t.Fatalf("sent = %d, want %d", sent, batchLimit+7)
	}
}

func TestDeliveryFailures(t *testing.T) {
	repo := &fakeRepo{due: map[Kind][]Notice{KindReminder: {
		notice(KindReminder, "blocked", 1),
		notice(KindReminder, "nochat", 2),
		notice(KindReminder, "flaky", 3),
		notice(KindReminder, "ok", 4),
	}}}
	bot := &fakeBot{errs: map[int64]error{
		1: &telegram.APIError{Method: "sendMessage", Code: 403, Description: "Forbidden: bot was blocked by the user"},
		2: &telegram.APIError{Method: "sendMessage", Code: 400, Description: "Bad Request: chat not found"},
		3: errors.New("telegram sendMessage: request failed (network or timeout)"),
	}}
	sent := newTestDispatcher(repo, bot).RunOnce(context.Background())

	if sent != 1 {
		t.Fatalf("sent = %d, want 1", sent)
	}
	// Заблокировавший бота чат отвязан, повтора нет.
	if len(repo.disabled) != 1 || repo.disabled[0] != 1 {
		t.Fatalf("disabled = %v, want [1]", repo.disabled)
	}
	// Временный сбой возвращается в очередь; «чат не найден» — нет.
	if len(repo.unclaimed) != 1 || repo.unclaimed[0] != "reminder:flaky" {
		t.Fatalf("unclaimed = %v, want [reminder:flaky]", repo.unclaimed)
	}
}

func TestWakeDoesNotBlock(t *testing.T) {
	d := newTestDispatcher(&fakeRepo{due: map[Kind][]Notice{}}, &fakeBot{})
	done := make(chan struct{})
	go func() {
		for i := 0; i < 10; i++ {
			d.Wake()
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Wake() blocked without a running dispatcher")
	}
}

func TestRunStopsOnContextCancel(t *testing.T) {
	claimed := make(chan struct{}, 16)
	repo := &fakeRepo{due: map[Kind][]Notice{}, onClaim: func() { claimed <- struct{}{} }}
	d := newTestDispatcher(repo, &fakeBot{})
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})
	go func() {
		d.Run(ctx)
		close(finished)
	}()
	// Первый проход — сразу при запуске, не через минуту.
	select {
	case <-claimed:
	case <-time.After(2 * time.Second):
		t.Fatal("Run() must make a pass right away")
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not stop after cancel")
	}
}

func TestTexts(t *testing.T) {
	confirm := Text(notice(KindConfirm, "b", 1), testNow)
	for _, want := range []string{
		"Бронь подтверждена", "Городское кольцо", "пятница, 18 сентября, 16:30 (мск)",
		"Паддок, вход 2", "Маршал: Марат", "Мест: 2, с экипировкой: 1", "5\u00a0800", "Напомню", "/stop",
	} {
		if !strings.Contains(confirm, want) {
			t.Errorf("confirm text lacks %q:\n%s", want, confirm)
		}
	}

	late := notice(KindConfirm, "b", 1)
	late.CreatedAt = late.StartAt.Add(-time.Hour)
	if strings.Contains(Text(late, testNow), "Напомню") {
		t.Error("booking made an hour before start must not promise a reminder")
	}

	if reminder := Text(notice(KindReminder, "b", 1), testNow); !strings.Contains(reminder, "Заезд через 1 ч 30 мин") {
		t.Errorf("reminder text:\n%s", reminder)
	}

	cancel := notice(KindCancel, "b", 1)
	cancel.Status = "cancelled"
	if text := Text(cancel, testNow); !strings.Contains(text, "Бронь отменена") || strings.Contains(text, "Поздняя") {
		t.Errorf("cancel text:\n%s", text)
	}
	cancel.Status = "late_cancel"
	if text := Text(cancel, testNow); !strings.Contains(text, "Поздняя отмена") {
		t.Errorf("late cancel text:\n%s", text)
	}
}

func TestUntilStartAndRubles(t *testing.T) {
	cases := map[time.Duration]string{
		30 * time.Second: "вот-вот начнётся",
		40 * time.Minute: "через 40 мин",
		2 * time.Hour:    "через 2 ч",
		time.Hour + 59*time.Minute + 50*time.Second: "через 2 ч",
		time.Hour + 5*time.Minute:                   "через 1 ч 5 мин",
	}
	for left, want := range cases {
		if got := untilStart(left); got != want {
			t.Errorf("untilStart(%s) = %q, want %q", left, got, want)
		}
	}
	for amount, want := range map[int]string{900: "900", 5800: "5\u00a0800", 1250000: "1\u00a0250\u00a0000"} {
		if got := rubles(amount); !strings.HasPrefix(got, want+"\u00a0") && !strings.HasPrefix(got, want+" ") {
			t.Errorf("rubles(%d) = %q, want prefix %q", amount, got, want)
		}
	}
}

func TestCancelButtonOnlyUnderConfirmAndReminder(t *testing.T) {
	for _, kind := range []Kind{KindConfirm, KindReminder} {
		keyboard, ok := Markup(notice(kind, "11111111-1111-1111-1111-111111111111", 1)).(telegram.InlineKeyboard)
		if !ok || len(keyboard.InlineKeyboard) != 1 || !strings.Contains(keyboard.InlineKeyboard[0][0].CallbackData, "11111111-1111-1111-1111-111111111111") {
			t.Fatalf("%s markup = %#v, want cancel button", kind, keyboard)
		}
	}
	cancelled := notice(KindCancel, "b", 1)
	cancelled.Status = "cancelled"
	if Markup(cancelled) != nil {
		t.Fatal("cancel notice must not have buttons")
	}
	// Подтверждение, пришедшее уже после отмены, кнопку не получает.
	stale := notice(KindConfirm, "b", 1)
	stale.Status = "cancelled"
	if Markup(stale) != nil {
		t.Fatal("inactive booking must not get a cancel button")
	}

	repo := &fakeRepo{due: map[Kind][]Notice{KindConfirm: {notice(KindConfirm, "11111111-1111-1111-1111-111111111111", 1)}}}
	bot := &fakeBot{}
	newTestDispatcher(repo, bot).RunOnce(context.Background())
	if len(bot.markups) != 1 || bot.markups[0] == nil {
		t.Fatalf("dispatcher must send the button, markups = %#v", bot.markups)
	}
}
