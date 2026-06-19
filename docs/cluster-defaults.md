# Cluster-wide GitOps defaults

Shared Argo CD settings (`allowedSourceRepos`, `allowedOIDCGroups`, `destinations`, legacy `applicationNamespace`) are configured **once per cluster** via this ConfigMap or `spec.gitOps` in YAML. They are **hidden in the OpenShift form** (removed in v0.0.40); per-tenant AppProjects and RBAC are documented in [guide.md — GitOps / Argo CD](guide.md#gitops--argo-cd).

The operator loads the ConfigMap at reconcile time and merges into each `ProjectOnboarding`. Tenant CR fields **override** cluster defaults on conflict.

Helm **helper-proj-onboarding** mapping:

| Helm key | ConfigMap / CR field |
| -------- | -------------------- |
| `application_gitops_namespace` | `namespaces[].applicationGitOpsNamespace` (preferred) or ConfigMap `gitOps.applicationNamespace` (fallback) |
| `allowed_source_repos` | `gitOps.allowedSourceRepos` |
| `allowed_oidc_groups` | `gitOps.allowedOIDCGroups` |
| `global.envs` / destinations | `gitOps.destinations` |

## ConfigMap

| Field | Value |
|-------|--------|
| Namespace | Operator namespace (default `project-onboarding-operator`) |
| Name | `onboarding-defaults` |
| Data key | `defaults.yaml` |

Sample: [`config/samples/onboarding_defaults_configmap.yaml`](../config/samples/onboarding_defaults_configmap.yaml).

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: onboarding-defaults
  namespace: project-onboarding-operator
data:
  defaults.yaml: |
    gitOps:
      applicationNamespace: gitops-application
      allowedSourceRepos:
        - https://my-git-repo.com/super-repo
      allowedOIDCGroups:
        - admin-group-1
      destinations:
        - name: in-cluster
          server: https://kubernetes.default.svc
```

Apply once per cluster (or manage via GitOps in the operator namespace). From the repository root:

```bash
oc apply -f config/samples/onboarding_defaults_configmap.yaml
```

## Merge rules

Loaded at reconcile time (`internal/onboarding/defaults.go`). For each `spec.gitOps` field on the tenant CR:

| CR field | Defaults used when |
|----------|-------------------|
| `applicationNamespace` | CR value is empty |
| `destinations` | CR list is empty |
| `allowedSourceRepos` | CR list is empty |
| `allowedOIDCGroups` | CR list is empty |
| `allowedSourceNamespaces` | CR list is empty |

Non-empty CR values always win. Defaults are **not** merged field-by-field inside nested structures.

## Tenant CR example

With defaults applied, a minimal tenant CR only needs namespace-specific settings:

```yaml
apiVersion: onboarding.stderr.at/v1beta1
kind: ProjectOnboarding
metadata:
  name: tenant3-onboarding
spec:
  namespaces:
    - name: tenant3-app-1
      projectSize: small
      limitRanges:
        enabled: true
```

GitOps AppProject creation requires `applicationGitOpsNamespace` on the namespace entry (or legacy `spec.gitOps.applicationNamespace` / ConfigMap fallback) and `spec.namespaces[].argoCDProjects[]` when AppProjects are enabled.

## Troubleshooting

| Symptom | Check |
|---------|--------|
| Defaults ignored | ConfigMap in **operator** namespace? Key `defaults.yaml` present? |
| Reconcile error on bad YAML | Fix YAML syntax; invalid content is logged and fails reconcile |
| AppProject missing | Defaults only fill `spec.gitOps`; `argoCDProjects` must be set on the namespace entry |

See also: [guide.md — Argo CD Project](guide.md#argo-cd-project), [config/samples/migration/](../config/samples/migration/), [operatorhub-install.md](operatorhub-install.md).
