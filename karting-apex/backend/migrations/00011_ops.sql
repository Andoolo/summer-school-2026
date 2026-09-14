-- +goose Up
-- Наблюдаемость без внешних сервисов: счётчики событий по часам и служебное состояние.
--
-- Счётчики пишутся пачками из памяти сервиса (раз в 30 секунд и при остановке), поэтому
-- переживают сон Render и перезапуски. Хранятся 30 дней — для сводки /stats больше не нужно.
CREATE TABLE ops_counters (
    hour timestamptz NOT NULL,
    name text NOT NULL,
    value bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (hour, name)
);

-- Служебные значения: например, какая версия уже объявлена администратору.
CREATE TABLE ops_state (
    key text PRIMARY KEY,
    value text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS ops_state;
DROP TABLE IF EXISTS ops_counters;
