# project-onboarding-operator

This operator declaratively onboards tenants on Kubernetes/OpenShift and manages the resources: namespaces, resourcequotas, limit ranges, network policies, egress IPs, and RBAC bindings. Optional GitOps (Argo CD) setup is included.

The operator comes with two custom resources:

- **ProjectOnboarding** — baseline for a new tenant (one CR can manage multiple tenant namespaces, for example when a tenant requires multiple namespaces such as "prod" and "dev")
- **TShirtSize** — optional catalogue of project sizes (for example Small, Medium, Large)

## Container image (Red Hat Hardened Images / Project Hummingbird)

The pod is using a **multi-stage build** image leveraging [Red Hat Hardened Images](https://docs.redhat.com/en/documentation/red_hat_hardened_images/) (Project Hummingbird) to keep the images small and secure:


| Stage   | Image                                               | Purpose                                                   |
| ------- | --------------------------------------------------- | --------------------------------------------------------- |
| Builder | `registry.access.redhat.com/hi/go:latest-builder`   | Compile static Go binary (toolchain stays out of runtime) |
| Runtime | `registry.access.redhat.com/hi/core-runtime:latest` | Minimal distroless base — no shell, no package manager    |


This replaces legacy Docker Hub builder/runtime images (for example `golang:*` and `gcr.io/distroless/static`) with Red Hat–maintained, hardened bases and a smaller attack surface. The project requires **Go 1.25.11+** (see `go.mod`; patch releases address stdlib CVEs scanned by `govulncheck`).

Pin digests or tags for production:

```bash
podman build \
  --platform=linux/amd64 \
  --build-arg HI_GO_BUILDER_IMAGE=registry.access.redhat.com/hi/go:latest-builder \
  --build-arg HI_CORE_RUNTIME_IMAGE=registry.access.redhat.com/hi/core-runtime:latest \
  -t quay.io/<user>/project-onboarding-operator:v$(cat VERSION) .
```

### Build and push (Mac Silicon → OpenShift)

**NOTE**: This example uses [quay.io](https://quay.io) as example image registry. Of course you are free to use any registry you prefer. 

```bash
export IMG=quay.io/<user>/project-onboarding-operator:v$(cat VERSION)
export CONTAINER_TOOL=podman

podman login quay.io
podman build --platform=linux/amd64 -t $IMG .
podman push $IMG

make install
make deploy IMG=$IMG
```

## Documentation

Start with [docs/install.md](docs/install.md), then [docs/guide.md](docs/guide.md) once the operator is on the cluster.


| Topic | Guide |
| ----- | ----- |
| **Install (all paths)** — start here | [docs/install.md](docs/install.md) |
| **User guide** — CRs, console form, lifecycle | [docs/guide.md](docs/guide.md) |
| OpenShift OperatorHub (UI) | [docs/operatorhub-install.md](docs/operatorhub-install.md) |
| OpenShift OLM install (CLI) | [docs/openshift-install.md](docs/openshift-install.md) |
| **Upgrade** (operator-sdk vs OperatorHub) | [docs/upgrade.md](docs/upgrade.md) |
| Architecture | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) |
| API design and versioning (`v1alpha1` / `v1beta1`) | [docs/api-design.md](docs/api-design.md) |
| T-shirt sizing (`TShirtSize`, `projectSize`) | [docs/project-size.md](docs/project-size.md) |
| Metrics & ServiceMonitor | [docs/metrics.md](docs/metrics.md) |
| Cluster GitOps defaults (`onboarding-defaults` ConfigMap) | [docs/cluster-defaults.md](docs/cluster-defaults.md) |
| Local testing (unit → Kind E2E → manual deploy) | [docs/local-testing.md](docs/local-testing.md) |
| OpenShift test cases (TC-00–TC-14) | [docs/openshift-testcases.md](docs/openshift-testcases.md) |
| Operational runbook | [docs/runbook.md](docs/runbook.md) |
| Supply chain (cosign, SBOM) | [docs/supply-chain.md](docs/supply-chain.md) |
| Contributing | [CONTRIBUTING.md](CONTRIBUTING.md) |


## Prerequisites

- Go 1.25.11+
- `podman` or `docker`
- `oc` or `kubectl` with cluster-admin (for install)
- OpenShift 4.15+ or Kubernetes 1.28+

## Deploy

```bash
make manifests generate build
make install
make deploy IMG=quay.io/<user>/project-onboarding-operator:v$(cat VERSION)
kubectl apply -k config/samples/
```

## OLM bundle

```bash
export IMG=quay.io/<user>/project-onboarding-operator:v$(cat VERSION)
make bundle bundle-build bundle-push
```

## Uninstall

Offboard tenant namespaces before deleting a `ProjectOnboarding` (`offboard: true` on each entry). Otherwise the CR can stay in `Terminating`. See [docs/guide.md — Lifecycle](docs/guide.md#lifecycle-enable-freeze-offboard-and-delete).

Development uninstall (`make deploy`):

```bash
# Offboard samples first, then remove CRs and operator
kubectl delete -k config/samples/
make undeploy
make uninstall
```

OpenShift OLM uninstall: [docs/openshift-install.md — Uninstall](docs/openshift-install.md#uninstall).

## Security posture

Hardened runtime image, non-root pod, leader election, HA (3 replicas + PDB), secure metrics, operator NetworkPolicies, reconcile-time GC, OLM **stable** channel. Details per release in [CHANGELOG.md](CHANGELOG.md).

Metrics are served over HTTPS with RBAC filtering (`--metrics-secure=true`). On OpenShift, the bundled `ServiceMonitor` scrapes port 8443 using a bearer token from a namespace `Secret` (`authorization` credentials); cert-manager is not required (OLM injects webhook serving certs separately).

Release version: [VERSION](VERSION) · [CHANGELOG.md](CHANGELOG.md) · [SECURITY.md](SECURITY.md) · [Supply chain](docs/supply-chain.md)

## License

Apache License 2.0 — see [LICENSE](LICENSE).

Copyright 2026 Thomas Jungbauer.
