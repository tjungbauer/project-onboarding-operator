#!/usr/bin/env bash
# Remove all OpenShift manual test resources (safe to re-run).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFESTS="${SCRIPT_DIR}/manifests"

oc delete -f "${MANIFESTS}/tc13-gitops-onboarding.yaml" --ignore-not-found
oc delete -f "${MANIFESTS}/tc05-custom-netpol.yaml" --ignore-not-found
oc delete -f "${MANIFESTS}/tc04-openshift-features.yaml" --ignore-not-found
oc delete -f "${MANIFESTS}/tc03-tshirt-onboarding.yaml" --ignore-not-found
oc delete -f "${MANIFESTS}/tc01-core-onboarding.yaml" --ignore-not-found
oc delete -f "${MANIFESTS}/tc02-tshirt-catalog.yaml" --ignore-not-found

# Tenant namespaces (in case CR delete is slow or stuck)
for ns in ocp-test-core-dev ocp-test-medium-dev ocp-test-egress-dev ocp-test-netpol-dev \
          ocp-test-gitops-dev; do
  oc delete namespace "${ns}" --ignore-not-found --wait=false 2>/dev/null || true
done

echo "Cleanup requested. Waiting for tenant namespaces to terminate..."
for ns in ocp-test-core-dev ocp-test-medium-dev ocp-test-egress-dev ocp-test-netpol-dev \
          ocp-test-gitops-dev; do
  for _ in $(seq 1 60); do
    if ! oc get namespace "${ns}" >/dev/null 2>&1; then
      break
    fi
    sleep 2
  done
done

echo "Cleanup complete. Remaining ocp-test namespaces:"
echo "  oc get namespace | grep ocp-test || true"
