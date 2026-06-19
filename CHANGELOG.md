# Changelog

All notable changes to this project are documented here. Version numbers match OLM bundle / image tags.

## [0.0.45] - 2026-06-19

### Fixed

- Namespace reconcile: reuse the namespace object from create/patch instead of a second cache read, avoiding a transient `Namespace not found` error on new tenants when the informer has not caught up yet.

## [0.0.44] - 2026-06-19

### Changed

- T-shirt sizing: catalogue quotas/limits apply only when `resourceQuotas.enabled` / `limitRanges.enabled` is explicitly `true` on the namespace entry; `projectSize` alone sets the `namespace-size` label only.
- T-shirt merge: field-level overrides require `overwriteTshirt: true` in addition to `enabled: true`; inline fields are ignored when overwrite is off.
- T-shirt limit ranges: no longer require a `resourceQuotas` block on the namespace entry (operator-only; Helm template had that gate).

### Added

- Operator metrics guide: [docs/metrics.md](docs/metrics.md) (ServiceMonitor, PrometheusRule, verification).

### Documentation

- Expanded T-shirt sizing and GitOps docs in `guide.md` and `project-size.md`; cluster-defaults cross-links.

## [0.0.43] - 2026-06-18

### Fixed

- ResourceQuota: removed invalid `limits.storage` hard limit (not supported by Kubernetes). Use `resourceQuotas.requests.storage` or `storageClasses[]` for storage quotas.

## [0.0.42] - 2026-06-18

### Changed

- Argo CD Casbin `proj:` prefix defaults to the **tenant namespace name** (`spec.namespaces[].name`); optional per-policy `appProjectName` overrides it (Helm `overwrite_appproject_name` / `policies[].namespace` equivalent without a role toggle).
- Application object path in policies still uses `argoCDProjects[].name`.

### Removed

- `overwriteAppProjectName` on roles (replaced by optional `appProjectName` on each policy).

## [0.0.40] - 2026-06-18

### Changed

- **Application GitOps Namespace** moved to `spec.namespaces[].applicationGitOpsNamespace` (per tenant entry; Helm `application_gitops_namespace`). Each namespace can target a different Argo CD instance.
- OpenShift form: hides the `gitOps` block; shows **Application GitOps Namespace** on the namespace row above **Argo CD Project**.
- Legacy fallback: `spec.gitOps.applicationNamespace` and ConfigMap `gitOps.applicationNamespace` still apply when the namespace field is empty.

## [0.0.39] - 2026-06-18

### Changed

- OpenShift form: **Application GitOps Namespace** (`spec.gitOps.applicationNamespace`) exposed above Argo CD Project — maps to Helm `application_gitops_namespace`.
- API/docs: clarified that this namespace is where the application-scoped Argo CD instance runs and AppProjects are created.

## [0.0.38] - 2026-06-18

### Changed

- OpenShift form: removed **GitOps** block (use `onboarding-defaults` ConfigMap or YAML for cluster GitOps defaults).
- OpenShift form: **Policy Resource**, **Policy Action**, and **Policy Permission** use dropdowns; removed CRD defaults for `resource` and `permission` so the form starts empty.

## [0.0.37] - 2026-06-18

### Changed

- OpenShift form: **Argo CD Project** — `enabled` is opt-in (defaults to false), listed first; removed pre-filled GitOps alm-example (`tenant1-onboarding`).
- OpenShift form: added **GitOps** specDescriptors for cluster-wide Argo CD defaults; expanded **Argo CD Project** field layout (name, overrides, roles, policies).

## [0.0.36] - 2026-06-18

### Changed

- OpenShift form: removed pre-filled **Custom Namespace Labels** (`team: payments`) from alm-examples; users add labels via **Add label** intentionally.

## [0.0.35] - 2026-06-18

### Changed

- Guarded `ProjectOnboarding` deletion: CR stays `Terminating` until every managed tenant namespace is offboarded (`offboard: true`) or deleted manually; removing a spec entry no longer auto-deletes the namespace.
- Status condition **`DeletionBlocked`** explains pending namespaces during blocked delete.

## [0.0.34] - 2026-06-18

### Changed

- **Namespace Admins** `enabled` is opt-in (defaults to false); checkbox unchecked in the OpenShift form until explicitly enabled.

## [0.0.33] - 2026-06-18

### Changed

- OpenShift form: **Namespace Admins** — Enabled checkbox above Cluster Role; display names **Cluster Role** and **Group Name**; field descriptions; removed pre-filled `admin` and `developer1` from alm-examples.

## [0.0.32] - 2026-06-18

### Changed

