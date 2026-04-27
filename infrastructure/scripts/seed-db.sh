#!/usr/bin/env bash
#
# seed-db.sh - Seed the database with test data
#
# Usage:
#   ./seed-db.sh [OPTIONS]
#
# Options:
#   -e, --environment ENV    Target environment (local|staging|production). Default: local
#   -d, --database DB        Database name. Default: interview_platform
#   -h, --host HOST          Database host. Default: localhost
#   -p, --port PORT          Database port. Default: 5432
#   -u, --user USER          Database user. Default: postgres
#   -f, --file FILE          Specific seed file to run. Default: all files
#   --dry-run                Show what would be done without executing
#   --help                   Show this help message
#
# Examples:
#   ./seed-db.sh                         # Seed local database with all seed files
#   ./seed-db.sh -e staging               # Seed staging database
#   ./seed-db.sh -f seeds/users.sql      # Run a specific seed file

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

ENVIRONMENT="local"
DATABASE="interview_platform"
DB_HOST="localhost"
DB_PORT="5432"
DB_USER="postgres"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
SEED_FILE=""
DRY_RUN=false

# Seed directories to look for
SEED_DIRS=(
  "${PROJECT_ROOT}/services/user-service/seeds"
  "${PROJECT_ROOT}/services/resume-service/seeds"
  "${PROJECT_ROOT}/services/github-service/seeds"
  "${PROJECT_ROOT}/services/interview-service/seeds"
  "${PROJECT_ROOT}/services/scoring-service/seeds"
  "${PROJECT_ROOT}/services/report-service/seeds"
  "${PROJECT_ROOT}/services/notification-service/seeds"
  "${PROJECT_ROOT}/services/analytics-service/seeds"
  "${PROJECT_ROOT}/services/admin-service/seeds"
)

# ─── Functions ────────────────────────────────────────────────────────────
usage() {
  head -15 "$0" | tail -13
  exit 0
}

validate_prerequisites() {
  log_info "Validating prerequisites..."

  if ! command -v psql &>/dev/null; then
    log_error "psql (PostgreSQL client) is not installed or not in PATH"
    exit 1
  fi

  log_success "PostgreSQL client found"
}

check_database_connection() {
  log_info "Checking database connection..."

  local db_url="postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DATABASE}"

  if [ "${DRY_RUN}" = true ]; then
    echo "  [DRY-RUN] psql ${db_url} -c 'SELECT 1;'"
    return 0
  fi

  if ! PGPASSWORD="${DB_PASSWORD}" psql \
    -h "${DB_HOST}" \
    -p "${DB_PORT}" \
    -U "${DB_USER}" \
    -d "${DATABASE}" \
    -c "SELECT 1;" &>/dev/null; then
    log_error "Cannot connect to database at ${DB_HOST}:${DB_PORT}/${DATABASE}"
    log_error "Check your connection settings and ensure the database is running"
    exit 1
  fi

  log_success "Database connection successful"
}

find_seed_files() {
  local seed_files=()

  for dir in "${SEED_DIRS[@]}"; do
    if [ -d "${dir}" ]; then
      while IFS= read -r -d '' file; do
        seed_files+=("${file}")
      done < <(find "${dir}" -name "*.sql" -print0 2>/dev/null | sort -z)
    fi
  done

  # If a specific file is requested, use only that
  if [ -n "${SEED_FILE}" ]; then
    if [ ! -f "${SEED_FILE}" ]; then
      # Try to find it relative to project root
      if [ -f "${PROJECT_ROOT}/${SEED_FILE}" ]; then
        seed_files=("${PROJECT_ROOT}/${SEED_FILE}")
      else
        log_error "Seed file not found: ${SEED_FILE}"
        exit 1
      fi
    else
      seed_files=("${SEED_FILE}")
    fi
  fi

  printf '%s\n' "${seed_files[@]}"
}

run_seed_file() {
  local file="$1"
  local filename
  filename="$(basename "${file}")"

  log_info "Running seed file: ${filename}"

  if [ "${DRY_RUN}" = true ]; then
    echo "  [DRY-RUN] psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${DATABASE} -f ${file}"
    return 0
  fi

  if ! PGPASSWORD="${DB_PASSWORD}" psql \
    -h "${DB_HOST}" \
    -p "${DB_PORT}" \
    -U "${DB_USER}" \
    -d "${DATABASE}" \
    -f "${file}" \
    -v ON_ERROR_STOP=1 2>&1; then
    log_error "Failed to run seed file: ${filename}"
    return 1
  fi

  log_success "Seed file completed: ${filename}"
}

