package ops

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type memoryStore struct {
	mu      sync.Mutex
	counts  map[time.Time]map[string]int64
	failing bool
	state   map[string]string
}

func (m *memoryStore) AddCounters(_ context.Context, hour time.Time, counts map[string]int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failing {
		return errors.New("db down")
	}
	if m.counts == nil {
		m.counts = map[time.Time]map[string]int64{}
	}
	if m.counts[hour] == nil {
		m.counts[hour] = map[string]int64{}
	}
	for name, value := range counts {
		m.counts[hour][name] += value
	}
	return nil
}

func (m *memoryStore) SwapState(_ context.Context, key, value string) (string, error) {
	if m.state == nil {
		m.state = map[string]string{}
	}
	previous := m.state[key]
	m.state[key] = value
	return previous, nil
}

type sentAlert struct {
	chat int64
	text string
}

type recordingBot struct {
	mu   sync.Mutex
	sent []sentAlert
}

func (b *recordingBot) SendMessage(_ context.Context, chatID int64, text string, _ any) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sent = append(b.sent, sentAlert{chatID, text})
	return nil
}

var base = time.Date(2026, 9, 14, 12, 30, 0, 0, time.UTC)

func newTestRecorder(store *memoryStore, bot *recordingBot, admin int64) (*Recorder, *time.Time) {
	clock := base
	r := NewRecorder(store, bot, admin, nil)
	r.now = func() time.Time { return clock }
	return r, &clock
}

func TestCountersAreBucketedByHourAndFlushed(t *testing.T) {
	store := &memoryStore{}
	r, clock := newTestRecorder(store, &recordingBot{}, 0)
	r.Inc(HTTPRequests)
	r.Inc(HTTPRequests)
	*clock = base.Add(40 * time.Minute) // следующий час
	r.Inc(HTTPRequests)

	if err := r.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := store.counts[base.Truncate(time.Hour)][HTTPRequests]; got != 2 {
		t.Fatalf("first hour = %d, want 2", got)
	}
	if got := store.counts[base.Add(40*time.Minute).Truncate(time.Hour)][HTTPRequests]; got != 1 {
		t.Fatalf("second hour = %d, want 1", got)
	}
	// Повторный сброс ничего не удваивает.
	_ = r.Flush(context.Background())
	if got := store.counts[base.Truncate(time.Hour)][HTTPRequests]; got != 2 {
		t.Fatalf("after second flush = %d, want 2", got)
	}
}

func TestFailedFlushKeepsCounts(t *testing.T) {
	store := &memoryStore{failing: true}
	r, _ := newTestRecorder(store, &recordingBot{}, 0)
	r.Inc(TelegramSent)
	if err := r.Flush(context.Background()); err == nil {
		t.Fatal("flush must report store failure")
	}
	store.failing = false
	r.Inc(TelegramSent)
	if err := r.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := store.counts[base.Truncate(time.Hour)][TelegramSent]; got != 2 {
		t.Fatalf("counts after retry = %d, want 2 (nothing lost)", got)
	}
}

func TestAlertCooldownPerKey(t *testing.T) {
	bot := &recordingBot{}
	r, clock := newTestRecorder(&memoryStore{}, bot, 42)
	r.Alert("db", "первый")
	r.Alert("db", "второй — подавлен")
	r.Alert("panic", "другой ключ")
	*clock = base.Add(31 * time.Minute)
	r.Alert("db", "после паузы")
	r.waitAlerts()

	if len(bot.sent) != 3 {
		t.Fatalf("sent %d alerts, want 3: %+v", len(bot.sent), bot.sent)
	}
	for _, alert := range bot.sent {
		if alert.chat != 42 || strings.Contains(alert.text, "подавлен") || !strings.Contains(alert.text, "не раньше чем через 30 мин") {
			t.Fatalf("unexpected alert %+v", alert)
		}
	}
}

func TestCountAndAlertThresholdWindow(t *testing.T) {
	bot := &recordingBot{}
	r, clock := newTestRecorder(&memoryStore{}, bot, 42)
	text := func(n int) string { return "ошибок: " + itoa(n) }

	r.CountAndAlert("http_5xx", 3, 5*time.Minute, text)
	r.CountAndAlert("http_5xx", 3, 5*time.Minute, text)
	*clock = base.Add(6 * time.Minute) // первые два вышли из окна
	r.CountAndAlert("http_5xx", 3, 5*time.Minute, text)
	r.waitAlerts()
	if len(bot.sent) != 0 {
		t.Fatalf("alert before threshold: %+v", bot.sent)
	}
	r.CountAndAlert("http_5xx", 3, 5*time.Minute, text)
	r.CountAndAlert("http_5xx", 3, 5*time.Minute, text)
	r.waitAlerts()
	if len(bot.sent) != 1 || !strings.HasPrefix(bot.sent[0].text, "ошибок: 3") {
		t.Fatalf("alerts = %+v, want one with count 3", bot.sent)
	}
}

