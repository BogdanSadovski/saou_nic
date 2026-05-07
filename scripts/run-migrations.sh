#!/bin/bash
# Apply all *.up.sql migrations from services/<svc>/migrations/ against
# the matching `${svc//-/_}` postgres database.
#
# Robust against transient postgres states (initdb shutdown/restart,
# slow initial bootstrapping with POSTGRES_MULTIPLE_DATABASES) by
# waiting for true readiness — not just the container being up — via
# `pg_isready` AND a successful trivial SELECT, then retrying each
# migration on transient errors.

set -u

CONTAINER="${POSTGRES_CONTAINER:-platform-postgres-1}"
PG_USER="${POSTGRES_USER:-postgres}"
SERVICES=(
  user-service
  interview-service
  scoring-service
  resume-service
  admin-service
  analytics-service
  notification-service
  report-service
)

# --- helpers ---------------------------------------------------------------

log() { printf '[migrate] %s\n' "$*"; }
err() { printf '[migrate] ERROR: %s\n' "$*" >&2; }

container_running() {
  [ -n "$(docker ps -q -f name="^${CONTAINER}$")" ]
}

# True only when the server fully accepts a real query (not just TCP open).
pg_truly_ready() {
  docker exec "$CONTAINER" pg_isready -U "$PG_USER" -h localhost -q >/dev/null 2>&1 || return 1
  docker exec "$CONTAINER" psql -U "$PG_USER" -d postgres -tAc 'SELECT 1' >/dev/null 2>&1
}

wait_for_postgres() {
  local timeout="${1:-180}" elapsed=0
  log "waiting for $CONTAINER to accept queries (timeout ${timeout}s)..."
  while [ "$elapsed" -lt "$timeout" ]; do
    if container_running && pg_truly_ready; then
      log "postgres is ready"
      return 0
    fi
    sleep 2
    elapsed=$((elapsed + 2))
  done
  err "postgres not ready after ${timeout}s"
  return 1
}

# Run a single .sql file, retrying on transient errors.
apply_migration() {
  local db="$1" file="$2" attempt
  for attempt in 1 2 3 4 5; do
    local out rc
    out=$(docker exec -i "$CONTAINER" psql -U "$PG_USER" -d "$db" \
      -v ON_ERROR_STOP=1 < "$file" 2>&1)
    rc=$?
    if [ "$rc" -eq 0 ]; then
      return 0
    fi
    # Treat "already exists" as success (idempotency on re-runs).
    if echo "$out" | grep -qE 'already exists|duplicate'; then
      log "  (already applied) $(basename "$file")"
      return 0
    fi
    # Retry on transient pg states.
    if echo "$out" | grep -qE 'shutting down|starting up|the database system is|connection refused|could not connect'; then
      log "  transient pg state, retry $attempt/5..."
      sleep $((attempt * 2))
      continue
    fi
    err "  $(basename "$file") failed permanently:"
    err "$out"
    return 1
  done
  err "  $(basename "$file") still failing after retries"
  return 1
}

# --- main ------------------------------------------------------------------

if ! container_running; then
  err "$CONTAINER is not running. Start the stack first (make dev-up)."
  exit 1
fi

wait_for_postgres 180 || exit 1

failed=0
for svc in "${SERVICES[@]}"; do
  db="${svc//-/_}"
  shopt -s nullglob
  migrations=("./services/$svc/migrations/"*.up.sql)
  shopt -u nullglob
  if [ "${#migrations[@]}" -eq 0 ]; then
    continue
  fi
  log ">>> $svc -> $db"
  for mig in "${migrations[@]}"; do
    if apply_migration "$db" "$mig"; then
      log "  ok: $(basename "$mig")"
    else
      failed=$((failed + 1))
    fi
  done
done

if [ "$failed" -gt 0 ]; then
  err "$failed migration(s) failed"
  exit 1
fi

log "all migrations applied"
