# Install via OpenShift OperatorHub (UI)

Use this path to install from **Operators → OperatorHub** instead of `operator-sdk run bundle`. This is the most convenient way to deploy. The steps below create a custom CatalogSource. For the **official Red Hat community OperatorHub** listing, see [operatorhub-community-submission.md](operatorhub-community-submission.md).

Set the release version once (must match the `VERSION` file). From the repository root:

```bash
export VERSION="$(tr -d ' \n\r' < VERSION)"
```

You need three images on Quay (all **public**, or configure pull secrets on the cluster):


| Image       | Example                                                            |
| ----------- | ------------------------------------------------------------------ |
| Operator    | `quay.io/tjungbau/project-onboarding-operator:v${VERSION}`         |
| Bundle      | `quay.io/tjungbau/project-onboarding-operator-bundle:v${VERSION}`  |
| **Catalog** | `quay.io/tjungbau/project-onboarding-operator-catalog:v${VERSION}` |


Create the `project-onboarding-operator-catalog` repository on Quay if it does not exist yet.

You can also build and push your own images to your repository. 

## Step 1 — Build and push images

From the repository root (after `podman login quay.io -u <user>`):

```bash
unset IMG BUNDLE_IMG REGISTRY # just in case

# Builds operator + bundle + catalog; does not install on the cluster
./scripts/release-openshift.sh "${VERSION}"
```

## Step 2 — Register the catalog (cluster admin)

From the repository root:

```bash
sed "s|:v0.0.0|:v${VERSION}|g" config/openshift/catalogsource-marketplace.yaml | oc apply -f -

oc get catalogsource project-onboarding-operator-catalog -n openshift-marketplace
oc get pods -n openshift-marketplace | grep project-onboarding-operator-catalog
```

Wait until the catalog pod is **Running** and the CatalogSource reports `READY`:

```bash
oc wait --for=jsonpath='{.status.connectionState.lastObservedState}'=READY \
  catalogsource/project-onboarding-operator-catalog \
  -n openshift-marketplace --timeout=5m
```

### Private Quay repositories

```bash
oc create secret docker-registry quay-pull-secret \
  --docker-server=quay.io \
  --docker-username=<user> \
  --docker-password='<token>' \
  -n openshift-marketplace

# Uncomment spec.secrets in catalogsource-marketplace.yaml, then re-apply
```

## Step 3 — Install from the OpenShift console

1. Log in as a user with permission to install operators (typically **cluster-admin** for AllNamespaces).
2. Go to **Ecosystem → Software Catalog** (previously: **Operators → OperatorHub**).
3. Search for **Project Onboarding** (publisher: Thomas Jungbauer).
4. Click **Project Onboarding** → **Install**.
5. Choose:
  - **Update channel:** `stable`
  - **Installation mode:** **All namespaces**
  - **Installed namespace:** should default to `**project-onboarding-operator`**; you can still pick another namespace, but the console may warn it is not operator-recommended
6. Click **Install** and approve the InstallPlan if prompted.
7. Wait until **Installed** / CSV phase **Succeeded**.

CLI check:

```bash
oc get csv -n project-onboarding-operator
oc get deploy -n project-onboarding-operator
```

## Step 4 — Apply samples

`ProjectOnboarding` and `TShirtSize` are **cluster-scoped**.

From the repository root:

```bash
oc apply -f config/samples/onboarding_v1beta1_tshirtsizes_catalog.yaml
oc apply -f config/samples/onboarding_v1beta1_projectonboarding_gitops.yaml

oc get projectonboarding,tshirtsize
oc get namespace tenant1-app-1
```

## Upgrade via UI

See **[upgrade.md](upgrade.md)** (Path B: OperatorHub / marketplace catalog).

Summary:

1. Push a new version: `./scripts/release-openshift.sh "${VERSION}"` (cumulative catalog index when the previous catalog tag exists on Quay).
2. Update the catalog image on the CatalogSource and restart the catalog pod:

```bash
export VERSION="$(tr -d ' \n\r' < VERSION)"

oc patch catalogsource project-onboarding-operator-catalog -n openshift-marketplace \
  --type merge -p "{\"spec\":{\"image\":\"quay.io/tjungbau/project-onboarding-operator-catalog:v${VERSION}\"}}"
oc delete pod -n openshift-marketplace -l olm.catalogSource=project-onboarding-operator-catalog
```

