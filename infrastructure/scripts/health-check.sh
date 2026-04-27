#!/usr/bin/env bash
#
# health-check.sh - Check health of all services in the AI Interview Platform
#
# Usage:
#   ./health-check.sh [OPTIONS]
#
# Options:
#   -e, --environment ENV    Target environment (local|staging|production). Default: local
#   -v, --verbose            Show detailed response information
#   -t, --timeout SEC        Request timeout in seconds. Default: 10
#   --json                   Output results in JSON format
#   --help                   Show this help message
#
# Examples:
#   ./health-check.sh                    # Check all services (local)
#   ./health-check.sh -e staging          # Check staging services
#   ./health-check.sh -v --json           # Verbose JSON output

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
VERBOSE=false
TIMEOUT=10
OUTPUT_JSON=false

# Service endpoints - format: "name|url|type"
# Types: http, grpc, tcp
declare -a SERVICES=()

# Results tracking
declare -A RESULTS=()
TOTAL=0
HEALTHY=0
UNHEALTHY=0
UNKNOWN=0

# ─── Functions ────────────────────────────────────────────────────────────
usage() {
  head -11 "$0" | tail -9
  exit 0
}

configure_endpoints() {
  case "${ENVIRONMENT}" in
    local)
      SERVICES=(
        "user-service|http://localhost:8080/health|http"
        "resume-service|http://localhost:8081/health|http"
        "github-service|http://localhost:8082/health|http"
        "interview-service|http://localhost:8083/health|http"
        "scoring-service|http://localhost:8084/health|http"
        "report-service|http://localhost:8085/health|http"
        "notification-service|http://localhost:8086/health|http"
        "analytics-service|http://localhost:8087/health|http"
        "admin-service|http://localhost:8088/health|http"
        "ai-service|http://localhost:8081/api/health|http"
        "frontend|http://localhost:3000|http"
        "postgresql|localhost:5432|tcp"
        "redis|localhost:6379|tcp"
        "rabbitmq|localhost:5672|tcp"
        "kafka|localhost:9092|tcp"
      )
      ;;
    staging)
      SERVICES=(
        "user-service|https://staging.interview-platform.io/api/user/health|http"
        "resume-service|https://staging.interview-platform.io/api/resume/health|http"
        "github-service|https://staging.interview-platform.io/api/github/health|http"
        "interview-service|https://staging.interview-platform.io/api/interview/health|http"
        "scoring-service|https://staging.interview-platform.io/api/scoring/health|http"
        "report-service|https://staging.interview-platform.io/api/report/health|http"
        "notification-service|https://staging.interview-platform.io/api/notification/health|http"
        "analytics-service|https://staging.interview-platform.io/api/analytics/health|http"
        "admin-service|https://staging.interview-platform.io/api/admin/health|http"
        "frontend|https://staging.interview-platform.io|http"
      )
      ;;
    production)
      SERVICES=(
        "user-service|https://interview-platform.io/api/user/health|http"
        "resume-service|https://interview-platform.io/api/resume/health|http"
        "github-service|https://interview-platform.io/api/github/health|http"
        "interview-service|https://interview-platform.io/api/interview/health|http"
        "scoring-service|https://interview-platform.io/api/scoring/health|http"
        "report-service|https://interview-platform.io/api/report/health|http"
        "notification-service|https://interview-platform.io/api/notification/health|http"
        "analytics-service|https://interview-platform.io/api/analytics/health|http"
        "admin-service|https://interview-platform.io/api/admin/health|http"
        "frontend|https://interview-platform.io|http"
      )
      ;;
    *)
      log_error "Unknown environment: ${ENVIRONMENT}"
      exit 1
      ;;
  esac
}

check_http() {
  local name="$1"
  local url="$2"

  local http_code
  local response
  local status="unknown"

  # Make HTTP request, capture status code and body
  response="$(curl -sf -o /dev/null -w '%{http_code}' \
    --max-time "${TIMEOUT}" \
    --retry 2 \
    --retry-delay 2 \
    "${url}" 2>/dev/null)" || response="000"

  http_code="${response}"

  if [[ "${http_code}" =~ ^2[0-9]{2}$ ]]; then
    status="healthy"
    HEALTHY=$((HEALTHY + 1))
  elif [ "${http_code}" = "000" ]; then
    status="unreachable"
    UNKNOWN=$((UNKNOWN + 1))
  else
    status="unhealthy (HTTP ${http_code})"
    UNHEALTHY=$((UNHEALTHY + 1))
  fi

  RESULTS["${name}"]="${status}"

  if [ "${VERBOSE}" = true ]; then
    log_info "${name}: ${status} (${url})"
  fi
}

check_tcp() {
  local name="$1"
  local host_port="$2"

  local host
  host="$(echo "${host_port}" | cut -d: -f1)"
  local port
  port="$(echo "${host_port}" | cut -d: -f2)"

  if timeout "${TIMEOUT}" bash -c "echo > /dev/tcp/${host}/${port}" 2>/dev/null; then
    RESULTS["${name}"]="healthy"
    HEALTHY=$((HEALTHY + 1))
  else
    RESULTS["${name}"]="unreachable"
    UNKNOWN=$((UNKNOWN + 1))
  fi

  if [ "${VERBOSE}" = true ]; then
    log_info "${name}: ${RESULTS["${name}"]} (${host_port})"
  fi
}

