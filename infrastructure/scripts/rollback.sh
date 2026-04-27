#!/usr/bin/env bash
#
# rollback.sh - Rollback Kubernetes deployments to the previous revision
#
# Usage:
#   ./rollback.sh [OPTIONS]
#
# Options:
#   -e, --environment ENV    Target environment (staging|production). Default: staging
#   -n, --namespace NS       Kubernetes namespace. Default: auto-detected
#   -s, --service SVC        Rollback a specific service. Default: all services
#   -r, --revision REV       Rollback to a specific revision number. Default: previous
#   -d, --dry-run            Show what would be done without executing
#   -h, --help               Show this help message
#
# Examples:
#   ./rollback.sh                         # Rollback all services to previous revision
#   ./rollback.sh -e production            # Rollback production services
#   ./rollback.sh -s user-service          # Rollback only user-service
#   ./rollback.sh -r 3                     # Rollback to revision 3

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

ENVIRONMENT="staging"
NAMESPACE=""
SERVICE=""
REVISION=""
DRY_RUN=false

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
  ai-service
  frontend
)

# ─── Functions ────────────────────────────────────────────────────────────
usage() {
  head -14 "$0" | tail -12
  exit 0
}

validate_prerequisites() {
  log_info "Validating prerequisites..."

  if ! command -v kubectl &>/dev/null; then
    log_error "kubectl is not installed or not in PATH"
    exit 1
  fi

  if ! kubectl cluster-info &>/dev/null; then
    log_error "Cannot connect to Kubernetes cluster. Check your kubeconfig."
    exit 1
  fi

  log_success "Kubernetes cluster is accessible"
}

get_namespace() {
  if [ -z "${NAMESPACE}" ]; then
    case "${ENVIRONMENT}" in
      staging)
        NAMESPACE="interview-platform-staging"
        ;;
      production)
        NAMESPACE="interview-platform-production"
        ;;
      *)
        log_error "Unknown environment: ${ENVIRONMENT}"
        exit 1
        ;;
    esac
  fi
  log_info "Target namespace: ${NAMESPACE}"
}

show_deployment_history() {
  local svc="$1"

  log_info "Deployment history for ${svc}:"
  kubectl rollout history "deployment/${svc}" -n "${NAMESPACE}" 2>/dev/null || true
  echo ""
}

rollback_service() {
  local svc="$1"

  log_info "Rolling back ${svc}..."

  # Show current revision before rollback
  local current_revision
  current_revision="$(kubectl rollout history "deployment/${svc}" -n "${NAMESPACE}" 2>/dev/null | tail -1 | awk '{print $1}' || echo "unknown")"
  log_info "Current revision: ${current_revision}"

  if [ "${DRY_RUN}" = true ]; then
    if [ -n "${REVISION}" ]; then
      echo "  [DRY-RUN] kubectl rollout undo deployment/${svc} -n ${NAMESPACE} --to-revision=${REVISION}"
    else
      echo "  [DRY-RUN] kubectl rollout undo deployment/${svc} -n ${NAMESPACE}"
    fi
    return 0
  fi

  if [ -n "${REVISION}" ]; then
    log_info "Rolling back to revision ${REVISION}..."
    if ! kubectl rollout undo "deployment/${svc}" \
      -n "${NAMESPACE}" --to-revision="${REVISION}" 2>&1; then
      log_error "Failed to rollback ${svc} to revision ${REVISION}"
      return 1
    fi
  else
    if ! kubectl rollout undo "deployment/${svc}" \
      -n "${NAMESPACE}" 2>&1; then
      log_error "Failed to rollback ${svc}"
      return 1
    fi
  fi

  log_success "Rollback initiated for ${svc}"
}

wait_for_rollback() {
  local svc="$1"
  local timeout="${2:-300s}"

  log_info "Waiting for ${svc} rollback to complete (timeout: ${timeout})..."

  if [ "${DRY_RUN}" = true ]; then
    echo "  [DRY-RUN] kubectl rollout status deployment/${svc} -n ${NAMESPACE} --timeout=${timeout}"
    return 0
  fi

  if ! kubectl rollout status "deployment/${svc}" \
    -n "${NAMESPACE}" --timeout="${timeout}" 2>&1; then
    log_error "Rollback timed out for ${svc}"
    return 1
  fi

  log_success "Rollback complete for ${svc}"
}

