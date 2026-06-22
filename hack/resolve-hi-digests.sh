#!/usr/bin/env bash
# Resolve Red Hat Hardened Image linux/amd64 digests into build/hi-images.lock.
#
# Run before a release when adopting new HI base images:
#   ./hack/resolve-hi-digests.sh
#
# Requires: podman or docker

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOCK="${ROOT}/build/hi-images.lock"
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
    raise SystemExit('linux/amd64 digest not found for ${ref}')
"
}

echo "==> Resolving HI linux/amd64 digests (${CONTAINER_TOOL})"
GO_REF="registry.access.redhat.com/hi/go:latest-builder"
RUNTIME_REF="registry.access.redhat.com/hi/core-runtime:latest"
GO_DIGEST="$(amd64_digest "${GO_REF}")"
RUNTIME_DIGEST="$(amd64_digest "${RUNTIME_REF}")"

cat > "${LOCK}" <<EOF
# Red Hat Hardened Images — platform-specific digests for reproducible linux/amd64 CI/release builds.
# Default Dockerfile ARGs use :latest-builder / :latest for local multi-arch builds.
# Update when adopting new HI builds:
#   ./hack/resolve-hi-digests.sh
HI_GO_BUILDER_IMAGE_LINUX_AMD64=registry.access.redhat.com/hi/go@${GO_DIGEST}
HI_CORE_RUNTIME_IMAGE_LINUX_AMD64=registry.access.redhat.com/hi/core-runtime@${RUNTIME_DIGEST}
EOF

echo "==> Wrote ${LOCK}"
cat "${LOCK}"
