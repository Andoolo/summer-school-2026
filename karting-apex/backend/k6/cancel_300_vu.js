import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    cancel_300_vu: {
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
const token = __ENV.TOKEN || '';
const bookingIDs = (__ENV.BOOKING_IDS || '').split(',').map((value) => value.trim()).filter(Boolean);

http.setResponseCallback(http.expectedStatuses(200, 401, 403, 404, 409, 422));

export default function () {
  if (!token || bookingIDs.length === 0) {
    check(null, { 'TOKEN and BOOKING_IDS are configured': () => false });
    return;
  }

  const bookingID = bookingIDs[(__VU + __ITER) % bookingIDs.length];
  const response = http.post(`${baseURL}/bookings/${bookingID}/cancel`, null, {
    headers: { Authorization: `Bearer ${token}` },
  });

  check(response, {
    'cancel documented status': (r) => [200, 401, 403, 404, 409, 422].includes(r.status),
    'cancel does not expose invalid status': (r) => {
      if (r.status !== 200) return true;
      return ['cancelled', 'late_cancel'].includes(r.json('status'));
    },
  });

  sleep(1);
}
