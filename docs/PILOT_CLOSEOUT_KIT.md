# Pilot Closeout Kit

Use this document to assemble a reusable, anonymized, evidence-backed pilot closeout packet.

This kit is meant to turn current pilot proof into buyer-safe external material without hiding ownership boundaries, open gaps, or support limits.

## Intended audiences

- **Delivery lead** — assembles the closeout packet from current evidence
- **Account team / solutions lead** — reuses the sanitized case study and outcome scorecard in buyer conversations
- **Customer sponsor** — reviews the decision packet and next-step conditions

## Kit contents

| Artifact | Purpose | Canonical location |
| --- | --- | --- |
| Pilot charter | Locks scope, owners, and success criteria | `docs/templates/PILOT_CHARTER.md` |
| Customer validation checklist | Tracks customer-owned validation and evidence | `docs/PILOT_CUSTOMER_VALIDATION_CHECKLIST.md` |
| Measurable outcomes scorecard | Turns pilot evidence into quantified closeout statements | `docs/templates/PILOT_MEASURABLE_OUTCOMES_SCORECARD.md` |
| Acceptance memo | Records what was proven, what remains customer-owned, and the decision | `docs/templates/PILOT_ACCEPTANCE_MEMO.md` |
| Anonymized case study | Buyer-facing narrative derived from the memo and scorecard | `docs/templates/PILOT_CASE_STUDY.md` |
| Operator handoff checklist | Confirms who owns day-2 execution after closeout | `docs/templates/PILOT_OPERATOR_HANDOFF_CHECKLIST.md` |
| Generated closeout bundle | Archives the dated source packet and readiness evidence | `make pilot-closeout-bundle` |

## Assembly workflow

1. **Start from the live decision record**
   - confirm the pilot still matches the scope in the charter
   - confirm customer-owned controls are current in the validation checklist
2. **Complete the measurable outcomes scorecard**
   - convert pilot evidence into counts, thresholds, and decision-grade statements
   - avoid invented KPIs or customer-environment claims you cannot prove
3. **Write the acceptance memo**
   - state what was proven
   - state what remained customer-owned
   - record `EXPAND`, `REMEDIATE_AND_CONTINUE`, or `NO_GO`
4. **Write the anonymized case study**
   - derive it from the memo and scorecard
   - keep the customer anonymous unless explicit approval exists
   - preserve open gaps and boundary statements
5. **Build the dated evidence packet**
   - run `make pilot-closeout-bundle`
   - verify it with `make pilot-closeout-bundle-verify`

## Minimum external packet

A reusable external-proof packet should contain:

- anonymized case study
- measurable outcomes scorecard
- decision summary from the acceptance memo
- explicit customer-owned controls and open gaps
- current claim boundary references

## Minimum internal packet

The internal closeout packet should additionally contain:

- full acceptance memo
- current customer validation checklist
- operator handoff checklist
- readiness evidence and verification outputs
- generated pilot closeout bundle path

## Required commands

```bash
make readiness-evidence
make readiness-evidence-verify
make pilot-closeout-bundle
make pilot-closeout-bundle-verify
```

## Writing guardrails

- anonymize the customer unless release approval exists
- keep measurable outcomes tied to evidence, not sales targets
- do not convert detective-only coverage into enforcement language
- do not remove open gaps just because the overall decision is positive
- prefer explicit next-step conditions over vague success language

## Starter references

- [Enterprise Pilot Package](ENTERPRISE_PILOT_PACKAGE.md)
- [Pilot Execution Model](PILOT_EXECUTION_MODEL.md)
- [Pilot Closeout Examples](PILOT_CLOSEOUT_EXAMPLES.md)
- [Pilot Sponsor One-Pager](PILOT_SPONSOR_ONE_PAGER.md)
- [Shared Responsibility Model](SHARED_RESPONSIBILITY_MODEL.md)
- [Examples README](../examples/README.md)
