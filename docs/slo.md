# SLOs and error budgets

Service level objectives for **project-onboarding-operator** when platform monitoring (Prometheus) is enabled.

## Objectives

| SLI | SLO target | Measurement |
| --- | ---------- | ----------- |
| Operator availability | **99.5%** monthly | `up{service=~".*controller-manager-metrics-service"}` or deployment replicas available |
| Reconcile success | **99%** of 5m windows without errors | `rate(controller_runtime_reconcile_errors_total{controller="projectonboarding"}[5m]) == 0` |
| Tenant sync latency | **95%** of namespace entries `Ready` within **15 minutes** of CR apply | `ProjectOnboarding` status conditions + manual spot checks |

## Error budget

- **Reconcile errors:** sustained `ProjectOnboardingReconcileErrors` or `ProjectOnboardingReconcileErrorsByReason` alerts consume budget. Investigate when firing > 1 hour/day.
- **Operator down:** `ProjectOnboardingOperatorDown` (critical) consumes full availability budget for the incident duration.

## Recording rules

Bundled `PrometheusRule` includes:

- `projectonboarding:reconcile_error_rate5m`
- `projectonboarding:tenant_count`

Use these in Grafana ([grafana/dashboard.json](grafana/dashboard.json)) for trend panels.

## Alert routing

Configure receivers per [metrics.md — Alert routing](metrics.md#alert-routing-alertmanager--on-call). Example `AlertmanagerConfig` is **not** in the OLM bundle — apply `config/prometheus/alertmanagerconfig.yaml` manually when the `AlertmanagerConfig` CRD is available.

## Related

- [metrics.md](metrics.md)
- [runbook.md](runbook.md)
