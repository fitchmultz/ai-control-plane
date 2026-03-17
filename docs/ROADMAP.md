# AI Control Plane Roadmap

This is the canonical execution-ordered roadmap for AI Control Plane.

It contains outstanding work only. When an item is complete, remove it from this file and capture the shipped outcome in release notes or a changelog entry.

## Scope truth

The roadmap must stay aligned to the current validated claim boundary:

- **Validated now:** local host-first Docker reference implementation, typed operator workflows, readiness evidence, and pilot closeout artifact generation.
- **Conditionally ready:** customer pilots on controlled Linux hosts where customer-owned network, IAM, SIEM, retention, and workspace controls are validated.
- **Not yet validated:** cloud-production enforcement claims, multi-tenant managed-service claims, and universal browser-bypass prevention.

Primary claim-discipline references:

- [GO_TO_MARKET_SCOPE.md](GO_TO_MARKET_SCOPE.md)
- [ENTERPRISE_STRATEGY.md](ENTERPRISE_STRATEGY.md)
- [ENTERPRISE_PILOT_PACKAGE.md](ENTERPRISE_PILOT_PACKAGE.md)
- [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md)

## Operating rules

- This file is the only canonical roadmap.
- Other docs may describe strategy, maturity progression, or support boundaries, but they should link here instead of carrying a competing backlog.
- Prioritize work that improves truthfulness, operator adoption, deployment confidence, and procurement credibility before adding differentiating scope.
- Harden the host-first deployment path before broadening cloud or managed-service claims.
- Do not keep completed items here.

## Execution order

### Wave 2 — Operator adoption and day-2 usability

Objective: collapse time-to-first-success, make routine operations obvious, and remove the need for tribal knowledge.

| # | Initiative | Why it matters | Done when |
| --- | --- | --- | --- |
| 7 | **Automate key lifecycle workflows.** Replace manual rotation and inspection runbooks with productized flows. | Key management is security-critical and should not depend on copy-pasted steps. | Operators can generate replacements, inspect current usage, stage cutover, and revoke old keys through typed workflows. |
| 8 | **Add one-command finance and operator workflows.** Remove the need for direct SQL and Docker knowledge for routine operations. | Operational confidence goes up when common tasks are easy and safe. | Monthly chargeback and showback reporting plus common operator reporting tasks have first-class commands with documented outputs. |
| 9 | **Extend doctor with safe remediation and alert adapters.** Turn diagnosis into controlled action. | Troubleshooting should be fast, guided, and low-risk. | `doctor --fix` safely handles common recoveries, and budget or security findings can emit notifications through pluggable channels. |

### Wave 3 — Host-first production readiness

Objective: make the validated host-first path operationally defensible for real deployments and easier to procure.

| # | Initiative | Why it matters | Done when |
| --- | --- | --- | --- |
| 10 | **Harden host deployment playbooks.** Make the supported deployment path production-defensible. | Production buyers care about repeatability, hardening, and support boundaries more than raw feature count. | Ansible covers baseline hardening, firewall defaults, package and update posture, and host preflight checks aligned to the support boundary. |
| 11 | **Automate backups and restore verification.** Make recovery boring. | Procurement and operations teams need evidence that recovery is routine, not theoretical. | Scheduled backups, retention, restore validation, and drill documentation exist for the supported host-first path. |
| 12 | **Add an upgrade and migration framework.** Close the gap between reference implementation and operable product. | Real deployments need a safe path forward, not just a clean install story. | Version-to-version upgrade steps, database and config migrations, rollback guidance, and compatibility expectations are documented and tested. |
| 13 | **Manage certificate lifecycle.** Remove TLS hand-waving from the production story. | Buyers expect TLS operations to be explicit and repeatable. | Certificate issuance, renewal, expiry detection, and operator runbooks are documented and at least partially automated for the supported path. |
| 14 | **Publish a reference HA and failover pattern.** Be explicit about single-node limits and the next credible step. | Honest topology guidance builds more trust than vague “enterprise ready” language. | The repo documents a supported primary topology, failure domains, backup and failover expectations, and what remains customer-owned. |
| 15 | **Validate the cloud path deliberately with AWS first.** Do not broaden claims without proof. | Cloud buyers need evidence, not architecture intent. | An AWS-first path is validated with tested IaC, cloud hardening guidance, and a basic cost-estimation model; unsupported cloud claims remain explicitly out. |

### Wave 4 — Procurement credibility and buyer proof

