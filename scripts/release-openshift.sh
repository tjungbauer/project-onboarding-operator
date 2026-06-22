#!/usr/bin/env bash
# Build, push, and optionally upgrade the operator on OpenShift via OLM.
#
# Usage:
#   ./scripts/release-openshift.sh 0.0.3              # build + push images only
#   UPGRADE=true ./scripts/release-openshift.sh 0.0.3 # build + push + cluster upgrade
#
# Cluster upgrade only (images already on Quay):
#   ./scripts/upgrade-cluster.sh 0.0.3
#
# Optional environment variables:
#   QUAY_USER         default: tjungbau
#   QUAY_USERNAME     optional; used with QUAY_TOKEN (GitHub Actions secrets)
#   QUAY_TOKEN        optional; non-interactive quay.io login (CI)
#   OPERATOR_NS       default: project-onboarding-operator
#   CONTAINER_TOOL    default: podman
#   PLATFORM          default: linux/amd64
#   CHANNELS          default: stable
#   DEFAULT_CHANNEL   default: stable
#   UPGRADE           default: false — after build/push, run cluster upgrade (see upgrade-cluster.sh)
#   SKIP_BUILD        default: false — skip operator image build (still builds bundle/catalog unless SKIP_CATALOG)
#   SKIP_CATALOG      default: false — skip catalog index image (OperatorHub needs it)
#   CATALOG_BASE_IMG  optional — previous catalog image for cumulative index (opm --from-index)
#   CATALOG_FRESH     default: false — build catalog from scratch (no --from-index)
#   APPLY_MARKETPLACE_CATALOG  default: false — apply CatalogSource to openshift-marketplace
#
# Note: CLI <VERSION> wins over a exported VERSION env var.
#       IMG/BUNDLE_IMG/REGISTRY from the shell are ignored (recomputed each run).
#       To clear stale values: unset VERSION IMG BUNDLE_IMG REGISTRY

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: release-openshift.sh [<VERSION>]

  VERSION   Release version (e.g. 0.0.6). CLI argument overrides VERSION env;
            if omitted, reads the VERSION file in the repo root.

Examples:
  ./scripts/release-openshift.sh 0.0.3
  UPGRADE=true ./scripts/release-openshift.sh 0.0.3   # publish + upgrade cluster

Cluster upgrade only (no build/push):
  ./scripts/upgrade-cluster.sh 0.0.3

Before first push:
  podman login quay.io -u tjungbau
  # Create repos on quay.io:
  #   project-onboarding-operator, project-onboarding-operator-bundle,
  #   project-onboarding-operator-catalog  (for OperatorHub UI)
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ -n "${1:-}" ]]; then
  if [[ -n "${VERSION:-}" && "${VERSION}" != "$1" ]]; then
    echo "warning: ignoring VERSION=${VERSION} from environment; using CLI version $1" >&2
  fi
  VERSION="$1"
elif [[ -n "${VERSION:-}" ]]; then
  :
elif [[ -f "${ROOT}/VERSION" ]]; then
  VERSION="$(tr -d ' \n\r' < "${ROOT}/VERSION")"
else
  echo "error: VERSION is required (argument, env, or VERSION file)" >&2
  usage >&2
  exit 1
fi

QUAY_USER="${QUAY_USER:-tjungbau}"
REGISTRY="quay.io/${QUAY_USER}"
OPERATOR_NS="${OPERATOR_NS:-project-onboarding-operator}"
CONTAINER_TOOL="${CONTAINER_TOOL:-podman}"
PLATFORM="${PLATFORM:-linux/amd64}"
CHANNELS="${CHANNELS:-stable}"
DEFAULT_CHANNEL="${DEFAULT_CHANNEL:-stable}"
UPGRADE="${UPGRADE:-false}"
SKIP_BUILD="${SKIP_BUILD:-false}"
SKIP_CATALOG="${SKIP_CATALOG:-false}"
CATALOG_FRESH="${CATALOG_FRESH:-false}"
APPLY_MARKETPLACE_CATALOG="${APPLY_MARKETPLACE_CATALOG:-false}"

