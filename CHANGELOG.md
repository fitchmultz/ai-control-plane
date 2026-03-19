# Changelog

All notable changes to this project will be documented in this file.

This repository uses:
- [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) formatting
- semantic versioning from the tracked [`VERSION`](VERSION) file

## [Unreleased]

### Added
- Machine-readable config contract and schema validation for supported config surfaces.
- `examples/` operator reference directory with reusable deployment and pilot artifacts.
- Troubleshooting index, ADR home, and explicit generated-doc drift validation.
- Shared `internal/health` and tracked `internal/rbac` packages.
- Typed key inventory, inspection, and rotation workflows via `acpctl key list|inspect|rotate` plus `make key-list|key-inspect|key-rotate`.
- One-command operator runtime reporting via `acpctl ops report` and `make operator-report` with private local archive output.
- Doctor budget/detection finding adapters, safe gateway/database remediation helpers, and webhook fanout for `acpctl doctor --notify`.
- Hardened host-playbook defaults for package/update posture, UFW ingress policy, unattended security updates, SSH hardening, and Debian/Ubuntu support-boundary enforcement.
- Automated host-first backup timer assets, typed backup retention cleanup, and a real scratch-restore DR drill workflow.
- Typed host-first certificate lifecycle workflows for listing, validating, renewing, and automating Caddy-managed TLS certificates.
- Typed host-first upgrade planning, rollback artifact capture, compatibility documentation, and future release-edge migration hooks.
- Truthful HA and failover topology guidance for the single-node host-first deployment, including explicit failure-domain, DR-vs-HA, and next-step active-passive reference documentation.
- Separate-host-aware off-host recovery manifests and evidence, including explicit `drill_mode`/`drill_host` labeling and a dedicated replacement-host manifest example.
- Typed active-passive HA failover drill evidence via `acpctl host failover-drill` and `make ha-failover-drill`, plus a dedicated HA runbook, manifest example, and active-passive inventory example.
- Buyer-safe compliance crosswalk covering SOC 2, ISO 27001, and NIST-style controls with explicit customer/shared/provider ownership.
- Canonical security whitepaper and threat model covering architecture, trust boundaries, attacker paths, mitigations, residual risks, and control ownership.
- `docs/PERFORMANCE_BASELINE.md` now serves as the published public benchmark and sizing artifact, with reproducible methodology, profile tables, hardware tiers, result bands, and explicit claim-boundary caveats.
- Canonical CVE governance documentation now formalizes remediation, time-bounded accepted-risk handling, quarterly review records, and buyer-safe vulnerability status communication.
- Pilot closeout materials now include a reusable closeout kit, an anonymized case-study template, a measurable outcomes scorecard template, and a filled Falcon example packet for external-proof reuse.
- External-review readiness documentation now packages the threat model, compliance crosswalk, evidence workflow, and claim-boundary guidance for future third-party assessments without overstating current validation.

### Changed
- Release bundle and readiness workflows now default to the tracked root `VERSION` file.
- Key-generation role validation, default-role resolution, model selection, and least-privileged role inference now derive from the tracked RBAC contract plus approved model catalog instead of duplicated hardcoded role/model mappings.
- Disaster-recovery, HA, production handoff, and production-contract docs now distinguish `staged-local` proof from `separate-host` proof and remove the completed off-host recovery roadmap item.
- Release bundles include explicit release metadata and version-source files.
- Root README now presents the validated support boundary, examples, and release discipline more clearly.
- Gateway host, URL, TLS, and secret ergonomics now follow one canonical operator contract across `make`, `acpctl`, onboarding, and operator docs.
- `acpctl onboard` now performs post-write config linting plus actionable verification summaries for local contract issues, gateway reachability, and authorized model access.
- Wave 2 operator adoption work is now productized through canonical make and `acpctl` entrypoints instead of manual key and reporting runbooks.
- Host production docs and preflight checks now enforce the hardened support boundary: Debian 12+/Ubuntu 24.04+, verified SSH host keys, loopback-only non-TLS base access, and TLS for remote ingress.

### Removed
- Redundant empty placeholder packages: `internal/validate`, `internal/release`, and `internal/key`.

## [0.1.0] - 2026-03-16

### Added
- Initial public baseline for the host-first Docker reference implementation.
- Typed operator workflows for validation, status, readiness evidence, release bundles, and pilot closeout artifacts.
- Public documentation for deployment, governance, demos, and service offerings.
