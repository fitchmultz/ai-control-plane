# AI Control Plane — Service Offerings

Four reusable offering templates aligned to this reference implementation.

---

## Overview

| Offering | Shape | Outcome | Key Artifacts |
|----------|-------|---------|---------------|
| **AI Usage & Exposure Assessment** | Discovery | Inventory, risk analysis, roadmap | Assessment report, architecture, 90-day plan |
| **AI Control Plane Implementation** | Project | Operational gateway, SIEM integration, pilot | Gateway, pipelines, runbooks |
| **Managed AI Security Operations** | Recurring | Continuous monitoring, IR, reporting | Monthly reports, quarterly reviews |
| **Vendor Workspace Governance** | Project/Recurring | Workspace config, RBAC, compliance pipeline | Policies, RBAC model, dashboards |

**Current maturity:** Delivery model defined and technically validated in the local host-first reference environment. Customer-specific implementation artifacts are produced during engagements. Cloud-specific production validation remains pending.

## Buyable Shape

The offerings are designed to answer four buyer questions directly:

1. What do you deploy?
2. What do you operate after deployment?
3. What must the customer still own?
4. What evidence is produced during the engagement?

If an offering cannot answer all four, it should not be sold as a defined service.

## Sellability Rule

Do not sell ambiguity. Every offering must say:

- the primary validated deployment path
- the control boundary
- the customer-owned prerequisites
- the decision artifact produced at the end

---

## Offering 1: AI Usage & Exposure Assessment

**Purpose:** Current-state visibility into AI usage, data exposure, control gaps. Prioritized roadmap.

**Typical Activities:**
- Inventory discovery (known tools, shadow usage, network indicators)
- Data classification mapping
- Control gap analysis (API-key vs. subscription flows)
- Target architecture design
- Roadmap prioritization

**Deliverables:**

| Artifact | Description |
|----------|-------------|
| AI Usage Assessment Report | Inventory, risk findings, gap analysis |
| Data Classification Matrix | Tool-by-data-type usage mapping |
| Target Architecture Document | Future-state design |
| 90-Day Roadmap | Prioritized initiatives |

**Prerequisites From Customer:**
- List of known AI tools
- Proxy/SWG logs if available
- Network architecture overview
- IAM/IdP contact

**Out of Scope:**
- Production changes
- Penetration testing
- Implementation (separate offering)

---

## Offering 2: AI Control Plane Implementation

**Purpose:** Deploy operational gateway, integrate SIEM, implement detections, pilot with users.

**Typical Activities:**
- Deploy AI gateway (single-node supported baseline; any HA design is customer-owned or separately validated)
- Configure allowlists, budgets, rate limits
- Implement evidence pipeline (logs → normalization → SIEM)
- Design egress/SWG patterns (customer implements)
- Configure workspace governance
- Develop detection rules
- Create IR runbooks
- Execute pilot (10-50 users)

**Deliverables:**

| Artifact | Description |
|----------|-------------|
| Deployed Gateway | Operational in customer environment |
| Policy Configuration | Allowlists, budgets, rate limits |
| Evidence Pipeline | Log collection, normalization, SIEM ingestion |
| Detection Pack | 5-10 initial detection rules |
| Runbook Library | IR procedures |
| Pilot Report | Metrics, issues, recommendations |

**Success Criteria:**
- Gateway healthy
- Allowlist enforced (test: unapproved rejected)
- Budget enforcement functional
- SIEM receiving normalized events
- Pilot users active

**Pilot Exit Criteria:**
- Customer network owner has reviewed enforce-vs-detect boundaries
- Customer SIEM owner confirms ingestion path and alert destination
- Customer sponsor signs off on pilot scope and next-step decision
- Pilot follows the phase model in `docs/PILOT_EXECUTION_MODEL.md`

---

## Offering 3: Managed AI Security Operations

**Purpose:** Ongoing monitoring, alert triage, incident response, governance reporting.

**Typical Activities:**
- 24x7 or business-hours monitoring
- Alert triage and classification
- Incident response per runbook
- Monthly governance reporting
- Monthly chargeback reporting
- Quarterly business reviews
- Policy maintenance and tuning

**Deliverables:**

| Artifact | Frequency |
|----------|-----------|
| Monthly Governance Report | Monthly |
| Monthly Chargeback Report | Monthly |
| Incident Reports | Per incident |
| Quarterly Business Review | Quarterly |

**Operating Boundaries:**
- Coverage windows in this document define staffing shape, not contractual SLA/SLO terms
- Incident response time, MTTR, and escalation commitments must be defined in the customer SOW or managed-service agreement
- Managed operations does not remove the customer's responsibility for network, IAM, or vendor-workspace ownership
- Shared responsibility should be agreed using [SHARED_RESPONSIBILITY_MODEL.md](SHARED_RESPONSIBILITY_MODEL.md)

