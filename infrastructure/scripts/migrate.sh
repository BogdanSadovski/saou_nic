#!/usr/bin/env bash
#
# migrate.sh - Run database migrations using golang-migrate
#
# Usage:
#   ./migrate.sh [OPTIONS] [COMMAND]
#
# Commands:
#   up [N]              Apply all or next N migrations (default: all)
#   down [N]            Revert all or last N migrations (default: 1)
#   redo                Revert and re-apply the last migration
#   status              Show current migration status
#   version             Show current migration version
#   force V             Force set version to V (no migration run)
#
# Options:
#   -s, --service SVC    Run migrations for a specific service
#   -d, --database URL   Database URL. Default: from environment
#   --dir PATH           Migrations directory. Default: per service
#   --help               Show this help message
#
# Examples:
#   ./migrate.sh up                         # Run all pending migrations
#   ./migrate.sh up 2                       # Run next 2 migrations
#   ./migrate.sh down                       # Revert last migration
#   ./migrate.sh status                     # Show migration status
#   ./migrate.sh -s user-service up         # Migrate user-service only
#   ./migrate.sh -s user-service status     # Check user-service migration status

set -euo pipefail

# ─── Colors ───────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ─── Logging ──────────────────────────────────────────────────────────────
log_info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
log_success() { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $*"; }

# ─── Configuration ────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

SERVICE=""
DATABASE_URL="${DATABASE_URL:-}"
CUSTOM_MIGRATIONS_DIR=""
COMMAND="up"
COMMAND_ARG=""

# Service migration directories
declare -A SERVICE_MIGRATIONS=(
  [user-service]="services/user-service/migrations"
  [resume-service]="services/resume-service/migrations"
  [github-service]="services/github-service/migrations"
  [interview-service]="services/interview-service/migrations"
  [scoring-service]="services/scoring-service/migrations"
  [report-service]="services/report-service/migrations"
  [notification-service]="services/notification-service/migrations"
  [analytics-service]="services/analytics-service/migrations"
  [admin-service]="services/admin-service/migrations"
)

ALL_SERVICES=(
  user-service
  resume-service
  github-service
  interview-service
  scoring-service
  report-service
  notification-service
  analytics-service
  admin-service
)

# ─── Functions ────────────────────────────────────────────────────────────
usage() {
  head -16 "$0" | tail -14
  exit 0
}

validate_prerequisites() {
  log_info "Validating prerequisites..."

  if command -v migrate &>/dev/null; then
    log_success "golang-migrate CLI found: $(migrate -version 2>&1 | head -1 || echo 'version unknown')"
    return 0
  fi

  # Try to install it
  log_warn "golang-migrate CLI not found"
  log_info "Attempting to install golang-migrate..."

  if command -v go &>/dev/null; then
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest 2>/dev/null || {
      log_error "Failed to install golang-migrate"
      log_info "Install manually: https://github.com/golang-migrate/migrate#cli-installation"
      exit 1
    }
    log_success "golang-migrate installed"
    return 0
  fi

  log_error "Install golang-migrate CLI: https://github.com/golang-migrate/migrate"
  exit 1
}

get_database_url() {
  if [ -n "${DATABASE_URL}" ]; then
    return 0
  fi

  # Try to read from .env file
  local env_file="${PROJECT_ROOT}/.env"
  if [ -f "${env_file}" ]; then
    DATABASE_URL="$(grep '^DATABASE_URL=' "${env_file}" 2>/dev/null | cut -d'=' -f2- | tr -d '"' || echo "")"
  fi

  if [ -z "${DATABASE_URL}" ]; then
    # Construct from individual variables
    local db_host="${DB_HOST:-localhost}"
    local db_port="${DB_PORT:-5432}"
    local db_name="${DB_NAME:-interview_platform}"
    local db_user="${DB_USER:-postgres}"
    local db_pass="${DB_PASSWORD:-postgres}"

    DATABASE_URL="postgresql://${db_user}:${db_pass}@${db_host}:${db_port}/${db_name}?sslmode=disable"
  fi

  if [ -z "${DATABASE_URL}" ]; then
    log_error "DATABASE_URL is not set. Set it as an environment variable or in .env"
    exit 1
  fi

  log_info "Using database URL: ${DATABASE_URL}"
}

get_service_migrations_dir() {
  local svc="$1"
  local migrations_dir=""

  if [ -n "${CUSTOM_MIGRATIONS_DIR}" ]; then
    echo "${CUSTOM_MIGRATIONS_DIR}"
    return 0
  fi

  migrations_dir="${PROJECT_ROOT}/${SERVICE_MIGRATIONS["${svc}"]}"

  if [ ! -d "${migrations_dir}" ]; then
    log_warn "Migrations directory not found for ${svc}: ${migrations_dir}"
    return 1
  fi

  echo "${migrations_dir}"
  return 0
}

check_database_connection() {
  log_info "Checking database connection..."

  if ! command -v psql &>/dev/null; then
    log_warn "psql not available, skipping connection check"
    return 0
  fi

  # Parse URL for connection check
  local db_url="${DATABASE_URL}"

  # Extract components from postgresql:// URL
  local db_host db_port db_name db_user db_pass
  db_host="$(echo "${db_url}" | sed -n 's|.*://\([^:@]*\)[:@].*|\1|p')"
  db_port="$(echo "${db_url}" | sed -n 's|.*:[0-9]*/|\1|p' | head -1)"
  db_name="$(echo "${db_url}" | sed -n 's|.*/\([^?]*\).*|\1|p')"
  db_user="$(echo "${db_url}" | sed -n 's|.*//\([^:@]*\):.*|\1|p')"
  db_pass="$(echo "${db_url}" | sed -n 's|.*//[^:]*:\([^@]*\)@.*|\1|p')"

  if PGPASSWORD="${db_pass}" psql \
    -h "${db_host}" \
    -p "${db_port}" \
    -U "${db_user}" \
    -d "${db_name}" \
    -c "SELECT 1;" &>/dev/null; then
    log_success "Database connection successful"
  else
    log_error "Cannot connect to database"
    return 1
  fi
}

run_migration() {
  local svc="$1"
  local cmd="$2"
  local migrations_dir
  local schema_name

  migrations_dir="$(get_service_migrations_dir "${svc}")" || return 0
  schema_name="${svc//-/_}"

  log_info "Running migration [${cmd}] for ${svc}..."
  log_info "Migrations: ${migrations_dir}"

  # Run migrate command
  if ! migrate \
    -path "${migrations_dir}" \
    -database "${DATABASE_URL}" \
    -source "file://${migrations_dir}" \
    "${cmd}" 2>&1; then
    log_error "Migration failed for ${svc}"
    return 1
  fi

  log_success "Migration completed for ${svc}"
}

show_migration_status() {
  local svc="$1"
  local migrations_dir

  migrations_dir="$(get_service_migrations_dir "${svc}")" || return 0

  log_info "Migration status for ${svc}:"

  # Show version
  migrate \
    -path "${migrations_dir}" \
    -database "${DATABASE_URL}" \
    version 2>&1 || log_warn "Could not retrieve version for ${svc}"

  # Show migration files
  echo ""
  log_info "Available migrations:"
  for f in "${migrations_dir}"/*.up.sql; do
    if [ -f "${f}" ]; then
      echo "  [ ] $(basename "${f}")"
    fi
  done
  echo ""
}

# ─── Argument Parsing ─────────────────────────────────────────────────────
parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -s|--service)
        SERVICE="$2"
        shift 2
        ;;
      -d|--database)
        DATABASE_URL="$2"
        shift 2
        ;;
      --dir)
        CUSTOM_MIGRATIONS_DIR="$2"
        shift 2
        ;;
      --help)
        usage
        ;;
      up|down|redo|status|version|force)
        COMMAND="$1"
        shift
        if [ $# -gt 0 ] && [[ ! "$1" =~ ^- ]]; then
          COMMAND_ARG="$1"
          shift
        fi
        ;;
      *)
        log_error "Unknown option: $1"
        usage
        ;;
    esac
  done
}

# ─── Main ─────────────────────────────────────────────────────────────────
main() {
  parse_args "$@"

  echo "============================================="
  echo "  AI Interview Platform - Migrations"
  echo "============================================="
  echo "  Command   : ${COMMAND} ${COMMAND_ARG}"
  echo "  Service   : ${SERVICE:-all}"
  echo "============================================="
  echo ""

  validate_prerequisites
  get_database_url
  check_database_connection

  # Build migration command
  local migrate_cmd="${COMMAND}"
  if [ -n "${COMMAND_ARG}" ]; then
    migrate_cmd="${COMMAND} ${COMMAND_ARG}"
  fi

  # Determine services to migrate
  local services_to_migrate=()
  if [ -n "${SERVICE}" ]; then
    services_to_migrate=("${SERVICE}")
  else
    services_to_migrate=("${ALL_SERVICES[@]}")
  fi

  # Execute migrations
  local failed=0
  for svc in "${services_to_migrate[@]}"; do
    if [ "${COMMAND}" = "status" ]; then
      show_migration_status "${svc}"
    else
      if ! run_migration "${svc}" "${migrate_cmd}"; then
        failed=$((failed + 1))
      fi
      echo ""
    fi
  done

  if [ ${failed} -gt 0 ]; then
    log_error "${failed} service(s) had migration failures"
    exit 1
  fi

  if [ "${COMMAND}" != "status" ]; then
    log_success "All migrations completed successfully!"
  fi
}

main "$@"
