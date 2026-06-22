#!/usr/bin/env bash
# Resolve Red Hat Hardened Image linux/amd64 digests into build/hi-images.lock.
#
#   ./hack/resolve-hi-digests.sh          # update lock file
#   ./hack/resolve-hi-digests.sh --check  # fail if lock differs from registry
#
# Run before a release when adopting new HI base images.
# Requires: podman or docker

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOCK="${ROOT}/build/hi-images.lock"
CHECK_ONLY=false
if [[ "${1:-}" == "--check" ]]; then
  CHECK_ONLY=true
fi

CONTAINER_TOOL="${CONTAINER_TOOL:-}"
if [[ -z "${CONTAINER_TOOL}" ]]; then
  if command -v podman >/dev/null 2>&1; then
    CONTAINER_TOOL=podman
  else
    CONTAINER_TOOL=docker
  fi
fi

amd64_digest() {
  local ref="$1"
  "${CONTAINER_TOOL}" manifest inspect "${ref}" | python3 -c "
import json, sys
manifest = json.load(sys.stdin)
for entry in manifest.get('manifests', []):
    p = entry.get('platform') or {}
    if p.get('os') == 'linux' and p.get('architecture') == 'amd64':
        print(entry['digest'])
        break
else:
    raise SystemExit('linux/amd64 digest not found')
"
}

GO_REF="registry.access.redhat.com/hi/go:latest-builder"
RUNTIME_REF="registry.access.redhat.com/hi/core-runtime:latest"
GO_DIGEST="$(amd64_digest "${GO_REF}")"
RUNTIME_DIGEST="$(amd64_digest "${RUNTIME_REF}")"
GO_IMAGE="registry.access.redhat.com/hi/go@${GO_DIGEST}"
RUNTIME_IMAGE="registry.access.redhat.com/hi/core-runtime@${RUNTIME_DIGEST}"

if [[ "${CHECK_ONLY}" == "true" ]]; then
  if [[ ! -f "${LOCK}" ]]; then
    echo "error: ${LOCK} not found; run ./hack/resolve-hi-digests.sh" >&2
    exit 1
  fi
  # shellcheck disable=SC1091
  source "${LOCK}"
  ok=true
  if [[ "${HI_GO_BUILDER_IMAGE_LINUX_AMD64}" != "${GO_IMAGE}" ]]; then
    echo "error: HI builder digest drift — lock has ${HI_GO_BUILDER_IMAGE_LINUX_AMD64}" >&2
    echo "       registry has ${GO_IMAGE}" >&2
    ok=false
  fi
  if [[ "${HI_CORE_RUNTIME_IMAGE_LINUX_AMD64}" != "${RUNTIME_IMAGE}" ]]; then
    echo "error: HI runtime digest drift — lock has ${HI_CORE_RUNTIME_IMAGE_LINUX_AMD64}" >&2
    echo "       registry has ${RUNTIME_IMAGE}" >&2
    ok=false
  fi
  if [[ "${ok}" != "true" ]]; then
    echo "hint: run ./hack/resolve-hi-digests.sh and commit build/hi-images.lock" >&2
    exit 1
  fi
  echo "==> HI image lock matches registry (${CONTAINER_TOOL})"
  exit 0
fi

echo "==> Resolving HI linux/amd64 digests (${CONTAINER_TOOL})"
cat > "${LOCK}" <<EOF
# Red Hat Hardened Images — platform-specific digests for reproducible linux/amd64 CI/release builds.
# Default Dockerfile ARGs use :latest-builder / :latest for local multi-arch builds.
# Update when adopting new HI builds:
#   ./hack/resolve-hi-digests.sh
HI_GO_BUILDER_IMAGE_LINUX_AMD64=${GO_IMAGE}
HI_CORE_RUNTIME_IMAGE_LINUX_AMD64=${RUNTIME_IMAGE}
EOF

echo "==> Wrote ${LOCK}"
cat "${LOCK}"
