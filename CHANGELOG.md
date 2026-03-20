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
- Host-first observability now includes typed traffic, backup, and readiness status collectors plus a browser-friendly static operator dashboard snapshot via `make operator-dashboard`.
- Typed vendor evidence ingest via `acpctl evidence ingest`, including compliance-export normalization, schema validation, artifact runs, and example payloads.
- ACP-native local custom policy evaluation via `acpctl policy eval`, including tracked rule validation, request/response guardrail inspection, auditable decision artifacts, and sample policy-engine payloads.
- ACP-native assessor handoff packaging via `acpctl deploy assessor-packet build|verify` and `make assessor-packet|assessor-packet-verify`, producing a verifiable external-review preparation packet with canonical reviewer docs, readiness evidence, and the referenced release bundle while keeping roadmap item #22 explicitly open.
- A design-only multi-tenant isolation and billing package via `demo/config/tenant_design.yaml`, `acpctl tenant inspect|validate`, `acpctl validate tenant`, `make validate-tenant`, and a new ADR/policy doc set that defines organization/workspace isolation, row-level predicates, tenant-safe reporting, and provider billing boundaries without overstating current runtime support.
- An AWS-first incubating cloud validation package via `make tf-fmt-check`, `make tf-validate`, `make tf-plan-aws`, `docs/deployment/TERRAFORM.md`, `docs/security/AWS_CLOUD_HARDENING.md`, `docs/deployment/AWS_COST_ESTIMATION.md`, and ADR 0003, keeping Terraform incubating while making AWS-specific claim boundaries explicit.

### Changed
- Support, topology, and scope documents now promote the customer-operated active-passive HA failover drill evidence workflow to a validated supported surface while keeping automatic failover, ACP-managed replication, ACP-managed fencing/promotion, and customer-owned traffic cutover automation explicitly out of scope.
- Cloud scope, roadmap, compliance, and whitepaper documents now recognize the AWS-first incubating Terraform validation package while keeping Azure/GCP and named-account/runtime cloud claims explicitly out of scope.
- Release bundle and readiness workflows now default to the tracked root `VERSION` file.
- Key-generation role validation, default-role resolution, model selection, and least-privileged role inference now derive from the tracked RBAC contract plus approved model catalog instead of duplicated hardcoded role/model mappings.
- Doctor now consumes `internal/health` directly for shared health levels instead of routing that vocabulary through `internal/status`.
- Typed domain ownership now aligns cleanly across RBAC, health, validation, release, and key workflows, closing the roadmap item for those package boundaries.
- Structured workflow logging now uses one canonical helper contract across command handlers plus the release-bundle, readiness, pilot-closeout, and onboarding workflows, including verify paths.
- Disaster-recovery, HA, production handoff, and production-contract docs now distinguish `staged-local` proof from `separate-host` proof and remove the completed off-host recovery roadmap item.
- Release bundles include explicit release metadata and version-source files.
- Root README now presents the validated support boundary, examples, and release discipline more clearly.
- Gateway host, URL, TLS, and secret ergonomics now follow one canonical operator contract across `make`, `acpctl`, onboarding, and operator docs.
- `acpctl onboard` now performs post-write config linting plus actionable verification summaries for local contract issues, gateway reachability, and authorized model access.
- Wave 2 operator adoption work is now productized through canonical make and `acpctl` entrypoints instead of manual key and reporting runbooks.
- Host production docs and preflight checks now enforce the hardened support boundary: Debian 12+/Ubuntu 24.04+, verified SSH host keys, loopback-only non-TLS base access, and TLS for remote ingress.
- Operator reporting now exposes recent routed traffic, backup freshness, and readiness-evidence freshness through `status`, `ops report`, and the new static HTML dashboard workflow.
- Evidence handling now includes typed local ingest and policy-evaluation surfaces for supported vendor exports and ACP-native request/response guardrail evaluation instead of requiring manual normalization.

### Removed
- Redundant empty placeholder packages: `internal/validate`, `internal/release`, and `internal/key`.

## [0.1.0] - 2026-03-16

### Added
- Initial public baseline for the host-first Docker reference implementation.
- Typed operator workflows for validation, status, readiness evidence, release bundles, and pilot closeout artifacts.
- Public documentation for deployment, governance, demos, and service offerings.
