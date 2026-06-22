# Capacity and performance

Guidance for sizing and operating **project-onboarding-operator** at scale.

## Operator deployment

| Setting | Default (OLM) | Notes |
| ------- | ------------- | ----- |
| Replicas | **3** | Leader election; only leader runs controllers |
| `MaxConcurrentReconciles` | **2** per controller | `projectonboarding`, `tshirtsize` |
| CPU request / limit | 50m / 500m | `config/manager/manager.yaml` |
| Memory request / limit | 128Mi / 256Mi | Increase if many CRs or large specs |
| Resync period | **5 minutes** | Full cache resync (`cmd/main.go`) |

## CR scale guidance

| Dimension | Practical starting point | Watch |
| --------- | ------------------------ | ----- |
| `ProjectOnboarding` CRs | 50–200 cluster-wide | Workqueue depth alert |
| Namespaces per CR | Up to **64** (CRD max) | Reconcile duration grows linearly |
| Total tenant namespaces | 500+ | Monitor `projectonboarding_tenants_total`, API latency |

## Under load

- **Workqueue backlog:** `ProjectOnboardingWorkqueueBacklog` fires when `workqueue_depth{name="projectonboarding"} > 10` for 15m. Check API server latency, RBAC denials, or hot-looping CRs.
- **API slowness:** Requeues use controller-runtime backoff; transient errors surface as `reason=transient`.
- **Pause reconciliation:** `onboarding.stderr.at/pause-reconciliation: "true"` on a CR skips reconcile without uninstalling.

## Load testing

- Unit load test: `internal/onboarding/load_test.go` (`TestReconcileManyNamespaceEntries`) exercises 32 namespace entries with a fake client.
- OpenShift: apply many CRs in a test cluster and watch metrics / Grafana dashboard.

## HA

- PodDisruptionBudget limits voluntary disruption during node drains.
- Pod anti-affinity prefers spreading controller pods across nodes.
- Webhooks and metrics hit the Service fronting all replicas.

## Related

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [metrics.md](metrics.md)
- [slo.md](slo.md)