verify_rollback() {
  local svc="$1"

  log_info "Verifying rollback for ${svc}..."

  if [ "${DRY_RUN}" = true ]; then
    echo "  [DRY-RUN] kubectl get deployment/${svc} -n ${NAMESPACE} -o wide"
    return 0
  fi

  kubectl get "deployment/${svc}" -n "${NAMESPACE}" -o wide 2>/dev/null || true

  # Show the last applied revision
  local new_revision
  new_revision="$(kubectl get "deployment/${svc}" -n "${NAMESPACE}" -o jsonpath='{.metadata.annotations.kubectl\.kubernetes\.io/last-applied-configuration}' 2>/dev/null || echo "N/A")"

  echo ""
}

show_status() {
  log_info "Post-rollback status:"
  echo ""

  if [ "${DRY_RUN}" = true ]; then
    echo "  [DRY-RUN] kubectl get pods -n ${NAMESPACE}"
    echo "  [DRY-RUN] kubectl get deployments -n ${NAMESPACE}"
    return 0
  fi

  echo -e "${BLUE}Pods:${NC}"
  kubectl get pods -n "${NAMESPACE}" -l app=interview-platform -o wide 2>/dev/null || true
  echo ""

  echo -e "${BLUE}Deployments:${NC}"
  kubectl get deployments -n "${NAMESPACE}" -l app=interview-platform 2>/dev/null || true
}

# ─── Argument Parsing ─────────────────────────────────────────────────────
parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -e|--environment)
        ENVIRONMENT="$2"
        shift 2
        ;;
      -n|--namespace)
        NAMESPACE="$2"
        shift 2
        ;;
      -s|--service)
        SERVICE="$2"
        shift 2
        ;;
      -r|--revision)
        REVISION="$2"
        shift 2
        ;;
      -d|--dry-run)
        DRY_RUN=true
        shift
        ;;
      -h|--help)
        usage
        ;;
      *)
        log_error "Unknown option: $1"
        usage
        ;;
    esac
  done

  # Validate environment
  if [[ ! "${ENVIRONMENT}" =~ ^(staging|production)$ ]]; then
    log_error "Invalid environment: ${ENVIRONMENT}. Must be 'staging' or 'production'."
    exit 1
  fi
}

# ─── Main ─────────────────────────────────────────────────────────────────
main() {
  parse_args "$@"

  echo "============================================="
  echo "  AI Interview Platform - Rollback"
  echo "============================================="
  echo "  Environment : ${ENVIRONMENT}"
  echo "  Service     : ${SERVICE:-all}"
  echo "  Revision    : ${REVISION:-previous}"
  echo "  Dry Run     : ${DRY_RUN}"
  echo "============================================="
  echo ""

  # Confirmation for production
  if [ "${ENVIRONMENT}" = "production" ] && [ "${DRY_RUN}" = false ]; then
    log_warn "You are about to rollback production services!"
    read -rp "Type 'yes' to confirm: " confirm
    if [ "${confirm}" != "yes" ]; then
      log_info "Rollback cancelled"
      exit 0
    fi
  fi

  validate_prerequisites
  get_namespace

  # Determine services to rollback
  local services_to_rollback=()
  if [ -n "${SERVICE}" ]; then
    services_to_rollback=("${SERVICE}")
  else
    services_to_rollback=("${ALL_SERVICES[@]}")
  fi

  log_info "Rolling back ${#services_to_rollback[@]} service(s)..."

  # Show history before rollback
  for svc in "${services_to_rollback[@]}"; do
    show_deployment_history "${svc}"
  done

  # Perform rollback
  local failed=0
  for svc in "${services_to_rollback[@]}"; do
    if ! rollback_service "${svc}"; then
      failed=$((failed + 1))
      log_error "Rollback failed for ${svc}"
    fi
  done

  if [ ${failed} -gt 0 ]; then
    log_error "${failed} service(s) failed to rollback"
    exit 1
  fi

  # Wait for rollback to complete
  log_info "Waiting for rollback to complete..."
  for svc in "${services_to_rollback[@]}"; do
    if ! wait_for_rollback "${svc}"; then
      log_error "Rollback timed out for ${svc}"
      exit 1
    fi
  done

  # Verify rollback
  for svc in "${services_to_rollback[@]}"; do
    verify_rollback "${svc}"
  done

  # Show final status
  echo ""
  show_status

  echo ""
  log_success "Rollback completed successfully!"
  log_info "Environment: ${ENVIRONMENT}"
  log_info "Namespace  : ${NAMESPACE}"
  log_info "Revision   : ${REVISION:-previous}"
}

main "$@"
