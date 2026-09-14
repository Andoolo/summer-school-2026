-- +goose Up
-- Лист ожидания на заполненные заезды.
--
-- Жизненный цикл записи:
--   waiting  — человек в очереди;
--   notified — место освободилось, человеку отправлено предложение. Место не закреплено:
--              записаться может любой. Через 15 минут без брони предложение истекает и
--              переходит к следующему в очереди;
--   booked   — человек записался на заезд;
--   left     — вышел из очереди сам (или удалил аккаунт);
--   expired  — предложение истекло, заезд начался или отменён.
CREATE TABLE waitlist_entries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slot_id uuid NOT NULL REFERENCES slots(id) ON DELETE CASCADE,
    client_id uuid NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    seats_count integer NOT NULL,
    status text NOT NULL DEFAULT 'waiting',
    created_at timestamptz NOT NULL DEFAULT now(),
    notified_at timestamptz,
    closed_at timestamptz,
    CONSTRAINT waitlist_entries_seats_chk CHECK (seats_count BETWEEN 1 AND 3),
    CONSTRAINT waitlist_entries_status_chk CHECK (status IN ('waiting', 'notified', 'booked', 'left', 'expired')),
    CONSTRAINT waitlist_entries_notified_chk CHECK (status <> 'notified' OR notified_at IS NOT NULL)
);

-- В очереди на один заезд человек стоит один раз.
CREATE UNIQUE INDEX waitlist_entries_active_uidx ON waitlist_entries (client_id, slot_id) WHERE status IN ('waiting', 'notified');
-- Очередь заезда в порядке записи.
CREATE INDEX waitlist_entries_queue_idx ON waitlist_entries (slot_id, created_at) WHERE status IN ('waiting', 'notified');

-- +goose Down
DROP TABLE IF EXISTS waitlist_entries;