**Required prerequisites before sale:**

- Customer has a deployed or deployment-ready AI Control Plane baseline
- Customer has named security, platform, SIEM, and finance contacts
- Escalation path and communications channel are defined
- Customer accepts the shared-responsibility model in writing
- Any 24x7 or extended-hours expectation is staffed and explicitly priced

**Do not sell this offering yet if any of these are false:**

- the customer has not accepted the shared-responsibility split in writing
- the customer expects provider ownership of cloud/network/IdP systems by implication
- the customer expects bypass prevention without owning endpoint and egress controls
- no decision-maker exists for accepted-risk and escalation calls

**Standard response model:**

| Area | Provider default role | Customer role |
|---|---|---|
| Alert triage | Monitor, classify, investigate, escalate | Receive escalations, approve material actions |
| Application incidents | Diagnose and remediate issues within the delivered platform | Approve change windows and support dependency access |
| Cloud/network/IdP issues | Identify likely dependency and coordinate evidence | Own root-cause remediation in customer-controlled systems |
| Governance reporting | Produce recurring reports and recommendations | Review decisions, approve budget or policy changes |
| Major security events | Execute agreed runbooks and escalation path | Own business-risk decisions and containment outside provider scope |

**Procurement-safe operating truth:**

- application issues: provider-owned within scope
- customer infrastructure issues: customer-owned
- cross-boundary incidents: shared coordination, customer risk ownership
- service language must match `docs/SHARED_RESPONSIBILITY_MODEL.md`

**Service Tiers:**

| Tier | Coverage | Best For |
|------|----------|----------|
| Essential | Business hours | Low-volume deployments |
| Professional | Extended hours | Growing deployments |
| Enterprise | 24×7 | Production-critical |

---

## Offering 4: Vendor Workspace Governance

**Purpose:** Configure and operate governance for AI vendor workspaces (ChatGPT Enterprise, Claude Enterprise, etc.)

**Typical Activities:**
- Workspace configuration: RBAC, retention, sharing policies
- User lifecycle management
- Compliance export setup: automated pulls, normalization, SIEM ingestion
- Usage analytics and reporting
- Policy enforcement alignment
- SSO/SAML, SCIM integration design

**Deliverables:**

| Artifact | Description |
|----------|-------------|
| Workspace Governance Policy | RBAC model, retention rules |
| Configuration Baseline | Standard settings |
| Compliance Export Pipeline | Automated export to SIEM |
| Governance Dashboard | Usage and compliance view |
| User Lifecycle Runbook | Onboard/use/offboard procedures |

---

## Guardrails Customization (Cross-Cutting)

Productized guardrail engineering across request lifecycle.

| Stage | Objective | Typical Controls |
|-------|-----------|------------------|
| **Pre-call** | Stop unsafe requests | Entity policies, injection filters, secret detection |
| **In-call** | Constrain behavior | Parameter constraints, tool allowlists, schema controls |
| **Post-call** | Detect drift | Detection tuning, SIEM correlation, reporting |

**Package Options:**

| Package | Scope |
|---------|-------|
| Foundation | Pre-call baseline |
| Extended | Pre-call + in-call |
| Operationalized | Full lifecycle with analytics |

---

## Data Handling (Default)

**Metadata-only:**
- Principal identity
- Model/provider used
- Timestamp, token counts, cost
- Policy outcome

**Transcripts (opt-in only):**
- Separate SOW amendment
- Restricted storage, encryption at rest
- Strict retention limits
- Access logging

---

## Commercial Models

| Model | Description | Best For |
|-------|-------------|----------|
| Fixed Scope | Defined deliverables, timeline, acceptance | Assessment, Implementation |
| Time & Materials | Flexible scope, effort-based | Managed Ops, ongoing |
| Retainer | Reserved capacity, priority response | Managed Ops, multiple needs |

## Managed-Service Rule

Do not position managed operations as “we own everything.” The buyable version of this service is:

- provider-operated application layer
- explicit escalation and reporting model
- customer-owned infrastructure, network, IAM, and workspace controls unless separately contracted

That boundary should stay consistent in proposals, SOWs, and pilot closeout material.

---

## Related Documentation

- `ENTERPRISE_STRATEGY.md` — Strategic overview
- `LOCAL_DEMO_PLAN.md` — Local reference environment scope
- `GO_TO_MARKET_SCOPE.md` — Readiness criteria
- `PILOT_CONTROL_OWNERSHIP_MATRIX.md` — Customer pilot owner matrix
- `SHARED_RESPONSIBILITY_MODEL.md` — Operating-boundary and commercial responsibility split

---

**Document status:** Enterprise-facing offering framework  
**Validation basis:** Local host-first reference environment, runbooks, and delivery templates  
**Last revised:** 2026-03-05
