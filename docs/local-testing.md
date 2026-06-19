# Local testing

Unit tests, Kind E2E, manual deploy, and optional OpenShift runs for **project-onboarding-operator**.

## Test levels


| Test level                    | Cluster needed?          | What it validates                                                        |
| ----------------------------- | ------------------------ | ------------------------------------------------------------------------ |
| Unit tests (`make test`)      | No                       | Go/controller wiring                                                     |
| E2E on Kind (`make test-e2e`) | Local Kind               | CRD install, Deployment, health, metrics                                 |
| Manual deploy                 | Any K8s/OCP              | Same as E2E, plus sample CR                                              |
| OpenShift-only features       | **OpenShift** (optional) | Groups, EgressIPs — see [openshift-testcases.md](openshift-testcases.md) |


You do **not** need OpenShift for levels 1–3. Kind covers core reconciliation; Groups and EgressIPs are skipped on plain Kubernetes.

Reconcile logic covers namespaces, ResourceQuotas, LimitRanges, NetworkPolicies, OpenShift Groups/RoleBindings, OVN EgressIPs, and Argo CD AppProjects when configured.

---

## Prerequisites

Install on your workstation:

- **Go 1.25+**
- **kubectl** or **oc**
- **podman** or **docker** (for image-based tests; `make` auto-uses podman when docker is not installed)
- **kind** (for automated E2E)
- **make**

Optional for OpenShift testing later:

- **CRC** (local OpenShift) or access to a dev/test OCP cluster

For the [hardened container image build](https://hummingbird-project.io/docs/using/overview/) ([Red Hat Hardened Images](https://docs.redhat.com/en/documentation/red_hat_hardened_images/)), Red Hat registry access may be required:

```bash
podman login registry.access.redhat.com
```

---

## Step 1 — No cluster (fastest)

From the repository root:

```bash
make test          # unit tests (uses envtest, no real cluster)
make lint          # golangci-lint
make build         # compile bin/manager
```

Verbose reconcile logging (development-style zap output) is **off** by default. Enable it on the Deployment or when running locally:

```bash
export DEBUG=true
make run
# or: oc set env deployment/project-onboarding-operator-controller-manager -n project-onboarding-operator DEBUG=true
```

This confirms the code compiles and the controller runs locally with verbose logging when needed.

---

## Step 2 — Automated E2E on Kind (recommended next step)

This matches what CI runs. It creates a Kind cluster, builds the image, installs CRDs, deploys the operator, and checks the pod and metrics endpoint.

From the repository root:

```bash
# Install kind if missing, e.g. on macOS:
# brew install kind

make test-e2e
```

What it does internally:

1. Creates Kind cluster `project-onboarding-operator-test-e2e`
2. Builds image `example.com/project-onboarding-operator:v<VERSION>` (reads the repo `VERSION` file)
3. Loads the image into Kind
4. Installs CertManager (skip with `CERT_MANAGER_INSTALL_SKIP=true`)
5. Runs `make install` and `make deploy`
6. Verifies the controller pod is Running and the metrics endpoint works
7. Deletes the Kind cluster

On Apple Silicon, Kind runs natively. The E2E image build uses your local architecture unless you override `IMG`.

If you only have podman (no Docker Desktop), `make test-e2e` should work without extra setup — the Makefile picks podman automatically. To force it:

```bash
export CONTAINER_TOOL=podman
make test-e2e
```

---

## Step 3 — Manual deploy to a local cluster

Use this when you want to inspect CRs yourself (`kubectl get projectonboardings`, logs, etc.).

### Option A: Kind (Kubernetes only)

From the repository root:

```bash
kind create cluster --name project-onboarding

export IMG=project-onboarding-operator:dev
export CONTAINER_TOOL=podman   # or docker

# Build and load into Kind
make docker-build
kind load docker-image $IMG --name project-onboarding

kubectl cluster-info

make install
make deploy IMG=$IMG

kubectl apply -k config/samples/

kubectl get pods -n project-onboarding-operator
kubectl get projectonboardings
kubectl logs -n project-onboarding-operator \
  -l control-plane=controller-manager -c manager -f
```

### Option B: Run controller on your laptop (no container)

Useful for quick reconcile debugging once logic is implemented. From the repository root:

```bash
# Cluster must match your current kubeconfig context
make install    # CRDs only

# Your kubeconfig user needs cluster-admin (or equivalent RBAC)
make run
```

**NOTE**: Do **not** run `make deploy` and `make run` at the same time — that would start two controllers.

### Cleanup

From the repository root:

```bash
kubectl delete -k config/samples/
make undeploy
make uninstall

# If using Kind:
kind delete cluster --name project-onboarding
```

---

## Step 4 — OpenShift

The operator targets OpenShift for full functionality:

- OpenShift **Projects** / **Namespaces**
- **EgressIP** (OVN), **NetworkPolicy**
- OpenShift **Groups** and **RoleBindings**
- SCC-aware workload (`nonroot-v2` binding in `config/openshift/`)

### You do not need a public cluster

Any of these work:


| Option                          | Good for                                 |
| ------------------------------- | ---------------------------------------- |
| **CRC** (`crc start`)           | Local OCP on your laptop                 |
| **Developer Sandbox** (Red Hat) | Free shared OCP without running infra    |
| **Company dev/test cluster**    | Realistic policies, GitOps, etc.         |
| **ROSA/OSD trial**              | Cloud OCP when shared access is required |


You need **cluster-admin** (or enough permissions to install CRDs and deploy operators). A public cloud OpenShift cluster is optional.

---



### Deploy to OpenShift

Use the OLM install guide: [openshift-install.md](openshift-install.md).

Quick summary: build/push image and bundle (`./scripts/release-openshift.sh $(cat VERSION)`), install via [operatorhub-install.md](operatorhub-install.md) or [openshift-install.md](openshift-install.md), then `oc apply -f config/samples/`. 

### Automated verification/tests on OpenShift

After the operator is installed:

```bash
export OPENSHIFT_E2E=true
make test-e2e-openshift
```

See [openshift-testcases.md](openshift-testcases.md) for TC-00–TC-14 details.

---

## Suggested order

1. `make test` — confirm baseline
2. `make test-e2e` — full operator smoke test on Kind
3. Manual Kind deploy + `kubectl apply -k config/samples/` — inspect CR and status
4. OpenShift (CRC or sandbox) — when building OpenShift-specific onboarding logic

---

## Uninstall

From the repository root:

```bash
kubectl delete -k config/samples/
make undeploy
make uninstall
```

Use `oc` instead of `kubectl` on OpenShift.