# AI Control Plane

![Version](https://img.shields.io/badge/version-0.1.0-blue)
![License](https://img.shields.io/badge/license-Apache--2.0-green)
![Validated](https://img.shields.io/badge/validated-make%20ci-brightgreen)

AI Control Plane is a host-first Docker reference implementation for enterprise AI governance: a minimal LiteLLM + PostgreSQL runtime wrapped in typed operator workflows, truthful runtime gates, security validation, and evidence-producing delivery artifacts.

![AI Control Plane architecture](docs/images/2026-03-05-19-30-architecture-hero.png)

## Why this repo exists

Most LiteLLM-based projects stop at “the gateway runs.”
This repository goes further:

- typed operator workflows via `acpctl`
- explicit support and claim boundaries
- deterministic validation and smoke gates
- local readiness evidence and release bundles
- buyer-safe documentation for pilots, procurement, and shared responsibility

The canonical execution roadmap for outstanding work lives in [docs/ROADMAP.md](docs/ROADMAP.md).

## Support Boundary

| Status | Boundary |
| --- | --- |
| Validated now | Host-first Docker reference implementation, typed operator workflows, scheduled backup and restore verification, the typed upgrade framework, readiness evidence, and pilot closeout artifacts |
| Conditionally ready | Customer pilots on controlled Linux hosts with customer-owned network, IAM, SIEM, retention, and workspace controls validated |
| Not yet validated | Broad cloud-production claims, multi-tenant managed-service claims, and universal browser-bypass prevention |

- Supported runtime: host-first Docker baseline plus explicit overlays
- Supported host boundary: Debian 12+ or Ubuntu 24.04+ with systemd, apt, Docker, Docker Compose, and verified SSH host keys
- Primary operator UX: `make`
- Typed workflow UX: `./scripts/acpctl.sh`
- Incubating only: deployment-exploration assets under `deploy/incubating/`

## Quick Start

Fastest useful reviewer flow:

```bash
make install
make up-offline
make health
./scripts/acpctl.sh status
./scripts/acpctl.sh doctor
```

Standard connected baseline:

```bash
make install
make up
make health
make operator-report
make operator-dashboard
make prod-smoke
```

Validate tracked config before changing deployment surfaces:

```bash
make validate-config
make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
```

## Operator Paths

- Local baseline: `make up`
- Managed UI overlay: `make up-ui`
- DLP overlay: `make up-dlp`
- Offline deterministic runtime: `make up-offline`
- TLS ingress overlay: `make up-tls`
- Day-2 reporting and recovery: `make operator-report`, `make operator-dashboard`, `make chargeback-report`, `make db-backup`, `make db-backup-retention`, `make dr-drill`, `make ha-failover-drill`
- Certificate lifecycle: `make cert-status`, `make cert-renew`, `sudo make cert-renew-install SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env`
- Host-first upgrade lifecycle: `make upgrade-plan FROM_VERSION=X.Y.Z`, `make upgrade-check FROM_VERSION=X.Y.Z`, `make upgrade-execute FROM_VERSION=X.Y.Z`, `make upgrade-rollback UPGRADE_RUN_DIR=...`
- Key lifecycle: `make key-list`, `make key-inspect ALIAS=<alias>`, `make key-rotate ALIAS=<alias>`
- Guided diagnostics and remediation: `make doctor FIX=1 NOTIFY=1`
- Production-like host path with baseline host hardening: [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)

## Repo Map

| Path | Purpose |
| --- | --- |
| `demo/` | Runnable local runtime and fixtures |
| `examples/` | Curated operator examples and sanitized pilot artifacts |
| `cmd/acpctl/` | Typed CLI surface |
| `internal/` | Typed implementation packages |
| `deploy/` | Supported host deployment assets plus incubating tracks |
| `docs/` | Canonical public docs, references, and architecture material |

## Start Here

- [Documentation index](docs/README.md)
- [Deployment guide](docs/DEPLOYMENT.md)
- [Observability signal reference](docs/observability/OPERATOR_SIGNAL_REFERENCE.md)
- [OTEL collector setup](docs/observability/OTEL_SETUP.md)
- [HA and failover topology](docs/deployment/HA_FAILOVER_TOPOLOGY.md)
- [Active-passive HA failover runbook](docs/deployment/HA_FAILOVER_RUNBOOK.md)
- [Compliance crosswalk](docs/COMPLIANCE_CROSSWALK.md)
- [Security whitepaper and threat model](docs/security/SECURITY_WHITEPAPER_AND_THREAT_MODEL.md)
- [CVE remediation and risk acceptance policy](docs/security/CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md)
- [Performance benchmarks and sizing guidance](docs/PERFORMANCE_BASELINE.md)
- [Security and governance](docs/SECURITY_GOVERNANCE.md)
- [Technical architecture](docs/technical-architecture.md)
- [Examples](examples/README.md)
- [Support matrix](docs/reference/support-matrix.md)
- [Roadmap](docs/ROADMAP.md)
- [Changelog](CHANGELOG.md)
- [Release notes convention](RELEASE_NOTES.md)

## Release Discipline

- Current tracked version: [`VERSION`](VERSION)
- Changes over time: [`CHANGELOG.md`](CHANGELOG.md)
- Release-note convention: [`RELEASE_NOTES.md`](RELEASE_NOTES.md)
- Generated release artifacts: `make release-bundle`, `make readiness-evidence`, `make pilot-closeout-bundle`

## Canonical Docs

- [Roadmap](docs/ROADMAP.md)
- [Compliance Crosswalk](docs/COMPLIANCE_CROSSWALK.md)
- [Security Whitepaper and Threat Model](docs/security/SECURITY_WHITEPAPER_AND_THREAT_MODEL.md)
- [External Review Readiness](docs/security/EXTERNAL_REVIEW_READINESS.md)
- [CVE Remediation and Risk Acceptance Policy](docs/security/CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md)
- [Performance Benchmarks and Sizing Guidance](docs/PERFORMANCE_BASELINE.md)
- [Pilot Closeout Kit](docs/PILOT_CLOSEOUT_KIT.md)
- [Support](docs/SUPPORT.md)
- [Architecture](docs/technical-architecture.md)
- [Operations And Deployment](docs/DEPLOYMENT.md)
- [HA And Failover Topology](docs/deployment/HA_FAILOVER_TOPOLOGY.md)
- [Active-Passive HA Failover Runbook](docs/deployment/HA_FAILOVER_RUNBOOK.md)
- [Certificate Lifecycle](docs/deployment/CERTIFICATE_LIFECYCLE.md)
- [Upgrade And Migration](docs/deployment/UPGRADE_MIGRATION.md)
- [Security And Governance](docs/SECURITY_GOVERNANCE.md)

Generated references:

- [ACPCTL Reference](docs/reference/acpctl.md)
- [Approved Models](docs/reference/approved-models.md)
- [Detection Rules](docs/reference/detections.md)
- [Support Matrix](docs/reference/support-matrix.md)

## Local Env Files

- `demo/.env` is local-demo only.
- `/etc/ai-control-plane/secrets.env` is the canonical host-production env source.
- Production workflows do not sync secrets back into the repository tree.

## License

AI Control Plane is licensed under [Apache-2.0](LICENSE).

- Project notice: [NOTICE](NOTICE)
- Third-party license boundary: [docs/policy/THIRD_PARTY_LICENSE_MATRIX.md](docs/policy/THIRD_PARTY_LICENSE_MATRIX.md)
- Customer-facing compliance summary: [docs/deployment/THIRD_PARTY_LICENSE_SUMMARY.md](docs/deployment/THIRD_PARTY_LICENSE_SUMMARY.md)
