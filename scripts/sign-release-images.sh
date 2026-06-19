#!/usr/bin/env bash
# Sign release images on Quay and attach SPDX SBOMs (cosign + syft).
#
# Usage:
#   ./scripts/sign-release-images.sh 0.0.46
#   QUAY_USER=tjungbau ./scripts/sign-release-images.sh
#
# Environment:
#   QUAY_USER              default: tjungbau
#   COSIGN_YES             default: true (non-interactive)
#   SKIP_SIGN              default: false
#   SKIP_SBOM              default: false
#
# CI (keyless): set COSIGN_EXPERIMENTAL=1 and grant id-token: write on the workflow job.
# Local (key): export COSIGN_PRIVATE_KEY and COSIGN_PASSWORD (or use cosign.key on disk).
#
# Verify:
#   cosign verify --certificate-identity-regexp=.* --certificate-oidc-issuer-regexp=.* \
#     quay.io/tjungbau/project-onboarding-operator:v0.0.46
#   cosign download sbom quay.io/tjungbau/project-onboarding-operator:v0.0.46

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"
QUAY_USER="${QUAY_USER:-tjungbau}"
REGISTRY="quay.io/${QUAY_USER}"
COSIGN_YES="${COSIGN_YES:-true}"
SKIP_SIGN="${SKIP_SIGN:-false}"
SKIP_SBOM="${SKIP_SBOM:-false}"
SBOM_DIR="${SBOM_DIR:-${ROOT}/dist/sbom}"

if [[ -z "${VERSION}" ]]; then
  if [[ -f "${ROOT}/VERSION" ]]; then
    VERSION="$(tr -d ' \n\r' < "${ROOT}/VERSION")"
  else
    echo "error: VERSION required (argument or VERSION file)" >&2
    exit 1
  fi
fi

IMAGES=(
  "${REGISTRY}/project-onboarding-operator:v${VERSION}"
  "${REGISTRY}/project-onboarding-operator-bundle:v${VERSION}"
  "${REGISTRY}/project-onboarding-operator-catalog:v${VERSION}"
)

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: $1 not found in PATH" >&2
    exit 1
  fi
}

cosign_sign() {
  local ref="$1"
  local yes_flag=()
  if [[ "${COSIGN_YES}" == "true" ]]; then
    yes_flag=(--yes)
  fi
  if [[ -n "${COSIGN_PRIVATE_KEY:-}" ]]; then
    cosign sign "${yes_flag[@]}" --key "env://COSIGN_PRIVATE_KEY" "${ref}"
  elif [[ -f "${COSIGN_KEY:-cosign.key}" ]]; then
    cosign sign "${yes_flag[@]}" --key "${COSIGN_KEY:-cosign.key}" "${ref}"
  else
    cosign sign "${yes_flag[@]}" "${ref}"
  fi
}

mkdir -p "${SBOM_DIR}"

for ref in "${IMAGES[@]}"; do
  name="${ref##*/}"
  name="${name//:/-}"
  sbom_file="${SBOM_DIR}/${name}.spdx.json"

  echo "==> ${ref}"

  if [[ "${SKIP_SBOM}" != "true" ]]; then
    require_cmd syft
    echo "    SBOM -> ${sbom_file}"
    syft scan "${ref}" -o spdx-json > "${sbom_file}"
  fi

  if [[ "${SKIP_SIGN}" != "true" ]]; then
    require_cmd cosign
    echo "    cosign sign"
    cosign_sign "${ref}"
    if [[ "${SKIP_SBOM}" != "true" && -f "${sbom_file}" ]]; then
      echo "    cosign attach sbom"
      # attach sbom does not support --yes (unlike cosign sign for keyless terms)
      cosign attach sbom --sbom "${sbom_file}" "${ref}"
    fi
  fi
done

echo "==> Supply chain artifacts in ${SBOM_DIR}"
