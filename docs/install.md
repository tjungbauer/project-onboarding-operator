# Install

## Get the source

Clone the repository and change into it (once per machine):

```bash
git clone https://github.com/tjungbauer/project-onboarding-operator.git
cd project-onboarding-operator
```

Commands that reference `make`, `config/`, `VERSION`, or `./scripts/` assume your shell is in the **repository root**.

## Choose an install path

| Path | Guide |
|------|--------|
| **OpenShift OperatorHub (UI)** | [operatorhub-install.md](operatorhub-install.md) |
| **OLM CLI** (`operator-sdk run bundle`) | [openshift-install.md](openshift-install.md) |
| **Development** (`make deploy`) | [local-testing.md](local-testing.md) |
| **Using the operator** (CRs, console form) | [guide.md](guide.md) |

After install:

- Apply samples: `oc apply -k config/samples/`
- Optional cluster GitOps defaults: [cluster-defaults.md](cluster-defaults.md)
- Helm chart migration samples: `config/samples/migration/`

Pause reconciliation on a tenant CR:

```yaml
metadata:
  annotations:
    onboarding.stderr.at/pause-reconciliation: "true"
```

Remove the annotation or set to `false` to resume.

**Offboard before delete:** set `offboard: true` on each namespace entry, then remove the CR. [guide.md — Lifecycle](guide.md#lifecycle-enable-freeze-offboard-and-delete).
