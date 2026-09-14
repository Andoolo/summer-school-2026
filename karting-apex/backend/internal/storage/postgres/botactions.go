package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"summer-school-2026/backend/internal/service/booking"
	"summer-school-2026/backend/internal/service/botactions"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BotActionsRepository — брони для действий из бота. Отмена — та же, что в приложении.
type BotActionsRepository struct {
	db       *pgxpool.Pool
	bookings *BookingRepository
}

func NewBotActionsRepository(db *pgxpool.Pool) *BotActionsRepository {
	return &BotActionsRepository{db: db, bookings: NewBookingRepository(db)}
}

func (r *BotActionsRepository) BookingForChat(ctx context.Context, bookingID string, chatID int64) (botactions.Booking, bool, error) {
	var found botactions.Booking
	err := r.db.QueryRow(ctx, `
SELECT b.client_id::text, b.status, s.start_at
FROM bookings b
JOIN clients c ON c.id = b.client_id
JOIN slots s ON s.id = b.slot_id
WHERE b.id = $1
  AND c.telegram_chat_id = $2
  AND c.deleted_at IS NULL`, bookingID, chatID).Scan(&found.ClientID, &found.Status, &found.StartAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return botactions.Booking{}, false, nil
	}
	if err != nil {
		return botactions.Booking{}, false, fmt.Errorf("query booking for telegram chat: %w", err)
	}
	return found, true, nil
}

func (r *BotActionsRepository) Cancel(ctx context.Context, clientID, bookingID string, now time.Time) (booking.Booking, error) {
	return r.bookings.Cancel(ctx, clientID, bookingID, now)
}
