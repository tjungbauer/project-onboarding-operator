# OpenShift test cases

Manual test cases to verify **project-onboarding-operator** on OpenShift 4.x after OLM install ([openshift-install.md](openshift-install.md)).

**Quick path:** apply manifests under `[test/openshift/manifests/](../test/openshift/manifests/)` and run `[test/openshift/cleanup.sh](../test/openshift/cleanup.sh)`. The detailed steps below expand each case for formal QA sign-off.

Manifests use `onboarding.stderr.at/v1beta1` only (`v1alpha1` removed in 0.0.50).

## Prerequisites


| Requirement                                    | Check                                                                                                              |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| Operator installed (OLM or `make deploy`)      | `oc get csv -n project-onboarding-operator` → **Succeeded**                                                        |
| Controller pods running                        | `oc get pods -n project-onboarding-operator -l control-plane=controller-manager` → **3/3 Running** (HA deployment) |
| Cluster-admin `oc` access                      | `oc auth can-i create projectonboardings --all-namespaces` → **yes**                                               |
| OVN-Kubernetes (for TC-04 EgressIP)            | `oc get network cluster -o jsonpath='{.spec.defaultNetwork}'`                                                      |
| Argo CD AppProject CRD (for TC-13)             | `oc get crd appprojects.argoproj.io`                                                                               |
| Prometheus Operator CRDs (for TC-14, optional) | `oc get crd servicemonitors.monitoring.coreos.com`                                                                 |


Set the operator namespace if yours differs:

```bash
export OPERATOR_NS=project-onboarding-operator
```

For TC-13, ensure the Argo CD namespace exists (create if your cluster has no GitOps yet):

```bash
oc get namespace gitops-application || oc new-project gitops-application
```

If your Argo CD runs elsewhere, edit `spec.gitOps.applicationNamespace` in `tc13-gitops-onboarding.yaml`.

## Kind vs OpenShift coverage

Some reconciliation paths require OpenShift APIs that **Kind / plain Kubernetes CI does not provide**:


| Feature                                     | API                            | Kind E2E | OpenShift E2E        |
| ------------------------------------------- | ------------------------------ | -------- | -------------------- |
| Namespaces, quotas, limits, NetworkPolicies | core / networking              | Yes      | Yes                  |
| `localAdminGroup`                           | `user.openshift.io/v1` `Group` | No       | Yes                  |
| `egressIPs`                                 | `k8s.ovn.org/v1` `EgressIP`    | No       | Yes (OVN-Kubernetes) |



| Layer             | What runs                                                                                       |
| ----------------- | ----------------------------------------------------------------------------------------------- |
| **Unit tests**    | `internal/onboarding/openshift_test.go` — Group + EgressIP reconcile/cleanup with a fake client |
| **Kind E2E**      | Pod health, metrics, core onboarding (namespace + quota)                                        |
| **OpenShift E2E** | TC-00–TC-15 via `make test-e2e-openshift` (TC-13/14 auto-skip when CRDs missing) |


---

## Test matrix


| ID    | Area                             | Required?              | Pass criteria                                                  |
| ----- | -------------------------------- | ---------------------- | -------------------------------------------------------------- |
| TC-00 | Operator health                  | Yes                    | CSV, pod, CRDs, webhooks, operator NetworkPolicies             |
| TC-01 | Core onboarding                  | Yes                    | Namespace + quota + limit range + network policies             |
| TC-02 | T-shirt catalogue                | Yes                    | `TShirtSize` Ready, reference count                            |
| TC-03 | T-shirt merge                    | Yes                    | Quota CPU overridden, memory from T-shirt                      |
| TC-04 | OpenShift APIs                   | Yes                    | Group, RoleBinding, EgressIP created                           |
| TC-05 | Custom NetworkPolicy             | Yes                    | Custom policy exists alongside defaults                        |
| TC-06 | Admission — invalid T-shirt      | Yes                    | Server rejects empty `TShirtSize`                              |
| TC-07 | Admission — unknown size         | Yes                    | Webhook rejects missing `projectSize`                          |
| TC-08 | Admission — T-shirt delete guard | Yes                    | Delete blocked while referenced                                |
| TC-09 | CR delete / cleanup              | Yes                    | Tenant namespace removed after offboard + CR delete            |
| TC-10 | Drift correction                 | Yes                    | Operator restores edited quota                                 |
| TC-11 | T-shirt update propagation       | Yes                    | Quota changes when `TShirtSize` is patched                     |
| TC-12 | v1beta1 API                      | Yes                    | `ProjectOnboarding` stored and served as `v1beta1`             |
| TC-13 | GitOps AppProject                | If Argo CD CRD present | AppProject + managed-by label                                  |
| TC-14 | Observability                    | Optional               | ServiceMonitor + PrometheusRule present                        |
| TC-15 | OLM upgrade path                 | Yes                    | CSV `spec.replaces` points at previous bundle version          |


