// Нагрузка на лист ожидания: три заполненных заезда по 8 мест.
//
//   VU 1–24   — «держатели» мест (vu-token-1..24, по 8 на заезд): отменяют бронь и сразу
//               пытаются забронировать снова. Так проверяется, что предложенное место
//               не закреплено: его может перехватить любой.
//   VU 25–224 — «ждущие» (vu-token-101..300): встают в очередь, следят за статусом,
//               бронируют по предложению, иногда выходят из очереди или отменяют бронь.
//
// Данные готовит `go run ./cmd/k6seed`. Корректность проверяется после прогона запросами
// из k6/waitlist_invariants.sql: нагрузочный тест сам по себе ловит только ошибки и время.
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter } from 'k6/metrics';

const holders = 24;
const waiters = Number(__ENV.WAITERS || 200);

export const options = {
  scenarios: {
    waitlist: {
      executor: 'constant-vus',
      vus: holders + waiters,
      duration: __ENV.DURATION || '3m',
    },
  },
  thresholds: {
    // 409 (мест нет, уже записан, места появились) — ожидаемые ответы, не ошибки.
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<1000'],
    checks: ['rate>0.99'],
  },
};

const baseURL = __ENV.BASE_URL || 'http://127.0.0.1:8080';
const slots = [
  '7a170000-0000-4000-8000-000000000001',
  '7a170000-0000-4000-8000-000000000002',
  '7a170000-0000-4000-8000-000000000003',
];

const joined = new Counter('waitlist_joined');
const left = new Counter('waitlist_left');
const offersSeen = new Counter('waitlist_offers_seen');
const bookedFromOffer = new Counter('waitlist_booked_from_offer');
const offerLostToOthers = new Counter('waitlist_offer_seat_taken');
const holderCancels = new Counter('holder_cancels');
const holderRebooks = new Counter('holder_rebooks');

// Последнее увиденное предложение этого VU (у каждого VU своя копия переменной).
let lastOffer = null;

http.setResponseCallback(http.expectedStatuses(200, 201, 204, 409, 422));

export default function () {
  if (__VU <= holders) {
    holder(__VU);
  } else {
    waiter(__VU);
  }
}

function holder(vu) {
  const token = `vu-token-${vu}`;
  const slotID = slots[Math.floor((vu - 1) / 8)];
  const own = activeBooking(token, slotID);
  if (own) {
    sleep(1 + Math.random() * 3);
    const res = http.post(`${baseURL}/bookings/${own}/cancel`, null, auth(token));
    check(res, { 'holder cancel: 200 or already cancelled': (r) => [200, 409].includes(r.status) });
    if (res.status === 200) holderCancels.add(1);
    sleep(Math.random() * 2);
    return;
  }
  const res = book(token, slotID);
  check(res, { 'holder rebook: 201 or slot full': (r) => [201, 409].includes(r.status) });
  if (res.status === 201) holderRebooks.add(1);
  sleep(1 + Math.random() * 2);
}

function waiter(vu) {
  const token = `vu-token-${vu + 76}`;
  const slotID = slots[vu % slots.length];
  const statusRes = http.get(`${baseURL}/slots/${slotID}/waitlist`, auth(token));
  check(statusRes, { 'waitlist status 200': (r) => r.status === 200 });
  if (statusRes.status !== 200) {
    sleep(1);
    return;
  }
  const entry = statusRes.json('entry');

  if (!entry) {
    const res = http.post(`${baseURL}/slots/${slotID}/waitlist`, JSON.stringify({ seats_count: 1 }), auth(token));
    check(res, { 'join: documented status': (r) => [200, 201, 409].includes(r.status) });
    if (res.status === 201) joined.add(1);
    if (res.status === 409) {
      const code = res.json('code');
      if (code === 'seats_available') {
        book(token, slotID);
      } else if (code === 'double_booking') {
        // Уже записан (по предложению или перехватил место) — через время отменяет, чтобы
        // места продолжали освобождаться.
        maybeCancel(token, slotID, 0.3);
      }
    }
  } else if (entry.status === 'notified') {
    check(entry, { 'offer has deadline': (e) => Boolean(e.offer_expires_at) });
    const res = book(token, slotID);
    check(res, { 'book from offer: 201 or seat taken': (r) => [201, 409].includes(r.status) });
    // Одно предложение считается один раз: с перехваченным местом ждущий пробует снова
    // на каждом шаге, пока предложение не истечёт.
    const firstAttempt = lastOffer !== entry.offer_expires_at;
    lastOffer = entry.offer_expires_at;
    if (firstAttempt) offersSeen.add(1);
    if (res.status === 201) {
      bookedFromOffer.add(1);
    } else if (firstAttempt && res.status === 409 && res.json('code') === 'slot_full') {
      // Место перехватил другой — предложение его не закрепляло.
      offerLostToOthers.add(1);
    }
    // 409 double_booking — уже записан по этому предложению: запись в очереди закроется
    // при следующем проходе рассыльщика, повторно не считаем.
  } else if (Math.random() < 0.05) {
    const res = http.del(`${baseURL}/slots/${slotID}/waitlist`, null, auth(token));
    check(res, { 'leave: 204': (r) => r.status === 204 });
    left.add(1);
  }
  sleep(1 + Math.random());
}

function maybeCancel(token, slotID, probability) {
  if (Math.random() >= probability) return;
  const own = activeBooking(token, slotID);
  if (!own) return;
  const res = http.post(`${baseURL}/bookings/${own}/cancel`, null, auth(token));
  check(res, { 'waiter cancel: 200 or already cancelled': (r) => [200, 409].includes(r.status) });
}

function activeBooking(token, slotID) {
  const res = http.get(`${baseURL}/bookings?status=active&limit=50`, auth(token));
  check(res, { 'list bookings 200': (r) => r.status === 200 });
  if (res.status !== 200) return null;
  const found = (res.json('items') || []).find((b) => b.slot_id === slotID);
  return found ? found.id : null;
}

function book(token, slotID) {
  return http.post(
    `${baseURL}/bookings`,
    JSON.stringify({ slot_id: slotID, seats_count: 1, rental_count: 0 }),
    { headers: { ...auth(token).headers, 'Idempotency-Key': uuid() } },
  );
}

function auth(token) {
  return { headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' } };
}

function uuid() {
  const hex = () => Math.floor(Math.random() * 16).toString(16);
  const part = (n) => Array.from({ length: n }, hex).join('');
  return `${part(8)}-${part(4)}-4${part(3)}-8${part(3)}-${part(12)}`;
}
