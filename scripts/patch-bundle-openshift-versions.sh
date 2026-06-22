#!/usr/bin/env bash
# Add com.redhat.openshift.versions to bundle metadata and bundle.Dockerfile (community OperatorHub).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ANNOTATIONS="${ROOT}/bundle/metadata/annotations.yaml"
DOCKERFILE="${ROOT}/bundle.Dockerfile"
OPENSHIFT_VERSIONS="${OPENSHIFT_VERSIONS:-v4.15-v4.22}"

if [[ ! -f "${ANNOTATIONS}" ]]; then
  echo "error: ${ANNOTATIONS} not found; run make bundle first" >&2
  exit 1
fi

python3 - "${ANNOTATIONS}" "${OPENSHIFT_VERSIONS}" <<'PY'
import pathlib
import re
import sys

path = pathlib.Path(sys.argv[1])
versions = sys.argv[2]
text = path.read_text()
key = "com.redhat.openshift.versions"
line = f"  {key}: {versions}"
if re.search(rf"^  {re.escape(key)}:", text, flags=re.M):
    text = re.sub(rf"^  {re.escape(key)}:.*$", line, text, count=1, flags=re.M)
else:
    marker = "  operators.operatorframework.io.metrics.project_layout:"
    if marker not in text:
        sys.exit(f"error: could not locate insertion point in {path}")
    text = text.replace(
        marker + " go.kubebuilder.io/v4\n",
        marker + " go.kubebuilder.io/v4\n\n  # OpenShift community OperatorHub catalog range.\n" + line + "\n",
        1,
    )
path.write_text(text)
print(f"==> Patched {path} ({key}={versions})")
PY

if [[ -f "${DOCKERFILE}" ]]; then
  python3 - "${DOCKERFILE}" "${OPENSHIFT_VERSIONS}" <<'PY'
import pathlib
import re
import sys

path = pathlib.Path(sys.argv[1])
versions = sys.argv[2]
text = path.read_text()
label = f"LABEL com.redhat.openshift.versions={versions}"
if re.search(r"^LABEL com.redhat.openshift.versions=", text, flags=re.M):
    text = re.sub(r"^LABEL com.redhat.openshift.versions=.*$", label, text, count=1, flags=re.M)
else:
    marker = "LABEL operators.operatorframework.io.metrics.project_layout=go.kubebuilder.io/v4"
    if marker not in text:
        sys.exit(f"error: could not locate insertion point in {path}")
    text = text.replace(marker + "\n", marker + "\n" + label + "\n", 1)
path.write_text(text)
print(f"==> Patched {path} (com.redhat.openshift.versions={versions})")
PY
fi
