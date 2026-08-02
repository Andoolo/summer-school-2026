import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    booking_300_vu: {
      executor: 'constant-vus',
      vus: Number(__ENV.VUS || 300),
      duration: __ENV.DURATION || '1m',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.10'],
    http_req_duration: ['p(95)<1000'],
    checks: ['rate>0.95'],
  },
};

const baseURL = __ENV.BASE_URL || 'http://127.0.0.1:8080';
const tokenPrefix = __ENV.TOKEN_PREFIX || 'vu-token-';
const slotID = __ENV.SLOT_ID || '55555555-5555-5555-5555-555555555555';

http.setResponseCallback(http.expectedStatuses(201, 401, 409, 410, 422));

export default function () {
  const token = __ENV.TOKEN || `${tokenPrefix}${__VU}`;
  const response = http.post(
    `${baseURL}/bookings`,
    JSON.stringify({ slot_id: slotID, seats_count: 1, rental_count: Number(__ENV.RENTAL_COUNT || 0) }),
    {
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
        'Idempotency-Key': deterministicUUID(__VU, __ITER),
      },
    },
  );

  check(response, {
    'create booking documented status': (r) => [201, 401, 409, 410, 422].includes(r.status),
    'no negative availability in response': (r) => {
      if (r.status !== 201) return true;
      return r.json('slot.free_seats') >= 0 && r.json('slot.free_rental_boards') >= 0;
    },
  });

  sleep(1);
}

function deterministicUUID(vu, iter) {
  const suffix = String((vu * 1000000) + iter).padStart(12, '0').slice(-12);
  return `10000000-0000-4000-8000-${suffix}`;
}
