import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: Number(__ENV.VUS || 1),
  iterations: Number(__ENV.ITERATIONS || 1),
  thresholds: {
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<500'],
  },
};

const baseURL = __ENV.BASE_URL || 'http://127.0.0.1:8080';
const token = __ENV.TOKEN || '';
const slotID = __ENV.SLOT_ID || '55555555-5555-5555-5555-555555555555';

http.setResponseCallback(http.expectedStatuses(200, 201, 204, 400, 401, 403, 404, 409, 410, 422, 429));

export default function () {
  check(http.get(`${baseURL}/healthz`), {
    'healthz is ok': (r) => r.status === 200 && r.json('status') === 'ok',
  });

  check(http.get(`${baseURL}/slots?limit=10&offset=0`), {
    'slots listed': (r) => r.status === 200 && Array.isArray(r.json('items')),
  });

  check(http.get(`${baseURL}/instructors?limit=10&offset=0`), {
    'instructors listed': (r) => r.status === 200 && Array.isArray(r.json('items')),
  });

  if (__ENV.PHONE) {
    check(http.post(`${baseURL}/auth/request-code`, JSON.stringify({ phone: __ENV.PHONE }), jsonHeaders()), {
      'auth code requested': (r) => r.status === 200 || r.status === 429,
    });
  }

  if (__ENV.PHONE && __ENV.OTP_CODE) {
    check(http.post(`${baseURL}/auth/verify-code`, JSON.stringify({ phone: __ENV.PHONE, code: __ENV.OTP_CODE }), jsonHeaders()), {
      'auth code verified': (r) => r.status === 200 || r.status === 400,
    });
  }

  if (token) {
    const createResponse = http.post(
      `${baseURL}/bookings`,
      JSON.stringify({ slot_id: slotID, seats_count: Number(__ENV.SEATS_COUNT || 1), rental_count: Number(__ENV.RENTAL_COUNT || 0) }),
      jsonHeaders({ Authorization: `Bearer ${token}`, 'Idempotency-Key': __ENV.IDEMPOTENCY_KEY || 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa' }),
    );
    check(createResponse, {
      'booking create has documented status': (r) => [201, 409, 410, 422].includes(r.status),
    });

    const bookingID = createResponse.status === 201 ? createResponse.json('id') : __ENV.BOOKING_ID;
    if (bookingID) {
      check(http.post(`${baseURL}/bookings/${bookingID}/cancel`, null, jsonHeaders({ Authorization: `Bearer ${token}` })), {
        'booking cancel has documented status': (r) => [200, 409, 422].includes(r.status),
      });
    }
  }

  sleep(1);
}

function jsonHeaders(headers = {}) {
  return { headers: { 'Content-Type': 'application/json', ...headers } };
}
