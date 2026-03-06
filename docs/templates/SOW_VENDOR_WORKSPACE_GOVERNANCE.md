# Statement of Work: Vendor Workspace Governance

**Template Version:** 1.0  
**Last Updated:** February 2026  

---

## 1. Engagement Overview

| Field | Value |
|-------|-------|
| **Customer** | [CUSTOMER] |
| **Service Provider** | Project Maintainer |
| **Engagement Type** | Configuration and Governance (Project or Managed) |
| **Start Date** | [START_DATE] |
| **End Date** | [END_DATE] (or "Ongoing" for managed service) |
| **Effective Date** | [EFFECTIVE_DATE] |
| **Workspace Provider(s)** | [WORKSPACE_PROVIDERS] |

### 1.1 Objectives

The primary objectives of this engagement are to:

1. **Configure AI vendor workspaces** (ChatGPT Business/Enterprise, Claude Enterprise, etc.) per governance requirements
2. **Implement RBAC and access controls** aligned with [CUSTOMER] identity and security policies
3. **Establish compliance export pipeline** for audit and governance (where available)
4. **Document workspace governance policies** and operational procedures
5. **Enable ongoing governance** through monitoring, access reviews, and policy maintenance

### 1.2 Background

[CUSTOMER] utilizes or plans to utilize AI vendor workspace offerings (e.g., ChatGPT Enterprise, Claude for Enterprise) that require governance configuration including user management, data retention policies, sharing controls, and compliance monitoring. This engagement establishes and maintains that governance posture.

### 1.3 Control Boundary

This engagement governs vendor-managed workspace settings and related operating procedures. It does not convert vendor product limits into provider guarantees, and it does not replace customer ownership of identity, legal retention decisions, or enterprise endpoint/network controls.

Use this template with:

- `docs/BROWSER_WORKSPACE_PROOF_TRACK.md`
- `docs/SHARED_RESPONSIBILITY_MODEL.md`
- `docs/PILOT_CUSTOMER_VALIDATION_CHECKLIST.md`

### 1.4 Hard-Stop Conditions

Do not describe this engagement as enterprise browser/workspace governance readiness unless all of the following are true:

- the vendor license tier required for the scoped features is confirmed
- the customer names owners for workspace administration, identity, compliance, and SIEM
- the customer accepts in writing which controls remain vendor-limited or detective-only
- the engagement has an evidence and closeout path rather than a pure configuration checklist

---

## 2. Scope

### 2.1 In Scope

#### Workspace Configuration
- [ ] Initial workspace assessment and inventory
- [ ] User role definition and RBAC model design
- [ ] Workspace policy configuration (sharing, retention, data handling)
- [ ] SSO/SAML integration design and validation
- [ ] SCIM provisioning configuration (if applicable)

#### User Lifecycle Management
- [ ] User onboarding procedures
- [ ] User offboarding procedures
- [ ] Access review process design and execution
- [ ] Transfer procedures (conversation/data ownership)

#### Compliance Exports (Where Available)
- [ ] Export configuration and automation setup
- [ ] Normalization and SIEM integration
- [ ] Retention policy implementation
- [ ] Export validation and monitoring

#### Governance Operations (Managed Service)
- [ ] Ongoing user access management
- [ ] Quarterly access reviews
- [ ] Policy compliance monitoring
- [ ] Usage analytics and reporting
- [ ] Workspace policy updates

#### Documentation
- [ ] Workspace governance policy document
- [ ] User lifecycle runbooks
- [ ] Configuration baseline and change log
- [ ] Governance dashboard setup

### 2.2 Out of Scope

The following are explicitly excluded:

- Procurement of workspace licenses or tier upgrades
- SSO/SAML infrastructure deployment (we integrate with existing)
- End-user training on AI tool features
- Content moderation or review of user conversations
- Forensic search of conversation history (unless explicitly scoped)
- Integration with proprietary systems beyond standard APIs
- Development of custom connectors (unless scoped as add-on)
- Legal or compliance advisory beyond workspace governance
- Customer-owned firewall, SWG, CASB, proxy, DNS, or endpoint-enforcement administration
- Claims that this engagement alone prevents direct browser bypass outside the scoped vendor and managed-device path
- Vendor roadmap commitments or unsupported export behaviors

### 2.3 Deliverable Interpretation Rules

- workspace settings prove the scoped vendor baseline only
- compliance exports are limited to what the vendor actually supports on the purchased tier
- detective evidence for direct browser usage is not the same as preventative blocking
- customer acceptance must name any vendor or customer limitations that remain open

---

## 3. Deliverables and Acceptance Criteria

### 3.1 Deliverables (Project)

