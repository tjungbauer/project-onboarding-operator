# Contributing

Install paths: [docs/install.md](docs/install.md).

1. Fork the repository and create a feature branch.
2. Run `make test` and `make lint` before opening a pull request.
3. Update `CHANGELOG.md` under `[Unreleased]` for user-visible behaviour changes.
4. For OLM/bundle changes, run `make bundle` and ensure CI scorecard passes (`make scorecard` locally requires Kind).
5. Bump `VERSION` only on release (maintainers).

## Changelog

- User-visible changes go under **`[Unreleased]`** at the top of `CHANGELOG.md`.
- On release: move `[Unreleased]` entries into a new `[X.Y.Z] - YYYY-MM-DD` section, clear `[Unreleased]`, bump `VERSION`, run `make bundle`, and commit the regenerated `bundle/`.

## Release checklist (maintainers)

1. Optional: refresh HI digests when adopting new Red Hat Hardened Image bases:
   ```bash
   ./hack/resolve-hi-digests.sh
   git add build/hi-images.lock
   ```
2. Ensure `[Unreleased]` in `CHANGELOG.md` is complete; move entries into `[X.Y.Z] - YYYY-MM-DD` and clear `[Unreleased]`.
3. Bump `VERSION`, run `make bundle`, commit `bundle/` changes.
4. Run `./scripts/pre-release-check.sh` locally (or use **Actions → Release → Run workflow** dry-run).
5. Tag `vX.Y.Z` and push — the release workflow validates, publishes to Quay, **signs images (cosign)**, attaches **SPDX SBOMs**, and creates the GitHub Release. See [docs/supply-chain.md](docs/supply-chain.md).

### Release dry-run (no publish)

**Actions → Release → Run workflow** with version `X.Y.Z` (must match `VERSION` in the repo). Runs tests, Kind E2E, bundle drift, and scorecard only. Publish runs automatically on **tag push**, not on workflow_dispatch.

## Branch protection

See **[docs/branch-protection.md](docs/branch-protection.md)** for step-by-step setup of required status checks on `main`.

Summary — enable on **Settings → Branches → main**:

| Check | Workflow |
|-------|----------|
| Run on Ubuntu | `test.yml` |
| Run on Ubuntu | `lint.yml` |
| Run on Ubuntu | `test-e2e.yml` |
| Generate and validate OLM bundle | `bundle.yml` |
| Operator SDK scorecard | `bundle.yml` |
| Go vulnerability scan | `security.yml` |
| Container image vulnerability scan | `security.yml` |
| OpenShift test cases (TC-00–TC-14) | `test-e2e-openshift.yml` (optional; skips when secret not set) |

Require branches to be up to date before merging.

Optional GitHub secrets:

- `COSIGN_PRIVATE_KEY` + `COSIGN_PASSWORD` for static signing instead of keyless OIDC
- `OPENSHIFT_KUBECONFIG` (base64 kubeconfig) — optional; enables OpenShift E2E (TC-00–TC-14). Without it, that workflow skips cleanly.

## Code layout

| Path | Purpose |
|------|---------|
| `api/` | CRD types (`v1beta1`) |
| `internal/onboarding/` | Reconcile logic |
| `internal/controller/` | Controller-runtime reconcilers |
| `internal/webhook/` | Admission webhooks |
| `config/` | Kustomize manifests, samples, OLM bundle inputs |
| `test/e2e/` | Kind and OpenShift E2E tests |

## Commits

Use imperative, concise subject lines (e.g. `Add pause annotation for ProjectOnboarding`).