---

## TC-00 — Operator health

**Steps**

```bash
oc get csv -n "${OPERATOR_NS}"
oc get deploy,pods -n "${OPERATOR_NS}" | grep controller-manager
oc api-resources | grep -E 'projectonboarding|tshirtsize'
oc get validatingwebhookconfiguration -o name | grep -E 'vprojectonboarding|vtshirtsize'
```

**Verify API versions**

```bash
# Storage version is v1beta1
oc get crd projectonboardings.onboarding.stderr.at \
  -o jsonpath='{.spec.versions[?(@.storage==true)].name}{"\n"}'
# Expected: v1beta1

oc get crd tshirtsizes.onboarding.stderr.at \
  -o jsonpath='{.spec.versions[?(@.storage==true)].name}{"\n"}'
# Expected: v1beta1

# v1beta1 validating webhooks (OLM names: vprojectonboarding.kb.io-*, etc.)
oc get validatingwebhookconfiguration -o name | grep -E 'vprojectonboarding|vtshirtsize' | sort
# Expected: v1beta1 webhooks for ProjectOnboarding and TShirtSize
```

**Verify operator hardening (0.0.10+ bundle)**

```bash
oc get networkpolicy -n "${OPERATOR_NS}" \
  -o custom-columns=NAME:.metadata.name,PORTS:.spec.ingress[0].ports[*].port
# Expected: allow-metrics-traffic (8443), allow-webhook-traffic (9443)

oc get servicemonitor,prometheusrule -n "${OPERATOR_NS}" 2>/dev/null || true
# Expected when Prometheus Operator CRDs exist:
#   project-onboarding-operator-controller-manager-metrics-monitor
#   project-onboarding-operator-controller-manager-rules
```

**Expected**

- CSV `phase: Succeeded`
- Controller Deployment available, pod `Running`
- CRDs: `projectonboardings.onboarding.stderr.at`, `tshirtsizes.onboarding.stderr.at` (`v1beta1` only)
- Validating webhooks for `ProjectOnboarding` and `TShirtSize` (`v1beta1`)

**Optional — controller logs**

```bash
oc logs -n "${OPERATOR_NS}" -l control-plane=controller-manager -c manager --tail=50
```

---

## TC-01 — Core onboarding

**Manifest:** `test/openshift/manifests/tc01-core-onboarding.yaml`  
**Tenant namespace:** `ocp-test-core-dev`

**Steps**

```bash
oc apply -f test/openshift/manifests/tc01-core-onboarding.yaml

# Wait for Ready (up to ~3 min)
oc wait --for=jsonpath='{.status.phase}'=Ready projectonboarding/tc01-core-onboarding --timeout=3m
```

**Verify**

