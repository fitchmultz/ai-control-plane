# Pilot Measurable Outcomes Scorecard Template

Use this template to translate pilot evidence into quantified, decision-grade outcomes.

This scorecard should be completed before the acceptance memo and the case study.

## 1. Pilot Summary

| Field | Value |
| --- | --- |
| Customer | [CUSTOMER_OR_ALIAS] |
| Pilot name | [PILOT_NAME] |
| Review date | [REVIEW_DATE] |
| Decision candidate | [EXPAND / REMEDIATE_AND_CONTINUE / NO_GO] |
| Delivery lead | [DELIVERY_LEAD] |

## 2. Outcome Scorecard

| Outcome area | Measurement method | Target | Observed result | Status | Evidence |
| --- | --- | --- | --- | --- | --- |
| Routed governance proof | Count of scoped routed-baseline objectives proven | [TARGET] | [RESULT] | [MET / PARTIAL / NOT_MET] | [EVIDENCE] |
| Attribution coverage | Count or percentage of scoped users/teams/services with validated attribution | [TARGET] | [RESULT] | [MET / PARTIAL / NOT_MET] | [EVIDENCE] |
| SIEM / evidence path | Count of required normalized evidence checks completed | [TARGET] | [RESULT] | [MET / PARTIAL / NOT_MET] | [EVIDENCE] |
| Customer-owned control validation | Count of critical customer control domains validated in writing | [TARGET] | [RESULT] | [MET / PARTIAL / NOT_MET] | [EVIDENCE] |
| Operator closeout readiness | Count of required closeout commands or checkpoints completed | [TARGET] | [RESULT] | [MET / PARTIAL / NOT_MET] | [EVIDENCE] |
| Open follow-up items | Count of unresolved follow-up items carried into the decision | [TARGET] | [RESULT] | [MET / PARTIAL / NOT_MET] | [EVIDENCE] |

## 3. Interpretation Rules

- Use counts, percentages, or explicit yes/no thresholds that can be traced to evidence.
- Prefer scoped pilot metrics over enterprise-wide vanity metrics.
- If the pilot proved only part of a category, mark it `PARTIAL` and name the missing owner or dependency.
- If a metric depends on customer systems outside the validated repo boundary, say so explicitly.

## 4. Open Variances and Owners

| Variance | Why it matters | Owner | Required next step |
| --- | --- | --- | --- |
| [VARIANCE] | [IMPACT] | [OWNER] | [ACTION] |
| [VARIANCE] | [IMPACT] | [OWNER] | [ACTION] |

## 5. Final Decision Mapping

Use this section to tie the scorecard to the acceptance memo.

- **If most scoped proof categories are `MET` and remaining gaps are customer-owned and bounded:** `EXPAND`
- **If routed proof is credible but one or more critical customer validations remain incomplete:** `REMEDIATE_AND_CONTINUE`
- **If required proof is missing or ownership remains ambiguous:** `NO_GO`
