#!/usr/bin/env bash
# Pre-release checks before tagging vX.Y.Z.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

VERSION_FILE="${ROOT}/VERSION"
CSV="${ROOT}/bundle/manifests/project-onboarding-operator.clusterserviceversion.yaml"
if [[ -n "${1:-}" ]]; then
  EXPECTED="$1"
elif [[ -f "${VERSION_FILE}" ]]; then
  EXPECTED="$(tr -d ' \n\r' < "${VERSION_FILE}")"
else
  echo "error: VERSION required" >&2
  exit 1
fi

echo "==> Pre-release checks for ${EXPECTED}"

if [[ "$(tr -d ' \n\r' < "${VERSION_FILE}")" != "${EXPECTED}" ]]; then
  echo "error: VERSION file ($(tr -d ' \n\r' < "${VERSION_FILE}")) does not match ${EXPECTED}" >&2
  exit 1
fi

if [[ ! -f "${CSV}" ]] || ! grep -qE "^  version: ${EXPECTED}$" "${CSV}"; then
  echo "error: bundle CSV version is not ${EXPECTED}; run make bundle" >&2
  exit 1
fi

chmod +x hack/resolve-hi-digests.sh
hack/resolve-hi-digests.sh --check

echo "==> All pre-release checks passed"