- Republish OLM catalog for OperatorHub (marketplace catalog image sync).

## [0.0.31] - 2026-06-17

### Changed

- User workload monitoring: apply `openshift.io/user-monitoring` instead of `openshift.io/cluster-monitoring`; explicit opt-out sets the label to `false` rather than removing it (absent label would re-enable monitoring).
- OpenShift form: **User Workload Monitoring** descriptor documents opt-out behaviour; removed pre-filled tenant namespace name from alm-examples.

## [0.0.30] - 2026-06-17

### Added

- Validating webhook: rejects memory, ephemeral storage, and storage quantities without a unit suffix (e.g. bare `4`); applies to `ProjectOnboarding` and `TShirtSize` resource quota / limit range fields.

### Changed

- OpenShift form: expanded descriptions on CPU, memory, storage, and ephemeral storage fields (unit suffix required for byte quantities).
- `docs/guide.md`: quantity notation table, admission test examples.

## [0.0.29] - 2026-06-17

### Fixed

- OpenShift user-workload monitoring: `ServiceMonitor` uses `authorization` + token `Secret` instead of forbidden `bearerTokenFile`; added `metrics-reader` ClusterRoleBinding and scrape token secret.
- Removed duplicate unprefixed `ServiceMonitor` from the OLM bundle (single prefixed resource via `config/default`).
- NetworkPolicy: allow metrics scrapes from namespaces with `network.openshift.io/policy-group=monitoring` (OpenShift monitoring stack).

## [0.0.28] - 2026-06-17

### Changed

- OpenShift form display names: **Additional Namespace Settings**, **Persistent Volume Claims** (limit ranges), **Default Network Policies**, **Namespace Admins**, **Network Policies**, **ArgoCD Project**, **Project T-Shirt Size (TSS Resource Name)**.

## [0.0.26] - 2026-06-17

### Changed

- `storageClasses` under `resourceQuotas` is a `{key, value}` list in the CR (OpenShift form); reconciled tenant `ResourceQuota.spec.hard` still uses dict-style StorageClass quota keys.
- OpenShift form: **Limit Ranges** field order, descriptions, and empty defaults (**Enabled** starts deselected; removed CRD default `enabled: true` on `limitRanges`).
- **Default policies** are opt-in (all toggles off in form/alm-examples); OLM descriptors with per-policy descriptions. Omitting `defaultPolicies` creates no default NetworkPolicies.

## [0.0.24] - 2026-06-17

### Changed

- Renamed `spec.namespaces[].additionalSettings.labels` to **`additionalLabels`** so the OpenShift console accepts the list field (avoids conflict with map-shaped `labels` elsewhere in the schema).
- OpenShift form: **Resource Quotas** field order, descriptions, and empty defaults (no pre-filled example values; **Enabled** starts deselected). `storageClasses` changed from map to `{key, value}` list so StorageClass quotas render in the console form.

## [0.0.23] - 2026-06-17

### Changed

- Custom namespace labels under `additionalSettings` as a list of `{key, value}` pairs for dynamic **Add label** in the operator form.

### Fixed

- Kind E2E: conditional Argo CD watch, metrics NetworkPolicy for in-namespace scrapers, pinned curl probe image, non-blocking teardown.
- Metrics NetworkPolicy: allow in-namespace scrapers; operator namespace labeled `metrics: enabled`.

## [0.0.22] - 2026-06-17

Initial release.

### Added

- Cluster-scoped `ProjectOnboarding` and `TShirtSize` CRDs with validating webhooks and conversion (`v1alpha1` → `v1beta1` storage).
- T-shirt sizing catalogue, namespace onboarding (quotas, limit ranges, network policies), and drift correction.
- OpenShift integration: local admin groups, EgressIP (OVN-Kubernetes), SCC binding, serving-cert webhook TLS.
- GitOps / Argo CD `AppProject` reconciliation (when the CRD is present).
- OLM bundle and OperatorHub install path; Helm → CR migration guide.
- HA controller deployment (3 replicas, PodDisruptionBudget), metrics and webhook NetworkPolicies, optional Prometheus Operator resources.
- Kind E2E (`make test-e2e`) and OpenShift E2E (`make test-e2e-openshift`, TC-00–TC-14).
- Cluster defaults via optional `onboarding-defaults` ConfigMap; reconciliation pause annotation.

### Changed

- `spec.namespaces[].enabled` freezes reconciliation when `false`; use `spec.namespaces[].offboard` to tear down tenant resources.
- T-shirt overrides require `overwriteTshirt: true` when `projectSize` is set.
- OLM channel **stable**; operator form and sample manifests aligned with current API.
