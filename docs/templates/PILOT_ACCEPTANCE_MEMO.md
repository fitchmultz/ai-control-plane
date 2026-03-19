# Pilot Acceptance Memo Template

Use this memo at pilot completion or checkpoint review to document what was proven, what remains customer-owned, and the next-step decision.

---

## 1. Decision Summary

| Field | Value |
|---|---|
| Customer | [CUSTOMER] |
| Pilot name | [PILOT_NAME] |
| Review date | [REVIEW_DATE] |
| Decision | [EXPAND / REMEDIATE_AND_CONTINUE / NO_GO] |
| Sponsor | [CUSTOMER_SPONSOR] |
| Delivery lead | [DELIVERY_LEAD] |

## 2. What Was Proven

- [PROVEN_OUTCOME]
- [PROVEN_OUTCOME]
- [PROVEN_OUTCOME]

## 3. Open Gaps

| Gap | Impact | Owner | Next action |
|---|---|---|---|
| [GAP] | [IMPACT] | [OWNER] | [ACTION] |
| [GAP] | [IMPACT] | [OWNER] | [ACTION] |

## 4. Customer-Owned Controls Confirmed

The customer confirmed ownership for:

- network egress and bypass prevention
- IdP and enterprise access policy
- SIEM retention and downstream response workflow
- workspace administration and vendor settings
- underlying cloud or host infrastructure outside the delivered application

Add any scoped exceptions here:
- [SCOPED_EXCEPTION]

## 5. Evidence Reviewed

- `make readiness-evidence`
- `make readiness-evidence-verify`
- `make pilot-closeout-bundle`
- `make pilot-closeout-bundle-verify`
- `make validate-detections`
- `make validate-siem-schema`
- [PILOT_CLOSEOUT_BUNDLE_REFERENCE]
- [PILOT_MEASURABLE_OUTCOMES_SCORECARD_REFERENCE]
- [PILOT_CASE_STUDY_REFERENCE]
- [CUSTOMER_EVIDENCE_REFERENCE]

## 6. Recommendation

### Recommended next step

[RECOMMENDATION]

### Conditions before expansion

- [CONDITION]
- [CONDITION]

## 7. Sign-Off

| Role | Name | Decision / Notes |
|---|---|---|
| Customer sponsor | [CUSTOMER_SPONSOR] | |
| Network owner | [NETWORK_OWNER] | |
| SIEM owner | [SIEM_OWNER] | |
| Delivery lead | [DELIVERY_LEAD] | |
