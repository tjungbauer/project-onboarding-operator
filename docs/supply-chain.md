# Supply chain: signed releases and SBOMs

Release images on Quay are **signed with [cosign](https://docs.sigstore.dev/cosign/overview/)** and ship an **SPDX SBOM** attached to each image. SBOM files are also published as assets on the matching **GitHub Release** (tag `vX.Y.Z`).

## Release images

| Image | Purpose |
| ----- | ------- |
| `quay.io/tjungbau/project-onboarding-operator:vX.Y.Z` | Operator manager |
| `quay.io/tjungbau/project-onboarding-operator-bundle:vX.Y.Z` | OLM bundle |
| `quay.io/tjungbau/project-onboarding-operator-catalog:vX.Y.Z` | OperatorHub catalog index |

Signing runs in the [Release workflow](../.github/workflows/release.yml) after images are pushed (tag push only).

### CI signing (keyless)

GitHub Actions uses **keyless cosign** (`COSIGN_EXPERIMENTAL=1`, workflow `id-token: write`). Signatures are recorded in the Sigstore transparency log and bound to the workflow identity.

No repository secrets are required for keyless signing. Optional: set `COSIGN_PRIVATE_KEY` + `COSIGN_PASSWORD` secrets to use a static key instead (see below).

## Verify a signature

Install cosign, then for a released tag:

```bash
export VERSION=0.0.51
export IMG=quay.io/tjungbau/project-onboarding-operator:v${VERSION}

# Keyless (GitHub Actions OIDC) — adjust identity regex to match your repo/workflow
cosign verify "${IMG}" \
  --certificate-identity-regexp='https://github.com/tjungbauer/project-onboarding-operator/.github/workflows/release.yml@refs/tags/v.*' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com'
```

Repeat for `project-onboarding-operator-bundle` and `project-onboarding-operator-catalog` with the same version tag.

If releases use a **static cosign key** instead, verify with:

```bash
cosign verify --key cosign.pub "${IMG}"
```

Publish `cosign.pub` in the repo or GitHub Release when using static keys.

## Download and inspect the SBOM

From the registry (attached by cosign):

```bash
cosign download sbom "${IMG}" > sbom-operator.spdx.json
```

From GitHub: open the **Release** for tag `vX.Y.Z` and download `project-onboarding-operator-vX.Y.Z-*.spdx.json` assets.

Generate locally (without registry access to signature):

```bash
syft scan quay.io/tjungbau/project-onboarding-operator:v${VERSION} -o spdx-json > sbom-local.spdx.json
```

## Maintainer: sign manually after push

```bash
# Keyless (logged into GitHub via cosign)
export COSIGN_EXPERIMENTAL=1
./scripts/sign-release-images.sh "${VERSION}"

# Or static key
cosign generate-key-pair   # once; store private key in GitHub secret COSIGN_PRIVATE_KEY
export COSIGN_PRIVATE_KEY="$(cat cosign.key)"
export COSIGN_PASSWORD='...'
./scripts/sign-release-images.sh "${VERSION}"
```

Skip steps when testing:

```bash
SKIP_SIGN=true ./scripts/sign-release-images.sh "${VERSION}"   # SBOM only
SKIP_SBOM=true ./scripts/sign-release-images.sh "${VERSION}"   # sign only
```

## SLSA provenance

| Artifact | Status |
| -------- | ------ |
| cosign image signatures | Published on every release (keyless OIDC in CI) |
| SPDX SBOM | Attached via cosign + GitHub Release assets |
| **SLSA provenance** | Attached via `cosign attest --type slsaprovenance` (`scripts/attest-slsa-provenance.sh`) |

Verify signatures and provenance after release:

```bash
./scripts/verify-release-images.sh 0.0.51
./scripts/verify-slsa-provenance.sh 0.0.51
```

Post-release, the Release workflow runs both verifications automatically.

## Container vulnerability scanning (Trivy)

| Image | PR security workflow | Release workflow |
| ----- | -------------------- | ---------------- |
| Operator | **Blocking** (CRITICAL/HIGH, unfixed only) | **Blocking** |
| Bundle | **Blocking** | **Blocking** |
| Catalog | **Informational** (`continue-on-error`) | **Informational** (`continue-on-error`) |

The catalog index image is built from **`opm`** base images. Those bases carry **HIGH** CVEs that this project cannot patch directly. We still build and scan the catalog in CI for visibility, but failures do not block merges or releases.

## Pinned base images (Red Hat Hardened Images)

CI and release builds for **linux/amd64** pin HI images by digest via `build/hi-images.lock` and `scripts/hi-build-args.sh`. The `Dockerfile` defaults to `:latest-builder` / `:latest` so local builds on other architectures still work.

**When to update:** Before a release, when Red Hat publishes new HI builds you want to adopt:

```bash
./hack/resolve-hi-digests.sh   # reads multi-arch manifests, writes linux/amd64 @sha256 into build/hi-images.lock
git add build/hi-images.lock
```

**Release / CI verification:** The Release workflow and `./scripts/pre-release-check.sh` run `./hack/resolve-hi-digests.sh --check` to fail if the lock file drifts from the registry (refresh the lock before tagging).

You do **not** edit digests by hand for routine operator upgrades — only when bumping HI base images. Operator cluster upgrades still use version tags (`./scripts/upgrade-cluster.sh X.Y.Z`).

## OLM operator image digests

| Context | CSV operator `image:` |
| ------- | --------------------- |
| **Git / PR CI** | Semver tag (`quay.io/...:vX.Y.Z`) — `make bundle` drift checks work without registry access |
| **Published bundle** | Immutable digest (`quay.io/...@sha256:…`) plus `spec.relatedImages` and `containerImage` annotation |

`scripts/release-openshift.sh` resolves the operator image digest after push and runs `scripts/pin-csv-operator-image-digest.py` before `bundle-build`. Set `PIN_CSV_IMAGE_DIGEST=false` to skip (local testing only).

Cluster upgrades still use version tags (`./scripts/upgrade-cluster.sh X.Y.Z`); you do not edit digests by hand for routine upgrades.

## Optional GitHub secrets

| Secret | When needed |
| ------ | ----------- |
| `QUAY_USERNAME` / `QUAY_TOKEN` | Push to Quay (already required) |
| `COSIGN_PRIVATE_KEY` | Static signing instead of keyless |
| `COSIGN_PASSWORD` | Passphrase for `COSIGN_PRIVATE_KEY` |

## Related

- [SECURITY.md](../SECURITY.md) — vulnerability reporting
- [CONTRIBUTING.md](../CONTRIBUTING.md) — release checklist
- [runbook.md](runbook.md) — operational troubleshooting
