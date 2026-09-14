package notify

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"summer-school-2026/backend/internal/telegram"
)

// Offer — предложение освободившегося места человеку из листа ожидания.
type Offer struct {
	EntryID        string
	ChatID         int64
	SeatsWanted    int
	FreeSeats      int
	RouteName      string
	InstructorName string
	StartAt        time.Time
	MeetingPoint   string
	ExpiresAt      time.Time
}

type WaitlistRepository interface {
	// ClaimOffers закрывает отработавшие записи очереди и отмечает новые предложения.
	ClaimOffers(ctx context.Context, now time.Time, ttl time.Duration) ([]Offer, error)
	// ReleaseOffer возвращает запись в очередь на прежнее место.
	ReleaseOffer(ctx context.Context, entryID string) error
	// ExpireOffer закрывает запись, которой предложение не доставить.
	ExpireOffer(ctx context.Context, entryID string, now time.Time) error
}

// WithWaitlist включает раздачу мест из листа ожидания. appURL — адрес приложения для
// ссылки в сообщении (может быть пустым); ttl — срок предложения.
func (d *Dispatcher) WithWaitlist(repo WaitlistRepository, ttl time.Duration, appURL string) *Dispatcher {
	d.waitlist = repo
	d.offerTTL = ttl
	d.appURL = strings.TrimRight(appURL, "/")
	return d
}

func (d *Dispatcher) runWaitlist(ctx context.Context) int {
	if d.waitlist == nil || ctx.Err() != nil {
		return 0
	}
	now := d.now().UTC()
	offers, err := d.waitlist.ClaimOffers(ctx, now, d.offerTTL)
	if err != nil {
		if ctx.Err() == nil {
			d.logger.Error("waitlist offers claim failed", "error", err)
		}
		return 0
	}
	sent := 0
	for _, offer := range offers {
		err := d.bot.SendMessage(ctx, offer.ChatID, OfferText(offer, d.offerTTL, d.appURL), nil)
		if err == nil {
			sent++
			continue
		}
		var apiErr *telegram.APIError
		switch {
		case errors.As(err, &apiErr) && apiErr.Permanent():
			d.logger.Warn("waitlist offer rejected by telegram", "entry_id", offer.EntryID, "error", err)
			if apiErr.Blocked() {
				if err := d.repo.DisableChat(detached(ctx), offer.ChatID); err != nil {
					d.logger.Error("disable telegram chat failed", "error", err)
				}
			}
			// Место уходит следующему в очереди при следующем проходе.
			if err := d.waitlist.ExpireOffer(detached(ctx), offer.EntryID, now); err != nil {
				d.logger.Error("expire waitlist offer failed", "entry_id", offer.EntryID, "error", err)
			}
		default:
			d.logger.Warn("waitlist offer failed, will retry", "entry_id", offer.EntryID, "error", err)
			releaseCtx, cancel := context.WithTimeout(detached(ctx), 5*time.Second)
			if err := d.waitlist.ReleaseOffer(releaseCtx, offer.EntryID); err != nil {
				d.logger.Error("release waitlist offer failed", "entry_id", offer.EntryID, "error", err)
			}
			cancel()
		}
	}
	return sent
}

// OfferText — «освободилось место». Честно говорим, что место не закреплено.
func OfferText(o Offer, ttl time.Duration, appURL string) string {
	var b strings.Builder
	b.WriteString("🎉 Освободилось место в заезде!\n\n")
	writeRace(&b, Notice{RouteName: o.RouteName, StartAt: o.StartAt, MeetingPoint: o.MeetingPoint, InstructorName: o.InstructorName})
	b.WriteString("🎟 Свободно мест: " + strconv.Itoa(o.FreeSeats) + ", вы ждали: " + strconv.Itoa(o.SeatsWanted) + "\n\n")
	minutes := int(ttl / time.Minute)
	b.WriteString("Запишитесь в течение " + strconv.Itoa(minutes) + " минут — до " + o.ExpiresAt.In(moscow).Format("15:04") +
		" (мск). Потом предложим место следующему в очереди.\n")
	b.WriteString("Место не закреплено: пока вы не записались, его может занять другой.\n")
	if appURL != "" {
		b.WriteString("\nЗаписаться: " + appURL + "\n")
	}
	b.WriteString("\n" + footer)
	return b.String()
}
