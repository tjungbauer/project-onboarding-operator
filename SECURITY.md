# Security

## Supported versions

Security fixes are provided for the **latest minor release** only.

| Version | Supported |
|---------|-----------|
| 0.0.50+ | Yes |
| 0.0.49 and older | No — upgrade to the latest release |

Supported release artifacts:

- Operator manager image (`quay.io/tjungbau/project-onboarding-operator`)
- OLM bundle and catalog images for the same tag

## Reporting a vulnerability

**Do not** open public GitHub issues for security-sensitive reports.

Email **dev@stdin.at** with:

1. Description of the issue and affected component (operator, webhook, OLM bundle, CI)
2. Steps to reproduce
3. Impact assessment (confidentiality, integrity, availability)
4. Your contact details and optional PGP-encrypted report

### Response timeline

| Milestone | Target |
|-----------|--------|
| Initial acknowledgement | 3 business days |
| Severity assessment | 7 business days |
| Fix or mitigation plan | 30 days for High/Critical (best effort for lower severity) |
| Coordinated disclosure | After fix is released and users can upgrade |

We will credit reporters in `CHANGELOG.md` unless you request anonymity.

## Secure consumption

- Verify release images with cosign — see [docs/supply-chain.md](docs/supply-chain.md)
- Upgrade via [docs/upgrade.md](docs/upgrade.md) when security releases are published
- Report cluster misconfiguration (exposed metrics, weak RBAC) through the same channel if it affects operator security posture

## Out of scope

- Vulnerabilities in OpenShift/Kubernetes platform components not introduced by this operator
- Issues in tenant workloads created by `ProjectOnboarding` CRs
- Unsupported forks or modified images not published from this repository's release workflow
