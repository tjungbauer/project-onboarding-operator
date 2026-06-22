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
3. Bump `VERSION`, run `make bundle`, commit `bundle/` changes. Bump `charts/project-onboarding-operator/Chart.yaml` `version` and `appVersion` to match. If **signed commits** are required on `main`, use `git commit -S` (see [Commits](#commits)).
4. Run `./scripts/pre-release-check.sh` locally (or use **Actions → Release → Run workflow** dry-run).
5. Tag `vX.Y.Z` and push — the release workflow validates, publishes to Quay, **signs images (cosign)**, attaches **SPDX SBOMs**, and creates the GitHub Release. See [docs/supply-chain.md](docs/supply-chain.md).

### Release dry-run (no publish)

**Actions → Release → Run workflow** with version `X.Y.Z` (must match `VERSION` in the repo). Runs tests, Kind E2E, bundle drift, and scorecard only. Publish runs automatically on **tag push**, not on workflow_dispatch.

## Branch protection

Configure on **Settings → Branches → main** (each job `name:` below is a distinct required status check):

| Required check | Workflow |
|----------------|----------|
| Unit tests | `test.yml` |
| Lint | `lint.yml` |
| Kind E2E tests | `test-e2e.yml` |
| Generate and validate OLM bundle | `bundle.yml` |
| Operator SDK scorecard | `bundle.yml` |
| Helm chart lint and template | `bundle.yml` |
| Go vulnerability scan | `security.yml` |
| Container image vulnerability scan | `security.yml` |
| OpenShift test cases (TC-00–TC-15) | `test-e2e-openshift.yml` (optional; skips when secret not set) |

Also enable:

- **Require branches to be up to date** before merging (`strict` status checks) when using pull requests.

**Direct push to `main`:** keep **Do not allow bypassing** (`enforce_admins`) **disabled** so maintainers can push without waiting for every check on every commit. Required checks still gate pull request merges.

After renaming workflow jobs, update required checks in GitHub **after** the new job names have run once on `main` (otherwise pushes are blocked waiting for checks that do not exist yet).

**Optional** (off by default on this repo):

| Setting | Effect |
|---------|--------|
| **Require signed commits** | Every commit on `main` must be signed — use `git commit -S` (see below). |
| **Require linear history** | No merge commits; squash or rebase only. |

Optional GitHub secrets:

- `COSIGN_PRIVATE_KEY` + `COSIGN_PASSWORD` for static signing instead of keyless OIDC
- `OPENSHIFT_KUBECONFIG` (base64 kubeconfig) — optional; enables OpenShift E2E (TC-00–TC-15). Without it, that workflow skips cleanly.

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

### Signed commits (when branch protection requires them)

A plain `git commit -m "..."` is **rejected** if **Require signed commits** is enabled on `main`. Sign each commit with `-S` (or enable signing by default).

**One-time setup — SSH signing (recommended on macOS):**

```bash
# Upload the public key to GitHub → Settings → SSH and GPG keys → New signing key
git config --global gpg.format ssh
git config --global user.signingkey ~/.ssh/id_ed25519.pub   # your signing public key
git config --global commit.gpgsign true                     # optional: sign every commit
```

**One-time setup — GPG:**

```bash
git config --global user.signingkey <YOUR_GPG_KEY_ID>
git config --global commit.gpgsign true
```

**Release commit example:**

```bash
git add -A
git commit -S -m "$(cat <<'EOF'
Release v0.0.51: dedupe SCC binding, Helm appVersion defaults, doc refresh.
EOF
)"
git push origin main
```

Use `-S` on every commit you push to a protected branch while the rule is on. Signed **tags** (`git tag -s v0.0.51 -m "..."`) are separate and optional unless you enforce them elsewhere.
