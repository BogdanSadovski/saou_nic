#!/bin/bash
# Promote a user to the 'admin' role so they can access the Admin panel.
#
# Usage:
#   ./scripts/grant-admin.sh <email>
#
# Requires the platform stack to be running (`make dev-up`). After this
# completes, the user must log out and log back in for their JWT to
# carry the new role.

set -eu

CONTAINER="${POSTGRES_CONTAINER:-platform-postgres-1}"
PG_USER="${POSTGRES_USER:-postgres}"
DB="${POSTGRES_USER_DB:-user_service}"

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <email>" >&2
  exit 1
fi

EMAIL="$1"

if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER}$"; then
  echo "ERROR: container '$CONTAINER' is not running. Start the stack with 'make dev-up'." >&2
  exit 1
fi

# Verify the user exists before mutating, so we fail loudly if the email
# is wrong instead of silently doing nothing.
EXISTS=$(docker exec -i "$CONTAINER" psql -U "$PG_USER" -d "$DB" -tAc \
  "SELECT 1 FROM users WHERE email = '$EMAIL' LIMIT 1" 2>/dev/null || true)

if [ "$EXISTS" != "1" ]; then
  echo "ERROR: no user with email '$EMAIL' in $DB" >&2
  exit 1
fi

docker exec -i "$CONTAINER" psql -U "$PG_USER" -d "$DB" -c \
  "UPDATE users SET role = 'admin' WHERE email = '$EMAIL'" >/dev/null

echo "Granted admin role to $EMAIL."
echo "Tell them to log out and log back in for the role to take effect."
