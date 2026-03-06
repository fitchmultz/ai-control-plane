# Statement of Work: AI Usage & Exposure Assessment

**Template Version:** 1.0  
**Last Updated:** February 2026  

---

## 1. Engagement Overview

| Field | Value |
|-------|-------|
| **Customer** | [CUSTOMER] |
| **Service Provider** | Project Maintainer |
| **Engagement Type** | Assessment and Advisory |
| **Start Date** | [START_DATE] |
| **End Date** | [END_DATE] |
| **Effective Date** | [EFFECTIVE_DATE] |

### 1.1 Objectives

The primary objectives of this engagement are to:

1. **Inventory AI tool usage** across the enterprise, including sanctioned and shadow usage
2. **Assess data exposure risks** from current AI usage patterns
3. **Identify control gaps** in the current governance posture
4. **Develop target architecture** for AI governance and control
5. **Produce prioritized roadmap** for achieving target state

### 1.2 Background

[CUSTOMER] seeks to understand the current state of AI tool adoption within the organization, assess associated risks, and develop a strategic approach to governance. This assessment will provide the foundation for potential subsequent implementation engagements.

### 1.3 Assessment Boundary

This engagement produces a decision-grade assessment and roadmap. It does not implement controls, validate production operations, or prove customer-environment enforcement.

Use this template with:

- `docs/ENTERPRISE_STRATEGY.md`
- `docs/ENTERPRISE_BUYER_OBJECTIONS.md`
- `docs/GO_TO_MARKET_SCOPE.md`

---

## 2. Scope

### 2.1 In Scope

Service Provider will perform the following activities:

#### Discovery and Inventory
- [ ] Stakeholder interviews (Security, IT, Platform, Legal, key business units)
- [ ] Network and proxy log analysis for AI tool indicators
- [ ] Inventory of known AI tools, providers, and use cases
- [ ] Identification of shadow AI usage patterns

#### Risk and Gap Analysis
- [ ] Data classification mapping: what data types interact with AI tools
- [ ] Control coverage assessment: preventive vs. detective controls
- [ ] Gap analysis: API-key vs. subscription/SaaS flow coverage
- [ ] Regulatory and compliance alignment review

#### Architecture and Roadmap
- [ ] Target architecture design: approved-path model
- [ ] Gateway placement and integration recommendations
- [ ] Evidence pipeline design (logging, SIEM, detection)
- [ ] Prioritized roadmap: near-term, mid-term, and strategic initiatives

#### Deliverable Production
- [ ] Assessment report with findings and recommendations
- [ ] Data classification matrix
- [ ] Target architecture document
- [ ] Implementation roadmap with phased priorities

### 2.2 Out of Scope

The following are explicitly excluded from this engagement:

- Implementation of recommended controls or infrastructure
- Configuration changes to production systems
- Penetration testing or active exploitation attempts
- Procurement of AI tools, licenses, or provider contracts
- Long-term operational support or monitoring
- End-user training on AI tools or governance policies
- Any promise that recommended controls can be implemented without separate delivery work
- Any claim that the assessment alone proves bypass prevention, compliance readiness, or rollout readiness

### 2.3 Deliverable Interpretation Rules

- inventories and heatmaps are decision inputs, not production proof
- identified usage gaps may be directional when customer visibility sources are incomplete
- the roadmap must separate customer-owned work from provider-delivered work
- implementation or managed-service language must not appear in the final readout unless separately scoped

---

## 3. Deliverables and Acceptance Criteria

### 3.1 Deliverables

| ID | Deliverable | Description | Format | Delivery Milestone |
|----|-------------|-------------|--------|--------------------|
| D1 | Assessment Report | Current-state inventory, risk findings, gap analysis, prioritized recommendations | PDF + PPT executive summary | Final assessment readout |
| D2 | Data Classification Matrix | Tool-by-data-type usage mapping with risk ratings | Spreadsheet | Analysis milestone |
| D3 | Control Coverage Heatmap | Preventive vs. detective coverage visualization | Visual + narrative | Analysis milestone |
| D4 | Target Architecture Document | Future-state design with integration points and data flows | Technical document | Final assessment readout |
| D5 | Prioritized Roadmap | Prioritized initiatives with dependencies and success metrics | Project plan format | Final assessment readout |

### 3.2 Acceptance Criteria

Each deliverable will be considered accepted when:

- **D1 (Assessment Report)**:
  - [ ] All identified stakeholders interviewed
  - [ ] Tool inventory covers >80% of known AI usage
  - [ ] Risk findings documented with severity and business impact
  - [ ] Executive summary suitable for C-level presentation
  - [ ] Reviewed and approved by [CUSTOMER_SPONSOR]

