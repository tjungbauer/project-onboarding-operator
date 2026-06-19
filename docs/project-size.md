# T-shirt sizing (`TShirtSize` + `projectSize`)

## Overview

A **TShirtSize** lets you predefine ResourceQuota and LimitRange presets once. Tenant `ProjectOnboarding` resources reference a size by name instead of repeating quota keys on every namespace.

| Resource              | Scope   | Purpose                                                                     |
| --------------------- | ------- | --------------------------------------------------------------------------- |
| **TShirtSize**        | Cluster | Catalogue of quota/limit presets (`small`, `large`, `extra-large`, …)       |
| **ProjectOnboarding** | Cluster | References a size via `spec.namespaces[].projectSize`                       |

Shipped samples use `v1beta1` and catalogue sizes `small`, `large`, and `extra-large`.

## Merge rules

`projectSize` alone only sets the `namespace-size` label on the tenant namespace. **Catalogue values are applied only when you explicitly enable** the matching block on the namespace entry.

| `projectSize` | `resourceQuotas.enabled` | `limitRanges.enabled` | `overwriteTshirt` | ResourceQuota result | LimitRange result |
| ------------- | ------------------------ | --------------------- | ----------------- | -------------------- | ----------------- |
| set | unset / `false` | any | any | **None** (inline fields ignored) | Per limit column |
| set | `true` | unset / `false` | `false` | Catalogue only | **None** |
| set | unset / `false` | `true` | `false` | **None** | Catalogue only |
| set | `true` | `true` | `false` | Catalogue only | Catalogue only |
| set | `true` | any | `true` | Catalogue + inline quota overrides | Per limit column |
| set | any | `true` | `true` | Per quota column | Catalogue + inline limit overrides |

**Key points:**

- Inline `resourceQuotas` / `limitRanges` fields without `enabled: true` are **never** merged — even when `overwriteTshirt: true`.
- `overwriteTshirt: true` merges field-by-field onto the catalogue for blocks that are **enabled**. Tenant fields win; unset fields inherit from `TShirtSize`.
- Quota and limit range are independent: you can enable one without the other.

## TShirtSize (cluster catalogue)

```yaml
apiVersion: onboarding.stderr.at/v1beta1
kind: TShirtSize
metadata:
  name: small
spec:
  description: Small tenant
  resourceQuotas:
    enabled: true
    limits:
      cpu: "1"
      memory: 1Gi
    requests:
      cpu: 500m
      memory: 1Gi
  limitRanges:
    enabled: true
    container:
      defaultRequest:
        cpu: 100m
        memory: 256Mi
```

From the repository root:

```bash
oc apply -f config/samples/onboarding_v1beta1_tshirtsizes_catalog.yaml
```

## ProjectOnboarding examples

### Catalogue only (opt-in)

```yaml
spec:
  namespaces:
    - name: team-beta-dev
      projectSize: small
      resourceQuotas:
        enabled: true
      limitRanges:
        enabled: true
```

### Override catalogue (tenant3 pattern)

T-shirt `small` has `limits.cpu: "1"`. Bump CPU for this tenant only:

```yaml
spec:
  namespaces:
    - name: tenant3-app-1
      projectSize: small
      overwriteTshirt: true
      resourceQuotas:
        enabled: true
        limits:
          cpu: "10"
      limitRanges:
        enabled: true
```

Result: **cpu limit 10**, other quota/limit fields from **`small`**.

### Limits only (quota not enabled)

```yaml
spec:
  namespaces:
    - name: tenant3-app-1
      projectSize: small
      resourceQuotas:
        limits:
          cpu: "10"    # ignored — enabled not true
      limitRanges:
        enabled: true
```

Result: **no ResourceQuota**, LimitRange from catalogue.

### No T-shirt (explicit quotas)

Leave `projectSize` empty and set `resourceQuotas` / `limitRanges` with `enabled: true` directly.

## Reconciliation

- Label `namespace-size: <projectSize>` is set on the tenant namespace.
- Changing a `TShirtSize` re-triggers affected `ProjectOnboarding` resources (watch on `TShirtSize` → enqueue parent CRs).
- A dedicated **`TShirtSize` reconciler** maintains catalogue status: `status.phase`, `status.referencedBy`, and `Ready` condition. It also watches `ProjectOnboarding` so reference counts stay accurate when CRs change.

## Validation

- **CEL** on `TShirtSize.spec`: at least one of `resourceQuotas` or `limitRanges` must be present.
- **Webhook**: `projectSize` must reference an existing `TShirtSize` with usable quota/limit values; delete of a referenced `TShirtSize` is rejected.

See also: [guide.md — T-shirt sizing](guide.md#t-shirt-sizing-projectsize--overwritetshirt), `config/samples/onboarding_v1beta1_projectonboarding_tshirt.yaml`, `config/samples/migration/tenant3_onboarding.yaml`.
