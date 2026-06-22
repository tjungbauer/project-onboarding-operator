#!/usr/bin/env bash
# Fail when total or per-package statement coverage is below configured minimums.
set -euo pipefail

COVER_FILE="${1:-cover.out}"
MIN_COVERAGE="${MIN_COVERAGE:-40}"

# package_import_path:minimum_percent
PACKAGE_MIN_COVERAGE="${PACKAGE_MIN_COVERAGE:-\
github.com/tjungbauer/project-onboarding-operator/internal/onboarding:45,\
github.com/tjungbauer/project-onboarding-operator/internal/validation:40,\
github.com/tjungbauer/project-onboarding-operator/internal/controller:30,\
github.com/tjungbauer/project-onboarding-operator/internal/webhook:30}"

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

IFS=',' read -r -a entries <<< "${PACKAGE_MIN_COVERAGE}"
for entry in "${entries[@]}"; do
  [[ -z "${entry}" ]] && continue
  pkg="${entry%%:*}"
  min_pkg="${entry##*:}"
  pkg_pct="$(go tool cover -func="${COVER_FILE}" | awk -v pkg="${pkg}/" 'index($1, pkg) == 1 {gsub(/%/,"",$3); sum+=$3; n++} END {if (n>0) printf "%.1f", sum/n; else print ""}')"
  if [[ -z "${pkg_pct}" ]]; then
    echo "warning: no coverage data for package ${pkg}" >&2
    continue
  fi
  echo "Package ${pkg}: ${pkg_pct}% (minimum ${min_pkg}%)"
  awk -v pct="${pkg_pct}" -v min="${min_pkg}" 'BEGIN { exit (pct + 0 >= min + 0) ? 0 : 1 }' || {
    echo "error: package ${pkg} coverage ${pkg_pct}% is below minimum ${min_pkg}%" >&2
    exit 1
  }
done
