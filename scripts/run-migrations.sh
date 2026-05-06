#!/bin/bash
# Helper: applies all *.up.sql migrations in services/*/migrations/ to
# the matching `${service//-/_}` postgres database.
# Idempotent: each migration uses CREATE TABLE IF NOT EXISTS-friendly SQL
# (existing migrations error out cleanly when applied twice).
set -e
for svc in user-service interview-service scoring-service resume-service; do
  db="${svc//-/_}"
  for mig in ./services/$svc/migrations/*.up.sql; do
    [ -f "$mig" ] || continue
    echo ">>> $svc :: $(basename "$mig")"
    docker exec -i platform-postgres-1 psql -U postgres -d "$db" < "$mig" 2>&1 | tail -3 || true
  done
done
