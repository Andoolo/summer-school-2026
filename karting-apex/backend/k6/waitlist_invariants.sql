-- Инварианты листа ожидания и броней после нагрузочного прогона k6/waitlist_300_vu.js.
-- Каждая строка — одна проверка и число нарушений; всё, кроме 0, — ошибка.
-- Запуск: psql "$DATABASE_URL" -f k6/waitlist_invariants.sql
--
-- Проверять после того, как рассыльщик сделал последний проход (подождать минуту после
-- окончания нагрузки): записи тех, кто записался, закрываются при следующем проходе.
WITH k6_slots AS (
    SELECT id, total_seats, free_seats
    FROM slots
    WHERE id::text LIKE '7a170000-0000-4000-8000-%'
),
active_seats AS (
    SELECT s.id, coalesce(sum(b.seats_count) FILTER (WHERE b.status = 'active'), 0)
                 + coalesce(sum(b.seats_count) FILTER (WHERE b.status = 'late_cancel'), 0) AS held
    FROM k6_slots s
    LEFT JOIN bookings b ON b.slot_id = s.id
    GROUP BY s.id
),
entries AS (
    SELECT w.*
    FROM waitlist_entries w
    JOIN k6_slots s ON s.id = w.slot_id
)
SELECT 'места не уходят в минус и не превышают вместимость' AS check_name,
       count(*) FILTER (WHERE free_seats < 0 OR free_seats > total_seats) AS violations
FROM k6_slots
UNION ALL
SELECT 'свободные места = вместимость − занятые бронями',
       count(*) FILTER (WHERE s.free_seats <> s.total_seats - a.held)
FROM k6_slots s JOIN active_seats a ON a.id = s.id
UNION ALL
SELECT 'не больше одной активной брони человека на заезд',
       count(*)
FROM (
    SELECT client_id, slot_id FROM bookings
    WHERE status = 'active' AND slot_id IN (SELECT id FROM k6_slots)
    GROUP BY client_id, slot_id HAVING count(*) > 1
) dup
UNION ALL
SELECT 'не больше одной записи человека в очереди заезда',
       count(*)
FROM (
    SELECT client_id, slot_id FROM entries
    WHERE status IN ('waiting', 'notified')
    GROUP BY client_id, slot_id HAVING count(*) > 1
) dup
UNION ALL
-- Ошибка, найденная ревью: вышедший из очереди снова получал предложение. Такая запись
-- выглядит как 'notified' с уже проставленным closed_at.
SELECT 'вышедший из очереди не получает предложение',
       count(*) FILTER (WHERE status = 'notified' AND closed_at IS NOT NULL)
FROM entries
UNION ALL
SELECT 'записавшийся не остаётся в очереди',
       count(*)
FROM entries w
WHERE w.status IN ('waiting', 'notified')
  AND EXISTS (SELECT 1 FROM bookings b WHERE b.client_id = w.client_id AND b.slot_id = w.slot_id AND b.status = 'active')
UNION ALL
-- Порядок очереди: никто не получил предложение, пока раньше него стоял ждущий, которому
-- хватило бы мест (все места в сценарии — по одному). Записи, созданные за 2 секунды до
-- предложения, не считаются: их транзакция могла ещё не завершиться к моменту раздачи.
SELECT 'предложения идут по порядку очереди',
       count(*)
FROM entries offered
WHERE offered.notified_at IS NOT NULL
  AND EXISTS (
      SELECT 1 FROM entries earlier
      WHERE earlier.slot_id = offered.slot_id
        AND earlier.id <> offered.id
        AND earlier.seats_count <= offered.seats_count
        AND earlier.created_at < offered.created_at
        AND earlier.created_at < offered.notified_at - interval '2 seconds'
        -- В момент предложения earlier всё ещё ждал: не получил предложения раньше и не
        -- закрыт до этого момента.
        AND (earlier.notified_at IS NULL OR earlier.notified_at > offered.notified_at)
        AND (earlier.closed_at IS NULL OR earlier.closed_at > offered.notified_at)
  );