| ID | Deliverable | Description | Format | Delivery Milestone |
|----|-------------|-------------|--------|--------------------|
| D1 | Workspace Assessment | Current configuration, user inventory, gap analysis | PDF | Assessment milestone |
| D2 | Governance Policy | RBAC model, retention rules, sharing restrictions | Document | Policy approval milestone |
| D3 | Configuration Baseline | Standard settings for all workspace tenants | Checklist/Code | Configuration baseline milestone |
| D4 | Compliance Export Pipeline | Automated export, normalization, SIEM routing (if available) | Configs + Docs | Integration readiness milestone |
| D5 | Governance Dashboard | Workspace usage, policy compliance, access status | Dashboard | Reporting readiness milestone |
| D6 | User Lifecycle Runbook | Onboarding, offboarding, transfer procedures | Markdown/PDF | Operations readiness milestone |
| D7 | Access Review Process | Quarterly review template and procedure | Template + Docs | Governance handoff milestone |

### 3.2 Deliverables (Ongoing Managed Service)

| Deliverable | Description | Frequency |
|-------------|-------------|-----------|
| User Management | Onboarding/offboarding per requests | As requested (SLA-defined cadence) |
| Access Review Execution | Quarterly user access audits | Quarterly |
| Compliance Report | Workspace governance status | Monthly |
| Policy Update Log | All configuration changes | Per change |
| Usage Analytics Report | Trends, optimization opportunities | Quarterly |

### 3.3 Acceptance Criteria

- **D1 (Workspace Assessment)**:
  - [ ] All workspace tenants inventoried
  - [ ] User roles and permissions documented
  - [ ] Configuration gaps identified
  - [ ] Current policies vs. target mapped

- **D2 (Governance Policy)**:
  - [ ] RBAC model approved by [CUSTOMER_SECURITY_LEAD]
  - [ ] Retention policies aligned with legal requirements
  - [ ] Sharing restrictions defined
  - [ ] Policy documented and accessible

- **D3 (Configuration Baseline)**:
  - [ ] All settings configured per baseline
  - [ ] Configuration drift detection enabled
  - [ ] Baseline version-controlled

- **D4 (Compliance Export Pipeline)**:
  - [ ] Exports automated (if vendor supports)
  - [ ] Normalization functional
  - [ ] SIEM receiving data
  - [ ] Validation tests passing
  - [ ] Vendor limits and unsupported fields documented explicitly

- **D5 (Governance Dashboard)**:
  - [ ] User access status visible
  - [ ] Policy compliance metrics displayed
  - [ ] Usage trends available

- **D6 (User Lifecycle Runbook)**:
  - [ ] Onboarding procedure tested
  - [ ] Offboarding procedure tested
  - [ ] Transfer procedure documented

- **D7 (Access Review Process)**:
  - [ ] Review template functional
  - [ ] Procedure validated with stakeholders
  - [ ] First review scheduled

---

## 4. Timeline and Phases (Project)

### 4.1 Phase Breakdown

| Phase | Activities | Duration |
|-------|------------|----------|
| **Assessment** | Workspace inventory, user audit, current config review | Per agreed project plan |
| **Design** | RBAC model, policy definition, retention rules | Per agreed project plan |
| **Implementation** | Configuration changes, pipeline setup, automation | Per agreed project plan |
| **Validation** | Testing, runbook validation, documentation | Per agreed project plan |
| **Handoff** | Knowledge transfer, first access review | Per agreed project plan |

### 4.2 Key Milestones

| Milestone | Target Date | Success Criteria |
|-----------|-------------|------------------|
| M1: Assessment Complete | Per project plan | All workspaces inventoried |
| M2: Design Approved | Per project plan | RBAC and policies approved |
| M3: Configuration Complete | Per project plan | All settings applied |
| M4: Pipeline Operational | Per project plan | Exports flowing to SIEM (if available) |
| M5: Engagement Complete | Per project plan | All deliverables accepted |

### 4.3 Pilot/Closeout Requirement

If this engagement is part of a pilot, closeout must reference the customer validation checklist and acceptance memo rather than treating “configured” as equivalent to “governed.”

---

## 5. Roles and Responsibilities

### 5.1 Project Team

| Role | Name | Responsibilities |
|------|------|------------------|
| Workspace Admin | [PROJECT_WORKSPACE_ADMIN] | Configuration, user management, policy implementation |
| Integration Engineer | [PROJECT_INTEGRATION_ENGINEER] | Compliance export pipeline, SIEM integration |
| Project Manager | [PROJECT_PM] | Coordination, status reporting, stakeholder management |

### 5.2 Customer Team

| Role | Name | Responsibilities |
|------|------|------------------|
| Workspace Owner | [CUSTOMER_WORKSPACE_OWNER] | Workspace ownership, escalated access issues |
| IT Admin | [CUSTOMER_IT_ADMIN] | Day-to-day workspace administration |
| Identity Team Lead | [CUSTOMER_IDENTITY_LEAD] | SSO/SAML integration, user provisioning |
| Compliance/Legal | [CUSTOMER_COMPLIANCE] | Retention policy approval, data handling guidance |
| Security Lead | [CUSTOMER_SECURITY_LEAD] | Security policy alignment, risk acceptance |

