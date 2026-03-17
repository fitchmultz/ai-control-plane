# Falcon Insurance Group Pilot Charter

## 1. Pilot Summary

| Field | Value |
| --- | --- |
| Customer | Falcon Insurance Group |
| Pilot name | Claims Governance Pilot |
| Pilot sponsor | VP of Claims Technology |
| Delivery lead | AI Control Plane delivery lead |
| Pilot start | 2026-04-01 |
| Pilot end | 2026-05-15 |
| Decision date | 2026-05-20 |

## 2. Pilot Objective

This pilot exists to prove the following in the customer environment:

1. Approved AI traffic is routed through the gateway.
2. Usage is attributable by user and cost center.
3. Normalized evidence reaches the customer SIEM.

## 3. In Scope

- Host-first Linux deployment for a claims pilot cohort
- Gateway-routed API and managed browser workflows
- Detection, reporting, and pilot closeout evidence

## 4. Out of Scope

- Blocking every possible bypass path without customer network controls
- Compliance certification or auditor attestation
- Multi-tenant managed-service claims
- Production SLA commitments outside the signed SOW

## 5. Customer Prerequisites

| Workstream | Customer owner | Required before kickoff |
| --- | --- | --- |
| Network | Network security lead | Confirm egress and approved endpoints |
| IAM | IAM architect | Confirm pilot cohort and admin groups |
| SIEM | SIEM engineering lead | Confirm ingestion path and alert routing |
| FinOps | Finance systems lead | Confirm cost-center mapping |
| Workspace admin | Collaboration lead | Confirm sanctioned workspace settings |
| Platform ops | Linux operations lead | Confirm target host and change window |

## 6. Success Criteria

| Success criterion | Evidence | Owner |
| --- | --- | --- |
| Gateway enforcement works for scoped traffic | Readiness evidence run | Delivery team |
| Attribution is accurate enough for showback | Chargeback report and validation review | Delivery team + customer |
| SIEM receives normalized pilot evidence | SIEM dashboard and alert test | Customer SIEM owner |
| Enforce-vs-detect boundary is documented | Signed boundary review | Sponsor + network owner |
| Operators can run the agreed command set | Runbook walkthrough | Delivery team + customer ops |

## 7. Exit Decision

At pilot close, the sponsor and delivery lead will select one outcome:

- Expand to production rollout
- Remediate and continue pilot
- Stop with documented no-go reasons
