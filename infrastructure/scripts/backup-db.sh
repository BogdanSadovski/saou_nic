#!/usr/bin/env bash
#
# backup-db.sh - Backup the PostgreSQL database
#
# Usage:
#   ./backup-db.sh [OPTIONS]
#
# Options:
#   -d, --database DB        Database name. Default: interview_platform
#   -h, --host HOST          Database host. Default: localhost
#   -p, --port PORT          Database port. Default: 5432
#   -u, --user USER          Database user. Default: postgres
#   -o, --output DIR         Output directory for backups. Default: ./backups
#   -f, --filename NAME      Custom backup filename. Default: auto-generated
#   --compress               Compress the backup with gzip
#   --clean                  Delete backups older than N days
#   --retention DAYS         Number of days to keep backups. Default: 30
#   --help                   Show this help message
#
# Examples:
#   ./backup-db.sh                           # Create backup with default settings
#   ./backup-db.sh --compress                 # Create compressed backup
#   ./backup-db.sh --clean --retention 7      # Clean backups older than 7 days
#   ./backup-db.sh -o /mnt/backups            # Save to custom directory

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

DATABASE="${DB_NAME:-interview_platform}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
BACKUP_DIR="${PROJECT_ROOT}/infrastructure/backups"
CUSTOM_FILENAME=""
COMPRESS=false
CLEAN=false
RETENTION_DAYS=30

# ─── Functions ────────────────────────────────────────────────────────────
usage() {
  head -15 "$0" | tail -13
  exit 0
}

validate_prerequisites() {
  log_info "Validating prerequisites..."

  local missing=0

  if ! command -v pg_dump &>/dev/null; then
    log_error "pg_dump is not installed or not in PATH"
    missing=1
  fi

  if ! command -v psql &>/dev/null; then
    log_error "psql is not installed or not in PATH"
    missing=1
  fi

  if [ ${missing} -eq 1 ]; then
    log_error "Install PostgreSQL client tools to proceed"
    exit 1
  fi

  log_success "PostgreSQL client tools found"
}

setup_backup_dir() {
  if [ ! -d "${BACKUP_DIR}" ]; then
    log_info "Creating backup directory: ${BACKUP_DIR}"
    mkdir -p "${BACKUP_DIR}"
  fi

  # Verify write permissions
  if [ ! -w "${BACKUP_DIR}" ]; then
    log_error "Cannot write to backup directory: ${BACKUP_DIR}"
    exit 1
  fi

  log_success "Backup directory ready: ${BACKUP_DIR}"
}

generate_filename() {
  if [ -n "${CUSTOM_FILENAME}" ]; then
    echo "${CUSTOM_FILENAME}"
    return
  fi

  local timestamp
  timestamp="$(date +%Y%m%d_%H%M%S)"
  echo "${DATABASE}_${timestamp}.sql"
}

check_database_connection() {
  log_info "Checking database connection..."

  if ! PGPASSWORD="${DB_PASSWORD}" psql \
    -h "${DB_HOST}" \
    -p "${DB_PORT}" \
    -U "${DB_USER}" \
    -d "${DATABASE}" \
    -c "SELECT 1;" &>/dev/null; then
    log_error "Cannot connect to database at ${DB_HOST}:${DB_PORT}/${DATABASE}"
    exit 1
  fi

  log_success "Database connection successful"
}

get_database_size() {
  local size
  size="$(PGPASSWORD="${DB_PASSWORD}" psql \
    -h "${DB_HOST}" \
    -p "${DB_PORT}" \
    -U "${DB_USER}" \
    -d "${DATABASE}" \
    -t -c "SELECT pg_size_pretty(pg_database_size('${DATABASE}'));" 2>/dev/null | xargs)"
  echo "${size}"
}

