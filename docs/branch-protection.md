# Branch protection for `main`

Enable required status checks so pull requests cannot merge unless CI is green.

## GitHub UI

1. Open **Settings → Branches → Branch protection rules → Add rule** (or edit `main`).
2. Branch name pattern: `main`
3. Enable:
   - **Require a pull request before merging** (optional for solo maintainers)
   - **Require status checks to pass before merging**
   - **Require branches to be up to date before merging**
4. Search and select these status checks (exact names from workflow jobs):

| Status check name | Workflow file |
|-------------------|---------------|
| `Run on Ubuntu` | `test.yml`, `lint.yml`, `test-e2e.yml` (three separate checks with the same name — enable all) |
| `Generate and validate OLM bundle` | `bundle.yml` |
| `Operator SDK scorecard` | `bundle.yml` |
| `Go vulnerability scan` | `security.yml` |
| `Container image vulnerability scan` | `security.yml` |
| `OpenShift test cases (TC-00–TC-14)` | `test-e2e-openshift.yml` (optional; passes when skipped) |

5. Save changes.

## Verify checks appear

After at least one successful run on `main`, checks show up in the branch protection picker. Push a small commit or re-run workflows if the list is empty.

## Release dry-run (no publish)

Use **Actions → Release → Run workflow**:

- **version:** e.g. `0.0.49` (must match `VERSION` in the repo for bundle drift check)
- Runs unit tests, Kind E2E, bundle drift, and scorecard only
- Does **not** push images or create a GitHub Release (publish runs on tag push only)

## Related

- [CONTRIBUTING.md](../CONTRIBUTING.md) — release checklist
- [supply-chain.md](supply-chain.md) — signing and SBOM on tag push
