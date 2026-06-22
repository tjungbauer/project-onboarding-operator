# Disaster recovery

`ProjectOnboarding` and `TShirtSize` CRs live in **etcd** (cluster state). Tenant namespaces and their workloads are the **operational data plane**. Recovery focuses on restoring the control plane and reconciling drift.

## What to back up

| Asset | Location | Recovery approach |
| ----- | -------- | ----------------- |
| `ProjectOnboarding` / `TShirtSize` CRs | Cluster etcd | Export with `oc get pob,tts -o yaml` or Velero |
| Tenant namespaces | Cluster | Namespace backup (Velero, etcd restore) or recreate from Git |
| Operator install | OLM CSV / Subscription | Reinstall from catalog or `./scripts/upgrade-cluster.sh` |
| Git source of truth | Your GitOps repo | Preferred — re-apply manifests after cluster restore |

## Operator failure scenarios

### Controller down

1. Check deployment: `oc get deploy -n project-onboarding-operator`
2. Follow [runbook.md — Alert response](runbook.md#alert-response)
3. Roll back if upgrade-related: `./scripts/rollback-cluster.sh <VERSION>`

### Cluster loss (new cluster)

1. Install OpenShift / Kubernetes 1.28+
2. Install operator ([install.md](install.md))
3. Re-apply CR manifests from Git or backup
4. Operator reconciles tenant namespaces; verify with `oc get pob -A`

### Accidental CR delete

- **ProjectOnboarding delete blocked** while tenant namespaces exist (finalizer). Offboard first.
- If finalizer was patched away, orphaned resources may remain — see [runbook.md — Stuck finalizers](runbook.md#stuck-finalizers)

## Rollback

```bash
./scripts/rollback-cluster.sh 0.0.49
```

Requires the target bundle/catalog images on Quay and the CSV in the catalog index. See [upgrade.md](upgrade.md).

## Related

- [upgrade.md](upgrade.md)
- [runbook.md](runbook.md)
- [capacity-performance.md](capacity-performance.md)
