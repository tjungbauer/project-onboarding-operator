# Operational runbook

Quick reference for common production issues with **project-onboarding-operator** on OpenShift/Kubernetes. For install and upgrade paths see [operatorhub-install.md](operatorhub-install.md) and [openshift-install.md](openshift-install.md).

## Symptoms index

| Symptom | Section |
| ------- | ------- |
| `ProjectOnboarding` stuck in `Terminating` | [Stuck finalizers](#stuck-finalizers) |
| Tenant namespace will not delete | [Stuck finalizers](#stuck-finalizers) |
| OLM Subscription `ResolutionFailed` | [Catalog ResolutionFailed](#catalog-resolutionfailed) |
| CSV stuck on old version after catalog update | [Failed upgrades](#failed-upgrades) |
| Operator pod `CrashLoopBackOff` after upgrade | [Failed upgrades](#failed-upgrades) |
| Webhook `connection refused` / TLS errors | [Webhook certificate issues](#webhook-certificate-issues) |
| Validating webhook rejects all CR creates | [Webhook certificate issues](#webhook-certificate-issues) |
| Reconcile errors / alerts firing | [Reconcile errors](#reconcile-errors) |

---

## Stuck finalizers

### ProjectOnboarding stuck in Terminating

**Cause:** Finalizer `onboarding.stderr.at/finalizer` blocks CR deletion while **managed tenant namespaces** still exist.

**Diagnose:**

```bash
export OPERATOR_NS=project-onboarding-operator

oc get pob <name> -o jsonpath='{.metadata.deletionTimestamp}{"\n"}{.status.conditions[?(@.type=="DeletionBlocked")].message}{"\n"}'
oc get pob <name> -o yaml | grep -A2 finalizers
```

Typical status message:

```text
Cannot delete ProjectOnboarding while managed tenant namespaces still exist. Set offboard=true on each namespace entry or delete the tenant namespaces manually. Pending: team-payments-dev
```

**Fix (preferred — ordered teardown):**

1. Ensure the operator is running (`oc get pods -n "${OPERATOR_NS}"`).
2. Edit the CR (works even while terminating):

   ```bash
   oc edit pob <name>
   ```

   Set `offboard: true` on **each** pending `spec.namespaces[]` entry.
3. Wait for reconcile — tenant namespaces and cluster-scoped children (Groups, EgressIPs, AppProjects) are removed.
4. CR should disappear once all managed namespaces are gone.

**Fix (manual — break-glass):**

```bash
# Delete tenant namespace directly (workloads are removed with the namespace)
oc delete namespace <tenant-ns>

# If CR still terminating and operator is down, remove finalizer only after namespaces are gone:
oc patch pob <name> --type merge -p '{"metadata":{"finalizers":[]}}'
```

WARNING: Patching away the finalizer while tenant namespaces still exist leaves orphaned platform resources (Groups, EgressIPs, RoleBindings, AppProjects).

**Prevention:** Always offboard before delete. See [guide.md — Lifecycle](guide.md#lifecycle-enable-freeze-offboard-and-delete).

### Tenant namespace stuck in Terminating

**Cause:** Usually finalizers on resources inside the namespace, or the namespace is still owned by a `ProjectOnboarding` that has not offboarded the entry.

**Diagnose:**

```bash
oc get namespace <tenant-ns> -o yaml | grep -A5 finalizers
oc get pob -A -o yaml | grep -B5 <tenant-ns>
```

**Fix:** Offboard via the parent `ProjectOnboarding` (`offboard: true`) rather than deleting the namespace in isolation when the operator manages it.

---

## Failed upgrades

### Catalog updated but CSV stuck on old version

**Symptoms:** Subscription shows `ResolutionFailed` or `ConstraintsNotSatisfiable`; installed CSV version does not match catalog.

**Steps:**

1. Refresh catalog image and wait for `READY`:

   ```bash
   export VERSION="$(tr -d ' \n\r' < VERSION)"
   export OPERATOR_NS=project-onboarding-operator

   oc patch catalogsource project-onboarding-operator-catalog -n openshift-marketplace \
     --type merge -p "{\"spec\":{\"image\":\"quay.io/tjungbau/project-onboarding-operator-catalog:v${VERSION}\"}}"
   oc delete pod -n openshift-marketplace -l olm.catalogSource=project-onboarding-operator-catalog
   oc wait --for=jsonpath='{.status.connectionState.lastObservedState}'=READY \
     catalogsource/project-onboarding-operator-catalog \
     -n openshift-marketplace --timeout=5m
   ```

2. Confirm subscription channel is `stable`:

   ```bash
   oc patch subscription project-onboarding-operator -n "${OPERATOR_NS}" --type merge \
     -p '{"spec":{"channel":"stable"}}'
   ```

3. Delete the **stuck old CSV** (replace version):

   ```bash
   oc delete csv project-onboarding-operator.v0.0.45 -n "${OPERATOR_NS}"
   ```

4. Watch for new InstallPlan and `Succeeded` CSV:

   ```bash
   oc get subscription,installplan,csv,pods -n "${OPERATOR_NS}" -w
   ```

Tenant `ProjectOnboarding` CRs are **not** removed when deleting a CSV.

### Operator CrashLoopBackOff after upgrade

**Cause:** Bad operator image, webhook cert mismatch, or SCC/RBAC regression.

**Steps:**

1. Check operator logs:

   ```bash
   oc logs -n "${OPERATOR_NS}" deployment/project-onboarding-operator-controller-manager --tail=100
   ```

2. If the release is broken, roll back by pointing the catalog at the last known-good catalog tag, or delete the broken CSV and stale InstallPlans (see [operatorhub-install.md — Last resort](operatorhub-install.md#last-resort-broken-intermediate-version)).

3. Push a fixed catalog **before** recreating the Subscription.

---

## Catalog ResolutionFailed

**Symptoms:** Subscription condition `ResolutionFailed=True`; OperatorHub shows no installable version.

**Common causes:**

| Cause | Check |
| ----- | ----- |
| Catalog pod not ready | `oc get catalogsource,pods -n openshift-marketplace` |
| Wrong catalog image tag | `oc get catalogsource project-onboarding-operator-catalog -o yaml` |
| Bundle not in index | Verify catalog image exists on Quay and contains the bundle version |
| Channel mismatch | Subscription `spec.channel` must exist in catalog (`stable`) |
| Pull secret missing | `oc get sa default -n openshift-marketplace -o yaml` (global pull secret) |

**Fix workflow:**

```bash
# Catalog pod logs
oc logs -n openshift-marketplace -l olm.catalogSource=project-onboarding-operator-catalog

# Subscription status message
oc describe subscription project-onboarding-operator -n "${OPERATOR_NS}"

# Force catalog refresh
oc delete pod -n openshift-marketplace -l olm.catalogSource=project-onboarding-operator-catalog
```

If the catalog index is corrupt, rebuild and push a fresh cumulative catalog (`scripts/release-openshift.sh`) or use `CATALOG_FRESH=true` for a from-scratch index (see script header).

---

## Webhook certificate issues

OLM on OpenShift injects serving certificates via Service annotations. Kind/local deploy uses cert-manager instead.

### OpenShift (OLM)

**Symptoms:** `Internal error occurred: failed calling webhook`; apiserver logs mention x509 or connection refused.

**Diagnose:**

```bash
export OPERATOR_NS=project-onboarding-operator

oc get validatingwebhookconfiguration project-onboarding-operator-validating-webhook-configuration \
  -o jsonpath='{.webhooks[0].clientConfig.caBundle}' | wc -c

oc get svc -n "${OPERATOR_NS}" project-onboarding-operator-webhook-service
oc get pods -n "${OPERATOR_NS}" -l control-plane=controller-manager
```

**Fix:**

1. Ensure webhook Service has OpenShift serving-cert annotations (bundled in OLM install).
2. Restart controller pods so they reload TLS material:

   ```bash
   oc rollout restart deployment/project-onboarding-operator-controller-manager -n "${OPERATOR_NS}"
   ```

3. If `caBundle` is empty on `ValidatingWebhookConfiguration`, re-apply the CSV or check the `service.beta.openshift.io/serving-cert-secret-name` annotation on the webhook Service.

### Kind / cert-manager (development)

**Symptoms:** Webhook TLS errors after deploy; CRD conversion fails.

**Fix:**

```bash
kubectl get certificate -n project-onboarding-operator
kubectl describe certificate serving-cert -n project-onboarding-operator
kubectl rollout restart deployment/project-onboarding-operator-controller-manager -n project-onboarding-operator
```

Ensure cert-manager is installed and `config/overlays/local` is used (`make deploy`). See [local-testing.md](local-testing.md).

---

## Reconcile errors

**Metrics to inspect:**

| Metric | Meaning |
| ------ | ------- |
| `projectonboarding_reconcile_errors_total{reason="..."}` | Errors by reason (`namespace_reconcile`, `prune`, `transient`, …) |
| `projectonboarding_tenants_total{project_onboarding="..."}` | Active tenant entries per CR |
| `controller_runtime_reconcile_errors_total{controller="projectonboarding"}` | Controller-runtime aggregate |

See [metrics.md](metrics.md) for scrape setup and alert rules.

**Common reasons:**

| `reason` | Likely cause |
| -------- | ------------ |
| `namespace_reconcile` | RBAC denied, invalid quota, missing TShirtSize, Argo CD API unavailable |
| `prune` | Cannot delete orphaned resources after namespace removed from spec |
| `defaults_load` | `onboarding-defaults` ConfigMap missing or invalid |
| `transient` | API conflict/timeout — should self-heal on requeue |
| `deletion_finalize` | Cleanup failed during CR delete |

**Steps:**

```bash
oc describe pob <name>
oc get events --field-selector involvedObject.name=<name> --sort-by='.lastTimestamp'
oc logs -n "${OPERATOR_NS}" deployment/project-onboarding-operator-controller-manager --tail=200 | grep -i error
```

Enable debug logging temporarily:

```bash
oc set env deployment/project-onboarding-operator-controller-manager -n "${OPERATOR_NS}" DEBUG=true
```

---

## Escalation checklist

Before opening an issue or change request, collect:

1. Operator version: `oc get csv -n "${OPERATOR_NS}"`
2. Subscription / InstallPlan status
3. Controller pod logs (last restart)
4. Affected `ProjectOnboarding` YAML (redact secrets)
5. Relevant metrics or PrometheusRule alerts
6. OpenShift/Kubernetes version

Security-sensitive findings: [SECURITY.md](../SECURITY.md) (email **dev@stdin.at**, not public issues).

## Related documentation

| Topic | Document |
| ----- | -------- |
| User lifecycle (offboard/delete) | [guide.md — Lifecycle](guide.md#lifecycle-enable-freeze-offboard-and-delete) |
| Metrics and alerts | [metrics.md](metrics.md) |
| OperatorHub upgrade detail | [operatorhub-install.md](operatorhub-install.md) |
| Manual test cases | [openshift-testcases.md](openshift-testcases.md) |
