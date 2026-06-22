# Install project-onboarding-operator via Helm (OLM)

Alternative to `oc apply` on `config/openshift/catalogsource-marketplace.yaml` and manual Subscription.

## Prerequisites

- OpenShift 4.15+ / Kubernetes 1.28+ with OLM installed
- Helm 3

## Install

`catalog.image` and `subscription.startingCSV` default from `Chart.yaml` `appVersion` (keep in sync with repo `VERSION` on release):

```bash
helm install project-onboarding-operator ./charts/project-onboarding-operator \
  --namespace project-onboarding-operator --create-namespace
```

Override for a specific tag:

```bash
helm install project-onboarding-operator ./charts/project-onboarding-operator \
  --namespace project-onboarding-operator --create-namespace \
  --set catalog.image=quay.io/tjungbau/project-onboarding-operator-catalog:v0.0.51 \
  --set subscription.startingCSV=project-onboarding-operator.v0.0.51
```

## Upgrade

Re-install or upgrade the chart after bumping `Chart.yaml` `appVersion` to the target release, or override `catalog.image` / `subscription.startingCSV`:

```bash
helm upgrade project-onboarding-operator ./charts/project-onboarding-operator ...
```

Or use `./scripts/upgrade-cluster.sh` when the catalog index already contains the new bundle.

## Uninstall

```bash
helm uninstall project-onboarding-operator -n project-onboarding-operator
```

OLM CSV and operand resources may remain; use `operator-sdk cleanup` or delete the Subscription manually.

See [docs/install.md](../../docs/install.md) and [docs/operatorhub-install.md](../../docs/operatorhub-install.md).
