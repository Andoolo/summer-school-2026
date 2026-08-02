-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE clients (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text,
    phone text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT clients_phone_e164_chk CHECK (phone ~ '^\+[1-9][0-9]{1,14}$'),
    CONSTRAINT clients_name_len_chk CHECK (name IS NULL OR char_length(name) BETWEEN 1 AND 100)
);

CREATE TABLE auth_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id uuid NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    token_hash text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    CONSTRAINT auth_sessions_expiry_chk CHECK (expires_at > created_at)
);

CREATE TABLE otp_codes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    phone text NOT NULL,
    purpose text NOT NULL,
    code_hash text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    attempt_count integer NOT NULL DEFAULT 0,
    CONSTRAINT otp_codes_phone_e164_chk CHECK (phone ~ '^\+[1-9][0-9]{1,14}$'),
    CONSTRAINT otp_codes_purpose_chk CHECK (purpose IN ('login', 'phone_change')),
    CONSTRAINT otp_codes_expiry_chk CHECK (expires_at > created_at),
    CONSTRAINT otp_codes_attempt_count_chk CHECK (attempt_count >= 0)
);

CREATE TABLE routes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    type text NOT NULL,
    capacity_cap integer NOT NULL,
    duration_min integer NOT NULL,
    geometry jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT routes_type_chk CHECK (type IN ('novice', 'experienced')),
    CONSTRAINT routes_capacity_chk CHECK (
        capacity_cap > 0
        AND ((type = 'novice' AND capacity_cap <= 8) OR (type = 'experienced' AND capacity_cap <= 12))
    ),
    CONSTRAINT routes_duration_chk CHECK (duration_min > 0)
);

CREATE TABLE instructors (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE slots (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    route_id uuid NOT NULL REFERENCES routes(id) ON DELETE RESTRICT,
    instructor_id uuid NOT NULL REFERENCES instructors(id) ON DELETE RESTRICT,
    start_at timestamptz NOT NULL,
    total_seats integer NOT NULL,
    free_seats integer NOT NULL,
    rental_boards_total integer NOT NULL DEFAULT 12,
    free_rental_boards integer NOT NULL,
    price integer NOT NULL,
    rental_price integer NOT NULL,
    meeting_point text NOT NULL,
    meeting_point_lat double precision NOT NULL,
    meeting_point_lng double precision NOT NULL,
    status text NOT NULL DEFAULT 'scheduled',
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT slots_status_chk CHECK (status IN ('scheduled', 'cancelled')),
    CONSTRAINT slots_seats_chk CHECK (total_seats > 0 AND free_seats >= 0 AND free_seats <= total_seats),
    CONSTRAINT slots_rental_boards_chk CHECK (rental_boards_total >= 0 AND free_rental_boards >= 0 AND free_rental_boards <= rental_boards_total),
    CONSTRAINT slots_price_chk CHECK (price >= 0 AND rental_price >= 0),
    CONSTRAINT slots_meeting_point_lat_chk CHECK (meeting_point_lat BETWEEN -90 AND 90),
    CONSTRAINT slots_meeting_point_lng_chk CHECK (meeting_point_lng BETWEEN -180 AND 180)
);

CREATE TABLE bookings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slot_id uuid NOT NULL REFERENCES slots(id) ON DELETE RESTRICT,
    client_id uuid NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
    seats_count integer NOT NULL,
    rental_count integer NOT NULL,
    status text NOT NULL DEFAULT 'active',
    created_at timestamptz NOT NULL DEFAULT now(),
    cancelled_at timestamptz,
    CONSTRAINT bookings_status_chk CHECK (status IN ('active', 'cancelled', 'late_cancel')),
    CONSTRAINT bookings_seats_chk CHECK (seats_count BETWEEN 1 AND 3),
    CONSTRAINT bookings_rental_chk CHECK (rental_count BETWEEN 0 AND seats_count),
    CONSTRAINT bookings_cancelled_at_chk CHECK ((status = 'active' AND cancelled_at IS NULL) OR (status <> 'active' AND cancelled_at IS NOT NULL))
);

CREATE TABLE idempotency_keys (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id uuid NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    key text NOT NULL,
    request_hash text NOT NULL,
    response_status integer,
    response_body jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    CONSTRAINT idempotency_keys_key_len_chk CHECK (char_length(key) BETWEEN 1 AND 255),
    CONSTRAINT idempotency_keys_expiry_chk CHECK (expires_at > created_at),
    UNIQUE (client_id, key)
);

CREATE UNIQUE INDEX bookings_active_client_slot_uidx ON bookings (client_id, slot_id) WHERE status = 'active';
CREATE INDEX auth_sessions_client_id_idx ON auth_sessions (client_id);
CREATE INDEX auth_sessions_token_hash_idx ON auth_sessions (token_hash);
CREATE INDEX otp_codes_phone_purpose_created_at_idx ON otp_codes (phone, purpose, created_at DESC);
CREATE INDEX slots_start_at_idx ON slots (start_at);
CREATE INDEX slots_status_idx ON slots (status);
CREATE INDEX slots_route_id_idx ON slots (route_id);
CREATE INDEX slots_instructor_id_idx ON slots (instructor_id);
CREATE INDEX bookings_slot_id_idx ON bookings (slot_id);
CREATE INDEX bookings_client_id_idx ON bookings (client_id);
CREATE INDEX bookings_status_idx ON bookings (status);
CREATE INDEX idempotency_keys_client_key_idx ON idempotency_keys (client_id, key);

-- +goose Down
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS bookings;
DROP TABLE IF EXISTS slots;
DROP TABLE IF EXISTS instructors;
DROP TABLE IF EXISTS routes;
DROP TABLE IF EXISTS otp_codes;
DROP TABLE IF EXISTS auth_sessions;
DROP TABLE IF EXISTS clients;
