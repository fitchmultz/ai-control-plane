# AI Control Plane Template Kit

Use this directory to assemble concise, enterprise-facing pilot and service documents.

These templates are working deal-delivery tools. They are not legal boilerplate, pricing schedules, or implementation runbooks.

## What Belongs Here

- Pilot documents that define scope, owners, and acceptance
- SOW outlines that define service shape, deliverables, and operating boundaries
- Reporting templates that help the customer understand what they are buying

## Template Index

| Template | Use When | Outcome |
|---|---|---|
| `PILOT_CHARTER.md` | Starting a pilot | States objective, scope, owners, prerequisites, and success criteria |
| `PILOT_ACCEPTANCE_MEMO.md` | Closing a pilot or checkpoint | Records what was proven, what remains customer-owned, and the next-step decision |
| `PILOT_OPERATOR_HANDOFF_CHECKLIST.md` | Handing off day-to-day pilot operations | Confirms commands, contacts, evidence flow, and incident expectations |
| `SOW_AI_USAGE_EXPOSURE_ASSESSMENT.md` | Discovery engagement | Produces current-state assessment and roadmap |
| `SOW_AI_CONTROL_PLANE_IMPLEMENTATION.md` | Build and deploy engagement | Delivers the gateway baseline, policy controls, and pilot setup |
| `SOW_MANAGED_AI_SECURITY_OPERATIONS.md` | Ongoing service | Defines operating coverage, response model, and shared responsibility |
| `SOW_VENDOR_WORKSPACE_GOVERNANCE.md` | Workspace governance scope | Defines vendor-workspace controls and admin operating model |
| `FINANCIAL_SHOWBACK_CHARGEBACK_REPORT.md` | FinOps reporting | Produces a reusable showback/chargeback report format |

## How To Use This Directory

1. Start with the pilot templates if the customer is not yet in steady-state operations.
2. Pair the implementation or managed-service SOW with the ownership matrix and shared responsibility model.
3. Replace placeholders with customer names, dates, contacts, and service commitments.
4. Delete sections that are not in scope instead of leaving aspirational language in place.
5. Attach pricing, legal terms, and procurement language outside these templates.
6. When closeout evidence matters, build a dated artifact set with `make pilot-closeout-bundle`.
7. Compare your filled-in pilot docs with the sanitized example set under `examples/falcon-insurance-group/`.

## Required Companion Docs

Use these documents with the templates instead of duplicating their content:

- `docs/ENTERPRISE_PILOT_PACKAGE.md`
- `docs/PILOT_SPONSOR_ONE_PAGER.md`
- `docs/PILOT_CONTROL_OWNERSHIP_MATRIX.md`
- `docs/SHARED_RESPONSIBILITY_MODEL.md`
- `docs/SERVICE_OFFERINGS.md`
- `docs/ENTERPRISE_BUYER_OBJECTIONS.md`
- `docs/PILOT_CLOSEOUT_EXAMPLES.md`

## Writing Standard

Keep every document:

- explicit about what is proven vs customer-validated
- clear about who owns cloud, network, IAM, SIEM, and workspace controls
- short enough to survive procurement and sponsor review
- free of legal filler, promises that are not staffed, or platform claims that are not validated