### 5.3 RACI Matrix

| Activity | Project Admin | Project Integration | Customer Workspace Owner | Customer Identity | Customer Compliance |
|----------|-----------|-----------------|--------------------------|-------------------|---------------------|
| Configuration | R/A | C | A | C | C |
| User Management | R/A | I | A/C | C | I |
| SSO Integration | C | R | C | R/A | I |
| Export Pipeline | C | R/A | A | I | C |
| Policy Definition | C | I | C | I | A/R |
| Access Reviews | R | I | A | C | C |
| Compliance Validation | C | R | C | I | A |

---

## 6. Customer-Provided Inputs and Access

### 6.1 Required Inputs

| Input | Purpose | Timing | Owner |
|-------|---------|--------|-------|
| Workspace admin credentials | Configuration access | Kickoff | [CUSTOMER_WORKSPACE_OWNER] |
| Enterprise license confirmation | Compliance export eligibility | Kickoff | [CUSTOMER_WORKSPACE_OWNER] |
| Data retention requirements | Retention policy configuration | Design phase | [CUSTOMER_COMPLIANCE] |
| Data classification scheme | Access control alignment | Design phase | [CUSTOMER_SECURITY_LEAD] |
| User list/HR feed | User lifecycle management | Implementation phase | [CUSTOMER_IDENTITY_LEAD] |
| SSO/SAML metadata | Identity integration | Implementation phase | [CUSTOMER_IDENTITY_LEAD] |
| SIEM integration point | Export pipeline destination | Integration phase | [CUSTOMER_IT_ADMIN] |

### 6.2 System Access

| System | Access Level | Purpose | Timing |
|--------|--------------|---------|--------|
| Workspace admin console | Admin | Configuration, user management | Kickoff |
| Identity provider | Read + config | SSO integration | Implementation phase |
| SIEM platform | Write/API | Export ingestion | Integration phase |
| HR system (optional) | Read | User provisioning automation | Implementation phase |

---

## 7. Data Handling and Security

### 7.1 Data Access

Project may access:
- Workspace configuration settings
- User list and role assignments
- Usage metadata (token counts, user activity)
- Compliance export metadata (who, when, not content)

Service Provider will NOT access:
- User conversation content (prompts/responses)
- Exported conversation transcripts (unless explicitly scoped separately)
- Customer data processed by AI tools

### 7.2 Compliance Export Content

**Standard Scope**: Metadata only
- Export timestamp
- User identification
- Conversation IDs
- Token counts (if provided by vendor)

**Transcript Content (If Required)**:
- Requires explicit addendum to this SOW
- Separate restricted storage
- Enhanced access controls
- Defined retention period
- Separate from primary governance data

### 7.3 Data Residency

- Workspace data resides per [CUSTOMER] vendor contract
- Project access from [PROJECT_REGION]
- Compliance exports stored per [CUSTOMER] SIEM policy

---

## 8. Assumptions and Dependencies

### 8.1 Assumptions

1. **Enterprise Tier**: Customer holds or obtains enterprise-tier licenses enabling compliance exports and advanced governance features
2. **Vendor API Stability**: Vendor compliance export APIs are stable and documented
3. **Identity Infrastructure**: SSO/SAML infrastructure exists and can be integrated
4. **Legal Approval**: Data retention and handling policies approved by [CUSTOMER] Legal/Compliance
5. **Timely Decisions**: [CUSTOMER] provides policy decisions according to the agreed governance cadence
6. **Feature Availability Confirmed**: vendor-tier features in scope are contractually available before implementation begins

### 8.2 External Dependencies

| Dependency | Impact if Unavailable | Mitigation |
|------------|----------------------|------------|
| Vendor compliance API | Cannot automate exports | Manual export process |
| SSO infrastructure | Cannot implement SSO | Delay SSO, use native auth interim |
| SIEM integration point | Cannot ingest exports | Local storage interim |
| Enterprise tier license | Limited governance features | Upgrade or accept limitations |
| Missing customer owners | Governance scope cannot close credibly | Reframe as advisory/configuration only |

---

## 9. Workspace-Specific Notes

### 9.1 OpenAI ChatGPT Business/Enterprise

| Feature | Availability | Notes |
|---------|--------------|-------|
| SSO/SAML | Enterprise | Required for centralized governance |
| Compliance exports | Enterprise | Includes conversation metadata |
| Admin controls | Business+ | User management, sharing controls |
| Data retention | Enterprise | Configurable retention periods |

### 9.2 Anthropic Claude for Enterprise

