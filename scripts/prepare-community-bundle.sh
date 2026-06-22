#!/usr/bin/env bash
# Prepare a community-operators-prod bundle directory (Phase 1 + validation).
#
# Usage:
#   ./scripts/prepare-community-bundle.sh [VERSION]
#
# Environment:
#   QUAY_USER                  default: tjungbau
#   OPENSHIFT_VERSIONS         default: v4.15-v4.22
#   COMMUNITY_FIRST_VERSION    default: true — remove spec.replaces for first community PR
#   OUTPUT_DIR                 default: dist/community-bundle/<VERSION>
#   SKIP_VALIDATE              default: false
#
# Produces dist/community-bundle/<VERSION>/{manifests,metadata,tests} ready to copy into
# community-operators-prod/operators/project-onboarding-operator/<VERSION>/.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

VERSION="${1:-}"
if [[ -z "${VERSION}" ]]; then
  VERSION="$(tr -d ' \n\r' < VERSION)"
fi

QUAY_USER="${QUAY_USER:-tjungbau}"
OPENSHIFT_VERSIONS="${OPENSHIFT_VERSIONS:-v4.15-v4.22}"
COMMUNITY_FIRST_VERSION="${COMMUNITY_FIRST_VERSION:-true}"
OUTPUT_DIR="${OUTPUT_DIR:-${ROOT}/dist/community-bundle/${VERSION}}"
SKIP_VALIDATE="${SKIP_VALIDATE:-false}"

IMG="quay.io/${QUAY_USER}/project-onboarding-operator:v${VERSION}"
CSV_NAME="project-onboarding-operator.clusterserviceversion.yaml"
SRC_CSV="${ROOT}/bundle/manifests/${CSV_NAME}"
OUT_CSV="${OUTPUT_DIR}/manifests/${CSV_NAME}"

