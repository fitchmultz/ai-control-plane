# Pilot Charter Template

Use this document to start an enterprise pilot with clear scope, owners, and exit criteria.

---

## 1. Pilot Summary

| Field | Value |
|---|---|
| Customer | [CUSTOMER] |
| Pilot name | [PILOT_NAME] |
| Pilot sponsor | [CUSTOMER_SPONSOR] |
| Delivery lead | [DELIVERY_LEAD] |
| Pilot start | [START_DATE] |
| Pilot end | [END_DATE] |
| Decision date | [DECISION_DATE] |

## 2. Pilot Objective

This pilot exists to prove the following in the customer environment:

1. [OBJECTIVE_1]
2. [OBJECTIVE_2]
3. [OBJECTIVE_3]

Example:
- approved AI traffic is routed through the gateway
- usage is attributable by person, team, or cost center
- normalized evidence reaches the customer SIEM

## 3. In Scope

- [IN_SCOPE_ITEM]
- [IN_SCOPE_ITEM]
- [IN_SCOPE_ITEM]

## 4. Out of Scope

- Blocking every possible bypass path without customer network controls
- Compliance certification or auditor attestation
- Custom software development not listed in this charter
- Production SLA commitments outside the signed SOW

## 5. Customer Prerequisites

| Workstream | Customer owner | Required before kickoff |
|---|---|---|
| Network | [NETWORK_OWNER] | Confirm egress pattern, firewall/SWG change path, approved endpoints |
| IAM | [IAM_OWNER] | Confirm IdP contact, group mapping, and pilot user list |
| SIEM | [SIEM_OWNER] | Confirm ingestion path, dashboard owner, and alert destination |
| FinOps | [FINOPS_OWNER] | Confirm cost-center model and report consumers |
| Workspace admin | [WORKSPACE_OWNER] | Confirm sanctioned workspace settings and admin access |
| Platform ops | [PLATFORM_OWNER] | Confirm target host/environment and change window |

## 6. Delivery Team Commitments

- Deploy and validate the agreed pilot baseline
- Configure the policy, attribution, and detection set in scope
- Provide current readiness evidence before major checkpoints
- Maintain runbooks and named escalation contacts
- State clearly where customer-owned controls are required

## 7. Success Criteria

| Success criterion | Evidence | Owner |
|---|---|---|
| Gateway enforcement works for scoped traffic | [EVIDENCE_REFERENCE] | Delivery team |
| Attribution is accurate enough for audit/showback | [EVIDENCE_REFERENCE] | Delivery team + customer |
| SIEM receives normalized pilot evidence | [EVIDENCE_REFERENCE] | Customer SIEM owner |
| Enforce-vs-detect boundary is documented | Signed boundary review | Customer sponsor + network owner |
| Operators can run the agreed command set | Runbook walkthrough | Delivery team + customer ops |

## 8. Governance Cadence

| Meeting | Frequency | Purpose |
|---|---|---|
| Working session | Weekly | Remove blockers and review validation progress |
| Technical checkpoint | Bi-weekly | Review evidence, detections, and open issues |
| Sponsor checkpoint | At midpoint and close | Confirm decision path and scope discipline |

## 9. Exit Decision

At pilot close, the sponsor and delivery lead will select one outcome:

- Expand to production rollout
- Remediate and continue pilot
- Stop with documented no-go reasons

Record the decision in `PILOT_ACCEPTANCE_MEMO.md`.