- **D2 (Data Classification Matrix)**:
  - [ ] All identified tools mapped to data types used
  - [ ] Risk ratings aligned with [CUSTOMER] data classification scheme
  - [ ] Validation with business unit representatives

- **D3 (Control Coverage Heatmap)**:
  - [ ] Coverage assessment for API-key flows
  - [ ] Coverage assessment for subscription/SaaS flows
  - [ ] Identification of coverage gaps

- **D4 (Target Architecture Document)**:
  - [ ] Architecture approved by [CUSTOMER] Security and Platform teams
  - [ ] Integration points defined and validated
  - [ ] Sizing and capacity guidance provided

- **D5 (Prioritized Roadmap)**:
  - [ ] Roadmap accepted by implementation stakeholders
  - [ ] Dependencies and prerequisites identified
  - [ ] Success metrics defined for each initiative
  - [ ] Customer-owned vs provider-delivered work split explicitly

---

## 4. Timeline and Milestones

### 4.1 Phase Breakdown

| Phase | Activities | Duration |
|-------|------------|----------|
| **Kickoff** | Project initiation, stakeholder identification, access provisioning | Per agreed plan |
| **Discovery** | Interviews, log analysis, tool inventory | Per agreed plan |
| **Analysis** | Risk assessment, gap analysis, architecture design | Per agreed plan |
| **Reporting** | Draft deliverables, review cycles, finalization | Per agreed plan |
| **Readout** | Executive presentation, handoff | Per agreed plan |

### 4.2 Key Milestones

| Milestone | Target Date | Success Criteria |
|-----------|-------------|------------------|
| M1: Kickoff Complete | Per project plan | Stakeholder list confirmed, access granted |
| M2: Discovery Complete | Per project plan | Interviews done, logs analyzed |
| M3: Analysis Complete | Per project plan | Findings validated with stakeholders |
| M4: Draft Deliverables | Per project plan | All deliverables in draft for review |
| M5: Engagement Complete | Per project plan | All deliverables accepted |

---

## 5. Roles and Responsibilities

### 5.1 Project Team

| Role | Name | Responsibilities |
|------|------|------------------|
| Engagement Lead | [PROJECT_PM] | Project management, stakeholder coordination, quality assurance |
| Security Architect | [PROJECT_SECURITY_ARCHITECT] | Technical assessment, architecture design, risk analysis |
| Data Analyst | [PROJECT_ANALYST] | Log analysis, usage pattern identification, metrics |

### 5.2 Customer Team

| Role | Name | Responsibilities |
|------|------|------------------|
| Executive Sponsor | [CUSTOMER_SPONSOR] | Strategic alignment, escalation, final acceptance |
| Project Manager | [CUSTOMER_PM] | Coordination, scheduling, internal communication |
| Security Lead | [CUSTOMER_SECURITY_LEAD] | Requirements, findings validation, security context |
| Network Team Lead | [CUSTOMER_NETWORK_LEAD] | Log provision, architecture context |
| IT/Platform Lead | [CUSTOMER_PLATFORM_LEAD] | Tool inventory, workspace access |

### 5.3 RACI Matrix

| Activity | Project Engagement Lead | Project Security Architect | Customer Security Lead | Customer Network | Customer IT/Platform |
|----------|---------------------|------------------------|------------------------|------------------|----------------------|
| Project Management | R/A | C | A/C | I | I |
| Stakeholder Interviews | R | R | A | C | C |
| Log Analysis | R | C | I | A/C | I |
| Risk Assessment | R | R/A | A/C | I | C |
| Architecture Design | C | R/A | A/C | C | C |
| Deliverable Review | R | R | A | I | C |
| Acceptance Sign-off | C | C | A | I | I |

---

## 6. Customer-Provided Inputs and Access

### 6.1 Required Inputs

| Input | Purpose | Timing | Owner |
|-------|---------|--------|-------|
| List of known AI tools/providers | Baseline inventory | Kickoff | [CUSTOMER_PLATFORM_LEAD] |
| Proxy/SWG logs (representative period) | Usage pattern analysis | Discovery phase | [CUSTOMER_NETWORK_LEAD] |
| Network architecture diagram | Gateway placement planning | Discovery phase | [CUSTOMER_NETWORK_LEAD] |
| Data classification scheme | Risk alignment | Kickoff | [CUSTOMER_SECURITY_LEAD] |
| Current security policies | Control gap assessment | Kickoff | [CUSTOMER_SECURITY_LEAD] |
| Compliance requirements | Regulatory alignment | Kickoff | [CUSTOMER_SECURITY_LEAD] |

### 6.2 System Access

| System | Access Level | Purpose | Timing |
|--------|--------------|---------|--------|
| SIEM (read-only) | Query access | Control verification | Discovery phase |
| Proxy/SWG admin | Read access | Log export, policy review | Discovery phase |
| Network monitoring | Read access | Traffic analysis | Discovery phase |