```bash
# Stored API version
oc get pob tc01-core-onboarding -o jsonpath='{.apiVersion}{"\n"}{.status.phase}{"\n"}'
# Expected: onboarding.stderr.at/v1beta1 / Ready

# Status
oc get pob tc01-core-onboarding -o jsonpath='{.status.namespaces[0].ready}{"\n"}'
# Expected: true

# Namespace + labels
oc get namespace ocp-test-core-dev -o yaml | grep -E 'pod-security.kubernetes.io|openshift.io/user-monitoring|onboarding.stderr.at'

# Quota and limits (names: <namespace>-quota, <namespace>-limitrange)
oc get resourcequota ocp-test-core-dev-quota -n ocp-test-core-dev -o jsonpath='{.spec.hard}' | grep -E 'cpu|memory|pods'
oc get limitrange ocp-test-core-dev-limitrange -n ocp-test-core-dev

# Default network policy set (full platform baseline)
oc get networkpolicy -n ocp-test-core-dev --no-headers | awk '{print $1}' | sort
```

**Expected network policies**

```
allow-from-kube-apiserver-operator
allow-from-openshift-ingress
allow-from-openshift-monitoring
allow-same-namespace
allow-to-openshift-dns
```

**Expected quota (high level)**

- `limits.cpu`: 8, `limits.memory`: 16Gi
- `requests.cpu`: 2, `requests.memory`: 4Gi
- `pods`: 20

---

## TC-02 — T-shirt catalogue

**Manifest:** `test/openshift/manifests/tc02-tshirt-catalog.yaml`

**Steps**

```bash
oc apply -f test/openshift/manifests/tc02-tshirt-catalog.yaml
oc wait --for=jsonpath='{.status.phase}'=Ready tshirtsize/ocp-test-medium --timeout=2m
```

**Verify**

```bash
oc get tts ocp-test-medium -o custom-columns=API:.apiVersion,PHASE:.status.phase,REFS:.status.referencedBy,DESC:.spec.description
```

**Expected:** `API=v1beta1`, `PHASE=Ready`, `REFS=0` (before TC-03).

---

## TC-03 — T-shirt reference with overwrite

**Depends on:** TC-02  
**Manifest:** `test/openshift/manifests/tc03-tshirt-onboarding.yaml`  
**Tenant namespace:** `ocp-test-medium-dev`

**Steps**

```bash
oc apply -f test/openshift/manifests/tc03-tshirt-onboarding.yaml
oc wait --for=jsonpath='{.status.phase}'=Ready projectonboarding/tc03-tshirt-onboarding --timeout=3m
```

**Verify**

```bash
# namespace-size label from projectSize
oc get namespace ocp-test-medium-dev -o jsonpath='{.metadata.labels.namespace-size}{"\n"}'
# Expected: ocp-test-medium

# Merged quota: cpu overridden to 3, memory stays 4Gi from T-shirt
oc get resourcequota ocp-test-medium-dev-quota -n ocp-test-medium-dev \
  -o jsonpath='cpu={.spec.hard.cpu}{" memory="}{.spec.hard.memory}{"\n"}'
# Expected: cpu=3 memory=4Gi

# T-shirt reference count incremented
oc get tts ocp-test-medium -o jsonpath='{.status.referencedBy}{"\n"}'
# Expected: 1

# Minimal network policies (explicit false in manifest)
oc get networkpolicy -n ocp-test-medium-dev --no-headers | wc -l
# Expected: 1 (allow-same-namespace only)
```

---

## TC-04 — OpenShift Group, RoleBinding, EgressIP

**Requires:** OpenShift with `user.openshift.io/Group` and `k8s.ovn.org/EgressIP` APIs.

**Manifest:** `test/openshift/manifests/tc04-openshift-features.yaml`  
**Tenant namespace:** `ocp-test-egress-dev`

> **Note:** The sample uses documentation IP `203.0.113.50` (TEST-NET-3). The `EgressIP` object is created and reconciled; assignment to a node only happens with a real pool IP. Replace the IP in the manifest if your cluster provides egress addresses.

**Steps**

```bash
oc apply -f test/openshift/manifests/tc04-openshift-features.yaml
oc wait --for=jsonpath='{.status.phase}'=Ready projectonboarding/tc04-openshift-features --timeout=3m
```

**Verify**

