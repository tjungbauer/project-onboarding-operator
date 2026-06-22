#!/usr/bin/env bash
# Roll back project-onboarding-operator on OpenShift to a previous OLM version.
#
# Usage:
#   ./scripts/rollback-cluster.sh 0.0.49
#   ./scripts/rollback-cluster.sh    # uses PREV_VERSION from VERSION file
#
# Delegates to upgrade-cluster.sh — rollback is an OLM upgrade to an older CSV
# that remains in the catalog index. The target bundle/catalog images must exist on Quay.
#
# See docs/upgrade.md and docs/disaster-recovery.md

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
  cat <<'EOF'
Usage: rollback-cluster.sh [<VERSION>]

  Roll back the operator to a previous release already published on Quay.

Examples:
  ./scripts/rollback-cluster.sh 0.0.49
  ./scripts/rollback-cluster.sh

Without VERSION, computes PREV_VERSION from the VERSION file (e.g. 0.0.50 -> 0.0.49).
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ -n "${1:-}" ]]; then
  TARGET="$1"
elif [[ -n "${ROLLBACK_VERSION:-}" ]]; then
  TARGET="${ROLLBACK_VERSION}"
elif [[ -f "${ROOT}/VERSION" ]]; then
  CURRENT="$(tr -d ' \n\r' < "${ROOT}/VERSION")"
  TARGET="$(python3 -c "v='${CURRENT}'.strip().split('.'); print(f'{v[0]}.{v[1]}.{int(v[2])-1}' if len(v)==3 and v[2].isdigit() and int(v[2])>0 else '')" 2>/dev/null || true)"
  if [[ -z "${TARGET}" ]]; then
    echo "error: could not derive PREV_VERSION from VERSION=${CURRENT}; pass VERSION explicitly" >&2
    exit 1
  fi
else
  echo "error: rollback VERSION required" >&2
  usage >&2
  exit 1
fi

echo "==> Rolling back project-onboarding-operator to ${TARGET}"
echo "    (OLM upgrade to previous CSV; images must exist on Quay)"

exec "${ROOT}/scripts/upgrade-cluster.sh" "${TARGET}"
