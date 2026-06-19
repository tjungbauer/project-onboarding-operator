# v1alpha1 conversion samples (not in OLM bundle)

These manifests are for manual or automated conversion webhook checks (see `test/openshift/manifests/tc12-api-conversion-v1alpha1.yaml`).

They are **not** included in `config/samples/kustomization.yaml` and therefore do not appear in OperatorHub **alm-examples**.

```bash
oc apply -f config/samples/conversion/
```