```bash
# OpenShift Group (default name: <namespace>-admins)
oc get group ocp-test-egress-dev-admins -o jsonpath='{.users}{"\n"}'
# Expected: ["ocp-test-admin"]

# RoleBinding to cluster-admin role "admin"
oc get rolebinding ocp-test-egress-dev-rb -n ocp-test-egress-dev \
  -o jsonpath='{.roleRef.name}{" -> "}{.subjects[0].name}{"\n"}'
# Expected: admin -> ocp-test-egress-dev-admins

# EgressIP named after tenant namespace; selector uses label env=<namespace>
oc get egressip ocp-test-egress-dev -o yaml | grep -E 'egressIPs:|env:'
oc get namespace ocp-test-egress-dev -o jsonpath='{.metadata.labels.env}{"\n"}'
# Expected: ocp-test-egress-dev
```

---

## TC-05 — Custom NetworkPolicy

**Manifest:** `test/openshift/manifests/tc05-custom-netpol.yaml`  
**Tenant namespace:** `ocp-test-netpol-dev`

**Steps**

```bash
oc apply -f test/openshift/manifests/tc05-custom-netpol.yaml
oc wait --for=jsonpath='{.status.phase}'=Ready projectonboarding/tc05-custom-netpol --timeout=3m
```

**Verify**

```bash
oc get networkpolicy allow-from-openshift-monitoring-custom -n ocp-test-netpol-dev
oc get networkpolicy allow-from-openshift-monitoring-custom -n ocp-test-netpol-dev \
  -o jsonpath='{.spec.podSelector.matchLabels.app}{"\n"}'
# Expected: metrics-exporter

oc get networkpolicy -n ocp-test-netpol-dev --no-headers
# Expected: allow-same-namespace, allow-to-openshift-dns, allow-from-openshift-monitoring-custom
```

---

## TC-06 — Admission: invalid TShirtSize (negative)

**Steps**

```bash
oc apply --dry-run=server -f test/openshift/manifests/tc06-invalid-tshirtsize.yaml
```

**Expected:** Command fails — CEL requires `resourceQuotas` and/or `limitRanges`.

---

## TC-07 — Admission: unknown projectSize (negative)

**Steps**

```bash
oc apply --dry-run=server -f test/openshift/manifests/tc07-bad-projectsize.yaml
```

**Expected:** Webhook rejects — `projectSize` must reference an existing `TShirtSize`.

---

## TC-08 — Admission: T-shirt delete guard (negative)

**Depends on:** TC-02 + TC-03

**Steps**

```bash
oc delete tshirtsize ocp-test-medium --dry-run=server
```

**Expected:** Webhook rejects — T-shirt is referenced by `tc03-tshirt-onboarding`.

---

## TC-09 — Delete cleans up tenant namespace

**Depends on:** TC-01 (or any applied `ProjectOnboarding`)

Since v0.0.35, deleting the CR alone is not enough — set `offboard: true` on the namespace entry first.

**Steps**

```bash
oc edit projectonboarding tc01-core-onboarding   # offboard: true on ocp-test-core-dev
oc wait --for=delete namespace/ocp-test-core-dev --timeout=3m
oc delete projectonboarding tc01-core-onboarding --ignore-not-found
```

**Verify**

```bash
oc get namespace ocp-test-core-dev
# Expected: NotFound
```

Repeat for other test CRs during full cleanup (`./test/openshift/cleanup.sh`).

---

## TC-10 — Drift correction

**Depends on:** TC-03 running and Ready

**Steps**

```bash
# Manually change quota CPU away from desired state (3)
oc patch resourcequota ocp-test-medium-dev-quota -n ocp-test-medium-dev \
  --type=merge -p '{"spec":{"hard":{"cpu":"99"}}}'

# Wait for operator to reconcile (watch or poll)
sleep 30
oc get resourcequota ocp-test-medium-dev-quota -n ocp-test-medium-dev \
  -o jsonpath='{.spec.hard.cpu}{"\n"}'
```

**Expected:** CPU restored to `3`.

---

## TC-11 — T-shirt update propagation

**Depends on:** TC-02 + TC-03

**Steps**