perform_backup() {
  local filename="$1"
  local backup_path="${BACKUP_DIR}/${filename}"

  log_info "Starting backup..."
  log_info "Database : ${DATABASE}"
  log_info "Host     : ${DB_HOST}:${DB_PORT}"
  log_info "Size     : $(get_database_size)"
  log_info "Output   : ${backup_path}"

  local start_time
  start_time="$(date +%s)"

  # Run pg_dump
  if ! PGPASSWORD="${DB_PASSWORD}" pg_dump \
    -h "${DB_HOST}" \
    -p "${DB_PORT}" \
    -U "${DB_USER}" \
    -d "${DATABASE}" \
    --verbose \
    --no-owner \
    --no-privileges \
    --format=plain \
    --file="${backup_path}" 2>&1; then
    log_error "Backup failed!"
    rm -f "${backup_path}"
    exit 1
  fi

  local end_time
  end_time="$(date +%s)"
  local duration=$((end_time - start_time))

  # Compress if requested
  if [ "${COMPRESS}" = true ]; then
    log_info "Compressing backup..."
    if gzip -9 "${backup_path}"; then
      backup_path="${backup_path}.gz"
      log_success "Backup compressed: $(du -sh "${backup_path}" | cut -f1)"
    else
      log_warn "Compression failed, keeping uncompressed backup"
    fi
  fi

  local file_size
  file_size="$(du -sh "${backup_path}" | cut -f1)"

  echo ""
  log_success "Backup completed successfully!"
  log_info "File     : ${backup_path}"
  log_info "Size     : ${file_size}"
  log_info "Duration : ${duration}s"
}

clean_old_backups() {
  if [ "${CLEAN}" = false ]; then
    return 0
  fi

  log_info "Cleaning backups older than ${RETENTION_DAYS} days..."

  local count
  count="$(find "${BACKUP_DIR}" -name "${DATABASE}_*" -type f -mtime +${RETENTION_DAYS} 2>/dev/null | wc -l | tr -d ' ')"

  if [ "${count}" -eq 0 ]; then
    log_info "No old backups to clean"
    return 0
  fi

  log_info "Found ${count} backup(s) to delete"

  find "${BACKUP_DIR}" \
    -name "${DATABASE}_*" \
    -type f \
    -mtime +"${RETENTION_DAYS}" \
    -print -delete 2>/dev/null | while read -r file; do
    log_info "Deleted: $(basename "${file}")"
  done

  log_success "Cleanup complete"
}

list_backups() {
  log_info "Available backups:"

  local backups
  mapfile -t backups < <(find "${BACKUP_DIR}" -name "${DATABASE}_*" -type f -printf '%T@ %p\n' 2>/dev/null | sort -rn | cut -d' ' -f2-)

  if [ ${#backups[@]} -eq 0 ]; then
    log_info "No backups found"
    return 0
  fi

  for backup in "${backups[@]}"; do
    local size
    size="$(du -sh "${backup}" | cut -f1)"
    local modified
    modified="$(stat -f '%Sm' -t '%Y-%m-%d %H:%M' "${backup}" 2>/dev/null || stat -c '%y' "${backup}" 2>/dev/null | cut -d' ' -f1,2 | cut -d'.' -f1 || echo "unknown")"
    echo "  ${size}  ${modified}  $(basename "${backup}")"
  done
}

# ─── Argument Parsing ─────────────────────────────────────────────────────
parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
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
      -o|--output)
        BACKUP_DIR="$2"
        shift 2
        ;;
      -f|--filename)
        CUSTOM_FILENAME="$2"
        shift 2
        ;;
      --compress)
        COMPRESS=true
        shift
        ;;
      --clean)
        CLEAN=true
        shift
        ;;
      --retention)
        RETENTION_DAYS="$2"
        shift 2
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
  echo "  AI Interview Platform - Database Backup"
  echo "============================================="
  echo "  Database    : ${DATABASE}"
  echo "  Host        : ${DB_HOST}:${DB_PORT}"
  echo "  Output      : ${BACKUP_DIR}"
  echo "  Compress    : ${COMPRESS}"
  echo "  Clean Old   : ${CLEAN} (${RETENTION_DAYS}d retention)"
  echo "============================================="
  echo ""

  validate_prerequisites
  setup_backup_dir
  check_database_connection

  # Generate backup filename
  local filename
  filename="$(generate_filename)"

  # Perform backup
  perform_backup "${filename}"

  # Clean old backups if requested
  if [ "${CLEAN}" = true ]; then
    echo ""
    clean_old_backups
  fi

  # List all backups
  echo ""
  list_backups
}

main "$@"
