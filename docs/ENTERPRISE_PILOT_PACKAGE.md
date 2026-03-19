# Enterprise Pilot Package

Single source of truth for running a credible customer pilot from this repository.

---

## Purpose

This package defines what the AI Control Plane pilot proves, what the customer must provide, and how success is measured.

Use it when a prospect asks one of these questions:
- What exactly are we piloting?
- What will we be able to prove in 30-60 days?
- What do you need from our security, network, IAM, and SIEM teams?
- What is enforced vs only detected?

This document is intentionally strict. It avoids aspirational language and only covers what this repository can support with a defensible validation path.

---

## Pilot Outcome

A successful pilot proves five things:

1. Approved routed AI usage can be enforced through the gateway.
2. Usage can be attributed to people, teams, or services with enough fidelity for audit and chargeback/showback.
3. Detections and normalized evidence can land in the customer SIEM with clear runbooks.
4. Direct or bypass usage can be identified and escalated as a network and governance problem.
5. The operating team can run the platform with repeatable commands, health checks, and release evidence.

The pilot is only complete when those outcomes are converted into an explicit closeout decision under the phase model in `docs/PILOT_EXECUTION_MODEL.md`.

## Pilot Phase Model

Every pilot should be run through these phases:

1. Qualify
2. Charter
3. Implement
4. Validate customer controls
5. Decide
6. Transition

Source of truth: `docs/PILOT_EXECUTION_MODEL.md`

If the team skips the customer-validation or decision phases, the engagement should be described as a reference validation exercise, not as rollout readiness.

## Pilot Document Set

Use these documents together instead of creating custom pilot prose from scratch:

- `docs/ENTERPRISE_PILOT_PACKAGE.md` for the pilot boundary and proof model
- `docs/PILOT_EXECUTION_MODEL.md` for the required phase model and hard-stop rules
- `docs/PILOT_SPONSOR_ONE_PAGER.md` for executive sponsor and procurement review
- `docs/PILOT_CONTROL_OWNERSHIP_MATRIX.md` for named customer and delivery-team owners
- `docs/PILOT_CUSTOMER_VALIDATION_CHECKLIST.md` for customer-environment validation tracking
- `docs/SHARED_RESPONSIBILITY_MODEL.md` for the operating boundary after handoff
- `docs/templates/PILOT_CHARTER.md` to start the pilot
- `docs/templates/PILOT_MEASURABLE_OUTCOMES_SCORECARD.md` to quantify closeout results
- `docs/templates/PILOT_ACCEPTANCE_MEMO.md` to close the pilot or record checkpoint decisions
- `docs/templates/PILOT_CASE_STUDY.md` to publish a sanitized buyer-facing narrative
- `docs/templates/PILOT_OPERATOR_HANDOFF_CHECKLIST.md` for day-to-day operator turnover
- `docs/PILOT_CLOSEOUT_EXAMPLES.md` for decision-grade closeout language
- `docs/PILOT_CLOSEOUT_KIT.md` for the reusable assembly workflow

---

## What This Repo Proves

| Area | Proven in repo | Proof path |
|---|---|---|
| Gateway health and operator workflow | Yes | `make ci`, `make health`, `make status`, `docs/RUNBOOK.md` |
| Approved-model enforcement contract | Yes | `demo/config/litellm.yaml`, `docs/policy/APPROVED_MODELS.md`, `make validate-detections`, `make validate-siem-schema` |
| Budget and rate-limit governance | Yes | `docs/policy/BUDGETS_AND_RATE_LIMITS.md`, `docs/policy/FINANCIAL_GOVERNANCE_AND_CHARGEBACK.md`, `make db-status` |
| Detection pack to SIEM mapping consistency | Yes | `demo/config/detection_rules.yaml`, `demo/config/siem_queries.yaml`, `make validate-detections`, `make validate-siem-schema` |
| Managed browser path architecture | Yes, architecture + config baseline | `docs/security/ENTERPRISE_AUTH_ARCHITECTURE.md`, `docs/tooling/LIBRECHAT.md` |
| Customer-network egress blocking effectiveness | No | Customer environment validation required |
| External compliance attestation | No | [Compliance crosswalk](COMPLIANCE_CROSSWALK.md) and [security/EXTERNAL_REVIEW_READINESS.md](security/EXTERNAL_REVIEW_READINESS.md) are preparation material, not completed external validation |
| Cloud-specific production hardening | Not yet in this repo baseline | `docs/GO_TO_MARKET_SCOPE.md` |

---

## Customer Inputs Required

The pilot is not a pure software exercise. These customer-owned inputs are required:

| Workstream | Customer input |
|---|---|
| Network | SWG/CASB/firewall owner, outbound policy process, approved egress model |
| Identity | IdP owner, workspace admins, SSO/SCIM contacts |
| Security operations | SIEM owner, log ingestion path, incident routing destination |
| FinOps | Cost-center model, budget owners, showback/chargeback reviewer |
| Endpoint / IT | Managed-device policy owner for browser and IDE controls |
| Pilot operations | Named pilot sponsor, operator, and user cohort |

If these owners are not available, the pilot can still show gateway enforcement locally, but it cannot credibly prove enterprise rollout readiness.

Canonical owner matrix: [PILOT_CONTROL_OWNERSHIP_MATRIX.md](PILOT_CONTROL_OWNERSHIP_MATRIX.md)
Canonical customer validation tracker: [PILOT_CUSTOMER_VALIDATION_CHECKLIST.md](PILOT_CUSTOMER_VALIDATION_CHECKLIST.md)

