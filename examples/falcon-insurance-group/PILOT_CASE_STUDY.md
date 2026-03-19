# Falcon Insurance Group Pilot Case Study

## 1. Customer Snapshot

| Field | Value |
| --- | --- |
| Customer alias | Falcon Insurance Group |
| Industry | Insurance |
| Pilot name | Claims Governance Pilot |
| Review date | 2026-05-20 |
| Closeout decision | EXPAND |

## 2. Problem Statement

Falcon wanted a decision-grade pilot for governed AI use in claims operations. The buyer needed proof that approved routed usage could be enforced, that usage could be attributed for review and showback, and that normalized evidence could reach the customer SIEM without overstating org-wide bypass prevention.

## 3. Scoped Pilot Boundary

The pilot covered the host-first routed gateway baseline for approved claims workflows and the named pilot cohort. It did not claim enterprise-wide blocking of direct SaaS usage; broader egress treatment and backup-retention policy remained customer-owned follow-up work.

## 4. What Was Proven

- Approved pilot traffic for the scoped claims path was routed through the gateway.
- Usage was attributable by user and department for pilot review.
- Normalized evidence reached the customer SIEM and supported detection review.

## 5. Measurable Outcomes Snapshot

| Outcome area | Result | Evidence reference |
| --- | --- | --- |
| Routed governance objective | 3 / 3 scoped routed-baseline objectives proven | `PILOT_MEASURABLE_OUTCOMES_SCORECARD.md` |
| Attribution objective | Verified for the pilot cohort | `PILOT_MEASURABLE_OUTCOMES_SCORECARD.md` |
| SIEM / evidence objective | Validated with the customer SIEM owner | `PILOT_MEASURABLE_OUTCOMES_SCORECARD.md` |
| Customer-owned follow-up items | 2 named follow-up items remained before broader expansion | `PILOT_MEASURABLE_OUTCOMES_SCORECARD.md` |
| Final closeout decision | `EXPAND` with conditions | `PILOT_ACCEPTANCE_MEMO.md` |

## 6. What Remained Customer-Owned

- Enterprise egress controls for direct SaaS AI endpoints
- Production backup-retention policy and restore expectations
- Ongoing ownership of customer network, SIEM retention, and infrastructure outside the delivered application

## 7. Closeout Decision and Next Step

### Decision statement

The pilot met its scoped objective for routed AI governance on the approved claims path and produced an explicit `EXPAND` decision.

### Conditions before expansion or re-entry

- Finalize customer-owned egress controls beyond the pilot cohort.
- Confirm backup, restore, and change-window ownership for the broader rollout.

## 8. Evidence Packet References

- `make readiness-evidence`
- `make readiness-evidence-verify`
- `make pilot-closeout-bundle`
- `make pilot-closeout-bundle-verify`
- `PILOT_ACCEPTANCE_MEMO.md`
- `PILOT_MEASURABLE_OUTCOMES_SCORECARD.md`

## 9. Case Study Guardrails

This case study preserves the routed-baseline claim boundary. It does not claim enterprise-wide bypass prevention, customer-owned infrastructure ownership transfer, or universal production readiness beyond the scoped pilot path.
