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

1. Ensure `[Unreleased]` in `CHANGELOG.md` is complete and `[Unreleased]` is empty after the version section is added.
2. Bump `VERSION`, run `make bundle`, commit `bundle/` changes.
3. Tag `vX.Y.Z` and push — the release workflow runs unit tests, Kind E2E, bundle drift check, operator-sdk scorecard, then publishes to Quay, **signs images (cosign)**, attaches **SPDX SBOMs**, and uploads SBOMs to the GitHub Release. See [docs/supply-chain.md](docs/supply-chain.md).

Optional GitHub secrets:

- `COSIGN_PRIVATE_KEY` + `COSIGN_PASSWORD` for static signing instead of keyless OIDC
- `OPENSHIFT_KUBECONFIG` (base64 kubeconfig) — optional; enables OpenShift E2E (TC-00–TC-14). Without it, that workflow skips cleanly.

## Branch protection (recommended)

On GitHub **Settings → Branches → main**, enable required status checks before merge:

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

## Code layout

| Path | Purpose |
|------|---------|
| `api/` | CRD types (`v1beta1` storage, `v1alpha1` conversion) |
| `internal/onboarding/` | Reconcile logic |
| `internal/controller/` | Controller-runtime reconcilers |
| `internal/webhook/` | Admission and conversion webhooks |
| `config/` | Kustomize manifests, samples, OLM bundle inputs |
| `test/e2e/` | Kind and OpenShift E2E tests |

## Commits

Use imperative, concise subject lines (e.g. `Add pause annotation for ProjectOnboarding`).