| Feature | Availability | Notes |
|---------|--------------|-------|
| SSO/SAML | Enterprise | Required for centralized governance |
| Compliance exports | Enterprise | Via enterprise agreement |
| Admin controls | Enterprise | Workspace-level controls |
| Data retention | Enterprise | Configurable per workspace |

### 9.3 Microsoft Copilot

| Feature | Availability | Notes |
|---------|--------------|-------|
| SSO/SAML | Included | Via Microsoft 365 identity |
| Compliance | Included | Via Microsoft Purview |
| Admin controls | Included | Microsoft 365 admin center |
| Data handling | Included | Microsoft 365 compliance |

---

## 10. Change Control

### 10.1 Change Types

| Change Type | Approval Required | Implementation |
|-------------|-------------------|----------------|
| Policy configuration | [CUSTOMER_SECURITY_LEAD] | Project |
| User role changes | [CUSTOMER_WORKSPACE_OWNER] | Project or Customer |
| Retention period changes | [CUSTOMER_COMPLIANCE] | Project |
| SSO configuration | [CUSTOMER_IDENTITY_LEAD] | Joint |
| Export pipeline changes | [CUSTOMER_IT_ADMIN] | Project |

### 10.2 Emergency Changes

In security-relevant situations (e.g., suspected breach), Project may implement emergency access restrictions with:
- Immediate notification to [CUSTOMER_SECURITY_LEAD]
- Documentation of change and rationale
- Formal approval within the emergency-change SLA
- Rollback if approval not granted

---

## 11. Success Criteria

### 11.1 Project Completion

- [ ] Workspace configured per governance policy baseline
- [ ] RBAC implemented with documented role-user mapping
- [ ] Compliance exports automated (if vendor supports)
- [ ] User lifecycle runbook tested (onboard → use → offboard)
- [ ] Quarterly access review process established
- [ ] Knowledge transfer completed

### 11.2 Managed Service (if applicable)

- [ ] User requests processed within SLA-defined cadence (target: 100%)
- [ ] Quarterly access reviews completed on schedule
- [ ] Policy violations detected and triaged within agreed detection SLAs
- [ ] Customer satisfaction ≥4/5

Meeting these criteria does not by itself prove enterprise-wide bypass prevention or universal coverage for unmanaged devices. Those claims require customer-owned controls and validation outside this template.

---

## 12. Commercial Terms (Structure)

### 12.1 Engagement Models

| Model | Description | Best For |
|-------|-------------|----------|
| **Project** | Fixed-scope initial configuration | First-time setup |
| **Managed** | Ongoing governance operations | Continuous management |
| **Hybrid** | Project + ongoing managed | Setup + long-term ops |

*Selected: [ENGAGEMENT_MODEL]*

### 12.2 Pricing Components

| Component | Basis |
|-----------|-------|
| Initial setup | Fixed fee per workspace |
| Per-user management | Per-user per month (managed service) |
| Compliance export setup | Fixed fee per integration |
| Access reviews | Per review or included in managed service |

*Specific pricing provided in separate commercial proposal.*

---

## 13. Signatures

By signing below, the parties agree to the terms of this Statement of Work.

**Project Maintainer:**

| Role | Name | Signature | Date |
|------|------|-----------|------|
| Workspace Admin Lead | [PROJECT_WORKSPACE_ADMIN] | | |
| Project Manager | [PROJECT_PM] | | |
| Practice Lead | [PROJECT_PRACTICE_LEAD] | | |

**[CUSTOMER]:**

| Role | Name | Signature | Date |
|------|------|-----------|------|
| Workspace Owner | [CUSTOMER_WORKSPACE_OWNER] | | |
| Security Lead | [CUSTOMER_SECURITY_LEAD] | | |

---

## Appendix A: Workspace Configuration Checklist

### Standard Configuration Baseline

| Setting | Recommended Value | Configured |
|---------|-------------------|------------|
| SSO Required | Yes | |
| Data Retention | Per [CUSTOMER] policy | |
| External Sharing | Disabled or Restricted | |
| Code Interpreter | Per role | |
| Web Browsing | Per role | |
| Plugin/Action Usage | Restricted | |
| Conversation History | Enabled (for compliance) | |

## Appendix B: Glossary

| Term | Definition |
|------|------------|
| RBAC | Role-Based Access Control |
| SCIM | System for Cross-domain Identity Management |
| SAML | Security Assertion Markup Language |
| SSO | Single Sign-On |
| Workspace | Vendor-managed enterprise AI environment |

## Appendix C: Reference Documents

| Document | Location |
|----------|----------|
| Service Offerings Catalog | `../SERVICE_OFFERINGS.md` |
| Managed Ops Template | `SOW_MANAGED_AI_SECURITY_OPERATIONS.md` |
| Implementation Template | `SOW_AI_CONTROL_PLANE_IMPLEMENTATION.md` |

---

*Template Version 1.0 — Project Maintainer*
