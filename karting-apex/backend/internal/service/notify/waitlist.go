package notify

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	"summer-school-2026/backend/internal/ops"
)

// Offer — предложение освободившегося места человеку из листа ожидания.
type Offer struct {
	EntryID        string
	SlotID         string
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

func (d *Dispatcher) runWaitlist(ctx context.Context) int {
	if d.waitlist == nil || ctx.Err() != nil {
		return 0
	}
	now := d.now().UTC()
	offers, err := d.waitlist.ClaimOffers(ctx, now, d.offerTTL)
	if err != nil {
		if ctx.Err() == nil {
			d.logger.Error("waitlist offers claim failed", "error", err)
			d.observer.Alert("waitlist_claim", "⚠️ Лист ожидания не может раздать места: "+ops.DescribeError(err))
		}
		return 0
	}
	sent := 0
	for _, offer := range offers {
		err := d.bot.SendMessage(ctx, offer.ChatID, OfferText(offer, d.offerTTL, d.appURL), nil)
		switch d.classifySend(ctx, offer.ChatID, err) {
		case outcomeSent:
			sent++
			d.observer.Inc(ops.WaitlistOffers)
		case outcomeBlocked, outcomeRejected:
			d.logger.Warn("waitlist offer rejected by telegram", "entry_id", offer.EntryID, "error", err)
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

// SlotLink — ссылка сразу на экран заезда в веб-приложении (#slot/<id>), а не на главную:
// на запись всего 15 минут, искать заезд в каталоге некогда. Если человек не вошёл,
// приложение откроет заезд после входа.
func SlotLink(appURL, slotID string) string {
	switch {
	case appURL == "":
		return ""
	case slotID == "":
		return appURL
	default:
		return appURL + "/#slot/" + url.PathEscape(slotID)
	}
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
	if link := SlotLink(appURL, o.SlotID); link != "" {
		b.WriteString("\nЗаписаться: " + link + "\n")
	}
	b.WriteString("\n" + footer)
	return b.String()
}
