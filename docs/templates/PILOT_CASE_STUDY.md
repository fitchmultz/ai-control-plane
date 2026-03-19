# Pilot Case Study Template

Use this template to produce a sanitized, buyer-facing case study after the acceptance memo and measurable outcomes scorecard are current.

This is not generic marketing copy. It is a controlled narrative built from real pilot evidence.

## 1. Customer Snapshot

| Field | Value |
| --- | --- |
| Customer alias | [ANONYMIZED_CUSTOMER] |
| Industry | [INDUSTRY] |
| Pilot name | [PILOT_NAME] |
| Review date | [REVIEW_DATE] |
| Closeout decision | [EXPAND / REMEDIATE_AND_CONTINUE / NO_GO] |

## 2. Problem Statement

Describe the business and governance problem the pilot was intended to prove.

Example prompts:
- what AI workflow needed governance?
- why was routed enforcement valuable?
- which enterprise teams needed evidence before rollout?

## 3. Scoped Pilot Boundary

State the exact boundary that was in scope.

Include:
- routed path covered
- user cohort or workflow covered
- validated environment shape
- what was explicitly out of scope

## 4. What Was Proven

- [PROVEN_OUTCOME]
- [PROVEN_OUTCOME]
- [PROVEN_OUTCOME]

Keep these statements aligned with the acceptance memo.

## 5. Measurable Outcomes Snapshot

Pull these from `PILOT_MEASURABLE_OUTCOMES_SCORECARD.md`.

| Outcome area | Result | Evidence reference |
| --- | --- | --- |
| Routed governance objective | [RESULT] | [EVIDENCE] |
| Attribution objective | [RESULT] | [EVIDENCE] |
| SIEM / evidence objective | [RESULT] | [EVIDENCE] |
| Customer-owned follow-up items | [RESULT] | [EVIDENCE] |
| Final closeout decision | [RESULT] | [EVIDENCE] |

## 6. What Remained Customer-Owned

List the remaining customer-owned controls or dependencies that were not transferred by the pilot.

- [CUSTOMER_OWNED_CONTROL]
- [CUSTOMER_OWNED_CONTROL]

## 7. Closeout Decision and Next Step

### Decision statement

[DECISION_SUMMARY]

### Conditions before expansion or re-entry

- [CONDITION]
- [CONDITION]

## 8. Evidence Packet References

- `make readiness-evidence`
- `make readiness-evidence-verify`
- `make pilot-closeout-bundle`
- `make pilot-closeout-bundle-verify`
- [PILOT_ACCEPTANCE_MEMO_REFERENCE]
- [PILOT_MEASURABLE_OUTCOMES_SCORECARD_REFERENCE]

## 9. Case Study Guardrails

- Keep the customer anonymous unless explicit publication approval exists.
- Preserve open gaps and ownership boundaries even for `EXPAND` outcomes.
- Do not claim enterprise-wide proof when the pilot covered only a scoped path.
- Keep every statement traceable to the acceptance memo or scorecard.
