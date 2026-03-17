# Documentation Index

Start here based on your role.

## I am an operator

- [Operations And Deployment](DEPLOYMENT.md)
- [Certificate Lifecycle](deployment/CERTIFICATE_LIFECYCLE.md)
- [Upgrade And Migration](deployment/UPGRADE_MIGRATION.md)
- [Support](SUPPORT.md)
- [Troubleshooting](troubleshooting/README.md)
- [Examples](../examples/README.md)
- [ACPCTL Reference](reference/acpctl.md)

## I am a buyer / reviewer

- [Root README](../README.md)
- [Security And Governance](SECURITY_GOVERNANCE.md)
- [Technical Architecture](technical-architecture.md)
- [Support Matrix](reference/support-matrix.md)
- [Roadmap](ROADMAP.md)

## I am a contributor

- [AGENTS.md](../AGENTS.md)
- [CONTRIBUTING.md](../CONTRIBUTING.md)
- [ADR Home](adr/README.md)
- [Tooling docs](tooling/ACPCTL.md)

## Generated References

- [ACPCTL Reference](reference/acpctl.md)
- [Approved Models](reference/approved-models.md)
- [Detection Rules](reference/detections.md)
- [Support Matrix](reference/support-matrix.md)

## Maintenance Rules

- Generated references and shell completions must stay in sync with the live typed command tree.
- Run `make generate` after command or reference-surface changes.
- Run `make validate-generated-docs` before merging docs-affecting command changes.
