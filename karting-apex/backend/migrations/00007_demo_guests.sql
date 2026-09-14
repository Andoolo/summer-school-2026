-- +goose Up
-- Гостевые (демо) аккаунты: вход без регистрации, чтобы посмотреть приложение.
--
-- Отдельной таблицы нет: гость — обычный клиент с ограниченным сроком жизни, поэтому
-- записи на заезды, профиль и сессии работают без особых путей в коде. NULL означает
-- обычного клиента; значение — момент, после которого гость удаляется уборкой.
ALTER TABLE clients ADD COLUMN demo_expires_at timestamptz;

-- Уборка истёкших гостей и подсчёт активных идут только по гостям.
CREATE INDEX clients_demo_expires_at_idx ON clients (demo_expires_at) WHERE demo_expires_at IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS clients_demo_expires_at_idx;
ALTER TABLE clients DROP COLUMN IF EXISTS demo_expires_at;
