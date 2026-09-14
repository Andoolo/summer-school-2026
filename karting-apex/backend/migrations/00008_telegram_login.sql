-- +goose Up
-- Вход через Telegram: приложение создаёт запрос, человек открывает бота по ссылке и
-- делится номером, приложение забирает сессию.
--
-- Два разных секрета, в базе — только их хеши:
--   start_token_hash — из ссылки t.me/<бот>?start=<токен>. Ссылку видит тот, кому её
--                      показали, поэтому одного её недостаточно, чтобы получить сессию;
--   poll_token_hash  — есть только у приложения, начавшего вход; только по нему
--                      выдаётся сессия.
-- confirm_code — короткий код сверки: бот показывает тот же код, что и приложение, чтобы
-- человек не подтвердил вход, начатый кем-то другим.
CREATE TABLE telegram_login_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    start_token_hash text NOT NULL UNIQUE,
    poll_token_hash text NOT NULL UNIQUE,
    confirm_code text NOT NULL,
    chat_id bigint,
    phone text,
    first_name text,
    status text NOT NULL DEFAULT 'pending',
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    CONSTRAINT telegram_login_requests_status_chk CHECK (status IN ('pending', 'confirmed', 'consumed')),
    CONSTRAINT telegram_login_requests_expiry_chk CHECK (expires_at > created_at),
    CONSTRAINT telegram_login_requests_confirmed_chk CHECK (status = 'pending' OR (phone IS NOT NULL AND chat_id IS NOT NULL))
);

-- Бот ищет последний незавершённый запрос своего чата, когда человек делится номером.
CREATE INDEX telegram_login_requests_chat_idx ON telegram_login_requests (chat_id, created_at DESC) WHERE status = 'pending';
CREATE INDEX telegram_login_requests_expires_at_idx ON telegram_login_requests (expires_at);

-- +goose Down
DROP TABLE IF EXISTS telegram_login_requests;