## Prerequisites Before Kickoff

Do not schedule pilot kickoff until the following are true:

- the customer sponsor has named owners for network, IAM, SIEM, FinOps, and platform operations
- the customer agrees whether bypass prevention is an enforcement goal or a detective-only goal for the pilot
- the target environment and change window are identified
- the evidence review cadence is agreed
- the exit decision date is agreed in writing

Hard-stop rule:
- if critical owners are missing, do not call the engagement a pilot

---

## Pilot Scope

### In scope

- Linux host deployment for the gateway baseline
- Approved-model allowlists
- Virtual-key identity and spend attribution
- Budget and rate-limit controls
- Detection rules and SIEM mappings
- Managed browser path architecture review
- Runbooks, release evidence, and operator handoff
- Customer ownership matrix and shared-responsibility model

### Out of scope unless separately validated

- Claims that all AI traffic is blocked outside the gateway
- Multi-tenant service-provider operation
- Cloud-specific egress-control guarantees
- Formal compliance certification or auditor attestation
- Production SLA commitments beyond customer-specific agreement

---

## Objective Success Criteria

| Success criterion | How to verify | Owner |
|---|---|---|
| Gateway baseline is healthy and repeatable | `make ci`, `make health`, `make status` | Delivery team |
| Detection pack and SIEM mappings are internally consistent | `make validate-detections`; `make validate-siem-schema` | Delivery team |
| Unapproved routed request is rejected or surfaced correctly | Run approved/unapproved demo scenarios and capture logs | Delivery team + customer security |
| SIEM receives normalized evidence with user/team attribution | Customer ingestion test using `docs/security/SIEM_INTEGRATION.md` | Customer SOC |
| Bypass path is classified as enforced or detective in writing | Review against `docs/ENTERPRISE_BUYER_OBJECTIONS.md` and network-owner signoff | Customer network + sponsor |
| Pilot operators can execute release/rollback/runbook steps | `make readiness-evidence`, `make readiness-evidence-verify`, runbook walkthrough | Delivery team + customer ops |

---

## Minimum Pilot Command Set

Run these before customer review or pilot checkpoint meetings:

```bash
make readiness-evidence
make readiness-evidence-verify
make pilot-closeout-bundle
make pilot-closeout-bundle-verify
make health
make validate-detections
make validate-siem-schema
```

To build a complete packet, start from:
- `docs/templates/PILOT_CHARTER.md`
- `docs/templates/PILOT_MEASURABLE_OUTCOMES_SCORECARD.md`
- `docs/templates/PILOT_ACCEPTANCE_MEMO.md`
- `docs/templates/PILOT_CASE_STUDY.md`
- `docs/PILOT_CUSTOMER_VALIDATION_CHECKLIST.md`
- `docs/templates/PILOT_OPERATOR_HANDOFF_CHECKLIST.md`

If the customer pilot includes browser governance, also review:
- `docs/security/ENTERPRISE_AUTH_ARCHITECTURE.md`
- `docs/tooling/LIBRECHAT.md`
- `docs/security/SIEM_INTEGRATION.md`
- `docs/BROWSER_WORKSPACE_PROOF_TRACK.md`

---

## Deliverables

The minimum deliverable set for a credible pilot is:

1. Pilot architecture and scope statement
2. Approved-model and budget policy baseline
3. Detection pack with mapped SIEM queries
4. Runbook set for health, status, and incident response
5. Release bundle and validation record
6. Explicit enforce-vs-detect boundary statement signed off by customer stakeholders
7. Named control-owner matrix and shared responsibility matrix
8. Pilot charter, measurable outcomes scorecard, and acceptance memo
9. An anonymized case study plus customer validation checklist with evidence references
10. Pilot closeout bundle containing the closeout packet plus current readiness evidence

Recommended supporting docs:
- `docs/ENTERPRISE_STRATEGY.md`
- `docs/ENTERPRISE_BUYER_OBJECTIONS.md`
- `docs/GO_TO_MARKET_SCOPE.md`
- `docs/KNOWN_LIMITATIONS.md`
- `docs/release/PRESENTATION_READINESS_TRACKER.md`
- `docs/PILOT_CONTROL_OWNERSHIP_MATRIX.md`
- `docs/PILOT_CUSTOMER_VALIDATION_CHECKLIST.md`
- `docs/PILOT_CLOSEOUT_EXAMPLES.md`
- `docs/PILOT_CLOSEOUT_KIT.md`
- `docs/SHARED_RESPONSIBILITY_MODEL.md`

---

## Exit Criteria

Do not call the pilot complete until all of the following are true:

- Customer owners for network, IAM, SIEM, and FinOps have reviewed the control boundaries.
- Gateway-routed enforcement has been demonstrated with current evidence.
- Detection/SIEM validation commands pass on the pilot configuration.
- Known limitations and customer-owned controls are documented in writing.
- The sponsor agrees whether the next step is expansion, remediation, or no-go.
- The acceptance memo and closeout bundle point to current evidence instead of historical meeting memory.

## Pilot Governance Rule

If customer-owned prerequisites are not met, the pilot should be reframed as a reference validation exercise, not an enterprise rollout readiness pilot. That distinction protects credibility and prevents false claims at closeout.
prevents false claims at closeout.
validation exercise, not an enterprise rollout readiness pilot. That distinction protects credibility and prevents false claims at closeout.
