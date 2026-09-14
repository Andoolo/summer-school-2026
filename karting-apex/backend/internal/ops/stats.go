package ops

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Snapshot — всё, что показывает /stats. Числа из базы переживают сон Render.
type Snapshot struct {
	BookingsCreated24h int
	BookingsCreated7d  int
	Cancelled24h       int
	Cancelled7d        int
	LateCancelled7d    int

	UpcomingRaces int
	UpcomingSeats int
	UpcomingTaken int
	UpcomingFull  int

	WaitingNow            int
	Offers24h             int
	BookedFromWaitlist24h int
	ExpiredOffers24h      int

	Clients         int
	TelegramClients int
	NewClients7d    int
	ActiveGuests    int

	Counters24h map[string]int64
}

type SnapshotSource interface {
	Snapshot(ctx context.Context, now time.Time) (Snapshot, error)
}

// Stats собирает сводку для администратора.
type Stats struct {
	source    SnapshotSource
	recorder  *Recorder
	version   string
	startedAt time.Time
	now       func() time.Time
}

func NewStats(source SnapshotSource, recorder *Recorder, version string) *Stats {
	return &Stats{source: source, recorder: recorder, version: version, startedAt: time.Now(), now: time.Now}
}

func (s *Stats) Text(ctx context.Context) (string, error) {
	// Сначала сбрасываем счётчики из памяти, иначе сводка отстанет на полминуты.
	if err := s.recorder.Flush(ctx); err != nil {
		return "", err
	}
	now := s.now().UTC()
	snapshot, err := s.source.Snapshot(ctx, now)
	if err != nil {
		return "", err
	}
	return StatsText(snapshot, s.version, now.Sub(s.startedAt)), nil
}

// StatsText — сводка простым текстом (без разметки: так её не сломают спецсимволы).
func StatsText(s Snapshot, version string, uptime time.Duration) string {
	var b strings.Builder
	b.WriteString("📊 Апекс — сводка\n")
	if version != "" {
		b.WriteString("Версия " + shortVersion(version) + " · ")
	}
	b.WriteString("работает " + durationText(uptime) + " (после сна Render отсчёт с нуля)\n\n")

	b.WriteString("🎟 Брони\n")
	fmt.Fprintf(&b, "Создано: %d за сутки · %d за неделю\n", s.BookingsCreated24h, s.BookingsCreated7d)
	fmt.Fprintf(&b, "Отменено: %d за сутки · %d за неделю (поздних %d)\n", s.Cancelled24h, s.Cancelled7d, s.LateCancelled7d)
	fmt.Fprintf(&b, "Из бота отменено за сутки: %d\n\n", s.Counters24h[BotCancellations])

	b.WriteString("🏁 Заезды на 7 дней\n")
	fmt.Fprintf(&b, "Заездов: %d, полных: %d\n", s.UpcomingRaces, s.UpcomingFull)
	fmt.Fprintf(&b, "Занято мест: %d из %d (%s)\n\n", s.UpcomingTaken, s.UpcomingSeats, percent(s.UpcomingTaken, s.UpcomingSeats))

	b.WriteString("⏳ Лист ожидания\n")
	fmt.Fprintf(&b, "В очереди сейчас: %d\n", s.WaitingNow)
	fmt.Fprintf(&b, "За сутки: предложений %d · записались %d · истекло %d\n\n", s.Offers24h, s.BookedFromWaitlist24h, s.ExpiredOffers24h)

	b.WriteString("👤 Клиенты\n")
	fmt.Fprintf(&b, "Всего: %d · с Telegram: %d · новых за неделю: %d\n", s.Clients, s.TelegramClients, s.NewClients7d)
	fmt.Fprintf(&b, "Гостей (демо) сейчас: %d\n\n", s.ActiveGuests)

	b.WriteString("🤖 Telegram за сутки\n")
	fmt.Fprintf(&b, "Отправлено: %d · ошибок: %d · заблокировали бота: %d\n\n",
		s.Counters24h[TelegramSent], s.Counters24h[TelegramFailed], s.Counters24h[TelegramBlocked])

	b.WriteString("🌐 API за сутки\n")
	fmt.Fprintf(&b, "Запросов: %d · ошибок 5xx: %d · упёрлись в лимит (429): %d · паник: %d\n",
		s.Counters24h[HTTPRequests], s.Counters24h[HTTPServerErrors], s.Counters24h[HTTPRateLimited], s.Counters24h[Panics])
	b.WriteString("Счётчики API и Telegram считаются, только пока сервис не спит.")
	return b.String()
}

func shortVersion(version string) string {
	if len(version) > 7 {
		return version[:7]
	}
	return version
}

func percent(part, total int) string {
	if total == 0 {
		return "—"
	}
	return strconv.Itoa(part*100/total) + "%"
}

// durationText — «3 мин», «2 ч 15 мин», «1 дн 4 ч».
func durationText(d time.Duration) string {
	minutes := int(d / time.Minute)
	switch {
	case minutes < 60:
		return itoa(minutes) + " мин"
	case minutes < 24*60:
		return itoa(minutes/60) + " ч " + itoa(minutes%60) + " мин"
	default:
		return itoa(minutes/(24*60)) + " дн " + itoa(minutes/60%24) + " ч"
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