truncate_tables() {
  log_warn "This will truncate all tables. Skipping in dry-run mode."

  if [ "${DRY_RUN}" = true ]; then
    echo "  [DRY-RUN] TRUNCATE all tables..."
    return 0
  fi

  log_info "Truncating tables..."

  PGPASSWORD="${DB_PASSWORD}" psql \
    -h "${DB_HOST}" \
    -p "${DB_PORT}" \
    -U "${DB_USER}" \
    -d "${DATABASE}" \
    -c "
    DO \$\$
    DECLARE
      r RECORD;
    BEGIN
      FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
        EXECUTE 'TRUNCATE TABLE ' || quote_ident(r.tablename) || ' RESTART IDENTITY CASCADE';
      END LOOP;
    END \$\$;
    " 2>/dev/null || log_warn "Some tables could not be truncated (may not exist)"

  log_success "Tables truncated"
}

# ─── Argument Parsing ─────────────────────────────────────────────────────
parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -e|--environment)
        ENVIRONMENT="$2"
        # Set defaults based on environment
        case "${ENVIRONMENT}" in
          staging)
            DB_HOST="${DB_HOST:-staging-db.interview-platform.io}"
            DATABASE="${DATABASE:-interview_platform_staging}"
            ;;
          production)
            DB_HOST="${DB_HOST:-production-db.interview-platform.io}"
            DATABASE="${DATABASE:-interview_platform_prod}"
            ;;
          *)
            DB_HOST="${DB_HOST:-localhost}"
            ;;
        esac
        shift 2
        ;;
      -d|--database)
        DATABASE="$2"
        shift 2
        ;;
      -h|--host)
        DB_HOST="$2"
        shift 2
        ;;
      -p|--port)
        DB_PORT="$2"
        shift 2
        ;;
      -u|--user)
        DB_USER="$2"
        shift 2
        ;;
      -f|--file)
        SEED_FILE="$2"
        shift 2
        ;;
      --dry-run)
        DRY_RUN=true
        shift
        ;;
      --help)
        usage
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
  echo "  AI Interview Platform - Database Seed"
  echo "============================================="
  echo "  Environment : ${ENVIRONMENT}"
  echo "  Database    : ${DATABASE}"
  echo "  Host        : ${DB_HOST}:${DB_PORT}"
  echo "  User        : ${DB_USER}"
  echo "  Seed File   : ${SEED_FILE:-all}"
  echo "  Dry Run     : ${DRY_RUN}"
  echo "============================================="
  echo ""

  validate_prerequisites
  check_database_connection

  # Ask for confirmation
  if [ "${ENVIRONMENT}" = "production" ] && [ "${DRY_RUN}" = false ]; then
    log_warn "You are about to seed the PRODUCTION database!"
    read -rp "Type 'yes' to confirm: " confirm
    if [ "${confirm}" != "yes" ]; then
      log_info "Database seeding cancelled"
      exit 0
    fi
  fi

  # Find seed files
  local seed_files
  mapfile -t seed_files < <(find_seed_files)

  if [ ${#seed_files[@]} -eq 0 ]; then
    log_warn "No seed files found. Create .sql files in service seed directories."
    log_info "Expected locations:"
    for dir in "${SEED_DIRS[@]}"; do
      echo "  - ${dir}"
    done
    exit 0
  fi

  log_info "Found ${#seed_files[@]} seed file(s)"

  # Run seed files
  local failed=0
  for file in "${seed_files[@]}"; do
    if ! run_seed_file "${file}"; then
      failed=$((failed + 1))
    fi
  done

  if [ ${failed} -gt 0 ]; then
    log_error "${failed} seed file(s) failed"
    exit 1
  fi

  echo ""
  log_success "Database seeding completed successfully!"
  log_info "Environment : ${ENVIRONMENT}"
  log_info "Database    : ${DATABASE}"
  log_info "Files run   : ${#seed_files[@]}"
}

main "$@"
