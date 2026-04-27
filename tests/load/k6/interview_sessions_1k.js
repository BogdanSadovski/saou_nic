import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    interview_sessions: {
      executor: 'constant-arrival-rate',
      rate: 100,
      timeUnit: '1s',
      duration: '10s',
      preAllocatedVUs: 300,
      maxVUs: 1200,
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.02'],
    http_req_duration: ['p(95)<2500'],
  },
};

const BASE = __ENV.BASE_URL || 'http://localhost:8000/api/v1';
const TOKEN = __ENV.JWT || '';

export default function () {
  const headers = {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${TOKEN}`,
    'Idempotency-Key': `k6-${__VU}-${__ITER}`,
  };

  const createPayload = JSON.stringify({
    role: 'backend',
    level: 'senior',
    duration_minutes: 20,
    question_limit: 6,
  });

  const createRes = http.post(`${BASE}/interviews/sessions`, createPayload, { headers });
  check(createRes, { 'create status 201': (r) => r.status === 201 });

  if (createRes.status !== 201) {
    return;
  }

  const sessionId = createRes.json('data.session_id');
  if (!sessionId) {
    return;
  }

  const messagePayload = JSON.stringify({
    content: 'Used CQRS with Redis caching and p95 latency budgets; we monitored error budget burn and tuned DB indexes.',
  });

  const msgRes = http.post(`${BASE}/interviews/sessions/${sessionId}/messages`, messagePayload, { headers });
  check(msgRes, { 'message status 202': (r) => r.status === 202 });

  sleep(0.2);
}
