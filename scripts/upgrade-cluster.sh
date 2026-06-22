#!/usr/bin/env bash
# Upgrade project-onboarding-operator on OpenShift via OLM (no image build/push).
#
# Usage:
#   ./scripts/upgrade-cluster.sh 0.0.47
#   VERSION=0.0.47 ./scripts/upgrade-cluster.sh
#
# Detects install method:
#   - operator-sdk run bundle: catalog in project-onboarding-operator → bundle-upgrade
#   - OperatorHub: catalog in openshift-marketplace → patch catalog + subscription
#
# Requires: oc, images for VERSION already on Quay. operator-sdk for Path A.
#
# See docs/upgrade.md

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
  cat <<'EOF'
Usage: upgrade-cluster.sh [<VERSION>]

  Upgrade the operator on the current OpenShift cluster. Does not build or push images.

Examples:
  ./scripts/upgrade-cluster.sh 0.0.47
  VERSION=0.0.47 ./scripts/upgrade-cluster.sh

To build, push, and upgrade in one step, use release-openshift.sh instead:
  UPGRADE=true ./scripts/release-openshift.sh 0.0.47
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ -n "${1:-}" ]]; then
  VERSION="$1"
elif [[ -n "${VERSION:-}" ]]; then
  :
elif [[ -f "${ROOT}/VERSION" ]]; then
  VERSION="$(tr -d ' \n\r' < "${ROOT}/VERSION")"
else
  echo "error: VERSION required (argument, env, or VERSION file)" >&2
  usage >&2
  exit 1
fi

if ! command -v oc >/dev/null 2>&1; then
  echo "error: oc not found" >&2
  exit 1
fi

QUAY_USER="${QUAY_USER:-tjungbau}"
REGISTRY="quay.io/${QUAY_USER}"
OPERATOR_NS="${OPERATOR_NS:-project-onboarding-operator}"
MARKETPLACE_NS="${MARKETPLACE_NS:-openshift-marketplace}"
CATALOG_NAME="${CATALOG_NAME:-project-onboarding-operator-catalog}"
SUB_NAME="${SUB_NAME:-project-onboarding-operator}"
SKIP_CATALOG="${SKIP_CATALOG:-false}"

BUNDLE_IMG="${REGISTRY}/project-onboarding-operator-bundle:v${VERSION}"
CATALOG_IMG="${REGISTRY}/project-onboarding-operator-catalog:v${VERSION}"

echo "==> Upgrade project-onboarding-operator to ${VERSION}"
echo "    BUNDLE_IMG=${BUNDLE_IMG}"
echo "    CATALOG_IMG=${CATALOG_IMG}"
echo "    (no build/push — images must already exist on Quay)"

if ! oc get subscription "${SUB_NAME}" -n "${OPERATOR_NS}" >/dev/null 2>&1; then
  found_sub="$(oc get subscription -n "${OPERATOR_NS}" -o jsonpath='{range .items[?(@.spec.name=="project-onboarding-operator")]}{.metadata.name}{"\n"}{end}' 2>/dev/null | head -1)"
  if [[ -n "${found_sub}" ]]; then
    echo "==> Using subscription ${found_sub} (package project-onboarding-operator)"
    SUB_NAME="${found_sub}"
  fi
fi

if oc get catalogsource "${CATALOG_NAME}" -n "${OPERATOR_NS}" >/dev/null 2>&1; then
  if ! command -v operator-sdk >/dev/null 2>&1; then
    echo "error: operator-sdk not found (required for operator-sdk run bundle installs)" >&2
    exit 1
  fi
  echo "==> Path A: operator-sdk run bundle-upgrade (catalog in ${OPERATOR_NS})"
  operator-sdk run bundle-upgrade "${BUNDLE_IMG}" \
    --namespace "${OPERATOR_NS}" \
    --timeout 10m
elif oc get subscription "${SUB_NAME}" -n "${OPERATOR_NS}" >/dev/null 2>&1 \
  && oc get catalogsource "${CATALOG_NAME}" -n "${MARKETPLACE_NS}" >/dev/null 2>&1; then
  echo "==> Path B: OperatorHub / marketplace catalog (${MARKETPLACE_NS})"
  if [[ "${SKIP_CATALOG}" != "true" ]]; then
    oc patch catalogsource "${CATALOG_NAME}" -n "${MARKETPLACE_NS}" \
      --type merge -p "{\"spec\":{\"image\":\"${CATALOG_IMG}\"}}"
    oc delete pod -n "${MARKETPLACE_NS}" -l "olm.catalogSource=${CATALOG_NAME}" --ignore-not-found
    oc wait --for=jsonpath='{.status.connectionState.lastObservedState}'=READY \
      "catalogsource/${CATALOG_NAME}" -n "${MARKETPLACE_NS}" --timeout=5m
  fi
  oc patch subscription "${SUB_NAME}" -n "${OPERATOR_NS}" --type merge \
    -p "{\"spec\":{\"channel\":\"stable\",\"startingCSV\":\"project-onboarding-operator.v${VERSION}\",\"installPlanApproval\":\"Automatic\"}}"
elif oc get catalogsource "${CATALOG_NAME}" -n "${MARKETPLACE_NS}" >/dev/null 2>&1; then
  echo "error: marketplace catalog exists but no subscription in ${OPERATOR_NS}; install first" >&2
  exit 1
else
  if ! command -v operator-sdk >/dev/null 2>&1; then
    echo "error: operator-sdk not found" >&2
    exit 1
  fi
  echo "==> No existing install detected; running first-time operator-sdk run bundle"
  operator-sdk run bundle "${BUNDLE_IMG}" \
    --namespace "${OPERATOR_NS}" \
    --install-mode AllNamespaces \
    --timeout 10m
fi

echo "==> Waiting for CSV project-onboarding-operator.v${VERSION}"
oc wait --for=jsonpath='{.status.phase}'=Succeeded \
  "csv/project-onboarding-operator.v${VERSION}" \
  -n "${OPERATOR_NS}" --timeout=10m

echo "==> Waiting for deployment rollout"
oc rollout status "deployment/project-onboarding-operator-controller-manager" \
  -n "${OPERATOR_NS}" --timeout=5m

echo "==> Current CSV and image"
oc get csv -n "${OPERATOR_NS}" | grep project-onboarding || true
oc get deploy -n "${OPERATOR_NS}" \
  -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.template.spec.containers[0].image}{"\n"}{end}'
echo
echo "==> Done"
