# AI Control Plane Roadmap

This is the canonical execution-ordered roadmap for AI Control Plane.

It contains outstanding work only. When an item is complete, remove it from this file and capture the shipped outcome in release notes or a changelog entry.

## Scope truth

The roadmap must stay aligned to the current validated claim boundary:

- **Validated now:** local host-first Docker reference implementation, typed operator workflows, readiness evidence, pilot closeout artifact generation, ACP-native assessor packet generation for external-review preparation, a validated customer-operated active-passive HA failover drill evidence surface, an AWS-first incubating cloud deployment package validated through explicit Terraform fmt, validate, and validation-only plan workflows plus AWS hardening guidance and a basic cost-estimation model, and an incubating design-only tenant isolation/billing package.
- **Conditionally ready:** customer pilots on controlled Linux hosts where customer-owned network, IAM, SIEM, retention, and workspace controls are validated.
- **Not yet validated:** Azure/GCP cloud deployment claims, AWS applied/runtime cloud-operation evidence beyond the explicit validation package described above, shared-runtime multi-tenant managed-service claims, and universal browser-bypass prevention.

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
- One-time operational or readiness drills for already-supported workflows belong in runbooks, evidence bundles, and changelog entries, not as roadmap initiatives.
- Do not keep completed items here.

## Execution order

Sequence is deliberate to minimize churn: build on the now-validated host-first recovery and HA evidence baseline plus the AWS-first cloud validation package, ship buyer-proof artifacts after that, and only then invest in deeper platform differentiation.

### Wave 4 — Procurement credibility and buyer proof

Objective: make security, procurement, and executive stakeholders comfortable saying yes.

| # | Initiative | Why it matters | Done when |
| --- | --- | --- | --- |
| 22 | **Obtain external validation.** Add third-party credibility beyond self-assertion. ACP can now generate a preparation-only assessor packet via `make assessor-packet`, but this item stays open until a real outside assessment exists. Prep work is collected in [security/EXTERNAL_REVIEW_READINESS.md](security/EXTERNAL_REVIEW_READINESS.md). | External review materially strengthens buyer confidence. | A security review, architecture review, or equivalent external assessment is completed and can be referenced in buyer conversations. |

### Wave 5 — Product differentiation and deeper platform capability

Objective: build capabilities that create separation from DIY LiteLLM deployments without weakening the core host-first story.

| # | Initiative | Why it matters | Done when |
| --- | --- | --- | --- |
| 29 | **Implement tenant-safe runtime enforcement from the tracked design package.** Turn the design-only package into a validated runtime surface. | The design package is now in place, but managed-service claims still need real runtime isolation, scoped reporting, and customer-safe operating evidence. | Tenant-aware key issuance, query/report scoping, billing boundaries, and managed-service operations are implemented and validated in a dedicated environment or pilot before any shared-runtime multi-tenant claim is made. |

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