3. Patch the subscription or approve in **Operators → Installed Operators → Project Onboarding**. Or run `./scripts/upgrade-cluster.sh "${VERSION}"`.

## Stuck or failed upgrade

Do **not** wait indefinitely if the controller pod is in **CrashLoopBackOff** or the Subscription shows `**ResolutionFailed`**.

### Typical fix: catalog updated but CSV stuck on old version

Example: catalog and subscription point at **v0.0.14** on channel `**stable`**, but the installed CSV remains v0.0.13 and the subscription reports `**ResolutionFailed`** / `**ConstraintsNotSatisfiable**` (both CSVs claim the same CRDs).

1. From the repository root, push the new catalog image and refresh the catalog pod:
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
2. Ensure the subscription uses `**stable**`:
  ```bash
   oc patch subscription project-onboarding-operator -n "${OPERATOR_NS}" --type merge \
     -p '{"spec":{"channel":"stable"}}'
  ```
3. Delete the **orphaned old CSV** so OLM can install the new one (replace `0.0.13` with your stuck version):
  ```bash
   oc delete csv project-onboarding-operator.v0.0.13 -n "${OPERATOR_NS}"
  ```
4. Watch for a new InstallPlan and CSV **Succeeded**:
  ```bash
   oc get subscription,installplan,csv,pods -n "${OPERATOR_NS}" -w
  ```

Tenant `ProjectOnboarding` CRs are **not** removed by deleting a CSV. Only remove the subscription if the steps above do not produce a new InstallPlan.

### Last resort: broken intermediate version

If the operator image itself is broken (CrashLoopBackOff on a bad release), delete the broken CSV, clear stale install plans, and recreate the subscription on `**stable`** after pushing a fixed catalog:

```bash
export OPERATOR_NS=project-onboarding-operator

oc delete csv project-onboarding-operator.v0.0.10 -n "${OPERATOR_NS}" --ignore-not-found
oc delete installplan --all -n "${OPERATOR_NS}"

oc delete subscription project-onboarding-operator -n "${OPERATOR_NS}" --ignore-not-found
oc apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: project-onboarding-operator
  namespace: ${OPERATOR_NS}
spec:
  channel: stable
  installPlanApproval: Automatic
  name: project-onboarding-operator
  source: project-onboarding-operator-catalog
  sourceNamespace: openshift-marketplace
EOF

oc get csv,pods -n "${OPERATOR_NS}" -w
```

Push the fixed catalog **before** recreating the subscription.

## Production notes

- **HA:** Leader election is enabled. You may run multiple controller replicas (odd count recommended); see `config/manager/manager.yaml`.
- **Metrics / alerts:** ServiceMonitor and PrometheusRule ship with the bundle; see [metrics.md](metrics.md) to enable scraping and alerts.
- **Stale CRDs:** After major upgrades, remove orphaned CRDs only if nothing references them (`oc get crd | grep onboarding.stderr.at`).

## Uninstall

1. **Operators → Installed Operators** → delete the operator (or delete Subscription).
2. Remove the catalog:
  ```bash
   oc delete catalogsource project-onboarding-operator-catalog -n openshift-marketplace
  ```
3. Delete CRs and CRDs per [openshift-install.md](openshift-install.md) uninstall section.

## Troubleshooting


| Symptom                                            | Check                                                                                                                                       |
| -------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| Operator not in OperatorHub                        | CatalogSource `READY`? Pod running in `openshift-marketplace`?                                                                              |
| `ImagePullBackOff` on catalog pod                  | Quay repo public or `spec.secrets` set?                                                                                                     |
| `exec /bin/opm: Exec format error`                 | Catalog built for wrong CPU arch (Mac ARM). Rebuild with `PLATFORM=linux/amd64` (default in `make catalog-build` / `release-openshift.sh`). |
| InstallPlan not approved                           | **Operators → Installation Plans** → Approve                                                                                                |
| Subscription stuck on old CSV after catalog update | Catalog index may lack prior bundles; rebuild with `CATALOG_BASE_IMG=…/catalog:v<prev>` and repush, then refresh catalog pod                |
| CSV `Pending`                                      | `oc describe csv` — often pull secret on operator Deployment                                                                                |
| `ResolutionFailed` on Subscription                 | See [Stuck or failed upgrade](#stuck-or-failed-upgrade)                                                                                     |


See also: [openshift-install.md](openshift-install.md) (CLI / `operator-sdk run bundle`).