usage() {
  cat <<EOF
Usage: prepare-community-bundle.sh [<VERSION>]

Prepare bundle tree for redhat-openshift-ecosystem/community-operators-prod.
Requires bundle/ in repo to match VERSION (run: make bundle IMG=${IMG}).
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

echo "==> Preparing community bundle for v${VERSION}"
echo "    Image: ${IMG}"
echo "    OpenShift versions: ${OPENSHIFT_VERSIONS}"
echo "    Output: ${OUTPUT_DIR}"

if [[ ! -f "${SRC_CSV}" ]] || ! grep -qE "^  version: ${VERSION}$" "${SRC_CSV}"; then
  echo "error: bundle CSV is not v${VERSION}; run:" >&2
  echo "  make bundle IMG=${IMG} VERSION=${VERSION}" >&2
  exit 1
fi

export OPENSHIFT_VERSIONS
chmod +x scripts/patch-bundle-openshift-versions.sh
scripts/patch-bundle-openshift-versions.sh

resolve_image_digest() {
  local ref="$1"
  if command -v skopeo >/dev/null 2>&1; then
    skopeo inspect --override-arch amd64 --override-os linux "docker://${ref}" --format '{{.Digest}}'
    return 0
  fi
  local repo_digest
  repo_digest="$(podman inspect --format='{{index .RepoDigests 0}}' "${ref}" 2>/dev/null || docker inspect --format='{{index .RepoDigests 0}}' "${ref}" 2>/dev/null || true)"
  if [[ -n "${repo_digest}" && "${repo_digest}" == *@sha256:* ]]; then
    echo "${repo_digest#*@}"
    return 0
  fi
  echo "error: could not resolve digest for ${ref} (install skopeo or pull image first)" >&2
  return 1
}

echo "==> Resolving operator image digest"
OPERATOR_DIGEST="$(resolve_image_digest "${IMG}")"
echo "    ${OPERATOR_DIGEST}"

rm -rf "${OUTPUT_DIR}"
mkdir -p "${OUTPUT_DIR}"
cp -a "${ROOT}/bundle/manifests" "${OUTPUT_DIR}/"
cp -a "${ROOT}/bundle/metadata" "${OUTPUT_DIR}/"
cp -a "${ROOT}/bundle/tests" "${OUTPUT_DIR}/"

python3 scripts/pin-csv-operator-image-digest.py \
  --csv "${OUT_CSV}" \
  --image "${IMG}" \
  --digest "${OPERATOR_DIGEST}"

if [[ "${COMMUNITY_FIRST_VERSION}" == "true" ]]; then
  python3 - "${OUT_CSV}" <<'PY'
import pathlib
import re
import sys

path = pathlib.Path(sys.argv[1])
text = path.read_text()
text, count = re.subn(r"^  replaces:.*\n", "", text, count=1, flags=re.M)
path.write_text(text)
print(f"==> Removed spec.replaces from {path.name} (COMMUNITY_FIRST_VERSION)")
PY
fi

verify_csv_fields() {
  local missing=0
  if ! grep -q 'containerImage:' "${OUT_CSV}"; then
    echo "error: missing containerImage annotation in CSV" >&2
    missing=1
  fi
  if ! grep -q '^  relatedImages:' "${OUT_CSV}"; then
    echo "error: missing relatedImages in CSV" >&2
    missing=1
  fi
  if ! grep -q "${OPERATOR_DIGEST}" "${OUT_CSV}"; then
    echo "error: CSV does not reference operator digest ${OPERATOR_DIGEST}" >&2
    missing=1
  fi
  if [[ "${COMMUNITY_FIRST_VERSION}" == "true" ]] && grep -q '^  replaces:' "${OUT_CSV}"; then
    echo "error: replaces still present with COMMUNITY_FIRST_VERSION=true" >&2
    missing=1
  fi
  if ! grep -q "com.redhat.openshift.versions: ${OPENSHIFT_VERSIONS}" "${OUTPUT_DIR}/metadata/annotations.yaml"; then
    echo "error: missing com.redhat.openshift.versions in metadata/annotations.yaml" >&2
    missing=1
  fi
  shopt -s nullglob
  if compgen -G "${OUTPUT_DIR}/manifests/*networkpolicy*.yaml" > /dev/null; then
    echo "error: community bundle must not contain operator-namespace NetworkPolicy manifests" >&2
    missing=1
  fi
  shopt -u nullglob
  if ! grep -q 'metadata.annotations.support:' "${OUT_CSV}"; then
    echo "error: missing metadata.annotations.support in CSV" >&2
    missing=1
  fi
  if ! awk '/^metadata:/{m=1} m && /^  annotations:/{a=1} a && /^    description:/{found=1; exit} END{exit !found}' "${OUT_CSV}"; then
    echo "error: missing metadata.annotations.description in CSV" >&2
    missing=1
  fi
  return "${missing}"
}

echo "==> Verifying community bundle fields"
verify_csv_fields

if [[ "${SKIP_VALIDATE}" != "true" ]]; then
  if ! command -v operator-sdk >/dev/null 2>&1; then
    echo "error: operator-sdk not found" >&2
    exit 1
  fi
  echo "==> operator-sdk bundle validate"
  operator-sdk bundle validate "${OUTPUT_DIR}"
  echo "==> operator-sdk community optional validator"
  operator-sdk bundle validate "${OUTPUT_DIR}" --select-optional name=community || true

  if command -v ocp-olm-catalog-validator >/dev/null 2>&1; then
    echo "==> ocp-olm-catalog-validator"
    ocp-olm-catalog-validator "${OUTPUT_DIR}" \
      --optional-values="file=${OUTPUT_DIR}/metadata/annotations.yaml"
  else
    echo "==> Skipping ocp-olm-catalog-validator (build from https://github.com/redhat-openshift-ecosystem/ocp-olm-catalog-validator)"
  fi
fi

echo "==> Community bundle ready: ${OUTPUT_DIR}"
echo "    Copy into community-operators-prod:"
echo "      operators/project-onboarding-operator/${VERSION}/"