func TestNoAdminOrNilRecorderIsSafe(t *testing.T) {
	bot := &recordingBot{}
	r, _ := newTestRecorder(&memoryStore{}, bot, 0)
	r.Alert("x", "только в журнал")
	r.waitAlerts()
	if len(bot.sent) != 0 {
		t.Fatal("without admin chat alerts must not be sent")
	}

	var nilRecorder *Recorder
	nilRecorder.Inc("x")
	nilRecorder.Alert("x", "y")
	nilRecorder.CountAndAlert("x", 1, time.Minute, func(int) string { return "" })
	if err := nilRecorder.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestAlertTextIsTruncated(t *testing.T) {
	bot := &recordingBot{}
	r, _ := newTestRecorder(&memoryStore{}, bot, 42)
	r.Alert("long", strings.Repeat("я", 5000))
	r.waitAlerts()
	if runes := len([]rune(bot.sent[0].text)); runes > alertTextMaxRunes+100 {
		t.Fatalf("alert has %d runes, Telegram limit is 4096 and alerts must stay short", runes)
	}
}

func TestAnnounceVersionOncePerVersion(t *testing.T) {
	store, bot := &memoryStore{}, &recordingBot{}
	r, _ := newTestRecorder(store, bot, 42)
	ctx := context.Background()

	for i := 0; i < 3; i++ { // три пробуждения Render с одной версией
		if err := AnnounceVersion(ctx, store, r, "abcdef1234567", base); err != nil {
			t.Fatal(err)
		}
	}
	_ = AnnounceVersion(ctx, store, r, "", base) // версия неизвестна — молчим
	_ = AnnounceVersion(ctx, store, r, "0123456789abc", base)
	r.waitAlerts()

	if len(bot.sent) != 2 {
		t.Fatalf("announcements = %d, want 2: %+v", len(bot.sent), bot.sent)
	}
	// Алерты уходят в фоне — порядок не гарантирован, ищем по содержимому.
	var first, second string
	for _, alert := range bot.sent {
		if strings.Contains(alert.text, "Предыдущая версия") {
			second = alert.text
		} else {
			first = alert.text
		}
	}
	if !strings.Contains(first, "версия abcdef1") || !strings.Contains(first, "15:30 (мск)") {
		t.Fatalf("first announcement = %q", first)
	}
	if !strings.Contains(second, "версия 0123456") || !strings.Contains(second, "Предыдущая версия: abcdef1") {
		t.Fatalf("second announcement = %q", second)
	}
}

func TestStatsText(t *testing.T) {
	text := StatsText(Snapshot{
		BookingsCreated24h: 3, BookingsCreated7d: 12, Cancelled24h: 1, Cancelled7d: 4, LateCancelled7d: 1,
		UpcomingRaces: 10, UpcomingSeats: 80, UpcomingTaken: 34, UpcomingFull: 2,
		WaitingNow: 3, Offers24h: 1, BookedFromWaitlist24h: 1,
		Clients: 41, TelegramClients: 12, NewClients7d: 5, ActiveGuests: 2,
		Counters24h: map[string]int64{TelegramSent: 15, HTTPRequests: 900, HTTPServerErrors: 2, BotCancellations: 1},
	}, "166b5cc0123", 135*time.Minute)

	for _, want := range []string{
		"Версия 166b5cc", "работает 2 ч 15 мин", "Создано: 3 за сутки · 12 за неделю",
		"Отменено: 1 за сутки · 4 за неделю (поздних 1)", "Из бота отменено за сутки: 1",
		"Занято мест: 34 из 80 (42%)", "В очереди сейчас: 3", "с Telegram: 12",
		"Отправлено: 15 · ошибок: 0", "Запросов: 900 · ошибок 5xx: 2",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("stats lack %q:\n%s", want, text)
		}
	}
	if !strings.Contains(StatsText(Snapshot{Counters24h: map[string]int64{}}, "", 0), "Занято мест: 0 из 0 (—)") {
		t.Error("empty week must not divide by zero")
	}
	if len([]rune(text)) > 4096 {
		t.Fatalf("stats are %d runes, Telegram limit is 4096", len([]rune(text)))
	}
}

func TestDurationText(t *testing.T) {
	cases := map[time.Duration]string{3 * time.Minute: "3 мин", 135 * time.Minute: "2 ч 15 мин", 28 * time.Hour: "1 дн 4 ч"}
	for d, want := range cases {
		if got := durationText(d); got != want {
			t.Errorf("durationText(%s) = %q, want %q", d, got, want)
		}
	}
}
