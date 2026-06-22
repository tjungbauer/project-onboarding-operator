#!/usr/bin/env bash
# Remove operator-namespace NetworkPolicy manifests from the OLM bundle.
# OLM cannot install or manage NetworkPolicy resources in the operator bundle.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MANIFESTS="${ROOT}/bundle/manifests"

removed=0
shopt -s nullglob
for f in "${MANIFESTS}"/*networkpolicy*.yaml; do
  rm -f "${f}"
  removed=$((removed + 1))
  echo "==> Removed $(basename "${f}") (OLM cannot install operator-namespace NetworkPolicies)"
done
shopt -u nullglob

if [[ "${removed}" -eq 0 ]]; then
  echo "==> No operator-namespace NetworkPolicy manifests in bundle"
fi
