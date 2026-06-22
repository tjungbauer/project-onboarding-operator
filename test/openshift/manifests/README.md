# OpenShift test manifests

YAML files for manual and automated OpenShift E2E (TC-00–TC-15). See [docs/openshift-testcases.md](../../docs/openshift-testcases.md).

| TC | Manifest / procedure |
|----|----------------------|
| TC-01 | `tc01-core-onboarding.yaml` |
| TC-02 | `tc02-tshirt-catalog.yaml` |
| TC-03 | `tc03-tshirt-onboarding.yaml` |
| TC-04 | `tc04-openshift-features.yaml` |
| TC-05 | `tc05-custom-netpol.yaml` |
| TC-06 | `tc06-invalid-tshirtsize.yaml` (dry-run; must fail) |
| TC-07 | `tc07-bad-projectsize.yaml` (dry-run; must fail) |
| TC-08 | Procedural: `oc delete tshirtsize ocp-test-medium --dry-run=server` after TC-02 + TC-03 |
| TC-09 | Procedural: delete `tc01-core-onboarding` CR (see testcases doc) |
| TC-10 | Procedural: patch quota in namespace from TC-03 |
| TC-11 | Procedural: patch `TShirtSize` from TC-02 |
| TC-12 | `tc01-core-onboarding.yaml` (v1beta1 API check; same as TC-01) |
| TC-13 | `tc13-gitops-onboarding.yaml` |
| TC-14 | Procedural: verify ServiceMonitor + PrometheusRule in operator namespace |
| TC-15 | Procedural: verify CSV `spec.replaces` (see testcases doc) |

Cleanup: [`../cleanup.sh`](../cleanup.sh)