check_k8s_pods() {
  if ! command -v kubectl &>/dev/null; then
    return 0
  fi

  if ! kubectl cluster-info &>/dev/null; then
    return 0
  fi

  log_info "Checking Kubernetes pod status..."

  local namespace
  case "${ENVIRONMENT}" in
    staging)    namespace="interview-platform-staging" ;;
    production) namespace="interview-platform-production" ;;
    *)          return 0 ;;
  esac

  echo ""
  kubectl get pods -n "${namespace}" -l app=interview-platform 2>/dev/null || true
  echo ""
}

print_results() {
  echo ""
  echo "============================================="
  echo "  Health Check Results - ${ENVIRONMENT}"
  echo "============================================="
  echo ""

  if [ "${OUTPUT_JSON}" = true ]; then
    print_json_results
    return
  fi

  printf "%-30s %-20s\n" "SERVICE" "STATUS"
  printf "%-30s %-20s\n" "------------------------------" "--------------------"

  for entry in "${SERVICES[@]}"; do
    local name
    name="$(echo "${entry}" | cut -d'|' -f1)"
    local status="${RESULTS["${name}"]:-unknown}"

    local color="${NC}"
    if [[ "${status}" == healthy* ]]; then
      color="${GREEN}"
    elif [[ "${status}" == unreachable* ]] || [[ "${status}" == unknown* ]]; then
      color="${YELLOW}"
    else
      color="${RED}"
    fi

    printf "%-30s ${color}%-20s${NC}\n" "${name}" "${status}"
  done

  echo ""
  echo "============================================="
  printf "  Total:     %d\n" "${TOTAL}"
  printf "  Healthy:   ${GREEN}%d${NC}\n" "${HEALTHY}"
  printf "  Unhealthy: ${RED}%d${NC}\n" "${UNHEALTHY}"
  printf "  Unknown:   ${YELLOW}%d${NC}\n" "${UNKNOWN}"
  echo "============================================="

  if [ ${UNHEALTHY} -gt 0 ]; then
    echo ""
    log_error "Some services are unhealthy!"
    return 1
  fi

  if [ ${UNKNOWN} -gt 0 ]; then
    echo ""
    log_warn "Some services are unreachable"
  fi
}

print_json_results() {
  echo "{"
  echo "  \"environment\": \"${ENVIRONMENT}\","
  echo "  \"timestamp\": \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\","
  echo "  \"summary\": {"
  echo "    \"total\": ${TOTAL},"
  echo "    \"healthy\": ${HEALTHY},"
  echo "    \"unhealthy\": ${UNHEALTHY},"
  echo "    \"unknown\": ${UNKNOWN}"
  echo "  },"
  echo "  \"services\": {"

  local first=true
  for entry in "${SERVICES[@]}"; do
    local name
    name="$(echo "${entry}" | cut -d'|' -f1)"
    local status="${RESULTS["${name}"]:-unknown}"

    if [ "${first}" = true ]; then
      first=false
    else
      echo ","
    fi
    printf "    \"%s\": \"%s\"" "${name}" "${status}"
  done

  echo ""
  echo "  }"
  echo "}"
}

# ─── Argument Parsing ─────────────────────────────────────────────────────
parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -e|--environment)
        ENVIRONMENT="$2"
        shift 2
        ;;
      -v|--verbose)
        VERBOSE=true
        shift
        ;;
      -t|--timeout)
        TIMEOUT="$2"
        shift 2
        ;;
      --json)
        OUTPUT_JSON=true
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

  # Validate environment
  if [[ ! "${ENVIRONMENT}" =~ ^(local|staging|production)$ ]]; then
    log_error "Invalid environment: ${ENVIRONMENT}. Must be 'local', 'staging', or 'production'."
    exit 1
  fi
}

# ─── Main ─────────────────────────────────────────────────────────────────
main() {
  parse_args "$@"

  echo "============================================="
  echo "  AI Interview Platform - Health Check"
  echo "============================================="
  echo "  Environment : ${ENVIRONMENT}"
  echo "  Timeout     : ${TIMEOUT}s"
  echo "  Verbose     : ${VERBOSE}"
  echo "  Time        : $(date)"
  echo "============================================="

  configure_endpoints

  TOTAL=${#SERVICES[@]}
  log_info "Checking ${TOTAL} services..."
  echo ""

  # Run health checks
  for entry in "${SERVICES[@]}"; do
    local name
    name="$(echo "${entry}" | cut -d'|' -f1)"
    local url
    url="$(echo "${entry}" | cut -d'|' -f2)"
    local type
    type="$(echo "${entry}" | cut -d'|' -f3)"

    case "${type}" in
      http) check_http "${name}" "${url}" ;;
      tcp)  check_tcp "${name}" "${url}" ;;
    esac
  done

  # Check K8s pods for remote environments
  if [ "${ENVIRONMENT}" != "local" ]; then
    check_k8s_pods
  fi

  # Print results
  print_results
}

main "$@"
