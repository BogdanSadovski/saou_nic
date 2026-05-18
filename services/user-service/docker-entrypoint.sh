#!/bin/sh
set -e

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL..."
for i in $(seq 1 30); do
  if pg_isready -h postgres -p 5432 -U postgres; then
    echo "PostgreSQL is ready!"
    break
  fi
  echo "Waiting... ($i/30)"
  sleep 1
done

echo "Starting user-service..."
exec ./user-service
