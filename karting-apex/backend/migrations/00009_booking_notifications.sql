-- +goose Up
-- Уведомления о бронях в Telegram: подтверждение, напоминание перед заездом, отмена.
--
-- Чат запоминается при входе через Telegram. Номер в этом чате подтверждён самим
-- Telegram, поэтому пишем только владельцу аккаунта.
ALTER TABLE clients
    ADD COLUMN telegram_chat_id bigint,
    ADD COLUMN telegram_notifications boolean NOT NULL DEFAULT true;

-- Отметки об отправке. Рассыльщик сначала ставит отметку, потом шлёт: одно и то же
-- уведомление не уйдёт дважды, даже если рассыльщиков несколько. Отметка ставится и
-- броням клиентов без Telegram — так частичные индексы ниже не разрастаются.
ALTER TABLE bookings
    ADD COLUMN confirm_notified_at timestamptz,
    ADD COLUMN reminder_notified_at timestamptz,
    ADD COLUMN cancel_notified_at timestamptz;

-- Задним числом подтверждений и отмен не шлём. Напоминания о будущих бронях — можно.
UPDATE bookings
SET confirm_notified_at = now(),
    cancel_notified_at = CASE WHEN status <> 'active' THEN now() END;

-- Кто уже входил через Telegram, получает уведомления сразу, без повторного входа.
-- Берутся ещё не удалённые уборкой запросы входа (удаляются через сутки). Пара «чат —
-- номер» выбирается однозначно: у чата — последний номер, у номера — последний чат.
UPDATE clients c
SET telegram_chat_id = latest.chat_id
FROM (
    SELECT DISTINCT ON (phone) phone, chat_id
    FROM (
        SELECT DISTINCT ON (chat_id) chat_id, phone, created_at
        FROM telegram_login_requests
        WHERE status = 'consumed'
        ORDER BY chat_id, created_at DESC
    ) per_chat
    ORDER BY phone, created_at DESC
) latest
WHERE c.phone = latest.phone
  AND c.deleted_at IS NULL;

CREATE UNIQUE INDEX clients_telegram_chat_id_uidx ON clients (telegram_chat_id) WHERE telegram_chat_id IS NOT NULL;

CREATE INDEX bookings_confirm_due_idx ON bookings (created_at) WHERE confirm_notified_at IS NULL;
CREATE INDEX bookings_cancel_due_idx ON bookings (cancelled_at) WHERE cancel_notified_at IS NULL AND status <> 'active';
CREATE INDEX bookings_reminder_due_idx ON bookings (slot_id) WHERE reminder_notified_at IS NULL AND status = 'active';

-- +goose Down
DROP INDEX IF EXISTS bookings_reminder_due_idx;
DROP INDEX IF EXISTS bookings_cancel_due_idx;
DROP INDEX IF EXISTS bookings_confirm_due_idx;
DROP INDEX IF EXISTS clients_telegram_chat_id_uidx;
ALTER TABLE bookings
    DROP COLUMN IF EXISTS cancel_notified_at,
    DROP COLUMN IF EXISTS reminder_notified_at,
    DROP COLUMN IF EXISTS confirm_notified_at;
ALTER TABLE clients
    DROP COLUMN IF EXISTS telegram_notifications,
    DROP COLUMN IF EXISTS telegram_chat_id;
