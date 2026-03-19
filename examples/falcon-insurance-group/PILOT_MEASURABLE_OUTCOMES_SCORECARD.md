# Falcon Insurance Group Pilot Measurable Outcomes Scorecard

## 1. Pilot Summary

| Field | Value |
| --- | --- |
| Customer | Falcon Insurance Group |
| Pilot name | Claims Governance Pilot |
| Review date | 2026-05-20 |
| Decision candidate | EXPAND |
| Delivery lead | AI Control Plane delivery lead |

## 2. Outcome Scorecard

| Outcome area | Measurement method | Target | Observed result | Status | Evidence |
| --- | --- | --- | --- | --- | --- |
| Routed governance proof | Count of scoped routed-baseline objectives proven | 3 core routed objectives | 3 / 3 proven | MET | `PILOT_ACCEPTANCE_MEMO.md` §2 |
| Attribution coverage | Count of scoped attribution objectives proven | User and department attribution for pilot cohort | Verified for the pilot cohort | MET | `PILOT_ACCEPTANCE_MEMO.md` §2 |
| SIEM / evidence path | Count of required normalized evidence checks completed | SIEM ingestion plus detection review path | Validated with customer SIEM owner | MET | `PILOT_ACCEPTANCE_MEMO.md` §§2,5 |
| Customer-owned control validation | Count of critical customer control domains validated in writing | 5 named domains | 5 / 5 ownership areas confirmed | MET | `PILOT_ACCEPTANCE_MEMO.md` §4 |
| Operator closeout readiness | Count of required closeout commands reviewed | 6 required commands | 6 / 6 reviewed | MET | `PILOT_ACCEPTANCE_MEMO.md` §5 |
| Open follow-up items | Count of unresolved follow-up items carried into the decision | 0 unowned gaps | 2 follow-up items, both named and customer-owned | PARTIAL | `PILOT_ACCEPTANCE_MEMO.md` §3 |

## 3. Open Variances and Owners

| Variance | Why it matters | Owner | Required next step |
| --- | --- | --- | --- |
| Customer egress controls not yet expanded org-wide | Bypass prevention remains scoped to the pilot cohort | Network security lead | Complete the enterprise egress change plan |
| Production backup retention policy still customer-owned | Expansion gate depends on documented restore expectations | Platform ops | Approve retention and drill cadence |

## 4. Final Decision Mapping

- Routed-baseline proof categories were met.
- Remaining gaps are bounded, named, and customer-owned.
- The correct closeout decision is `EXPAND` with explicit follow-up conditions.
