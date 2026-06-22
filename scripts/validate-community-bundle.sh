#!/usr/bin/env bash
# Re-run Phase 1 validators on an existing dist/community-bundle/<VERSION> tree.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"
if [[ -z "${VERSION}" ]]; then
  VERSION="$(tr -d ' \n\r' < "${ROOT}/VERSION")"
fi

BUNDLE_DIR="${ROOT}/dist/community-bundle/${VERSION}"
if [[ ! -d "${BUNDLE_DIR}/manifests" ]]; then
  echo "error: ${BUNDLE_DIR} not found; run ./scripts/prepare-community-bundle.sh ${VERSION}" >&2
  exit 1
fi

echo "==> Validating community bundle at ${BUNDLE_DIR}"

operator-sdk bundle validate "${BUNDLE_DIR}"
operator-sdk bundle validate "${BUNDLE_DIR}" --select-optional name=community || true

if command -v ocp-olm-catalog-validator >/dev/null 2>&1; then
  ocp-olm-catalog-validator "${BUNDLE_DIR}" \
    --optional-values="file=${BUNDLE_DIR}/metadata/annotations.yaml"
else
  echo "warning: ocp-olm-catalog-validator not installed" >&2
  exit 1
fi

echo "==> Community bundle validation passed for v${VERSION}"
