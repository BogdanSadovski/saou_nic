#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8000/api/v1}"
JWT="${JWT:-}"

if [[ -z "${JWT}" ]]; then
  echo "JWT is required"
  exit 1
fi

echo "[e2e] start -> chat -> finish -> report"

SESSION_JSON=$(curl -sS -X POST "${BASE_URL}/interviews/sessions" \
  -H "Authorization: Bearer ${JWT}" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: e2e-create-$(date +%s%N)" \
  -d '{"role":"backend","level":"middle","duration_minutes":20,"question_limit":4}')

SESSION_ID=$(echo "${SESSION_JSON}" | sed -n 's/.*"session_id":"\([^"]*\)".*/\1/p')
if [[ -z "${SESSION_ID}" ]]; then
  echo "cannot parse session id"
  echo "${SESSION_JSON}"
  exit 1
fi

echo "session=${SESSION_ID}"

for i in {1..3}; do
  CODE=$(curl -s -o /tmp/e2e_message.json -w "%{http_code}" -X POST "${BASE_URL}/interviews/sessions/${SESSION_ID}/messages" \
    -H "Authorization: Bearer ${JWT}" \
    -H "Content-Type: application/json" \
    -H "Idempotency-Key: e2e-msg-${i}-$(date +%s%N)" \
    -d '{"content":"Implemented optimistic locking, retries with exponential backoff, and query tuning by p95 metrics."}')
  echo "message[$i] => ${CODE}"
done

curl -sS -X POST "${BASE_URL}/interviews/sessions/${SESSION_ID}/finish" \
  -H "Authorization: Bearer ${JWT}" \
  -H "Content-Type: application/json" >/tmp/e2e_finish.json

for t in {1..30}; do
  CODE=$(curl -s -o /tmp/e2e_report.json -w "%{http_code}" -X GET "${BASE_URL}/interviews/sessions/${SESSION_ID}/report" \
    -H "Authorization: Bearer ${JWT}")
  if [[ "${CODE}" == "200" ]]; then
    echo "report ready"
    cat /tmp/e2e_report.json
    exit 0
  fi
  sleep 1
done

echo "report timeout"
exit 1
