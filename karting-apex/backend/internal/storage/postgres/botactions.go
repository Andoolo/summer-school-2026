package postgres

import (
	"context"
	"errors"
	"fmt"

	"summer-school-2026/backend/internal/service/botactions"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BotActionsRepository ищет брони для действий из бота. Отменяет — BookingRepository, как в
// приложении.
type BotActionsRepository struct {
	db *pgxpool.Pool
}

func NewBotActionsRepository(db *pgxpool.Pool) *BotActionsRepository {
	return &BotActionsRepository{db: db}
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
