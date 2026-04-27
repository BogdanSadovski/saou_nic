#!/usr/bin/env bash
#
# deploy.sh - Deploy the AI Interview Platform to Kubernetes
#
# Usage:
#   ./deploy.sh [OPTIONS]
#
# Options:
#   -e, --environment ENV    Target environment (staging|production). Default: staging
#   -n, --namespace NS       Kubernetes namespace. Default: auto-detected
#   -t, --tag TAG            Docker image tag to deploy. Default: latest commit SHA
#   -s, --service SVC        Deploy a specific service only. Default: all services
#   -d, --dry-run            Show what would be done without executing
#   -h, --help               Show this help message
#
# Examples:
#   ./deploy.sh                          # Deploy all services to staging
#   ./deploy.sh -e production             # Deploy all services to production
#   ./deploy.sh -s user-service -t v1.2.3 # Deploy user-service with specific tag
#   ./deploy.sh --dry-run                 # Preview deployment

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
K8S_DIR="${PROJECT_ROOT}/infrastructure/k8s"

ENVIRONMENT="staging"
NAMESPACE=""
IMAGE_TAG=""
SERVICE=""
DRY_RUN=false

REGISTRY="${REGISTRY:-ghcr.io}"
IMAGE_PREFIX="${IMAGE_PREFIX:-interview-platform}"

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

get_image_tag() {
  if [ -z "${IMAGE_TAG}" ]; then
    IMAGE_TAG="$(git rev-parse --short HEAD 2>/dev/null || echo "latest")"
    log_info "No tag specified, using current commit: ${IMAGE_TAG}"
  fi
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

deploy_service() {
  local svc="$1"

  log_info "Deploying ${svc}..."

  local image="${REGISTRY}/${IMAGE_PREFIX}/${svc}:${IMAGE_TAG}"

  if [ "${DRY_RUN}" = true ]; then
    echo "  [DRY-RUN] kubectl set image deployment/${svc} ${svc}=${image} -n ${NAMESPACE}"
    return 0
  fi

  if ! kubectl set image "deployment/${svc}" \
    "${svc}=${image}" \
    -n "${NAMESPACE}" --record 2>&1; then
    log_error "Failed to update image for ${svc}"
    return 1
  fi

  log_success "Image updated for ${svc}: ${image}"
}

wait_for_rollout() {
  local svc="$1"
  local timeout="${2:-300s}"

  log_info "Waiting for ${svc} rollout to complete (timeout: ${timeout})..."

  if [ "${DRY_RUN}" = true ]; then
    echo "  [DRY-RUN] kubectl rollout status deployment/${svc} -n ${NAMESPACE} --timeout=${timeout}"
    return 0
  fi

  if ! kubectl rollout status "deployment/${svc}" \
    -n "${NAMESPACE}" --timeout="${timeout}" 2>&1; then
    log_error "Rollout timed out for ${svc}"
    return 1
  fi

  log_success "Rollout complete for ${svc}"
}

apply_manifests() {
  log_info "Applying Kubernetes manifests..."

  local dirs=("base" "services" "databases" "messaging" "ingress" "monitoring")

  for dir in "${dirs[@]}"; do
    local path="${K8S_DIR}/${dir}"
    if [ -d "${path}" ]; then
      log_info "Applying manifests from ${dir}/"
      if [ "${DRY_RUN}" = true ]; then
        echo "  [DRY-RUN] kubectl apply -f ${path}/ -n ${NAMESPACE}"
      else
        if ! kubectl apply -f "${path}/" -n "${NAMESPACE}" 2>&1; then
          log_error "Failed to apply manifests in ${dir}/"
          return 1
        fi
      fi
    fi
  done

  log_success "Manifests applied"
}

show_status() {
  log_info "Deployment status:"
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
      -t|--tag)
        IMAGE_TAG="$2"
        shift 2
        ;;
      -s|--service)
        SERVICE="$2"
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
  echo "  AI Interview Platform - Deployment"
  echo "============================================="
  echo "  Environment : ${ENVIRONMENT}"
  echo "  Image Tag   : ${IMAGE_TAG:-<auto-detect>}"
  echo "  Service     : ${SERVICE:-all}"
  echo "  Dry Run     : ${DRY_RUN}"
  echo "============================================="
  echo ""

  validate_prerequisites
  get_image_tag
  get_namespace

  # Apply base manifests first
  apply_manifests

  # Deploy services
  local services_to_deploy=()
  if [ -n "${SERVICE}" ]; then
    services_to_deploy=("${SERVICE}")
  else
    services_to_deploy=("${ALL_SERVICES[@]}")
  fi

  log_info "Deploying ${#services_to_deploy[@]} service(s)..."

  local failed=0
  for svc in "${services_to_deploy[@]}"; do
    if ! deploy_service "${svc}"; then
      failed=$((failed + 1))
      log_error "Deployment failed for ${svc}"
    fi
  done

  if [ ${failed} -gt 0 ]; then
    log_error "${failed} service(s) failed to deploy"
    exit 1
  fi

  # Wait for rollout
  log_info "Waiting for rollout to complete..."
  for svc in "${services_to_deploy[@]}"; do
    if ! wait_for_rollout "${svc}"; then
      log_error "Rollout failed for ${svc}"
      exit 1
    fi
  done

  # Show final status
  echo ""
  show_status

  echo ""
  log_success "Deployment completed successfully!"
  log_info "Environment: ${ENVIRONMENT}"
  log_info "Namespace  : ${NAMESPACE}"
  log_info "Image Tag  : ${IMAGE_TAG}"
}

main "$@"
