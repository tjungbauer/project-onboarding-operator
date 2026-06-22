#!/usr/bin/env bash
# Promote unprefixed ServiceMonitor/PrometheusRule bundle manifests to prefixed names.
# operator-sdk emits short resource names from config/prometheus; OLM bundles keep prefixed copies.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MANIFESTS="${ROOT}/bundle/manifests"

promote() {
  local file_suffix="$1"
  local old_name="$2"
  local new_name="$3"
  local src="${MANIFESTS}/${file_suffix}"
  local dst="${MANIFESTS}/project-onboarding-operator-${file_suffix}"
  if [[ -f "${src}" ]]; then
    sed "s/name: ${old_name}/name: ${new_name}/" "${src}" > "${dst}"
    rm -f "${src}"
  fi
}

promote "controller-manager-metrics-monitor_monitoring.coreos.com_v1_servicemonitor.yaml" \
  "controller-manager-metrics-monitor" "project-onboarding-operator-controller-manager-metrics-monitor"
promote "controller-manager-rules_monitoring.coreos.com_v1_prometheusrule.yaml" \
  "controller-manager-rules" "project-onboarding-operator-controller-manager-rules"
promote "controller-manager-nonroot-v2_rbac.authorization.k8s.io_v1_clusterrolebinding.yaml" \
  "controller-manager-nonroot-v2" "project-onboarding-operator-controller-manager-nonroot-v2"