```bash
# Patch T-shirt memory from 4Gi to 6Gi
oc patch tshirtsize ocp-test-medium --type=merge \
  -p '{"spec":{"resourceQuotas":{"memory":"6Gi"}}}'

sleep 30
oc get resourcequota ocp-test-medium-dev-quota -n ocp-test-medium-dev \
  -o jsonpath='memory={.spec.hard.memory}{"\n"}'
```

**Expected:** Memory updated to `6Gi` (CPU override `3` unchanged).

**Cleanup note:** Revert the patch or delete test resources when finished.

---

## TC-12 — v1beta1 API

**Manifest:** `test/openshift/manifests/tc01-core-onboarding.yaml`  
**Tenant namespace:** `ocp-test-core-dev`

**Steps**

```bash
oc apply -f test/openshift/manifests/tc01-core-onboarding.yaml
oc wait --for=jsonpath='{.status.phase}'=Ready projectonboarding/tc01-core-onboarding --timeout=3m
```

**Verify**

```bash
oc get pob tc01-core-onboarding -o jsonpath='{.apiVersion}{"\n"}'
# Expected: onboarding.stderr.at/v1beta1

oc get namespace ocp-test-core-dev
oc get resourcequota ocp-test-core-dev-quota -n ocp-test-core-dev
```

**Cleanup:** covered by TC-09 or `./test/openshift/cleanup.sh`

---

## TC-15 — OLM upgrade path

**Procedure only** (no manifest). Verifies the installed CSV declares an OLM upgrade edge via `spec.replaces`.

**Steps**

```bash
oc get csv -n "${OPERATOR_NS}" \
  -o jsonpath='{range .items[?(@.spec.displayName=="Project Onboarding")]}{.metadata.name}{" replaces="}{.spec.replaces}{"\n"}{end}'
```

**Expected**

- CSV name matches `project-onboarding-operator.vX.Y.Z`
- `spec.replaces` is set to the previous bundle version when upgrading from an earlier release (empty on first install)

Automated: `make test-e2e-openshift` (set `OPENSHIFT_E2E_PREV_VERSION` to assert a specific `replaces` target).

---

## TC-13 — GitOps AppProject

**Requires:** `appprojects.argoproj.io` CRD; namespace matching `spec.gitOps.applicationNamespace`.

**Manifest:** `test/openshift/manifests/tc13-gitops-onboarding.yaml`  
**Tenant namespace:** `ocp-test-gitops-dev`  
**AppProject namespace:** `gitops-application`

**Steps**

```bash
oc get crd appprojects.argoproj.io
oc get namespace gitops-application || oc new-project gitops-application

oc apply -f test/openshift/manifests/tc13-gitops-onboarding.yaml
oc wait --for=jsonpath='{.status.phase}'=Ready projectonboarding/tc13-gitops-onboarding --timeout=3m
```

**Verify**

```bash
# Argo CD managed-by label on tenant namespace
oc get namespace ocp-test-gitops-dev \
  -o jsonpath='{.metadata.labels.argocd\.argoproj\.io/managed-by}{"\n"}'
# Expected: gitops-application

# AppProject in Argo CD namespace
oc get appproject ocp-test-gitops-dev -n gitops-application
oc get appproject ocp-test-gitops-dev -n gitops-application \
  -o jsonpath='roles={.spec.roles[*].name}{"\n"}destinations={.spec.destinations[*].namespace}{"\n"}'
# Expected: roles include write and read; destinations namespace ocp-test-gitops-dev

# Source repos from spec.gitOps
oc get appproject ocp-test-gitops-dev -n gitops-application \
  -o jsonpath='{.spec.sourceRepos}{"\n"}'
# Expected includes https://github.com/example-org/example-repo
```

**Cleanup**

```bash
oc delete -f test/openshift/manifests/tc13-gitops-onboarding.yaml
oc wait --for=delete appproject/ocp-test-gitops-dev -n gitops-application --timeout=2m 2>/dev/null || true
oc wait --for=delete namespace/ocp-test-gitops-dev --timeout=3m
```

Helm chart (`helper-proj-onboarding`) → CR samples: `config/samples/migration/`.

---

## TC-14 — Observability (optional)

