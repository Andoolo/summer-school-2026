-- Mock data for exercising client app states.
-- Safe to rerun: clears only the tables used by the client-facing MVP.

BEGIN;

TRUNCATE TABLE
    idempotency_keys,
    bookings,
    slots,
    instructors,
    routes,
    otp_codes,
    auth_sessions,
    clients
RESTART IDENTITY CASCADE;

INSERT INTO routes (id, name, type, capacity_cap, duration_min, geometry)
VALUES
    ('11111111-1111-1111-1111-111111111111', 'Острова и каналы', 'novice', 8, 90, '[[59.978,30.262],[59.981,30.271],[59.976,30.285]]'),
    ('22222222-2222-2222-2222-222222222222', 'Большая вода', 'experienced', 12, 120, '[[59.942,30.226],[59.951,30.214],[59.963,30.232]]'),
    ('33333333-3333-3333-3333-333333333333', 'Закатный маршрут', 'experienced', 12, 110, '[[59.963,30.232],[59.970,30.241],[59.975,30.255]]'),
    ('44444444-4444-4444-4444-444444444444', 'Городской маршрут', 'novice', 8, 75, '[[59.965,30.300],[59.968,30.312],[59.971,30.321]]');

INSERT INTO instructors (id, name)
VALUES
    ('aaaa1111-1111-1111-1111-111111111111', 'Мария'),
    ('bbbb2222-2222-2222-2222-222222222222', 'Алексей'),
    ('cccc3333-3333-3333-3333-333333333333', 'Ирина');

INSERT INTO clients (id, phone, name)
VALUES
    ('90000000-0000-4000-8000-000000000001', '+79990000001', 'Иван'),
    ('90000000-0000-4000-8000-000000000002', '+79990000002', 'Анна'),
    ('90000000-0000-4000-8000-000000000003', '+79990000003', 'Ольга'),
    ('90000000-0000-4000-8000-000000000004', '+79990000004', 'Сергей');

INSERT INTO auth_sessions (client_id, token_hash, expires_at)
VALUES
    ('90000000-0000-4000-8000-000000000001', 'mock-token-1', now() + interval '30 days'),
    ('90000000-0000-4000-8000-000000000002', 'mock-token-2', now() + interval '30 days'),
    ('90000000-0000-4000-8000-000000000003', 'mock-token-3', now() + interval '30 days'),
    ('90000000-0000-4000-8000-000000000004', 'mock-token-4', now() + interval '30 days');

