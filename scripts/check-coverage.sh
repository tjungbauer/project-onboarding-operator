#!/usr/bin/env bash
# Fail when total statement coverage is below MIN_COVERAGE (default 25).
set -euo pipefail

COVER_FILE="${1:-cover.out}"
MIN_COVERAGE="${MIN_COVERAGE:-25}"

if [[ ! -f "${COVER_FILE}" ]]; then
  echo "error: coverage file not found: ${COVER_FILE}" >&2
  exit 1
fi

pct="$(go tool cover -func="${COVER_FILE}" | awk '/^total:/ {gsub(/%/,"",$3); print $3}')"
if [[ -z "${pct}" ]]; then
  echo "error: could not parse coverage from ${COVER_FILE}" >&2
  exit 1
fi

echo "Total coverage: ${pct}% (minimum ${MIN_COVERAGE}%)"
awk -v pct="${pct}" -v min="${MIN_COVERAGE}" 'BEGIN { exit (pct + 0 >= min + 0) ? 0 : 1 }' || {
  echo "error: coverage ${pct}% is below minimum ${MIN_COVERAGE}%" >&2
  exit 1
}
