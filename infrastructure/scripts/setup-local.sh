#!/usr/bin/env bash
#
# setup-local.sh - Set up local development environment
#
# Usage:
#   ./setup-local.sh [OPTIONS]
#
# Options:
#   --skip-docker          Skip Docker setup
#   --skip-go              Skip Go toolchain check
#   --skip-node            Skip Node.js check
#   --skip-python          Skip Python check
#   --skip-env             Skip .env file generation
#   --skip-proto           Skip protobuf tool installation
#   --force-env             Overwrite existing .env files
#   --help                 Show this help message
#
# Examples:
#   ./setup-local.sh              # Full setup
#   ./setup-local.sh --skip-docker # Skip Docker, only check toolchains

set -euo pipefail

# ─── Colors ───────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# ─── Logging ──────────────────────────────────────────────────────────────
log_info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
log_success() { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $*"; }
log_step()    { echo -e "\n${CYAN}--- $* ---${NC}"; }

# ─── Configuration ────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

SKIP_DOCKER=false
SKIP_GO=false
SKIP_NODE=false
SKIP_PYTHON=false
SKIP_ENV=false
SKIP_PROTO=false
FORCE_ENV=false

GO_MIN_VERSION="1.21"
NODE_MIN_VERSION="20"
PYTHON_MIN_VERSION="3.11"

# ─── Functions ────────────────────────────────────────────────────────────
usage() {
  head -12 "$0" | tail -10
  exit 0
}

check_command() {
  local cmd="$1"
  local name="${2:-${cmd}}"

  if command -v "${cmd}" &>/dev/null; then
    log_success "${name} is installed: $(${cmd} --version 2>/dev/null | head -1 || echo 'version unknown')"
    return 0
  else
    log_warn "${name} is not installed"
    return 1
  fi
}

version_gte() {
  # Compare two version numbers. Returns 0 if $1 >= $2
  local v1="${1//v/}"
  local v2="${2//v/}"

  if [ "${v1}" = "${v2}" ]; then
    return 0
  fi

  local largest
  largest="$(printf '%s\n%s' "${v1}" "${v2}" | sort -V | tail -1)"
  [ "${v1}" = "${largest}" ]
}

# ─── Checks ───────────────────────────────────────────────────────────────
check_docker() {
  if [ "${SKIP_DOCKER}" = true ]; then
    log_warn "Skipping Docker check"
    return 0
  fi

  log_step "Checking Docker"

  local ok=true

  if ! check_command docker; then
    log_error "Install Docker: https://docs.docker.com/get-docker/"
    ok=false
  fi

  if ! check_command "docker-compose" "Docker Compose" && ! docker compose version &>/dev/null; then
    log_error "Install Docker Compose (bundled with Docker Desktop or separately)"
    ok=false
  fi

  # Check Docker daemon is running
  if ! docker info &>/dev/null; then
    log_error "Docker daemon is not running. Start Docker and try again."
    ok=false
  else
    log_success "Docker daemon is running"
  fi

  if [ "${ok}" = false ]; then
    return 1
  fi
}

check_go() {
  if [ "${SKIP_GO}" = true ]; then
    log_warn "Skipping Go check"
    return 0
  fi

  log_step "Checking Go"

  if ! check_command go; then
    log_error "Install Go ${GO_MIN_VERSION}+: https://golang.org/dl/"
    return 1
  fi

  local version
  version="$(go version | grep -oP 'go\K[0-9]+\.[0-9]+(\.[0-9]+)?' || go version | awk '{print $3}' | sed 's/go//')"

  if version_gte "${version}" "${GO_MIN_VERSION}"; then
    log_success "Go version ${version} meets minimum requirement (${GO_MIN_VERSION})"
  else
    log_error "Go version ${version} is below minimum requirement (${GO_MIN_VERSION})"
    return 1
  fi
}

check_node() {
  if [ "${SKIP_NODE}" = true ]; then
    log_warn "Skipping Node.js check"
    return 0
  fi

  log_step "Checking Node.js"

  if ! check_command node "Node.js"; then
    log_error "Install Node.js ${NODE_MIN_VERSION}+: https://nodejs.org/"
    return 1
  fi

  local version
  version="$(node --version | sed 's/v//')"

  if version_gte "${version}" "${NODE_MIN_VERSION}"; then
    log_success "Node.js version ${version} meets minimum requirement (${NODE_MIN_VERSION})"
  else
    log_error "Node.js version ${version} is below minimum requirement (${NODE_MIN_VERSION})"
    return 1
  fi

  # Check npm
  if ! check_command npm; then
    log_error "npm is required but not found"
    return 1
  fi
}

check_python() {
  if [ "${SKIP_PYTHON}" = true ]; then
    log_warn "Skipping Python check"
    return 0
  fi

  log_step "Checking Python"

  if ! check_command python3 "Python 3"; then
    log_error "Install Python ${PYTHON_MIN_VERSION}+: https://www.python.org/downloads/"
    return 1
  fi

  local version
  version="$(python3 --version | awk '{print $2}')"

  if version_gte "${version}" "${PYTHON_MIN_VERSION}"; then
    log_success "Python version ${version} meets minimum requirement (${PYTHON_MIN_VERSION})"
  else
    log_error "Python version ${version} is below minimum requirement (${PYTHON_MIN_VERSION})"
    return 1
  fi
}

check_protobuf() {
  if [ "${SKIP_PROTO}" = true ]; then
    log_warn "Skipping Protobuf check"
    return 0
  fi

  log_step "Checking Protobuf Tooling"

  if check_command protoc "Protoc"; then
    log_success "protoc is available"
  else
    log_warn "protoc not found. Install for gRPC code generation."
    log_info "  macOS: brew install protobuf"
    log_info "  Linux: apt install protobuf-compiler"
  fi

  if check_command "protoc-gen-go" "protoc-gen-go"; then
    log_success "protoc-gen-go is available"
  else
    log_warn "protoc-gen-go not found. Install with: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
  fi

  if check_command "protoc-gen-go-grpc" "protoc-gen-go-grpc"; then
    log_success "protoc-gen-go-grpc is available"
  else
    log_warn "protoc-gen-go-grpc not found. Install with: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
  fi
}

# ─── Environment Files ────────────────────────────────────────────────────
generate_env_files() {
  if [ "${SKIP_ENV}" = true ]; then
    log_warn "Skipping .env file generation"
    return 0
  fi

  log_step "Generating Environment Files"

  local env_template="${PROJECT_ROOT}/.env.example"
  local env_file="${PROJECT_ROOT}/.env"

  # Root .env
  if [ -f "${env_file}" ] && [ "${FORCE_ENV}" = false ]; then
    log_warn "Root .env already exists. Use --force-env to overwrite."
  elif [ -f "${env_template}" ]; then
    log_info "Creating .env from .env.example..."
    cp "${env_template}" "${env_file}"
    log_success "Root .env created"
  else
    log_info "Creating default .env..."
    cat > "${env_file}" <<'EOF'
# AI Interview Platform - Environment Variables
APP_ENV=development
APP_PORT=8080

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=interview_platform
DB_USER=postgres
DB_PASSWORD=postgres

# Redis
REDIS_URL=redis://localhost:6379/0

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672

# Kafka
KAFKA_BROKERS=localhost:9092

# JWT
JWT_SECRET=change-me-in-production
JWT_EXPIRATION=24h

# OAuth
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=

# AI Service
OPENAI_API_KEY=
AI_SERVICE_URL=http://localhost:8081

# Frontend
FRONTEND_URL=http://localhost:3000
EOF
    log_success "Default .env created"
  fi

  # Frontend .env
  local frontend_env="${PROJECT_ROOT}/frontend/.env"
  local frontend_env_example="${PROJECT_ROOT}/frontend/.env.example"
  if [ -f "${frontend_env_example}" ]; then
    if [ -f "${frontend_env}" ] && [ "${FORCE_ENV}" = false ]; then
      log_warn "Frontend .env already exists. Use --force-env to overwrite."
    else
      cp "${frontend_env_example}" "${frontend_env}"
      log_success "Frontend .env created"
    fi
  fi

  # AI Service .env
  local ai_env="${PROJECT_ROOT}/services/ai-service/.env"
  local ai_env_example="${PROJECT_ROOT}/services/ai-service/.env.example"
  if [ -f "${ai_env_example}" ]; then
    if [ -f "${ai_env}" ] && [ "${FORCE_ENV}" = false ]; then
      log_warn "AI service .env already exists. Use --force-env to overwrite."
    else
      cp "${ai_env_example}" "${ai_env}"
      log_success "AI service .env created"
    fi
  fi

  # Docker .env
  local docker_env="${PROJECT_ROOT}/infrastructure/docker/.env"
  local docker_env_example="${PROJECT_ROOT}/infrastructure/docker/.env.example"
  if [ -f "${docker_env_example}" ]; then
    if [ -f "${docker_env}" ] && [ "${FORCE_ENV}" = false ]; then
      log_warn "Docker .env already exists. Use --force-env to overwrite."
    else
      cp "${docker_env_example}" "${docker_env}"
      log_success "Docker .env created"
    fi
  fi
}

# ─── Install Dependencies ─────────────────────────────────────────────────
install_go_deps() {
  log_step "Installing Go Dependencies"

  log_info "Downloading Go module dependencies for all services..."

  local services=(
    user-service resume-service github-service
    interview-service scoring-service report-service
    notification-service analytics-service admin-service
  )

  for svc in "${services[@]}"; do
    local svc_dir="${PROJECT_ROOT}/services/${svc}"
    if [ -f "${svc_dir}/go.mod" ]; then
      log_info "Downloading deps for ${svc}..."
      (cd "${svc_dir}" && go mod download) || log_warn "Failed to download deps for ${svc}"
    fi
  done

  log_success "Go dependencies downloaded"
}

install_node_deps() {
  log_step "Installing Node.js Dependencies"

  if [ -f "${PROJECT_ROOT}/frontend/package.json" ]; then
    log_info "Installing frontend dependencies..."
    (cd "${PROJECT_ROOT}/frontend" && npm install) || {
      log_error "Failed to install frontend dependencies"
      return 1
    }
    log_success "Frontend dependencies installed"
  fi
}

install_python_deps() {
  log_step "Installing Python Dependencies"

  local ai_req="${PROJECT_ROOT}/services/ai-service/requirements.txt"
  if [ -f "${ai_req}" ]; then
    log_info "Installing AI service Python dependencies..."
    pip3 install -r "${ai_req}" || {
      log_error "Failed to install Python dependencies"
      return 1
    }
    log_success "Python dependencies installed"
  fi
}

# ─── Docker Services ──────────────────────────────────────────────────────
start_infrastructure() {
  log_step "Starting Infrastructure Services"

  local compose_file="${PROJECT_ROOT}/infrastructure/docker/docker-compose.yml"

  if [ ! -f "${compose_file}" ]; then
    log_warn "Docker Compose file not found at ${compose_file}"
    log_info "Make sure Docker Compose files exist in infrastructure/docker/"
    return 0
  fi

  log_info "Starting PostgreSQL, Redis, RabbitMQ, and Kafka..."

  # Start only infrastructure services (not app services)
  (cd "${PROJECT_ROOT}" && docker compose -f "${compose_file}" up -d postgres redis rabbitmq kafka 2>/dev/null) || {
    log_warn "Some infrastructure services failed to start. Check docker compose logs."
    return 1
  }

  # Wait for services to be healthy
  log_info "Waiting for services to be ready..."
  sleep 10

  log_success "Infrastructure services started"
}

# ─── Summary ──────────────────────────────────────────────────────────────
show_summary() {
  echo ""
  echo "============================================="
  echo "  Setup Complete!"
  echo "============================================="
  echo ""
  echo "Next steps:"
  echo "  1. Review and update .env files with your credentials"
  echo "  2. Start infrastructure: make dev-up"
  echo "  3. Run migrations:    make db-migrate"
  echo "  4. Seed test data:    make db-seed"
  echo "  5. Build services:    make build-all"
  echo "  6. Start frontend:    cd frontend && npm run dev"
  echo ""
  echo "Documentation: ${PROJECT_ROOT}/docs/development/getting-started.md"
  echo "============================================="
}

# ─── Argument Parsing ─────────────────────────────────────────────────────
parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --skip-docker)  SKIP_DOCKER=true; shift ;;
      --skip-go)      SKIP_GO=true; shift ;;
      --skip-node)    SKIP_NODE=true; shift ;;
      --skip-python)  SKIP_PYTHON=true; shift ;;
      --skip-env)     SKIP_ENV=true; shift ;;
      --skip-proto)   SKIP_PROTO=true; shift ;;
      --force-env)    FORCE_ENV=true; shift ;;
      --help)         usage ;;
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
  echo "  AI Interview Platform - Local Setup"
  echo "============================================="
  echo "  Project: ${PROJECT_ROOT}"
  echo "  Date   : $(date)"
  echo "============================================="

  local failed=0

  check_docker || failed=$((failed + 1))
  check_go || failed=$((failed + 1))
  check_node || failed=$((failed + 1))
  check_python || failed=$((failed + 1))
  check_protobuf || true  # Non-fatal

  if [ ${failed} -gt 0 ]; then
    echo ""
    log_error "${failed} prerequisite(s) missing. Please install them and re-run."
    exit 1
  fi

  generate_env_files
  install_go_deps
  install_node_deps
  install_python_deps
  start_infrastructure || true

  show_summary
}

main "$@"
