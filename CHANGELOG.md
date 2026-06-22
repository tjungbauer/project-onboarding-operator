# Changelog

All notable changes to this project are documented here. Version numbers match OLM bundle / image tags.

## [Unreleased]

### Fixed

- CI: OpenShift E2E skips when `OPENSHIFT_KUBECONFIG` is unset (no longer fails `main` pushes).
- CI: `catalog-build` uses local bundle images (`--pull-tool none --build-tool`) in PR validation.
- CI: Security workflow scans operator and bundle images only (catalog inherits opm CVEs).

## [0.0.48] - 2026-06-22

### Added

- `docs/upgrade.md` and `scripts/upgrade-cluster.sh` — cluster upgrade without build/push; documents operator-sdk vs OperatorHub paths.
- `docs/ARCHITECTURE.md` — component and reconcile flow overview.
- `docs/grafana/dashboard.json` — Grafana dashboard for operator metrics.
- `build/hi-images.lock` and `hack/resolve-hi-digests.sh` — pin Red Hat Hardened Image linux/amd64 digests for CI/release.
- `scripts/hi-build-args.sh` — pass pinned HI digests to container builds.
- `scripts/check-coverage.sh` — enforce minimum unit test coverage (25%) in `make test`.
- Scorecard runs against PR-built operator image in Kind; deployment behaviour covered by Kind E2E.
- Critical Prometheus alert `ProjectOnboardingOperatorDown`.
- API stability policy in [docs/api-design.md](docs/api-design.md).

### Changed

- `release-openshift.sh`: `UPGRADE=true` delegates to `upgrade-cluster.sh`; published bundle uses `USE_IMAGE_DIGESTS=true` after operator image push.
- CI: bundle scorecard builds and loads the PR operator image into Kind; installs OLM for integration tests.
- CI: release scorecard uses the same PR-built image as Kind E2E (not pre-published Quay).
- CI: catalog index build validated on pull requests.
- CI: Trivy scans operator, bundle, and catalog images (security + release workflows).
- CI: OpenShift E2E fails on `main` / schedule when `OPENSHIFT_KUBECONFIG` is missing.
- Expanded [SECURITY.md](SECURITY.md) with supported versions, disclosure timeline, and scope.
- [CONTRIBUTING.md](CONTRIBUTING.md): recommended branch protection checks.
- [docs/supply-chain.md](docs/supply-chain.md): pinned HI images and OLM digest policy.

## [0.0.47] - 2026-06-19

### Added

- Custom Prometheus metrics: `projectonboarding_tenants_total`, `projectonboarding_reconcile_errors_total{reason}`; operational runbook [docs/runbook.md](docs/runbook.md).
- Supply chain: cosign image signing + SPDX SBOM on release ([docs/supply-chain.md](docs/supply-chain.md)); GitHub `CODEOWNERS`, issue templates, PR template.

### Changed

- CI: release workflow validates Kind E2E and operator-sdk scorecard before publishing images.
- CI: operator-sdk scorecard on pull requests; `govulncheck` in security workflow.
- OLM bundle: deduplicate unprefixed `PrometheusRule` manifest (keep prefixed copy only).
- Go toolchain: require **1.25.11** (stdlib security fixes for govulncheck / release builds).

### Fixed

- Kind E2E metrics probe: mount cert-manager metrics TLS in `config/overlays/local`, declare container port `8443`, regenerate OLM bundle (`targetPort: https`), and wait for TLS secrets before curling (fixes flaky connection refused).

## [0.0.46] - 2026-06-19

### Fixed

- Release workflow: pass `QUAY_TOKEN` to `release-openshift.sh` so non-interactive Quay login works on GitHub Actions runners (`docker/login-action` credentials are not visible to `docker login --get-login`).

### Changed

- CI: remove operator-sdk scorecard from the bundle pull-request workflow (requires a Kubernetes cluster).

## [0.0.45] - 2026-06-19

### Fixed

- Namespace reconcile: reuse the namespace object from create/patch instead of a second cache read, avoiding a transient `Namespace not found` error on new tenants when the informer has not caught up yet.

### Changed

- CI: pin bundle CSV `createdAt` for reproducible OLM bundle checks; fix Trivy action version; Kind E2E preloads images in workflow; OpenShift E2E skips cleanly when kubeconfig secret is missing; reduce Dependabot PR limit.
- CI: use `aquasecurity/trivy-action@v0.36.0` (v-prefixed tag); run `make lint` so golangci-lint matches Go 1.25; portable `PREV_VERSION` in Makefile; Kind deploy excludes Prometheus Operator CRs (ServiceMonitor/PrometheusRule stay in OLM bundle only).
- E2E: offboard tenant namespace before deleting `ProjectOnboarding` (matches finalizer behaviour; avoids hung `kubectl delete`).

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
