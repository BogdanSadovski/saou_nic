#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8000/api/v1}"
JWT="${JWT:-}"

if [[ -z "${JWT}" ]]; then
  echo "JWT is required"
  exit 1
fi

create_session() {
  curl -sS -X POST "${BASE_URL}/interviews/sessions" \
    -H "Authorization: Bearer ${JWT}" \
    -H "Content-Type: application/json" \
    -H "Idempotency-Key: chaos-create-$(date +%s%N)" \
    -d '{"role":"backend","level":"senior","duration_minutes":15,"question_limit":5}'
}

echo "[chaos] creating session"
SESSION_ID=$(create_session | sed -n 's/.*"session_id":"\([^"]*\)".*/\1/p')
if [[ -z "${SESSION_ID}" ]]; then
  echo "failed to create session"
  exit 1
fi

echo "[chaos] session=${SESSION_ID}"

for i in {1..12}; do
  CODE=$(curl -s -o /tmp/chaos_msg.json -w "%{http_code}" -X POST "${BASE_URL}/interviews/sessions/${SESSION_ID}/messages" \
    -H "Authorization: Bearer ${JWT}" \
    -H "Content-Type: application/json" \
    -H "Idempotency-Key: chaos-msg-${i}" \
    -d '{"content":"Simulating resilience: retry logic on 429/500 and timeout budget handling."}')
  echo "message[$i] => ${CODE}"
  sleep 0.2
done

FINISH_CODE=$(curl -s -o /tmp/chaos_finish.json -w "%{http_code}" -X POST "${BASE_URL}/interviews/sessions/${SESSION_ID}/finish" \
  -H "Authorization: Bearer ${JWT}" \
  -H "Content-Type: application/json")

echo "finish => ${FINISH_CODE}"

for t in {1..30}; do
  REPORT_CODE=$(curl -s -o /tmp/chaos_report.json -w "%{http_code}" -X GET "${BASE_URL}/interviews/sessions/${SESSION_ID}/report" \
    -H "Authorization: Bearer ${JWT}")
  if [[ "${REPORT_CODE}" == "200" ]]; then
    echo "report ready in ${t}s"
    cat /tmp/chaos_report.json
    exit 0
  fi
  sleep 1
done

echo "report not ready within timeout"
exit 1
