// Package notify рассылает уведомления о бронях в Telegram: подтверждение, напоминание
// перед заездом и отмену.
//
// Очереди нет: состояние хранится в самих бронях (отметки *_notified_at). Рассыльщик
// раз в минуту — и сразу после создания или отмены брони — забирает то, что пора
// отправить. На бесплатном Render сервис засыпает; после пробуждения рассыльщик
// дошлёт всё, что ещё актуально: подтверждения и отмены — в течение часа, напоминания —
// пока заезд не начался.
package notify

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"summer-school-2026/backend/internal/ops"
	"summer-school-2026/backend/internal/telegram"
)

type Kind string

const (
	KindConfirm  Kind = "confirm"
	KindCancel   Kind = "cancel"
	KindReminder Kind = "reminder"
)

const (
	// ReminderLead — за сколько до старта напоминать.
	ReminderLead = 2 * time.Hour
	// ReminderMinAdvance — бронь, сделанная ближе к старту, напоминания не получает:
	// подтверждение пришло только что, второе сообщение было бы лишним.
	ReminderMinAdvance = 3 * time.Hour
	// FreshWindow — подтверждение или отмена старше этого уже неактуальны.
	FreshWindow = time.Hour

	DefaultInterval = time.Minute
	batchLimit      = 50
)

// Notice — уведомление, которое пора отправить.
type Notice struct {
	Kind           Kind
	BookingID      string
	ChatID         int64
	Status         string
	SeatsCount     int
	RentalCount    int
	PriceTotal     int
	CreatedAt      time.Time
	RouteName      string
	InstructorName string
	StartAt        time.Time
	MeetingPoint   string
}

type Repository interface {
	// ClaimDue ставит отметку об отправке на всё, что пора отправить, и возвращает
	// уведомления для клиентов с Telegram. Брони остальных клиентов только отмечаются.
	ClaimDue(ctx context.Context, kind Kind, now time.Time, limit int) ([]Notice, error)
	// Unclaim снимает отметку, чтобы отправка повторилась при следующем проходе.
	Unclaim(ctx context.Context, kind Kind, bookingID string) error
	// DisableChat отвязывает чат, который заблокировал бота.
	DisableChat(ctx context.Context, chatID int64) error
}

type Bot interface {
	SendMessage(ctx context.Context, chatID int64, text string, markup any) error
}

type Dispatcher struct {
	repo     Repository
	bot      Bot
	logger   *slog.Logger
	now      func() time.Time
	interval time.Duration
	wake     chan struct{}

	waitlist WaitlistRepository
	offerTTL time.Duration
	appURL   string

	observer Observer
}

// Observer — счётчики и алерты (ops.Recorder).
type Observer interface {
	Inc(name string)
	Alert(key, text string)
	CountAndAlert(key string, threshold int, window time.Duration, text func(count int) string)
}

type noopObserver struct{}

func (noopObserver) Inc(string)                                                 {}
func (noopObserver) Alert(string, string)                                       {}
func (noopObserver) CountAndAlert(string, int, time.Duration, func(int) string) {}

// WithObserver подключает счётчики отправок и алерты о сбоях рассылки.
func (d *Dispatcher) WithObserver(observer Observer) *Dispatcher {
	if observer != nil {
		d.observer = observer
	}
	return d
}

// sendFailed считает временный сбой отправки; три за 10 минут — алерт.
func (d *Dispatcher) sendFailed(err error) {
	d.observer.Inc(ops.TelegramFailed)
	d.observer.CountAndAlert("telegram_failed", 3, 10*time.Minute, func(count int) string {
		return fmt.Sprintf("⚠️ Рассылка в Telegram сбоит: %d ошибок за 10 минут. Последняя: %v", count, err)
	})
}

func NewDispatcher(repo Repository, bot Bot, logger *slog.Logger) *Dispatcher {
	if logger == nil {
		logger = slog.Default()
	}
	return &Dispatcher{repo: repo, bot: bot, logger: logger, now: time.Now, interval: DefaultInterval, wake: make(chan struct{}, 1), observer: noopObserver{}}
}

// Wake просит рассыльщика пройтись раньше срока. Не блокирует: если просьба уже
// ждёт, вторая не нужна.
func (d *Dispatcher) Wake() {
	select {
	case d.wake <- struct{}{}:
	default:
	}
}

func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()
	for {
		d.RunOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-d.wake:
		}
	}
}

// RunOnce отправляет всё, что пора, и возвращает число отправленных сообщений.
func (d *Dispatcher) RunOnce(ctx context.Context) int {
	sent := 0
	for _, kind := range []Kind{KindConfirm, KindCancel, KindReminder} {
		for ctx.Err() == nil {
			now := d.now().UTC()
			notices, err := d.repo.ClaimDue(ctx, kind, now, batchLimit)
			if err != nil {
				if ctx.Err() == nil {
					d.logger.Error("booking notifications claim failed", "kind", kind, "error", err)
					d.observer.Alert("notify_claim", fmt.Sprintf("⚠️ Рассылка уведомлений не может прочитать базу: %v", err))
				}
				break
			}
			retryLater := false
			for _, notice := range notices {
				ok, retry := d.deliver(ctx, notice, now)
				if ok {
					sent++
				}
				retryLater = retryLater || retry
			}
			// Неудачные отправки вернулись в очередь — в этом проходе их не крутим.
			if retryLater || len(notices) < batchLimit {
				break
			}
		}
	}
	return sent + d.runWaitlist(ctx)
}

// deliver возвращает (отправлено, нужно повторить позже).
func (d *Dispatcher) deliver(ctx context.Context, notice Notice, now time.Time) (bool, bool) {
	err := d.bot.SendMessage(ctx, notice.ChatID, Text(notice, now), Markup(notice))
	if err == nil {
		d.observer.Inc(ops.TelegramSent)
		return true, false
	}

	var apiErr *telegram.APIError
	switch {
	case errors.As(err, &apiErr) && apiErr.Blocked():
		d.observer.Inc(ops.TelegramBlocked)
		d.logger.Info("telegram chat blocked the bot, notifications disabled", "booking_id", notice.BookingID)
		if err := d.repo.DisableChat(detached(ctx), notice.ChatID); err != nil {
			d.logger.Error("disable telegram chat failed", "error", err)
		}
		return false, false
	case errors.As(err, &apiErr) && apiErr.Permanent():
		d.observer.Inc(ops.TelegramFailed)
		d.logger.Warn("booking notification rejected by telegram", "booking_id", notice.BookingID, "kind", notice.Kind, "error", err)
		return false, false
	default:
		d.sendFailed(err)
		d.logger.Warn("booking notification failed, will retry", "booking_id", notice.BookingID, "kind", notice.Kind, "error", err)
		unclaimCtx, cancel := context.WithTimeout(detached(ctx), 5*time.Second)
		defer cancel()
		if err := d.repo.Unclaim(unclaimCtx, notice.Kind, notice.BookingID); err != nil {
			d.logger.Error("booking notification unclaim failed", "booking_id", notice.BookingID, "error", err)
		}
		return false, true
	}
}

// detached — контекст без отмены: вернуть отметку нужно, даже если сервер
// останавливается посреди отправки.
func detached(ctx context.Context) context.Context { return context.WithoutCancel(ctx) }
