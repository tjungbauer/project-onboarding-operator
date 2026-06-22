# Community OperatorHub submission (Red Hat)

Official listing uses [redhat-openshift-ecosystem/community-operators-prod](https://github.com/redhat-openshift-ecosystem/community-operators-prod). That is **separate** from the custom Quay catalog in [operatorhub-install.md](operatorhub-install.md).

## Phase 1 — Prepare bundle (this repo)

After `make bundle` for the target version:

```bash
export VERSION=0.0.51
export IMG=quay.io/tjungbau/project-onboarding-operator:v${VERSION}

make bundle IMG="${IMG}" VERSION="${VERSION}"   # if bundle drift
make bundle-community                          # or: ./scripts/prepare-community-bundle.sh "${VERSION}"
```

This writes **`dist/community-bundle/<VERSION>/`** with:

| Item | Value |
| ---- | ----- |
| `com.redhat.openshift.versions` | `v4.15-v4.22` (override with `OPENSHIFT_VERSIONS`) |
| `containerImage` + `relatedImages` | Digest-pinned operator image from Quay |
| `spec.replaces` | Removed (`COMMUNITY_FIRST_VERSION=true`, first community PR) |

Validate locally:

```bash
./scripts/validate-community-bundle.sh "${VERSION}"
```

Requires `operator-sdk` and `ocp-olm-catalog-validator` ([releases](https://github.com/redhat-openshift-ecosystem/ocp-olm-catalog-validator/releases) or `make install` from source).

`make bundle` always patches `bundle/metadata/annotations.yaml` and `bundle.Dockerfile` with `com.redhat.openshift.versions` via `scripts/patch-bundle-openshift-versions.sh`.

## Phase 2 — community-operators-prod (two PRs)

FBC submissions require **split PRs**:

| PR | Changes | When |
|----|---------|------|
| **1 — bundle** | `operators/project-onboarding-operator/` only (no `catalogs/`) | First |
| **2 — catalog** | `catalogs/v4.*/project-onboarding-operator/` only | After PR 1 merges |

1. Fork [community-operators-prod](https://github.com/redhat-openshift-ecosystem/community-operators-prod).
2. **PR 1:** add under `operators/project-onboarding-operator/`:
   - `ci.yaml` (FBC enabled — **no** `updateGraph` when `fbc.enabled: true`)
   - `Makefile` (from upstream template)
   - `catalog-templates/basic.yaml`
   - `0.0.51/` — copy from `dist/community-bundle/0.0.51/`
3. Open PR with DCO sign-off (`git commit -s`).
4. After merge, **PR 2:** `cd operators/project-onboarding-operator && make catalogs`, commit only `catalogs/`.

Upstream docs:

- [Contributing prerequisites](https://redhat-openshift-ecosystem.github.io/operator-pipelines/users/contributing-prerequisites/)
- [Where to place operator](https://redhat-openshift-ecosystem.github.io/operator-pipelines/users/contributing-where-to/)
- [ci.yaml](https://redhat-openshift-ecosystem.github.io/operator-pipelines/users/operator-ci-yaml/)

### Example `ci.yaml` (FBC — do not set `updateGraph`)

```yaml
---
fbc:
  enabled: true
  version_promotion_strategy: review-needed
  catalog_mapping:
    - template_name: basic.yaml
      catalog_names:
        - v4.15
        - v4.16
        - v4.17
        - v4.18
        - v4.19
        - v4.20
        - v4.21
        - v4.22
      type: olm.template.basic

reviewers:
  - tjungbau
```

## Subsequent releases

1. Tag and publish from this repo (existing release workflow).
2. Run `make bundle-community` for the new version.
3. **Bundle PR:** new version under `operators/.../<VERSION>/` only.
4. **Catalog PR:** `make catalogs` and commit `catalogs/` only. Use `release-config.yaml` `replaces` / catalog template for upgrade graph (not `updateGraph` in `ci.yaml` when FBC is enabled).

## Related

- [operatorhub-install.md](operatorhub-install.md) — custom CatalogSource (until community listing is live)
- [supply-chain.md](supply-chain.md) — cosign, SBOM, image pinning on release
