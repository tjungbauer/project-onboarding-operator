#!/usr/bin/env bash
# Resolve Red Hat Hardened Image digests and update build/hi-images.lock + Dockerfile ARG defaults.
#
# Run before a release when you want to pick up new HI base images:
#   ./hack/resolve-hi-digests.sh
#
# Requires: podman or docker

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOCK="${ROOT}/build/hi-images.lock"
DOCKERFILE="${ROOT}/Dockerfile"
CONTAINER_TOOL="${CONTAINER_TOOL:-}"
if [[ -z "${CONTAINER_TOOL}" ]]; then
  if command -v podman >/dev/null 2>&1; then
    CONTAINER_TOOL=podman
  else
    CONTAINER_TOOL=docker
  fi
fi

resolve() {
  local ref="$1"
  "${CONTAINER_TOOL}" pull "${ref}" >/dev/null
  "${CONTAINER_TOOL}" inspect --format='{{.Digest}}' "${ref}"
}

echo "==> Resolving HI image digests (${CONTAINER_TOOL})"
GO_DIGEST="$(resolve registry.access.redhat.com/hi/go:latest-builder)"
RUNTIME_DIGEST="$(resolve registry.access.redhat.com/hi/core-runtime:latest)"

GO_IMAGE="registry.access.redhat.com/hi/go@${GO_DIGEST}"
RUNTIME_IMAGE="registry.access.redhat.com/hi/core-runtime@${RUNTIME_DIGEST}"

cat > "${LOCK}" <<EOF
# Red Hat Hardened Images — pinned by digest for reproducible builds.
# Update before a release when RH publishes new HI builds:
#   ./hack/resolve-hi-digests.sh
HI_GO_BUILDER_IMAGE=${GO_IMAGE}
HI_CORE_RUNTIME_IMAGE=${RUNTIME_IMAGE}
EOF

python3 - <<PY
from pathlib import Path
import re

dockerfile = Path("${DOCKERFILE}")
text = dockerfile.read_text()
text = re.sub(
    r'^ARG HI_GO_BUILDER_IMAGE=.*$',
    f'ARG HI_GO_BUILDER_IMAGE=${GO_IMAGE}',
    text,
    flags=re.M,
)
text = re.sub(
    r'^ARG HI_CORE_RUNTIME_IMAGE=.*$',
    f'ARG HI_CORE_RUNTIME_IMAGE=${RUNTIME_IMAGE}',
    text,
    flags=re.M,
)
dockerfile.write_text(text)
print("Updated", dockerfile)
PY

echo "==> Wrote ${LOCK}"
echo "    HI_GO_BUILDER_IMAGE=${GO_IMAGE}"
echo "    HI_CORE_RUNTIME_IMAGE=${RUNTIME_IMAGE}"
