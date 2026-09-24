// Package ops — наблюдаемость прода без внешних сервисов: счётчики событий, алерты
// администратору в Telegram и сводка по команде /stats.
//
// Внешнего мониторинга нет сознательно: частые проверки снаружи не дали бы бесплатному
// Render засыпать. Цена — полное падение сервиса отсюда не видно: упавший сервис сам о
// себе не сообщит.
package ops

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

// Имена счётчиков.
const (
	HTTPRequests      = "http_requests"
	HTTPServerErrors  = "http_5xx"
	HTTPRateLimited   = "http_429"
	TelegramSent      = "tg_sent"
	TelegramFailed    = "tg_failed"
	TelegramBlocked   = "tg_blocked"
	WaitlistOffers    = "waitlist_offers"
	BotCancellations  = "bot_cancellations"
	Panics            = "panics"
	flushInterval     = 30 * time.Second
	defaultCooldown   = 30 * time.Minute
	alertSendTimeout  = 10 * time.Second
	alertTextMaxRunes = 700
)

// Observer — счётчики и алерты, как их видят HTTP-слой и рассыльщик. Реализация — *Recorder,
// выключенное наблюдение — Discard.
type Observer interface {
	Inc(name string)
	Alert(key, text string)
	CountAndAlert(key string, threshold int, window time.Duration, text func(count int) string)
}

// Discard — наблюдение выключено: события никуда не пишутся.
var Discard Observer = discard{}

type discard struct{}

func (discard) Inc(string)                                                 {}
func (discard) Alert(string, string)                                       {}
func (discard) CountAndAlert(string, int, time.Duration, func(int) string) {}

type Store interface {
	// AddCounters прибавляет значения к счётчикам часа hour.
	AddCounters(ctx context.Context, hour time.Time, counts map[string]int64) error
}

type Sender interface {
	SendMessage(ctx context.Context, chatID int64, text string, markup any) error
}

// Recorder копит счётчики в памяти и сбрасывает их в базу, а также шлёт алерты. Методы
// безопасны для nil: там, где наблюдаемость не подключена, вызовы просто ничего не делают.
type Recorder struct {
	store     Store
	bot       Sender
	adminChat int64
	logger    *slog.Logger
	now       func() time.Time
	cooldown  time.Duration

	mu        sync.Mutex
	pending   map[time.Time]map[string]int64
	lastAlert map[string]time.Time
	windows   map[string][]time.Time
	sending   sync.WaitGroup
}

// NewRecorder: adminChat == 0 — алерты только в журнал (администратор не задан).
func NewRecorder(store Store, bot Sender, adminChat int64, logger *slog.Logger) *Recorder {
	if logger == nil {
		logger = slog.Default()
	}
	return &Recorder{
		store: store, bot: bot, adminChat: adminChat, logger: logger, now: time.Now, cooldown: defaultCooldown,
		pending: map[time.Time]map[string]int64{}, lastAlert: map[string]time.Time{}, windows: map[string][]time.Time{},
	}
}

func (r *Recorder) Inc(name string) {
	if r == nil {
		return
	}
	hour := r.now().UTC().Truncate(time.Hour)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pending[hour] == nil {
		r.pending[hour] = map[string]int64{}
	}
	r.pending[hour][name]++
}

// Alert шлёт администратору сообщение. Один и тот же key — не чаще раза в cooldown, чтобы
// сбой, повторяющийся каждую секунду, не превратился в спам.
func (r *Recorder) Alert(key, text string) {
	if r == nil {
		return
	}
	now := r.now()
	r.mu.Lock()
	if last, ok := r.lastAlert[key]; ok && now.Sub(last) < r.cooldown {
		r.mu.Unlock()
		return
	}
	r.lastAlert[key] = now
	r.mu.Unlock()

	text = truncate(text, alertTextMaxRunes)
	r.logger.Warn("ops alert", "key", key, "text", text)
	if r.adminChat == 0 || r.bot == nil {
		return
	}
	message := text + "\n\nПовтор этого алерта — не раньше чем через " + minutesText(r.cooldown) + "."
	r.sending.Add(1)
	// В фоне: алерт не должен тормозить запрос или рассылку, из которых он возник.
	go func() {
		defer r.sending.Done()
		ctx, cancel := context.WithTimeout(context.Background(), alertSendTimeout)
		defer cancel()
		if err := r.bot.SendMessage(ctx, r.adminChat, message, nil); err != nil {
			r.logger.Error("ops alert send failed", "key", key, "error", err)
		}
	}()
}

// Notify отправляет администратору сообщение сразу и возвращает ошибку отправки — для
// того, что нужно пометить сделанным только после доставки. Без паузы между повторами.
// Администратор не задан — ErrNoAdmin.
func (r *Recorder) Notify(ctx context.Context, text string) error {
	if r == nil || r.adminChat == 0 || r.bot == nil {
		return ErrNoAdmin
	}
	r.logger.Info("ops notify", "text", truncate(text, alertTextMaxRunes))
	sendCtx, cancel := context.WithTimeout(ctx, alertSendTimeout)
	defer cancel()
	return r.bot.SendMessage(sendCtx, r.adminChat, truncate(text, alertTextMaxRunes), nil)
}

// ErrNoAdmin — администратор для сообщений не задан.
var ErrNoAdmin = errors.New("admin telegram chat is not configured")

// CountAndAlert считает событие и поднимает алерт, когда за window их набралось threshold.
func (r *Recorder) CountAndAlert(key string, threshold int, window time.Duration, text func(count int) string) {
	if r == nil {
		return
	}
	now := r.now()
	r.mu.Lock()
	events := append(r.windows[key], now)
	fresh := events[:0]
	for _, at := range events {
		if now.Sub(at) < window {
			fresh = append(fresh, at)
		}
	}
	r.windows[key] = fresh
	count := len(fresh)
	r.mu.Unlock()

	if count >= threshold {
		r.Alert(key, text(count))
	}
}

// Run сбрасывает счётчики в базу раз в flushInterval и последний раз — при остановке.
func (r *Recorder) Run(ctx context.Context) {
	if r == nil {
		return
	}
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := r.Flush(flushCtx); err != nil {
				r.logger.Error("ops counters final flush failed", "error", err)
			}
			cancel()
			return
		case <-ticker.C:
			if err := r.Flush(ctx); err != nil && ctx.Err() == nil {
				r.logger.Error("ops counters flush failed", "error", err)
			}
		}
	}
}

// Flush записывает накопленное. Не записанное из-за ошибки возвращается в копилку.
func (r *Recorder) Flush(ctx context.Context) error {
	if r == nil || r.store == nil {
		return nil
	}
	r.mu.Lock()
	batch := r.pending
	r.pending = map[time.Time]map[string]int64{}
	r.mu.Unlock()

	for hour, counts := range batch {
		if err := r.store.AddCounters(ctx, hour, counts); err != nil {
			r.mu.Lock()
			for h, c := range batch {
				if r.pending[h] == nil {
					r.pending[h] = map[string]int64{}
				}
				for name, value := range c {
					r.pending[h][name] += value
				}
			}
			r.mu.Unlock()
			return err
		}
		delete(batch, hour)
	}
	return nil
}

// waitAlerts ждёт фоновые отправки (для тестов).
func (r *Recorder) waitAlerts() { r.sending.Wait() }

func truncate(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "…"
}

func minutesText(d time.Duration) string {
	return strconv.Itoa(int(d/time.Minute)) + " мин"
}
