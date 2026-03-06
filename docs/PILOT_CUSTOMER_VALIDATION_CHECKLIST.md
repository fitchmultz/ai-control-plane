# Pilot Customer Validation Checklist

Use this checklist to keep a customer pilot honest.

The AI Control Plane repository can prove the local baseline. This checklist captures what the customer must validate in their own environment before anyone calls the pilot rollout-ready.

## How To Use It

1. Review this checklist during pilot kickoff.
2. Assign a named customer owner to each section.
3. Mark each item with one of: `Not Started`, `In Progress`, `Validated`, or `Accepted Risk`.
4. Attach evidence references instead of relying on meeting memory.
5. Reuse the results in the pilot acceptance memo.

## Network / Egress Controls

Owner: `[NETWORK_OWNER]`

| Validation item | Status | Evidence |
|---|---|---|
| Approved AI endpoints for routed traffic are documented | [STATUS] | [EVIDENCE] |
| Firewall, SWG, CASB, proxy, or DNS controls for unapproved AI endpoints are identified | [STATUS] | [EVIDENCE] |
| Customer has decided whether bypass prevention is enforcement or detective-only for this pilot | [STATUS] | [EVIDENCE] |
| Direct SaaS AI traffic test was performed and outcome recorded | [STATUS] | [EVIDENCE] |
| Escalation path for bypass findings is agreed | [STATUS] | [EVIDENCE] |

## Identity / IAM

Owner: `[IAM_OWNER]`

| Validation item | Status | Evidence |
|---|---|---|
| Pilot user cohort is defined | [STATUS] | [EVIDENCE] |
| IdP or workspace identity mapping for routed usage is tested | [STATUS] | [EVIDENCE] |
| Required MFA and device posture policy is confirmed | [STATUS] | [EVIDENCE] |
| Joiner/mover/leaver ownership is documented | [STATUS] | [EVIDENCE] |
| Privileged admin access path is documented | [STATUS] | [EVIDENCE] |

## SIEM / SOC

Owner: `[SIEM_OWNER]`

| Validation item | Status | Evidence |
|---|---|---|
| Gateway evidence reaches the customer SIEM | [STATUS] | [EVIDENCE] |
| Detection mappings are validated against the pilot config | [STATUS] | [EVIDENCE] |
| Alert routing destination is configured | [STATUS] | [EVIDENCE] |
| Investigation owner for pilot alerts is named | [STATUS] | [EVIDENCE] |
| Retention and case-management expectations are documented | [STATUS] | [EVIDENCE] |

## FinOps / Chargeback

Owner: `[FINOPS_OWNER]`

| Validation item | Status | Evidence |
|---|---|---|
| Cost-center or team mapping exists for the pilot cohort | [STATUS] | [EVIDENCE] |
| Report consumers are named | [STATUS] | [EVIDENCE] |
| Exceptions path for unattributed usage is documented | [STATUS] | [EVIDENCE] |
| Showback or chargeback review cadence is agreed | [STATUS] | [EVIDENCE] |

## Workspace / Browser Governance

Owner: `[WORKSPACE_OWNER]`

| Validation item | Status | Evidence |
|---|---|---|
| Workspace admin owner is named | [STATUS] | [EVIDENCE] |
| Approved browser or chat entrypoint is documented | [STATUS] | [EVIDENCE] |
| Sanctioned model exposure is reviewed | [STATUS] | [EVIDENCE] |
| Direct SaaS/browser usage detection path is documented | [STATUS] | [EVIDENCE] |
| Data-retention and audit-export settings are reviewed | [STATUS] | [EVIDENCE] |

## Platform Operations

Owner: `[PLATFORM_OWNER]`

| Validation item | Status | Evidence |
|---|---|---|
| Target host or environment is defined | [STATUS] | [EVIDENCE] |
| Change window is approved | [STATUS] | [EVIDENCE] |
| Backup and restore expectations are documented | [STATUS] | [EVIDENCE] |
| Named operator is available for checkpoint reviews | [STATUS] | [EVIDENCE] |
| Release evidence review cadence is agreed | [STATUS] | [EVIDENCE] |

## Minimum Exit Rule

Do not call the pilot rollout-ready until:

- all critical customer-owned items are either `Validated` or explicitly marked `Accepted Risk`
- the status of bypass prevention is written down clearly
- the pilot acceptance memo references the evidence used for these decisions

Reference documents:

- `docs/ENTERPRISE_PILOT_PACKAGE.md`
- `docs/PILOT_CONTROL_OWNERSHIP_MATRIX.md`
- `docs/templates/PILOT_ACCEPTANCE_MEMO.md`
- `docs/BROWSER_WORKSPACE_PROOF_TRACK.md`