# Previous patch release (0.0.6 -> 0.0.5). Empty when not semver or no prior patch.
semver_prev_patch() {
  local v="$1"
  if [[ ! "$v" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    return 1
  fi
  local major="${BASH_REMATCH[1]}" minor="${BASH_REMATCH[2]}" patch="${BASH_REMATCH[3]}"
  if (( patch > 0 )); then
    echo "${major}.${minor}.$((patch - 1))"
  elif (( minor > 0 )); then
    echo "${major}.$((minor - 1)).0"
  elif (( major > 0 )); then
    echo "$((major - 1)).0.0"
  else
    return 1
  fi
}

catalog_image_exists() {
  local ref="$1"
  if command -v skopeo >/dev/null 2>&1; then
    skopeo inspect --override-arch amd64 --override-os linux "docker://${ref}" >/dev/null 2>&1 && return 0
  fi
  # podman manifest inspect fails on single-platform images (common for opm catalogs).
  if [[ "${ref}" =~ ^([^/]+)/([^/]+)/([^:]+):(.+)$ ]]; then
    local registry="${BASH_REMATCH[1]}" namespace="${BASH_REMATCH[2]}" repo="${BASH_REMATCH[3]}" tag="${BASH_REMATCH[4]}"
    curl -sf -o /dev/null \
      -H "Accept: application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json" \
      "https://${registry}/v2/${namespace}/${repo}/manifests/${tag}" && return 0
  fi
  "${CONTAINER_TOOL}" manifest inspect "${ref}" >/dev/null 2>&1
}

resolve_catalog_base_img() {
  if [[ "${CATALOG_FRESH}" == "true" ]]; then
    echo ""
    return 0
  fi
  if [[ -n "${CATALOG_BASE_IMG:-}" ]]; then
    echo "${CATALOG_BASE_IMG}"
    return 0
  fi
  local prev
  if ! prev="$(semver_prev_patch "${VERSION}")"; then
    echo ""
    return 0
  fi
  local candidate="${REGISTRY}/project-onboarding-operator-catalog:v${prev}"
  if catalog_image_exists "${candidate}"; then
    echo "${candidate}"
  else
    echo "warning: previous catalog ${candidate} not found in registry; building fresh index (set CATALOG_BASE_IMG to override)" >&2
    echo ""
  fi
}

# Always derive image names from QUAY_USER + VERSION (ignore stale shell exports).
IMG="${REGISTRY}/project-onboarding-operator:v${VERSION}"
BUNDLE_IMG="${REGISTRY}/project-onboarding-operator-bundle:v${VERSION}"
CATALOG_IMG="${REGISTRY}/project-onboarding-operator-catalog:v${VERSION}"

cd "${ROOT}"

echo "==> Release ${VERSION}"
echo "    QUAY_USER=${QUAY_USER}"
echo "    IMG=${IMG}"
echo "    BUNDLE_IMG=${BUNDLE_IMG}"
echo "    CATALOG_IMG=${CATALOG_IMG}"
echo "    UPGRADE=${UPGRADE}"

ensure_quay_login() {
  local user="${QUAY_USERNAME:-${QUAY_USER}}"
  if [[ -n "${QUAY_TOKEN:-}" ]]; then
    echo "==> Logging in to quay.io as ${user}"
    echo "${QUAY_TOKEN}" | "${CONTAINER_TOOL}" login quay.io -u "${user}" --password-stdin
    return
  fi
  if "${CONTAINER_TOOL}" login quay.io --get-login >/dev/null 2>&1; then
    return
  fi
  echo "error: not logged in to quay.io — run: ${CONTAINER_TOOL} login quay.io -u ${QUAY_USER}" >&2
  echo "       or set QUAY_TOKEN (and optionally QUAY_USERNAME) for non-interactive login" >&2
  exit 1
}

if [[ "${SKIP_BUILD}" != "true" ]]; then
  ensure_quay_login

  echo "==> Building operator image (${PLATFORM})"
  GIT_COMMIT="$(git -C "${ROOT}" rev-parse --short HEAD 2>/dev/null || echo unknown)"
  "${CONTAINER_TOOL}" build --platform="${PLATFORM}" \
    --build-arg VERSION="${VERSION}" \
    --build-arg GIT_COMMIT="${GIT_COMMIT}" \
    -t "${IMG}" "${ROOT}"

  echo "==> Pushing operator image"
  "${CONTAINER_TOOL}" push "${IMG}"
else
  echo "==> Skipping operator image build (SKIP_BUILD=true)"
fi

echo "==> Generating OLM bundle"
PREV_VERSION="$(semver_prev_patch "${VERSION}" 2>/dev/null || true)"
# Resolve operator image tag to digest in the published bundle (git-committed bundle keeps tags for dev/CI drift).
make bundle IMG="${IMG}" VERSION="${VERSION}" CHANNELS="${CHANNELS}" DEFAULT_CHANNEL="${DEFAULT_CHANNEL}" \
  USE_IMAGE_DIGESTS=true \
  ${PREV_VERSION:+PREV_VERSION="${PREV_VERSION}"}

echo "==> Building bundle image"
make bundle-build BUNDLE_IMG="${BUNDLE_IMG}" CONTAINER_TOOL="${CONTAINER_TOOL}"

echo "==> Pushing bundle image"
make bundle-push BUNDLE_IMG="${BUNDLE_IMG}" CONTAINER_TOOL="${CONTAINER_TOOL}"

if [[ "${SKIP_CATALOG}" != "true" ]]; then
  CATALOG_BASE_IMG="$(resolve_catalog_base_img)"
  echo "==> Building catalog image (OperatorHub)"
  if [[ -n "${CATALOG_BASE_IMG}" ]]; then
    echo "    Cumulative index from ${CATALOG_BASE_IMG}"
    make catalog-build CATALOG_IMG="${CATALOG_IMG}" BUNDLE_IMGS="${BUNDLE_IMG}" \
      CATALOG_BASE_IMG="${CATALOG_BASE_IMG}" CONTAINER_TOOL="${CONTAINER_TOOL}" PLATFORM="${PLATFORM}"
  else
    echo "    Fresh index (single bundle ${BUNDLE_IMG})"
    make catalog-build CATALOG_IMG="${CATALOG_IMG}" BUNDLE_IMGS="${BUNDLE_IMG}" \
      CONTAINER_TOOL="${CONTAINER_TOOL}" PLATFORM="${PLATFORM}"
  fi

  echo "==> Pushing catalog image"
  make catalog-push CATALOG_IMG="${CATALOG_IMG}" CONTAINER_TOOL="${CONTAINER_TOOL}"
else
  echo "==> Skipping catalog image (SKIP_CATALOG=true)"
fi

if [[ "${APPLY_MARKETPLACE_CATALOG}" == "true" ]]; then
  if ! command -v oc >/dev/null 2>&1; then
    echo "error: oc not found; set APPLY_MARKETPLACE_CATALOG=false" >&2
    exit 1
  fi
  echo "==> Applying CatalogSource to openshift-marketplace"
  sed "s|:v0.0.0|:v${VERSION}|g; s|quay.io/tjungbau|${REGISTRY}|g" \
    config/openshift/catalogsource-marketplace.yaml | oc apply -f -
  if oc get catalogsource project-onboarding-operator-catalog -n openshift-marketplace >/dev/null 2>&1; then
    echo "==> Updating catalog image to ${CATALOG_IMG}"
    oc patch catalogsource project-onboarding-operator-catalog -n openshift-marketplace \
      --type merge -p "{\"spec\":{\"image\":\"${CATALOG_IMG}\"}}"
    oc delete pod -n openshift-marketplace -l olm.catalogSource=project-onboarding-operator-catalog --ignore-not-found
  fi
  echo "    Wait for READY: oc wait --for=jsonpath='{.status.connectionState.lastObservedState}'=READY \\"
  echo "      catalogsource/project-onboarding-operator-catalog -n openshift-marketplace --timeout=5m"
fi

if [[ "${UPGRADE}" == "true" ]]; then
  echo "==> Cluster upgrade (after build/push)"
  export VERSION QUAY_USER REGISTRY OPERATOR_NS SKIP_CATALOG
  "${ROOT}/scripts/upgrade-cluster.sh" "${VERSION}"
else
  echo "==> Images published."
  echo "    Upgrade cluster (images already on Quay): ./scripts/upgrade-cluster.sh ${VERSION}"
  echo "    OperatorHub UI: see docs/operatorhub-install.md"
  echo "    Register catalog:"
  echo "      APPLY_MARKETPLACE_CATALOG=true ${0} ${VERSION}"
  echo "    Build, push, and upgrade:"
  echo "      UPGRADE=true ${0} ${VERSION}"
fi

echo "==> Done"
