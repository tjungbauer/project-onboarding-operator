# Contributing

Install paths: [docs/install.md](docs/install.md).

1. Fork the repository and create a feature branch.
2. Run `make test` and `make lint` before opening a pull request.
3. Update `CHANGELOG.md` under `[Unreleased]` for user-visible behaviour changes.
4. Bump `VERSION` only on release branches/tags (maintainers).

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
