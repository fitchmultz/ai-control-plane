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
- One-time operational or readiness drills for already-supported workflows belong in runbooks, evidence bundles, and changelog entries, not as roadmap initiatives.
- Do not keep completed items here.

## Execution order

Sequence is deliberate to minimize churn: prove the host-first HA path first, validate AWS on top of that baseline second, ship buyer-proof artifacts next, and only then invest in deeper platform differentiation.

### Wave 3 — Host-first production readiness

Objective: make the validated host-first path operationally defensible for real deployments and easier to procure.

| # | Initiative | Why it matters | Done when |
| --- | --- | --- | --- |
| 15 | **Validate the next credible HA path as a host-first active-passive reference pattern.** Turn the documented topology guidance into tested evidence. | Buyers and operators will trust a proven two-host failover pattern more than abstract HA language. It also resolves the newly documented single-node limitation more directly than cloud work. | A two-host active-passive reference path is validated with PostgreSQL replication guidance, fencing/promotion runbooks, customer-owned traffic-cutover guidance, and repeatable failover-drill evidence; unsupported automation remains explicitly out. |
| 16 | **Validate the cloud path deliberately with AWS first.** Do not broaden claims without proof. | Cloud buyers need evidence, not architecture intent, but cloud expansion should follow the strongest host-first recovery and HA baseline. | An AWS-first path is validated with tested IaC, cloud hardening guidance, and a basic cost-estimation model; unsupported cloud claims remain explicitly out. |

### Wave 4 — Procurement credibility and buyer proof

Objective: make security, procurement, and executive stakeholders comfortable saying yes.

| # | Initiative | Why it matters | Done when |
| --- | --- | --- | --- |
| 22 | **Obtain external validation.** Add third-party credibility beyond self-assertion. Prep work is collected in [security/EXTERNAL_REVIEW_READINESS.md](security/EXTERNAL_REVIEW_READINESS.md), but this item stays open until a real outside assessment exists. | External review materially strengthens buyer confidence. | A security review, architecture review, or equivalent external assessment is completed and can be referenced in buyer conversations. |

### Wave 5 — Product differentiation and deeper platform capability

Objective: build capabilities that create separation from DIY LiteLLM deployments without weakening the core host-first story.

| # | Initiative | Why it matters | Done when |
| --- | --- | --- | --- |
| 28 | **Design multi-tenant isolation and service-provider billing.** Create a credible managed-service expansion path. | Multi-tenant claims need real isolation and billing boundaries, not marketing language. | Organization and workspace isolation, row-level access design, tenant-safe reporting, and chargeback boundaries are designed, documented, and validated before any multi-tenant claim is made. |

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
