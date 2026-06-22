# Upgrade the operator

Images for each release are on Quay (`quay.io/tjungbau/project-onboarding-operator*`). **Upgrading the cluster is separate from building and pushing images** — use the path that matches how you originally installed the operator.

## Which install method do I have?

Run on the cluster:

```bash
oc get subscription -n project-onboarding-operator
oc get catalogsource -A | grep project-onboarding-operator
```

| What you see | Install method | Upgrade section |
|--------------|----------------|-----------------|
| Subscription name like `project-onboarding-operator-v0-0-22-sub`, channel `operator-sdk-run-bundle`, CatalogSource in **`project-onboarding-operator`** | `operator-sdk run bundle` | [Path A](#path-a-operator-sdk-run-bundle) |
| Subscription `project-onboarding-operator`, channel `stable`, CatalogSource in **`openshift-marketplace`** | OperatorHub / custom catalog | [Path B](#path-b-operatorhub--marketplace-catalog) |

Patching the marketplace CatalogSource **does not** upgrade an `operator-sdk run bundle` install. The subscription points at the in-namespace catalog, not `openshift-marketplace`.

## Prerequisites

- `oc` logged in with permission to manage operators in `project-onboarding-operator`
- Target version **already published** on Quay (GitHub Release tag `vX.Y.Z` or `./scripts/release-openshift.sh` without cluster upgrade)
- For Path A: `operator-sdk` on your PATH

## Path A: `operator-sdk run bundle`

Typical after [openshift-install.md](openshift-install.md) (`operator-sdk run bundle`).

**One command** (replace version):

```bash
operator-sdk run bundle-upgrade quay.io/tjungbau/project-onboarding-operator-bundle:v0.0.51 \
  --namespace project-onboarding-operator \
  --timeout 10m
```

Or from the repository root:

```bash
./scripts/upgrade-cluster.sh 0.0.51
```

The script detects the in-namespace catalog and runs `operator-sdk run bundle-upgrade`.

**Verify:**

```bash
oc wait --for=jsonpath='{.status.phase}'=Succeeded \
  csv/project-onboarding-operator.v0.0.51 \
  -n project-onboarding-operator --timeout=10m

oc rollout status deployment/project-onboarding-operator-controller-manager \
  -n project-onboarding-operator --timeout=5m
```

Refresh **Operators → Installed Operators → Project Onboarding** — version should match the release.

## Path B: OperatorHub / marketplace catalog

Typical after [operatorhub-install.md](operatorhub-install.md).

1. Ensure the new **catalog** image is on Quay (release workflow or `./scripts/release-openshift.sh` without `UPGRADE`).

2. Point the CatalogSource at the new catalog tag and restart the catalog pod:

```bash
export VERSION=0.0.51

oc patch catalogsource project-onboarding-operator-catalog -n openshift-marketplace \
  --type merge -p "{\"spec\":{\"image\":\"quay.io/tjungbau/project-onboarding-operator-catalog:v${VERSION}\"}}"

oc delete pod -n openshift-marketplace -l olm.catalogSource=project-onboarding-operator-catalog

oc wait --for=jsonpath='{.status.connectionState.lastObservedState}'=READY \
  catalogsource/project-onboarding-operator-catalog -n openshift-marketplace --timeout=5m
```

3. Trigger OLM upgrade on the subscription:

```bash
oc patch subscription project-onboarding-operator -n project-onboarding-operator --type merge \
  -p "{\"spec\":{\"channel\":\"stable\",\"installPlanApproval\":\"Automatic\",\"startingCSV\":\"project-onboarding-operator.v${VERSION}\"}}"
```

4. Approve in the console if `installPlanApproval` is `Manual`, or wait for automatic approval.

Or use the helper script (detects marketplace catalog):

```bash
./scripts/upgrade-cluster.sh "${VERSION}"
```

## Build + push + upgrade (maintainers)

To publish images **and** upgrade a cluster in one go:

```bash
./scripts/release-openshift.sh 0.0.51
UPGRADE=true ./scripts/release-openshift.sh 0.0.51
```

`UPGRADE=true` always runs the full **build, bundle, catalog push** pipeline first. That is not the same as cluster-only upgrade.

## Stuck upgrades

See [operatorhub-install.md — Stuck or failed upgrade](operatorhub-install.md#stuck-or-failed-upgrade) and [runbook.md — Failed upgrades](runbook.md#failed-upgrades).

Common fix when CSV stays on an old version: delete the old CSV, patch subscription `startingCSV`, watch InstallPlans:

```bash
export VERSION=0.0.51
export OPERATOR_NS=project-onboarding-operator

oc delete csv project-onboarding-operator.v0.0.50 -n "${OPERATOR_NS}"   # adjust old version
oc patch subscription project-onboarding-operator -n "${OPERATOR_NS}" --type merge \
  -p "{\"spec\":{\"startingCSV\":\"project-onboarding-operator.v${VERSION}\",\"installPlanApproval\":\"Automatic\"}}"

oc get subscription,installplan,csv -n "${OPERATOR_NS}" -w
```

## Related

- [install.md](install.md) — first-time install paths
- [openshift-install.md](openshift-install.md) — OLM CLI install
- [operatorhub-install.md](operatorhub-install.md) — OperatorHub UI install
- [runbook.md](runbook.md) — production troubleshooting
