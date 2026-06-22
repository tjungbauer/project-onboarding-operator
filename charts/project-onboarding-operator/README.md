# Install project-onboarding-operator via Helm (OLM)

Alternative to `oc apply` on `config/openshift/catalogsource-marketplace.yaml` and manual Subscription.

## Prerequisites

- OpenShift 4.15+ / Kubernetes 1.28+ with OLM installed
- Helm 3

## Install

```bash
helm install project-onboarding-operator ./charts/project-onboarding-operator \
  --namespace project-onboarding-operator --create-namespace \
  --set catalog.image=quay.io/tjungbau/project-onboarding-operator-catalog:v0.0.50 \
  --set subscription.startingCSV=project-onboarding-operator.v0.0.50
```

## Upgrade

Bump `catalog.image` and `subscription.startingCSV` to the target release, then:

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
