# Pilot Closeout Examples

Use these examples to keep closeout language crisp and decision-grade.

They are not marketing copy. They are examples of what a real pilot closeout should sound like when the evidence is mixed, constrained, or strong.

## Example 1: Expand

**Decision:** `EXPAND`

**Why this is credible**

- Routed gateway enforcement was demonstrated for the scoped pilot traffic.
- Attribution to user and team was verified in the pilot evidence path.
- SIEM ingestion and alert routing were validated with the customer SOC owner.
- Customer stakeholders agreed the remaining gaps were outside the routed baseline and had named owners.

**Suggested memo language**

> The pilot met its primary objective for routed AI governance on the agreed Linux-host baseline. The customer validated SIEM ingestion, attribution, and operating ownership for the scoped user cohort. Remaining gaps are limited to customer-owned network and workspace controls outside the routed baseline and do not block expansion of the controlled deployment path.

**Conditions before expansion**

- Finalize customer network treatment for direct SaaS AI endpoints
- Keep readiness evidence fresh before each rollout checkpoint
- Reconfirm named owners for FinOps and workspace administration

## Example 2: Remediate And Continue

**Decision:** `REMEDIATE_AND_CONTINUE`

**Why this is credible**

- The product baseline worked, but one or more customer-owned dependencies were not validated.
- The gap is real, bounded, and fixable.
- The next step is more validation, not a forced yes or a vague maybe.

**Suggested memo language**

> The pilot proved the application-layer baseline, including routed enforcement, operator workflow, and normalized evidence generation. The customer environment did not yet complete the required validation for bypass controls and workspace governance. The correct decision is to continue the pilot under a remediation plan rather than overstate rollout readiness.

**Common reasons to use this**

- SIEM ingestion exists but alert routing is not final
- Network owners have not completed direct SaaS AI endpoint testing
- Workspace admin settings are still under review
- Cost-center mapping exists for some but not all pilot users

## Example 3: No-Go

**Decision:** `NO_GO`

**Why this is credible**

- The pilot did not satisfy the minimum conditions for a safe or honest next step.
- The blocker is material, not cosmetic.
- The closeout explains whether the issue is product-side, customer-side, or a mismatch in pilot intent.

**Suggested memo language**

> The pilot should not proceed to rollout. The routed baseline was not enough to offset unresolved customer-environment control gaps, specifically around direct SaaS AI usage and ownership of the downstream response path. Closing the pilot as a no-go is the correct outcome because the remaining issues are material to governance credibility and safe operations.

**Common reasons to use this**

- Sponsor and control owners never aligned on enforce-vs-detect boundaries
- Customer could not support the required IAM, SIEM, or network validation
- Pilot objective depended on controls the repository does not currently prove
- The customer wanted claims stronger than the evidence supported

## Closeout Writing Rules

Use these rules in every pilot closeout:

- state what was actually proven
- state what remained customer-owned
- state whether the gap is product, process, or customer-environment validation
- make the next decision explicit: expand, remediate and continue, or no-go
- link to evidence instead of retelling every command run

Avoid:

- “successful pilot” without boundaries
- “enterprise-ready” without naming the environment
- vague phrases like “further collaboration required”
- burying customer-owned gaps in an appendix