**Requires:** Prometheus Operator CRDs installed (OpenShift user-workload monitoring or cluster monitoring).

**Steps**

```bash
oc get servicemonitor project-onboarding-operator-controller-manager-metrics-monitor \
  -n "${OPERATOR_NS}"
oc get prometheusrule project-onboarding-operator-controller-manager-rules \
  -n "${OPERATOR_NS}"
```

**Verify metrics endpoint still reachable**

```bash
# Label a scraper namespace if using operator NetworkPolicy (metrics: enabled)
oc label namespace openshift-user-workload-monitoring metrics=enabled --overwrite 2>/dev/null || true

oc get endpoints -n "${OPERATOR_NS}" \
  project-onboarding-operator-controller-manager-metrics-service \
  -o jsonpath='{.subsets[0].ports[0].port}{"\n"}'
# Expected: 8443
```

**Expected:** ServiceMonitor selects controller pods on port `8443`; PrometheusRule defines operator alert rules.

Skip this case if `servicemonitors.monitoring.coreos.com` is not installed.

---

## Automated OpenShift E2E

The manual test cases TC-00–TC-14 are automated as a Ginkgo suite. The operator must **already be installed** on the target cluster (OLM or `make deploy`).

From the repository root:

```bash
export OPENSHIFT_E2E=true
export OPERATOR_NS=project-onboarding-operator   # optional; default shown

make test-e2e-openshift
```

Or directly, from the repository root:

```bash
OPENSHIFT_E2E=true go test ./test/e2e/ -v -ginkgo.v -ginkgo.focus="OpenShift test cases" -timeout 45m
```


| Behaviour           | Detail                                                 |
| ------------------- | ------------------------------------------------------ |
| Kind setup          | Skipped when `OPENSHIFT_E2E=true`                      |
| TC-13 GitOps        | Auto-skips if `appprojects.argoproj.io` CRD is missing |
| TC-14 observability | Auto-skips if Prometheus Operator CRDs are missing     |
| Cleanup             | `test/openshift/cleanup.sh` runs in `AfterAll`         |


CI: GitHub Actions workflow **Test E2E (OpenShift)** (`workflow_dispatch` + weekly schedule) with secret `OPENSHIFT_KUBECONFIG` (base64 kubeconfig). The workflow skips when the secret is not configured.

---

## Full cleanup

```bash
./test/openshift/cleanup.sh
oc get pob,tts
oc get namespace | grep ocp-test || echo "all test namespaces gone"
oc get appproject -n gitops-application 2>/dev/null | grep ocp-test || true
```

If a `ProjectOnboarding` stays in `Terminating`, managed tenant namespaces probably still exist (`offboard: false`). Check `DeletionBlocked`:

```bash
oc get pob -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.deletionTimestamp}{"\t"}{range .status.conditions[?(@.type=="DeletionBlocked")]}{.message}{"\n"}{end}{end}'
```

Example:

```text
Cannot delete ProjectOnboarding while managed tenant namespaces still exist. Set offboard=true on each namespace entry or delete the tenant namespaces manually. Pending: ocp-test-01
```

Set `offboard: true` on pending entries (`oc edit pob <name>`) or delete the tenant namespace, then wait for the CR to go away.

If a tenant namespace stays in `Terminating`, check `ProjectOnboarding` finalizers:

```bash
oc get pob -o yaml | grep -A2 finalizers
oc logs -n "${OPERATOR_NS}" -l control-plane=controller-manager -c manager --tail=100
```

---

## Suggested run order

1. TC-00 (health)
2. TC-01 → TC-09 (core smoke + delete)
3. TC-12 (v1beta1 API)
4. TC-15 (OLM upgrade path)
5. TC-02 → TC-03 → TC-08 → TC-10 → TC-11 (T-shirt path)
5. TC-04 (OpenShift APIs)
6. TC-05 (custom netpol)
7. TC-13 (GitOps — skip if no AppProject CRD)
8. TC-14 (observability — optional)
9. TC-06, TC-07 (negative admission)
10. `./test/openshift/cleanup.sh`

