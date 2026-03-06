# Statement of Work: Managed AI Security Operations

Use this template when the customer is buying ongoing operation of the delivered AI Control Plane application layer.

This is an operating document, not a legal appendix. Pricing, indemnities, and procurement language should live in the customer paper.

---

## 1. Engagement Summary

| Field | Value |
|---|---|
| Customer | [CUSTOMER] |
| Service term | [START_DATE] to [END_DATE] |
| Effective date | [EFFECTIVE_DATE] |
| Service tier | [ESSENTIAL / PROFESSIONAL / ENTERPRISE] |
| Coverage window | [BUSINESS_HOURS / EXTENDED_HOURS / 24X7] |
| Delivery lead | [DELIVERY_LEAD] |
| Customer security lead | [CUSTOMER_SECURITY_LEAD] |
| Customer platform lead | [CUSTOMER_PLATFORM_LEAD] |

## 2. Service Objective

The objective of this service is to operate the AI Control Plane safely after deployment by providing:

- alert monitoring and triage
- application-incident response
- detection and policy tuning
- recurring governance and chargeback reporting
- a clear escalation path when issues sit in customer-controlled systems

## 3. Preconditions

This managed service should not start until all of the following are true:

- the AI Control Plane baseline is deployed and operational
- named customer contacts exist for security, platform, SIEM, and finance
- the customer has approved the shared responsibility model
- the communications path for incidents and approvals is defined
- transcript handling, if any, is explicitly scoped

If these conditions are not met, sell implementation or stabilization work first, not managed operations.

## 3.1 Operating Boundary

This service operates the AI Control Plane application layer and the agreed governance workflow around it. It does not transfer ownership of customer-controlled cloud, network, IAM, workspace, or SIEM dependencies.

Use this template with:

- `docs/SHARED_RESPONSIBILITY_MODEL.md`
- `docs/SERVICE_OFFERINGS.md`
- `docs/PILOT_EXECUTION_MODEL.md`

## 3.2 Hard-Stop Conditions

Do not sell or start managed operations if any of the following are unresolved:

- no named customer owner exists for platform, network, security, SIEM, and workspace administration
- the incident path for customer-owned dependencies is not documented
- transcript handling scope is ambiguous
- the customer expects provider ownership of infrastructure, IdP, vendor licensing, or egress enforcement without explicit additional scope
- the baseline platform is not yet stable enough for steady-state operations

## 4. In-Scope Service

### 4.1 Core Operating Activities

- Monitor gateway health, alert streams, and policy exceptions during the agreed coverage window
- Triage alerts and classify them as true positive, false positive, or informational
- Execute agreed runbooks for application-layer incidents
- Tune detections and policy thresholds under agreed change control
- Produce recurring governance, spend, and incident reporting
- Coordinate handoff when the root cause is in customer-owned infrastructure, IAM, network, SIEM, or workspace controls

### 4.2 Optional Add-Ons

- Guardrail lifecycle tuning
- Additional reporting packs
- Expanded provider/model onboarding support
- Quarterly control-review workshops

## 5. Out of Scope

- Customer cloud infrastructure administration
- Firewall, SWG, CASB, DNS, or proxy enforcement changes
- IdP ownership, MFA policy, and user-lifecycle administration
- Vendor contract, billing, or seat-management disputes
- Legal, compliance, or audit opinions
- End-user help desk support unless explicitly added
- Customer procurement approvals, vendor-seat administration, or workspace commercial administration unless explicitly added
- Any claim that the provider can independently prevent bypass without customer-operated controls

## 6. Response Model

### 6.1 Severity Model

| Severity | Meaning | Typical example |
|---|---|---|
| Critical | Active security event or major service outage | Suspected credential compromise, gateway unavailable |
| High | Material policy breach or significant degradation | Unapproved usage pattern, sustained budget overrun |
| Medium | Operational issue requiring planned response | Detection drift, repeated false positives |
| Low | Informational or optimization item | Reporting clarification, backlog tuning item |

### 6.2 Standard Handling Model