WITH fixed_slots (
    id,
    route_id,
    instructor_id,
    start_at,
    total_seats,
    free_seats,
    rental_boards_total,
    free_rental_boards,
    price,
    rental_price,
    meeting_point,
    meeting_point_lat,
    meeting_point_lng,
    status
) AS (
    VALUES
    (
        '50000000-0000-4000-8000-000000000001',
        '11111111-1111-1111-1111-111111111111',
        'aaaa1111-1111-1111-1111-111111111111',
        timestamptz '2026-06-25 10:00:00+03',
        8,
        8,
        12,
        12,
        2500,
        800,
        'Лодочная станция у Елагина моста',
        59.978,
        30.262,
        'scheduled'
    ),
    (
        '50000000-0000-4000-8000-000000000002',
        '22222222-2222-2222-2222-222222222222',
        'bbbb2222-2222-2222-2222-222222222222',
        timestamptz '2026-06-26 12:00:00+03',
        12,
        2,
        12,
        0,
        3200,
        800,
        'Пирс у Крестовского острова',
        59.942,
        30.226,
        'scheduled'
    ),
    (
        '50000000-0000-4000-8000-000000000003',
        '33333333-3333-3333-3333-333333333333',
        'cccc3333-3333-3333-3333-333333333333',
        timestamptz '2026-06-27 14:00:00+03',
        12,
        0,
        12,
        0,
        3400,
        900,
        'Набережная у закатного пирса',
        59.963,
        30.232,
        'scheduled'
    ),
    (
        '50000000-0000-4000-8000-000000000004',
        '44444444-4444-4444-4444-444444444444',
        'aaaa1111-1111-1111-1111-111111111111',
        timestamptz '2026-06-28 08:00:00+03',
        8,
        5,
        12,
        9,
        2200,
        700,
        'Городской причал',
        59.965,
        30.300,
        'scheduled'
    ),
    (
        '50000000-0000-4000-8000-000000000005',
        '11111111-1111-1111-1111-111111111111',
        'bbbb2222-2222-2222-2222-222222222222',
        timestamptz '2026-06-29 16:00:00+03',
        8,
        8,
        12,
        12,
        2600,
        800,
        'Северный залив',
        59.981,
        30.271,
        'cancelled'
    )
),
generated_slots AS (
    SELECT
        gen_random_uuid() AS id,
        CASE ((slot_idx - 1) % 4)
            WHEN 0 THEN '11111111-1111-1111-1111-111111111111'
            WHEN 1 THEN '22222222-2222-2222-2222-222222222222'
            WHEN 2 THEN '33333333-3333-3333-3333-333333333333'
            ELSE '44444444-4444-4444-4444-444444444444'
        END::uuid AS route_id,
        CASE ((slot_idx - 1) % 3)
            WHEN 0 THEN 'aaaa1111-1111-1111-1111-111111111111'
            WHEN 1 THEN 'bbbb2222-2222-2222-2222-222222222222'
            ELSE 'cccc3333-3333-3333-3333-333333333333'
        END::uuid AS instructor_id,
        ((d::date + make_interval(hours => hour_offset))::timestamp at time zone 'Europe/Moscow') AS start_at,
        CASE WHEN slot_idx IN (1, 4) THEN 8 ELSE 12 END AS total_seats,
        CASE slot_idx
            WHEN 1 THEN 8
            WHEN 2 THEN 4
            WHEN 3 THEN 0
            WHEN 4 THEN 1
            ELSE 6
        END AS free_seats,
        12 AS rental_boards_total,
        CASE slot_idx
            WHEN 1 THEN 12
            WHEN 2 THEN 5
            WHEN 3 THEN 0
            WHEN 4 THEN 2
            ELSE 7
        END AS free_rental_boards,
        CASE slot_idx
            WHEN 1 THEN 2500
            WHEN 2 THEN 3200
            WHEN 3 THEN 3400
            WHEN 4 THEN 2200
            ELSE 2800
        END AS price,
        CASE slot_idx
            WHEN 1 THEN 800
            WHEN 2 THEN 800
            WHEN 3 THEN 900
            WHEN 4 THEN 700
            ELSE 750
        END AS rental_price,
        CASE slot_idx
            WHEN 1 THEN 'Лодочная станция у Елагина моста'
            WHEN 2 THEN 'Пирс у Крестовского острова'
            WHEN 3 THEN 'Набережная у закатного пирса'
            WHEN 4 THEN 'Городской причал'
            ELSE 'Северный залив'
        END AS meeting_point,
        CASE slot_idx
            WHEN 1 THEN 59.978
            WHEN 2 THEN 59.942
            WHEN 3 THEN 59.963
            WHEN 4 THEN 59.965
            ELSE 59.981
        END AS meeting_point_lat,
        CASE slot_idx
            WHEN 1 THEN 30.262
            WHEN 2 THEN 30.226
            WHEN 3 THEN 30.232
            WHEN 4 THEN 30.300
            ELSE 30.271
        END AS meeting_point_lng,
        CASE slot_idx
            WHEN 3 THEN 'cancelled'
            ELSE 'scheduled'
        END AS status
    FROM generate_series(date '2026-07-01', date '2026-07-31', interval '1 day') AS d
    CROSS JOIN generate_series(1, 5) AS slot_idx
    CROSS JOIN LATERAL (
        SELECT CASE slot_idx
            WHEN 1 THEN 8
            WHEN 2 THEN 10
            WHEN 3 THEN 12
            WHEN 4 THEN 14
            ELSE 16
        END AS hour_offset
    ) h
)
INSERT INTO slots (
    id,
    route_id,
    instructor_id,
    start_at,
    total_seats,
    free_seats,
    rental_boards_total,
    free_rental_boards,
    price,
    rental_price,
    meeting_point,
    meeting_point_lat,
    meeting_point_lng,
    status
)
SELECT
    id::uuid, route_id::uuid, instructor_id::uuid, start_at::timestamptz, total_seats::integer, free_seats::integer,
    rental_boards_total::integer, free_rental_boards::integer, price::integer, rental_price::integer,
    meeting_point::text, meeting_point_lat::double precision, meeting_point_lng::double precision, status::text
FROM fixed_slots
UNION ALL
SELECT
    id, route_id, instructor_id, start_at, total_seats, free_seats, rental_boards_total,
    free_rental_boards, price, rental_price, meeting_point, meeting_point_lat, meeting_point_lng, status
FROM generated_slots;

INSERT INTO bookings (id, slot_id, client_id, seats_count, rental_count, status, created_at, cancelled_at)
VALUES
    (
        '60000000-0000-4000-8000-000000000001',
        '50000000-0000-4000-8000-000000000001',
        '90000000-0000-4000-8000-000000000001',
        2,
        1,
        'active',
        now() - interval '30 minutes',
        NULL
    ),
    (
        '60000000-0000-4000-8000-000000000002',
        '50000000-0000-4000-8000-000000000002',
        '90000000-0000-4000-8000-000000000001',
        3,
        0,
        'active',
        now() - interval '1 hour',
        NULL
    ),
    (
        '60000000-0000-4000-8000-000000000003',
        '50000000-0000-4000-8000-000000000003',
        '90000000-0000-4000-8000-000000000002',
        1,
        1,
        'cancelled',
        now() - interval '2 days',
        now() - interval '1 day 20 hours'
    ),
    (
        '60000000-0000-4000-8000-000000000004',
        '50000000-0000-4000-8000-000000000004',
        '90000000-0000-4000-8000-000000000003',
        2,
        1,
        'late_cancel',
        now() - interval '6 hours',
        now() - interval '4 hours'
    ),
    (
        '60000000-0000-4000-8000-000000000005',
        '50000000-0000-4000-8000-000000000004',
        '90000000-0000-4000-8000-000000000004',
        1,
        0,
        'active',
        now() - interval '6 hours',
        NULL
    );

COMMIT;
