# Operator metrics and monitoring

The controller exposes **Prometheus-compatible metrics** over HTTPS. The OLM bundle ships a **ServiceMonitor** and **PrometheusRule** so OpenShift cluster monitoring (or any Prometheus Operator stack) can scrape the operator and fire alerts on reconcile problems.

## What gets exposed

The manager listens on **`:8443`** (`--metrics-bind-address=:8443`) with **RBAC filtering** enabled (`--metrics-secure=true`). Metrics come from controller-runtime and include standard series such as:

| Metric (examples) | Meaning |
| ----------------- | ------- |
| `controller_runtime_reconcile_total` | Reconcile attempts per controller |
| `controller_runtime_reconcile_errors_total` | Failed reconciles (`projectonboarding`, `tshirtsize`, …) |
| `projectonboarding_tenants_total{project_onboarding="..."}` | Active tenant namespace entries (enabled, not offboarded) per CR |
| `projectonboarding_reconcile_errors_total{reason="..."}` | ProjectOnboarding reconcile errors by reason |
| `workqueue_depth` | Pending items in the controller work queue |
| `workqueue_adds_total` | Items added to the work queue |

These help you spot stuck reconciles, error spikes, or backlog before tenant namespaces drift out of sync.

## What the ServiceMonitor does

The `ServiceMonitor` selects controller pods (`control-plane: controller-manager`) and scrapes:

```yaml
path: /metrics
port: https
scheme: https
authorization:
  type: Bearer
  credentials:
    name: <metrics-reader-token-secret>
    key: token
tlsConfig:
  insecureSkipVerify: true   # self-signed controller certs in default bundle
```

Prometheus Operator watches `ServiceMonitor` CRs. When your Prometheus instance uses a `serviceMonitorSelector` that matches these labels, it **automatically** adds this target — no manual `scrape_config` edits.

On OpenShift, user-workload or platform monitoring Prometheus picks up ServiceMonitors in the operator namespace once the **Prometheus Operator CRDs** are present.

## NetworkPolicy and scraper access

Metrics traffic is restricted by `allow-metrics-traffic` NetworkPolicy. Scrapers may reach `:8443` when they run:

- **Inside the operator namespace** (debug curl pod, Kind E2E), or
- In a namespace labeled **`metrics: enabled`**, or
- In OpenShift monitoring namespaces (`network.openshift.io/policy-group: monitoring`).

The operator namespace is labeled `metrics: enabled` at deploy time. For custom Prometheus in another namespace:

```bash
oc label namespace <prometheus-namespace> metrics=enabled --overwrite
```

## PrometheusRule alerts

The bundled `PrometheusRule` defines warning alerts (10–15 minute `for` windows):


| Alert | Severity | Condition |
| ----- | -------- | --------- |
| `ProjectOnboardingOperatorDown` | **critical** | `kube_deployment_status_replicas_available` for controller-manager &lt; 1 (requires kube-state-metrics) |
| `ProjectOnboardingReconcileErrors` | warning | `controller_runtime_reconcile_errors_total{controller="projectonboarding"}` rate > 0 |
| `ProjectOnboardingReconcileErrorsByReason` | `projectonboarding_reconcile_errors_total` rate > 0 by `reason` |
| `ProjectOnboardingWorkqueueBacklog` | `workqueue_depth{name="projectonboarding"}` > 10 |
| `TShirtSizeReconcileErrors` | `controller_runtime_reconcile_errors_total{controller="tshirtsize"}` rate > 0 |

Ensure your Prometheus `ruleSelector` includes the operator’s `PrometheusRule` labels.

## Grafana dashboard

Import [grafana/dashboard.json](grafana/dashboard.json) into Grafana (Dashboards → Import). Panels cover tenant count, reconcile errors, workqueue depth, and controller deployment availability.

## Prerequisites

| Requirement | Notes |
| ----------- | ----- |
| Prometheus Operator CRDs | `servicemonitors.monitoring.coreos.com`, `prometheusrules.monitoring.coreos.com` |
| Running Prometheus | OpenShift cluster monitoring, user-workload monitoring, or self-managed Prometheus with Operator |

Without Prometheus Operator CRDs, the operator still serves metrics on `:8443`, but **no** `ServiceMonitor` / `PrometheusRule` objects are created (OLM skips unknown CRDs).

## Verify on OpenShift

```bash
export OPERATOR_NS=project-onboarding-operator

# Resources present (skip if Prometheus Operator not installed)
oc get servicemonitor,prometheusrule -n "${OPERATOR_NS}" | grep controller-manager

# Metrics Service endpoint
oc get endpoints -n "${OPERATOR_NS}" \
  project-onboarding-operator-controller-manager-metrics-service \
  -o jsonpath='{.subsets[0].ports[0].port}{"\n"}'
# Expected: 8443

# Optional: manual scrape from a pod in a labeled namespace
oc label namespace "${OPERATOR_NS}" metrics=enabled --overwrite
```

Full test procedure: [openshift-testcases.md — TC-14](openshift-testcases.md#tc-14--observability-optional).

## Manual scrape (debug)

From a pod in the operator namespace (NetworkPolicy allows in-namespace traffic):

```bash
TOKEN=$(oc get secret -n "${OPERATOR_NS}" \
  project-onboarding-operator-controller-manager-metrics-reader-token \
  -o jsonpath='{.data.token}' | base64 -d)

oc run metrics-curl --rm -i -n "${OPERATOR_NS}" --image=curlimages/curl:latest --restart=Never -- \
  curl -sk -H "Authorization: Bearer ${TOKEN}" \
  "https://project-onboarding-operator-controller-manager-metrics-service.${OPERATOR_NS}.svc:8443/metrics" \
  | head -20
```

## TLS with cert-manager (optional)

The default bundle uses controller-runtime self-signed TLS with `insecureSkipVerify: true` on the ServiceMonitor. For cert-manager–signed metrics certs, apply overlay `config/overlays/metrics-tls/` (non-OLM / `make deploy` flows). See [local-testing.md](local-testing.md).

## Related documentation

| Topic | Document |
| ----- | -------- |
| OpenShift security summary | [guide.md — Security](guide.md#security-and-platform-integration) |
| TC-14 observability test | [openshift-testcases.md](openshift-testcases.md) |
| README metrics note | [README.md](../README.md) |