| Area | Provider action | Customer action |
|---|---|---|
| Alert triage | Review, classify, investigate, document | Be available for escalations and approvals |
| Application incident | Diagnose and remediate within delivered platform scope | Approve change windows and provide dependency access |
| Network/IdP/workspace issue | Escalate with evidence and recommended action | Own root-cause remediation in customer systems |
| Governance reporting | Deliver reports and recommendations | Review decisions and approve follow-up actions |
| Major incident | Run agreed escalation path | Own business-risk decisions and external communications |

### 6.3 Response Commitments

Document actual response targets in the commercial schedule or service exhibit, not in narrative prose here.

| Severity | Acknowledgment target | Update cadence |
|---|---|---|
| Critical | [ACK_CRITICAL] | [UPDATE_CRITICAL] |
| High | [ACK_HIGH] | [UPDATE_HIGH] |
| Medium | [ACK_MEDIUM] | [UPDATE_MEDIUM] |
| Low | [ACK_LOW] | Next scheduled review unless otherwise agreed |

## 7. Deliverables

| Deliverable | Cadence | Purpose |
|---|---|---|
| Alert triage record | Continuous | Documents investigated alerts and outcomes |
| Incident report | Per qualifying incident | Records impact, response, and remediation |
| Governance report | Monthly | Summarizes usage, policy outcomes, and recommended actions |
| Chargeback/showback report | Monthly if scoped | Provides cost allocation and exceptions |
| Business review | Quarterly | Reviews trends, risks, and next actions |
| Change log | Ongoing | Records policy and detection changes |

## 8. Ownership Boundary

This service operates the application layer. It does not transfer ownership of customer-controlled dependencies.

| Domain | Customer owns | Provider owns |
|---|---|---|
| Cloud and infrastructure | Account, tenancy, host lifecycle, backup platform, platform patching unless separately scoped | Application guidance only |
| Network and endpoint controls | Egress policy, firewall, SWG/CASB, DNS, managed-device controls | Required endpoint inventory and escalation evidence |
| IAM and workspace administration | IdP, MFA, user lifecycle, vendor workspace settings | Supported claim mapping and application-side role behavior |
| AI Control Plane application | Change approvals and business policy decisions | Defects, releases, runbooks, application-layer operations in scope |
| SIEM and downstream response | Retention, routing, case management, analyst workflow | Evidence feed, mappings, detections, and operational recommendations |

Reference: `docs/SHARED_RESPONSIBILITY_MODEL.md`

The commercial paper should attach service levels and pricing, but this template should remain explicit about the control boundary even when commercial language changes.

## 9. Customer Responsibilities

- Maintain named primary and backup contacts
- Maintain the agreed communications and escalation channel
- Approve material policy changes and emergency actions
- Provide timely access to required logs, dashboards, and support contacts
- Own remediation in infrastructure or enterprise systems outside provider control

## 10. Operating Rhythm

| Meeting | Frequency | Purpose |
|---|---|---|
| Service review | Monthly | Review incidents, open actions, reporting, and changes |
| Technical review | Monthly or bi-weekly | Review tuning items and dependency risks |
| Business review | Quarterly | Review value, renewal posture, and control maturity |

## 11. Success Measures

Use a small set of measurable service outcomes:

- agreed reports delivered on schedule
- qualifying incidents acknowledged within the contracted target
- high-severity issues have a documented owner and next action
- policy and detection changes are tracked and approved
- customer dependencies blocking service quality are documented explicitly

Success under this SOW means the provider operated the agreed application layer responsibly. It does not mean the customer environment is universally governed or that customer-owned dependencies were remediated by the provider.

## 12. Sign-Off

| Role | Name | Notes |
|---|---|---|
| Customer security lead | [CUSTOMER_SECURITY_LEAD] | |
| Customer platform lead | [CUSTOMER_PLATFORM_LEAD] | |
| Delivery lead | [DELIVERY_LEAD] | |
| Commercial owner | [COMMERCIAL_OWNER] | |
