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

if ! grep -q 'com.redhat.openshift.versions:' bundle/metadata/annotations.yaml; then
  echo "error: missing com.redhat.openshift.versions in bundle/metadata/annotations.yaml; run make bundle" >&2
  exit 1
fi

shopt -s nullglob
if compgen -G "bundle/manifests/*networkpolicy*.yaml" > /dev/null; then
  echo "error: OLM bundle must not contain operator-namespace NetworkPolicy manifests; run make bundle" >&2
  exit 1
fi
shopt -u nullglob

if ! awk '/^metadata:/{m=1} m && /^  annotations:/{a=1} a && /^    description:/{found=1; exit} END{exit !found}' "${CSV}"; then
  echo "error: missing metadata.annotations.description in CSV; run make bundle" >&2
  exit 1
fi

if ! grep -q 'metadata.annotations.support:' "${CSV}"; then
  echo "error: missing metadata.annotations.support in CSV; run make bundle" >&2
  exit 1
fi

echo "==> All pre-release checks passed"