### 6.3 Personnel Access

| Stakeholder Group | Interview Duration | Timing |
|-------------------|-------------------|--------|
| Security Leadership | Engagement-defined | Discovery phase |
| IT/Platform Team | Engagement-defined | Discovery phase |
| Network Team | Engagement-defined | Discovery phase |
| Legal/Compliance | Engagement-defined | Discovery phase |
| Business Unit Reps (2–3) | Engagement-defined | Discovery phase |

---

## 7. Data Handling and Security

### 7.1 Data Classification

Data handled during this engagement:

| Data Type | Classification | Handling |
|-----------|----------------|----------|
| Network logs | Internal/Confidential | Encrypted storage, limited access |
| Tool inventories | Internal | Standard handling |
| Risk findings | Confidential | Encrypted storage, need-to-know access |
| Architecture designs | Confidential | Encrypted storage, limited access |

### 7.2 Data Retention

- Assessment data retained for [RETENTION_PERIOD] post-engagement for support purposes
- All data deleted upon customer written request
- No customer data used for Project marketing without explicit permission

### 7.3 Security Practices

- All Project consultants undergo background checks
- Data transferred via encrypted channels only
- No customer data stored on personal devices
- Access logs maintained for audit purposes

---

## 8. Assumptions and Dependencies

### 8.1 Assumptions

1. **Stakeholder Availability**: Key stakeholders available for interviews according to the agreed discovery schedule
2. **Log Availability**: Customer can provide reasonable network visibility (proxy/SWG logs or flow data)
3. **No Active Incidents**: No active security incidents requiring immediate remediation during assessment period
4. **Good Faith Cooperation**: Customer provides accurate information and timely responses
5. **Environment Stability**: No major infrastructure changes during assessment period

### 8.2 External Dependencies

| Dependency | Impact if Not Available | Mitigation |
|------------|------------------------|------------|
| Network logs | Reduced visibility into shadow usage | Interview-based estimation |
| SIEM access | Cannot validate existing controls | Documentation review |
| Stakeholder time | Delayed findings validation | Extended timeline |

---

## 9. Change Control

### 9.1 Change Request Process

1. Either party may request changes via written change request
2. Service Provider will assess impact (timeline, cost, resources) per the change-management SLA defined for the engagement
3. Changes require written approval from both parties
4. Unapproved changes are out of scope

### 9.2 Common Change Types

| Change Type | Impact Assessment | Approval Authority |
|-------------|-------------------|-------------------|
| Scope expansion | Additional cost/time | Both parties |
| Timeline extension | Cost impact | Both parties |
| Stakeholder substitution | Minimal | [PROJECT_PM] + [CUSTOMER_PM] |
| Deliverable modification | Variable | Both parties |

---

## 10. Exit Criteria

This engagement will be considered complete when:

- [ ] All deliverables (D1–D5) produced and accepted
- [ ] Executive readout completed
- [ ] Knowledge transfer sessions completed (if applicable)
- [ ] Final invoice submitted and paid
- [ ] Lessons learned documented (internal Project)

Completion of this assessment does not authorize claims such as `pilot-ready`, `enterprise-ready`, or `managed service ready` without follow-on implementation and validation evidence.

### 10.1 Early Termination

Either party may terminate with [NOTICE_PERIOD] written notice. Customer pays for:
- All work completed and accepted prior to termination
- Work in progress (prorated)
- Non-cancelable commitments made by Project

---

## 11. Signatures

By signing below, the parties agree to the terms of this Statement of Work.

**Project Maintainer:**

| Role | Name | Signature | Date |
|------|------|-----------|------|
| Engagement Lead | [PROJECT_PM] | | |
| Practice Lead | [PROJECT_PRACTICE_LEAD] | | |

**[CUSTOMER]:**

| Role | Name | Signature | Date |
|------|------|-----------|------|
| Executive Sponsor | [CUSTOMER_SPONSOR] | | |
| Project Manager | [CUSTOMER_PM] | | |

---

## Appendix A: Glossary

| Term | Definition |
|------|------------|
| AI Control Plane | Unified governance approach for enterprise AI usage |
| API-Key Mode | AI access via programmatic API keys (enforceable at gateway) |
| Shadow AI | Unsanctioned AI tool usage outside IT governance |
| Subscription Mode | AI access via SaaS/workspace subscriptions (gateway-routed where supported; detection + response on bypass) |
| SIEM | Security Information and Event Management platform |

## Appendix B: Reference Documents

| Document | Location |
|----------|----------|
| Service Offerings Catalog | `../SERVICE_OFFERINGS.md` |
| Technical Implementation Guide | `../LOCAL_DEMO_PLAN.md` |
| Detection Patterns | `../security/DETECTION.md` |

---

*Template Version 1.0 — Project Maintainer*
