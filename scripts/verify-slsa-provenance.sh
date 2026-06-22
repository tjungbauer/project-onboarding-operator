#!/usr/bin/env bash
# Verify cosign SLSA provenance attestations on release images.
#
# Usage:
#   ./scripts/verify-slsa-provenance.sh 0.0.50

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"
QUAY_USER="${QUAY_USER:-tjungbau}"
REGISTRY="quay.io/${QUAY_USER}"

if [[ -z "${VERSION}" ]]; then
  if [[ -f "${ROOT}/VERSION" ]]; then
    VERSION="$(tr -d ' \n\r' < "${ROOT}/VERSION")"
  else
    echo "error: VERSION required" >&2
    exit 1
  fi
fi

IMAGES=(
  "${REGISTRY}/project-onboarding-operator:v${VERSION}"
  "${REGISTRY}/project-onboarding-operator-bundle:v${VERSION}"
  "${REGISTRY}/project-onboarding-operator-catalog:v${VERSION}"
)

if ! command -v cosign >/dev/null 2>&1; then
  echo "error: cosign not found in PATH" >&2
  exit 1
fi

identity="${COSIGN_CERTIFICATE_IDENTITY_REGEXP:-https://github.com/tjungbauer/project-onboarding-operator/.github/workflows/release.yml@refs/tags/v.*}"
issuer="${COSIGN_CERTIFICATE_OIDC_ISSUER:-https://token.actions.githubusercontent.com}"

verify_args=(
  --type slsaprovenance
  --certificate-identity-regexp "${identity}"
  --certificate-oidc-issuer "${issuer}"
)

if [[ -n "${COSIGN_PRIVATE_KEY:-}" ]]; then
  verify_args=(--key "env://COSIGN_PRIVATE_KEY" --type slsaprovenance)
elif [[ -f "${COSIGN_KEY:-cosign.pub}" ]]; then
  verify_args=(--key "${COSIGN_KEY:-cosign.pub}" --type slsaprovenance)
else
  export COSIGN_EXPERIMENTAL="${COSIGN_EXPERIMENTAL:-1}"
fi

echo "==> Verifying SLSA provenance attestations for v${VERSION}"
for ref in "${IMAGES[@]}"; do
  echo "    ${ref}"
  cosign verify-attestation "${verify_args[@]}" "${ref}"
done

echo "==> All SLSA provenance attestations verified"
