#!/usr/bin/env bash
# Attach SLSA build provenance attestations to release images (cosign attest).
#
# Usage:
#   ./scripts/attest-slsa-provenance.sh 0.0.50
#
# Requires cosign. In CI set COSIGN_EXPERIMENTAL=1 and id-token: write on the job.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"
QUAY_USER="${QUAY_USER:-tjungbau}"
REGISTRY="quay.io/${QUAY_USER}"
PREDICATE_DIR="${PREDICATE_DIR:-${ROOT}/dist/slsa}"
COSIGN_YES="${COSIGN_YES:-true}"

if [[ -z "${VERSION}" ]]; then
  if [[ -f "${ROOT}/VERSION" ]]; then
    VERSION="$(tr -d ' \n\r' < "${ROOT}/VERSION")"
  else
    echo "error: VERSION required" >&2
    exit 1
  fi
fi

IMAGES=(
  "${REGISTRY}/project-onboarding-operator:v${VERSION}"
  "${REGISTRY}/project-onboarding-operator-bundle:v${VERSION}"
  "${REGISTRY}/project-onboarding-operator-catalog:v${VERSION}"
)

if ! command -v cosign >/dev/null 2>&1; then
  echo "error: cosign not found in PATH" >&2
  exit 1
fi

mkdir -p "${PREDICATE_DIR}"

repo="${GITHUB_REPOSITORY:-tjungbauer/project-onboarding-operator}"
run_id="${GITHUB_RUN_ID:-local}"
run_attempt="${GITHUB_RUN_ATTEMPT:-1}"
sha="${GITHUB_SHA:-unknown}"
ref="${GITHUB_REF_NAME:-v${VERSION}}"
workflow="${GITHUB_WORKFLOW:-Release}"
server="${GITHUB_SERVER_URL:-https://github.com}"

predicate="${PREDICATE_DIR}/predicate-v${VERSION}.json"
cat > "${predicate}" <<EOF
{
  "_type": "https://in-toto.io/Statement/v1",
  "subject": [],
  "predicateType": "https://slsa.dev/provenance/v1",
  "predicate": {
    "buildDefinition": {
      "buildType": "https://github.com/tjungbauer/project-onboarding-operator/.github/workflows/release.yml",
      "externalParameters": {
        "workflow": {
          "ref": "${ref}",
          "repository": "${repo}",
          "path": ".github/workflows/release.yml"
        }
      },
      "internalParameters": {
        "github": {
          "event_name": "${GITHUB_EVENT_NAME:-push}",
          "repository_id": "${GITHUB_REPOSITORY_ID:-}",
          "repository_owner_id": "${GITHUB_REPOSITORY_OWNER_ID:-}",
          "runner_environment": "${RUNNER_ENVIRONMENT:-github-hosted}"
        }
      },
      "resolvedDependencies": [
        {
          "uri": "git+${server}/${repo}@${sha}",
          "digest": {
            "gitCommit": "${sha}"
          }
        }
      ]
    },
    "runDetails": {
      "builder": {
        "id": "${server}/${repo}/.github/workflows/release.yml@refs/tags/v${VERSION}"
      },
      "metadata": {
        "invocationId": "${server}/${repo}/actions/runs/${run_id}/attempts/${run_attempt}"
      }
    }
  }
}
EOF

yes_flag=()
if [[ "${COSIGN_YES}" == "true" ]]; then
  yes_flag=(--yes)
fi

sign_args=("${yes_flag[@]}")
if [[ -n "${COSIGN_PRIVATE_KEY:-}" ]]; then
  sign_args+=(--key "env://COSIGN_PRIVATE_KEY")
fi

echo "==> Attaching SLSA provenance for v${VERSION}"
for ref in "${IMAGES[@]}"; do
  echo "    ${ref}"
  cosign attest "${sign_args[@]}" --type slsaprovenance --predicate "${predicate}" "${ref}"
done

echo "==> SLSA provenance attestations attached"