Objective: make security, procurement, and executive stakeholders comfortable saying yes.

| # | Initiative | Why it matters | Done when |
| --- | --- | --- | --- |
| 16 | **Publish a compliance crosswalk.** Give security and procurement teams something concrete to review. | Mapped controls make due diligence faster and more credible. | A detailed control mapping exists for at least SOC 2, ISO 27001, and NIST-style controls, with customer, shared, and provider ownership called out. |
| 17 | **Publish a security whitepaper and threat model.** Strengthen technical credibility. | Procurement confidence improves when the risk story is explicit and professionally documented. | Architecture, trust boundaries, threats, mitigations, residual risks, and control ownership are documented in one buyer-safe artifact. |
| 18 | **Publish public performance benchmarks and sizing guidance.** Move from “reference host only” to reproducible capacity evidence. | Buyers need sizing and performance evidence before they approve production spend. | Methodology, workload profiles, hardware and software baseline, results, and sizing caveats are published and reproducible. |
| 19 | **Formalize CVE remediation and accepted-risk handling.** Make known limitations look governed rather than ad hoc. | Open vulnerabilities can be acceptable, but only when governance is credible and current. | Open CVEs have remediation plans or time-bounded accepted-risk records, a quarterly review cadence, and buyer-safe status communication. |
| 20 | **Create an anonymized case-study and pilot closeout kit.** Turn delivery maturity into reusable external proof. | Social proof and measurable outcomes reduce sales friction. | A reusable case-study template, measurable outcomes format, and evidence-backed closeout packet exist for successful pilots. |
| 21 | **Obtain external validation.** Add third-party credibility beyond self-assertion. | External review materially strengthens buyer confidence. | A security review, architecture review, or equivalent external assessment is completed and can be referenced in buyer conversations. |

### Wave 5 — Product differentiation and deeper platform capability

Objective: build capabilities that create separation from DIY LiteLLM deployments without weakening the core host-first story.

| # | Initiative | Why it matters | Done when |
| --- | --- | --- | --- |
| 22 | **Unify typed ownership for RBAC, health, validation, release, and key domains.** Close the gap between documented architecture and package reality. | Clean domain ownership reduces maintenance drag and makes the operator surface more coherent. | Domain boundaries are implemented cleanly in typed packages, CLI surfaces use them directly, and tests cover the resulting contracts. |
| 23 | **Standardize structured logging.** Make operations and forensics consistent across Go packages. | Consistent telemetry is foundational for supportability and incident response. | One logging contract exists, fields are consistent across workflows, and critical paths emit structured events suitable for SIEM and observability tooling. |
| 24 | **Add metrics, tracing, and operational dashboards.** Improve operator visibility. | Great operator UX requires visibility, not just command success or failure. | Key request, enforcement, cost, error, backup, and readiness signals are exposed; dashboards and tracing guidance exist for supported deployments. |
| 25 | **Add a vendor webhook and export ingestion surface.** Receive audit evidence programmatically instead of manually. | Automated evidence ingestion raises the ceiling on real-world usefulness. | A typed receiver and normalizer exist for supported vendor export flows, with schema validation and evidence-pipeline integration. |
| 26 | **Add a custom policy engine.** Differentiate beyond LiteLLM-native guardrails. | Proprietary enforcement and policy logic are where long-term platform value can compound. | ACP-native policy evaluation can inspect requests and responses, apply custom guardrails, and emit auditable policy decisions. |
| 27 | **Design multi-tenant isolation and service-provider billing.** Create a credible managed-service expansion path. | Multi-tenant claims need real isolation and billing boundaries, not marketing language. | Organization and workspace isolation, row-level access design, tenant-safe reporting, and chargeback boundaries are designed, documented, and validated before any multi-tenant claim is made. |

## Review cadence

Reassess this roadmap whenever one of these changes:

- the support boundary changes
- a previously unvalidated claim becomes validated
- major limitations change
- a wave is materially completed

## Related documents

- [GO_TO_MARKET_SCOPE.md](GO_TO_MARKET_SCOPE.md)
- [ENTERPRISE_STRATEGY.md](ENTERPRISE_STRATEGY.md)
- [SERVICE_OFFERINGS.md](SERVICE_OFFERINGS.md)
- [ENTERPRISE_PILOT_PACKAGE.md](ENTERPRISE_PILOT_PACKAGE.md)
- [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md)
- [release/READINESS_EVIDENCE_WORKFLOW.md](release/READINESS_EVIDENCE_WORKFLOW.md)
