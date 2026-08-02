-- +goose Up
-- Результаты кругов заезда (картинг): времена кругов участников по брони.
-- Только схема; демо-данные — в seed/karting_lap_results.sql (не попадают в тестовую БД).
CREATE TABLE lap_results (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id uuid NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    lap_number integer NOT NULL,
    lap_time_ms integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT lap_results_lap_number_chk CHECK (lap_number > 0),
    CONSTRAINT lap_results_lap_time_chk CHECK (lap_time_ms > 0)
);

CREATE INDEX lap_results_booking_id_idx ON lap_results (booking_id);

-- +goose Down
DROP TABLE lap_results;
