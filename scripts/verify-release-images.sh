#!/usr/bin/env bash
# Verify cosign signatures (and optional SBOM presence) for release images on Quay.
#
# Usage:
#   ./scripts/verify-release-images.sh 0.0.49
#
# Environment:
#   QUAY_USER              default: tjungbau
#   COSIGN_EXPERIMENTAL    set to 1 for keyless GitHub Actions signatures (default in CI)
#   COSIGN_CERTIFICATE_IDENTITY_REGEXP
#   COSIGN_CERTIFICATE_OIDC_ISSUER
#   SKIP_SBOM_CHECK        default: false

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"
QUAY_USER="${QUAY_USER:-tjungbau}"
REGISTRY="quay.io/${QUAY_USER}"
SKIP_SBOM_CHECK="${SKIP_SBOM_CHECK:-false}"

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

verify_args=()
if [[ -n "${COSIGN_PRIVATE_KEY:-}" ]]; then
  verify_args=(--key "env://COSIGN_PRIVATE_KEY")
elif [[ -f "${COSIGN_KEY:-cosign.pub}" ]]; then
  verify_args=(--key "${COSIGN_KEY:-cosign.pub}")
else
  export COSIGN_EXPERIMENTAL="${COSIGN_EXPERIMENTAL:-1}"
  verify_args=(
    --certificate-identity-regexp "${identity}"
    --certificate-oidc-issuer "${issuer}"
  )
fi

echo "==> Verifying cosign signatures for v${VERSION}"
for ref in "${IMAGES[@]}"; do
  echo "    ${ref}"
  cosign verify "${verify_args[@]}" "${ref}"
  if [[ "${SKIP_SBOM_CHECK}" != "true" ]]; then
    cosign download sbom "${ref}" >/dev/null
    echo "    SBOM present"
  fi
done

echo "==> All release image signatures verified"
