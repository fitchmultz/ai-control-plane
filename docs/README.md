# Documentation Index

Start here based on your role.

## I am an operator

- [Operations And Deployment](DEPLOYMENT.md)
- [HA And Failover Topology](deployment/HA_FAILOVER_TOPOLOGY.md)
- [Active-Passive HA Failover Runbook](deployment/HA_FAILOVER_RUNBOOK.md)
- [Certificate Lifecycle](deployment/CERTIFICATE_LIFECYCLE.md)
- [Upgrade And Migration](deployment/UPGRADE_MIGRATION.md)
- [Support](SUPPORT.md)
- [Troubleshooting](troubleshooting/README.md)
- [Examples](../examples/README.md)
- [ACPCTL Reference](reference/acpctl.md)

## I am a buyer / reviewer

- [Root README](../README.md)
- [Compliance Crosswalk](COMPLIANCE_CROSSWALK.md)
- [Security Whitepaper and Threat Model](security/SECURITY_WHITEPAPER_AND_THREAT_MODEL.md)
- [External Review Readiness](security/EXTERNAL_REVIEW_READINESS.md)
- [CVE Remediation and Risk Acceptance Policy](security/CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md)
- [Performance Benchmarks and Sizing Guidance](PERFORMANCE_BASELINE.md)
- [Pilot Closeout Kit](PILOT_CLOSEOUT_KIT.md)
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
