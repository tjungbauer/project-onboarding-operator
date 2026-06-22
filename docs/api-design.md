# API design

## Cluster-scoped `ProjectOnboarding` CR

`ProjectOnboarding` is **cluster-scoped**. Each entry in `spec.namespaces[]` declares a tenant `Namespace` and the resources the operator creates inside it (`ResourceQuota`, `LimitRange`, `NetworkPolicy`, OpenShift `Group`, `RoleBinding`, `EgressIP`, …).


| Aspect                 | Rationale                                                                                                                                                                                 |
| ---------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Cluster onboarding** | The operator provisions cluster-scoped and namespaced platform resources; the CR matches that scope.                                                                                      |
| **Owner references**   | A cluster-scoped `ProjectOnboarding` can own cluster-scoped children (`Namespace`, `Group`, `EgressIP`). Namespaced children in the tenant namespace are owned by the tenant `Namespace`. |
| **Discovery**          | `kubectl get projectonboarding` / `oc get pob` — no operator namespace required for the CR itself.                                                                                        |
| **RBAC**               | Use `ClusterRole` bindings (`projectonboarding-editor-role`, etc.) to grant teams create/update on `projectonboardings`.                                                                  |


### Recommended usage

1. Install the operator once (Deployment or OLM, AllNamespaces).
2. Create `TShirtSize` catalogue entries cluster-wide (optional).
3. Create `ProjectOnboarding` objects cluster-wide; `metadata.name` is the onboarding object name (e.g. `team-beta-onboarding`).
4. Use `spec.namespaces[].name` for the **tenant** namespace (e.g. `team-beta-dev`).

### What this is not

- The CR is **not** stored in the tenant namespace.
- `metadata.name` is **not** the tenant namespace — only `spec.namespaces[].name` controls provisioning.

## Companion CR: `TShirtSize`

`TShirtSize` is also cluster-scoped. `spec.namespaces[].projectSize` references `TShirtSize.metadata.name`.

## Garbage collection and owner references


| Resource                                                                     | GC / ownership                                                                      |
| ---------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| Tenant `Namespace`, OpenShift `Group`, `EgressIP`                            | **Controller reference** on `ProjectOnboarding` + finalizer teardown                |
| `ResourceQuota`, `LimitRange`, `NetworkPolicy`, `RoleBinding` (in tenant NS) | **Owner reference** on tenant `Namespace`                                           |
| Drift correction                                                             | Controller watches managed child types and re-queues the parent `ProjectOnboarding` |


Finalizer cleanup remains for ordered teardown (network policies before namespace delete, etc.) even when owner references exist.

**Deleting `ProjectOnboarding`:** a finalizer blocks completion while any operator-managed tenant namespace still exists. Offboard each entry (`offboard: true`) or delete the namespace manually. Dropping a namespace from `spec.namespaces[]` does not delete it.

Status condition `DeletionBlocked` / `AwaitingOffboard` includes a message like:

```text
Cannot delete ProjectOnboarding while managed tenant namespaces still exist. Set offboard=true on each namespace entry or delete the tenant namespaces manually. Pending: team-payments-dev
```

Steps: [guide.md — Lifecycle](guide.md#lifecycle-enable-freeze-offboard-and-delete).

## Admission validation

Structural rules use **CRD CEL** (`ProjectOnboardingSpec`, `NamespaceSpec`, `TShirtSizeSpec`).

Cross-object checks use **validating admission webhooks**:

- `projectSize` must reference an existing `TShirtSize` with usable quota/limit data
- `TShirtSize` delete blocked while referenced
- Duplicate `spec.namespaces[].name` values (webhook)

Webhooks use OpenShift service serving certificates (`config/openshift/webhook_cabundle_patch.yaml`, `config/openshift/webhook_service_tls_patch.yaml`).

## API version (`v1alpha1` / `v1beta1`)


| Version   | Role |
| --------- | ---- |
| `v1beta1` | Storage version (preferred for new manifests) |
| `v1alpha1` | Served; converted via conversion webhook (`/convert` on the operator webhook service) |

## API stability policy

| Level | Meaning for this project |
|-------|--------------------------|
| **OLM maturity `stable`** | Operator packaging and reconcile behaviour are production-supported; upgrades are documented in [upgrade.md](upgrade.md). |
| **API group `onboarding.stderr.at`** | Domain is stable; breaking group renames require a new API group and migration. |
| **`v1beta1`** | Preferred storage version. Field additions are backward-compatible. Breaking field removals or semantic changes require a deprecation period (≥ one minor release) and `CHANGELOG.md` notice. |
| **`v1alpha1`** | Legacy served version. New features land in `v1beta1` first. `v1alpha1` may be removed after deprecation when no clients remain. |

Before promoting to `v1` (if ever): publish a migration guide, run conversion tests, and bump CSV/API docs.

## GitOps / Argo CD AppProject

When `spec.gitOps` is set, the operator:

- Applies `argocd.argoproj.io/managed-by: <applicationNamespace>` on tenant namespaces
- Creates `AppProject` resources in the Argo CD namespace from `spec.namespaces[].argoCDProjects[]`

This mirrors `helper-proj-onboarding` `argocd-project.yaml` and tenant `argocd_rbac_setup` values.

