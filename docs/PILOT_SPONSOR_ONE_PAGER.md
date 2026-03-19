# Pilot Sponsor One-Pager

Use this document with executive sponsors and procurement stakeholders before pilot kickoff and again at closeout.

Its purpose is to keep the pilot honest, short, and decision-grade.

## What This Pilot Can Prove

A credible pilot from this repository can prove:

- governed AI traffic routes through the approved gateway for the scoped user path
- approved models, budgets, attribution, and detections work on that routed path
- evidence is generated consistently enough to support customer review and closeout
- customer-owned controls required for bypass treatment are named and validated in writing

## What This Pilot Does Not Prove By Itself

This pilot does not, by itself, prove:

- enterprise-wide bypass prevention
- compliance certification or auditor sign-off
- unmanaged-device governance
- customer network, IAM, SIEM, or workspace ownership transfer to the provider
- production readiness outside the scoped path and named environment

## Customer Prerequisites

Do not start a customer pilot until all of the following exist:

- named sponsor
- named platform owner
- named security owner
- named network owner
- named SIEM owner
- named workspace and identity owners when browser/workspace governance is in scope
- written agreement on whether bypass prevention is enforced or detective-only for this pilot
- written agreement on what remains customer-owned

## Hard Stops

Pause, reframe, or stop the pilot if any of the following are true:

- no named owner exists for a critical customer-controlled dependency
- the customer expects the provider to own cloud, IdP, workspace licensing, or network enforcement without explicit added scope
- closeout evidence cannot be produced for the scoped path
- bypass status is still ambiguous at sponsor review time
- the team is using “configured” or “deployed” as a substitute for “validated”

## Allowed Closeout Decisions

Every pilot must end in one of these decisions:

- `EXPAND`
- `REMEDIATE_AND_CONTINUE`
- `NO_GO`

`Successful pilot` is not a sufficient closeout statement on its own.

## Minimum Evidence Set

The sponsor packet should point to:

- `make readiness-evidence`
- `make pilot-closeout-bundle`
- `docs/PILOT_CUSTOMER_VALIDATION_CHECKLIST.md`
- `docs/templates/PILOT_MEASURABLE_OUTCOMES_SCORECARD.md`
- `docs/templates/PILOT_ACCEPTANCE_MEMO.md`
- `docs/templates/PILOT_CASE_STUDY.md`

## Sponsor Review Questions

Before approving expansion, the sponsor should be able to answer:

1. What did we actually prove in our environment?
2. What still depends on customer-owned controls?
3. What is enforced today versus only detected?
4. What open risks are we accepting if we expand?
5. Who owns those risks after pilot close?

## Companion Documents

- `docs/ENTERPRISE_PILOT_PACKAGE.md`
- `docs/PILOT_EXECUTION_MODEL.md`
- `docs/PILOT_CONTROL_OWNERSHIP_MATRIX.md`
- `docs/SHARED_RESPONSIBILITY_MODEL.md`
