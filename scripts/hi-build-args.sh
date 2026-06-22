#!/usr/bin/env bash
# Print docker/podman --build-arg flags for pinned HI images (linux/amd64 by default).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${ROOT}/build/hi-images.lock"

PLATFORM="${PLATFORM:-linux/amd64}"

case "${PLATFORM}" in
  linux/amd64)
    printf '%s\n' \
      "--build-arg" "HI_GO_BUILDER_IMAGE=${HI_GO_BUILDER_IMAGE_LINUX_AMD64}" \
      "--build-arg" "HI_CORE_RUNTIME_IMAGE=${HI_CORE_RUNTIME_IMAGE_LINUX_AMD64}"
    ;;
  *)
    # Local arm64/s390x builds use Dockerfile tag defaults.
    ;;
esac
