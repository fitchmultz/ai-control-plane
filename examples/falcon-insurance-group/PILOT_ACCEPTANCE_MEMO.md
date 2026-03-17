# Falcon Insurance Group Pilot Acceptance Memo

## 1. Decision Summary

| Field | Value |
| --- | --- |
| Customer | Falcon Insurance Group |
| Pilot name | Claims Governance Pilot |
| Review date | 2026-05-20 |
| Decision | EXPAND |
| Sponsor | VP of Claims Technology |
| Delivery lead | AI Control Plane delivery lead |

## 2. What Was Proven

- Approved AI traffic for the pilot cohort was routed through the gateway.
- Usage could be attributed by user and department for showback.
- Normalized evidence reached the customer SIEM and supported detection review.

## 3. Open Gaps

| Gap | Impact | Owner | Next action |
| --- | --- | --- | --- |
| Customer egress controls not yet expanded org-wide | Bypass prevention remains scoped to the pilot cohort | Network security lead | Complete enterprise egress change plan |
| Production backup retention policy still customer-owned | Expansion gate depends on documented restore expectations | Platform ops | Approve retention and drill cadence |

## 4. Customer-Owned Controls Confirmed

The customer confirmed ownership for network egress, IdP and enterprise access policy, SIEM retention and response workflow, workspace administration, and underlying infrastructure outside the delivered application.

## 5. Evidence Reviewed

- `make readiness-evidence`
- `make readiness-evidence-verify`
- `make pilot-closeout-bundle`
- `make pilot-closeout-bundle-verify`
- `make validate-detections`
- `make validate-siem-schema`

## 6. Recommendation

### Recommended next step

Proceed to a production-scoped host-first rollout for the approved claims workflows.

### Conditions before expansion

- Finalize customer-owned egress controls.
- Confirm backup, restore, and change-window ownership.

## 7. Sign-Off

| Role | Name | Decision / Notes |
| --- | --- | --- |
| Customer sponsor | VP of Claims Technology | Approved |
| Network owner | Network security lead | Approved with egress follow-up |
| SIEM owner | SIEM engineering lead | Approved |
| Delivery lead | AI Control Plane delivery lead | Approved |
