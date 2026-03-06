# Statement of Work: AI Control Plane Implementation

**Template Version:** 1.0  
**Last Updated:** February 2026  

---

## 1. Engagement Overview

| Field | Value |
|-------|-------|
| **Customer** | [CUSTOMER] |
| **Service Provider** | Project Maintainer |
| **Engagement Type** | Implementation and Deployment |
| **Start Date** | [START_DATE] |
| **End Date** | [END_DATE] |
| **Effective Date** | [EFFECTIVE_DATE] |

### 1.1 Objectives

The primary objectives of this engagement are to:

1. **Deploy operational AI gateway** with approved model policies and budget controls
2. **Implement evidence pipeline** for unified logging and SIEM integration
3. **Develop detection capabilities** for anomalous usage and policy violations
4. **Design network/endpoint controls** (customer implements egress/SWG patterns)
5. **Execute pilot rollout** with validated success metrics

### 1.2 Background

[CUSTOMER] has completed the AI Usage & Exposure Assessment (or equivalent) and identified the need for an AI Control Plane. This implementation will establish the technical foundation for ongoing AI governance.

### 1.3 Control Boundary

This engagement implements the AI Control Plane application baseline and the evidence path around it. It does not by itself prove enterprise-wide bypass prevention, compliance certification, or customer-environment rollout readiness.

Use this template with:

- `docs/ENTERPRISE_PILOT_PACKAGE.md`
- `docs/PILOT_EXECUTION_MODEL.md`
- `docs/PILOT_CONTROL_OWNERSHIP_MATRIX.md`
- `docs/SHARED_RESPONSIBILITY_MODEL.md`

### 1.4 Hard-Stop Conditions

Do not call this engagement a pilot-readiness implementation until all of the following are true:

- named customer owners exist for platform, network, security, SIEM, and pilot sponsorship
- customer has decided whether bypass prevention is an enforced objective or a detective-only objective for the scoped pilot
- customer has agreed in writing that network, endpoint, IAM, SIEM, and workspace controls remain customer-owned unless explicitly added to scope
- the engagement has a documented closeout decision path rather than an open-ended “stabilize and see” expectation

If any of those conditions are not met, sell this engagement as implementation or reference validation only, not as a rollout-readiness pilot.

---

## 2. Scope

### 2.1 In Scope

Service Provider will perform the following activities:

#### Gateway Deployment
- [ ] Deploy AI gateway (LiteLLM) in [CUSTOMER] environment
- [ ] Configure high availability (if specified)
- [ ] Implement model allowlist per approved provider list
- [ ] Configure budget enforcement and rate limits
- [ ] Set up authentication (virtual keys)

#### Evidence Pipeline
- [ ] Implement log collection and normalization
- [ ] Configure SIEM integration ([SIEM_PLATFORM])
- [ ] Set up dashboard and alerting
- [ ] Validate data flow end-to-end

#### Detection Engineering
- [ ] Develop initial detection rules (5–10 rules)
- [ ] Tune rules to minimize false positives
- [ ] Document detection logic and expected output

#### Guardrail Package (Lifecycle)
- [ ] Select guardrail package scope: Foundation / Extended / Operationalized
- [ ] Implement pre-call controls (DLP, prompt-injection, secrets policy)
- [ ] Implement in-call controls (tool/parameter/schema constraints) when in scope
- [ ] Implement post-call controls (detection correlation, tuning workflow, reporting)
- [ ] Validate guardrail outcomes using deterministic test cases

#### Network/Endpoint Design
- [ ] Design egress control patterns (default-deny)
- [ ] Document SWG/CASB integration approach
- [ ] Provide MDM configuration guidance
- [ ] Validation plan written with named customer owners and test evidence requirements
- [ ] **Note**: Customer implements network/endpoint controls; Project designs, documents, and validates the plan only

#### Workspace Governance (as applicable)
- [ ] Configure enterprise workspace policies (ChatGPT, Claude, etc.)
- [ ] Set up compliance export ingestion (if available)

#### Pilot and Documentation
- [ ] Onboard pilot users (10–50 users)
- [ ] Monitor and resolve issues
- [ ] Create incident response runbooks
- [ ] Produce pilot report with metrics and explicit closeout recommendation
- [ ] Deliver production rollout plan

### 2.2 Out of Scope

The following are explicitly excluded from this engagement:

- Procurement of AI provider API keys or enterprise licenses
- Implementation of network/egress controls (design only; customer implements)
- SWG/CASB hardware/software procurement or configuration
- MDM deployment or device management
- End-user training on AI tools (gateway onboarding only)
- Production rollout beyond pilot group
- Long-term operational support (available via separate Managed Ops SOW)
- Customer-owned network, firewall, SWG, CASB, DNS, MDM, or IdP administration
- Customer SIEM retention, case management, or analyst workflow ownership
- Enterprise-wide bypass prevention claims without customer validation evidence
- Compliance attestation, certification, or audit sign-off

### 2.3 Deliverable Interpretation Rules

This SOW must be interpreted using the following rules:

- a deployed gateway proves the application baseline, not enterprise-wide control coverage
- designed controls are not the same as implemented customer controls
- a pilot report is not a production-readiness declaration unless the closeout decision says so explicitly
- customer-owned prerequisites that remain incomplete must be written down in the acceptance memo and closeout packet

---

## 3. Deliverables and Acceptance Criteria

### 3.1 Deliverables

| ID | Deliverable | Description | Format | Delivery Milestone |
|----|-------------|-------------|--------|--------------------|
| D1 | Deployed Gateway | Operational AI gateway in [CUSTOMER] environment | Infrastructure | Core platform readiness |
| D2 | Policy Configuration | Model allowlists, budgets, rate limits | Code/Config | Policy baseline approval |
| D3 | Evidence Pipeline | Log collection, normalization, SIEM integration | Documentation + Configs | Observability readiness |
| D4 | Detection Pack | Detection rules with documentation | SIEM Code/Queries | Detection baseline approval |
| D5 | Runbook Library | Incident response procedures for common scenarios | Markdown/PDF | Operational readiness |
| D6 | Network Design Docs | Egress/SWG/MDM design patterns | Technical Document | Network design approval |
| D7 | Pilot Report | Usage metrics, issues, recommendations | PDF | Pilot closure |
| D8 | Production Rollout Plan | Scaling guidance, change control procedures | Project Plan | Production readiness review |
| D9 | Guardrail Policy Package | Approved pre-call/in-call/post-call policy matrix, tests, and operations workflow | Policy Matrix + Test Evidence | Guardrail readiness review |

### 3.2 Acceptance Criteria

Each deliverable will be considered accepted when:

- **D1 (Deployed Gateway)**:
  - [ ] Gateway deployed on [GATEWAY_HOST]
  - [ ] Health checks passing consistently
  - [ ] Authentication functional (virtual key generation works)
  - [ ] [CUSTOMER] Platform Lead sign-off

- **D2 (Policy Configuration)**:
  - [ ] Model allowlist configured per [PROVIDER_LIST]
  - [ ] Budget enforcement tested and validated
  - [ ] Rate limits configured and tested
  - [ ] Configuration documented and version-controlled

- **D3 (Evidence Pipeline)**:
  - [ ] Gateway logs flowing to normalization layer
  - [ ] SIEM receiving events with <5 minute latency
  - [ ] Dashboard functional with key metrics visible
  - [ ] Alerting configured for critical conditions

- **D4 (Detection Pack)**:
  - [ ] 5–10 detection rules implemented
  - [ ] Rules tested with sample data
  - [ ] No critical false negatives on test cases
  - [ ] Detection logic documented

- **D5 (Runbook Library)**:
  - [ ] Runbooks cover: budget exhaustion, anomalous usage, suspected bypass
  - [ ] Escalation procedures defined
  - [ ] Response steps validated with [CUSTOMER] Security

- **D6 (Network Design Docs)**:
  - [ ] Egress pattern documented (default-deny, gateway-only)
  - [ ] SWG/CASB integration approach specified
  - [ ] MDM configuration guidance provided
  - [ ] Validation test cases defined

- **D7 (Pilot Report)**:
  - [ ] Pilot user usage metrics documented
  - [ ] Issues encountered and resolutions documented
  - [ ] Recommendations for production rollout
  - [ ] Success metrics vs. targets
  - [ ] Explicit `EXPAND`, `REMEDIATE_AND_CONTINUE`, or `NO_GO` recommendation recorded

- **D8 (Production Rollout Plan)**:
  - [ ] Phased rollout approach defined
  - [ ] Change control procedures documented
  - [ ] Rollback procedures defined
  - [ ] Training plan for operations team
  - [ ] Open customer-owned prerequisites listed explicitly

- **D9 (Guardrail Policy Package)**:
  - [ ] Policy matrix approved by [CUSTOMER_SECURITY_LEAD]
  - [ ] Pre-call controls pass deterministic block/mask/allow tests
  - [ ] In-call constraints tested for scoped tools/models
  - [ ] Post-call review process documented (detection + exception workflow)

---

## 4. Timeline and Phases

### 4.1 Phase Breakdown

| Phase | Activities | Duration |
|-------|------------|----------|
| **Setup** | Environment prep, gateway deployment, basic config | Per agreed project plan |
| **Integration** | SIEM integration, evidence pipeline, detection rules | Per agreed project plan |
| **Design** | Egress/SWG patterns, workspace config | Per agreed project plan |
| **Pilot** | User onboarding, monitoring, issue resolution | Per agreed project plan |
| **Hardening** | Security review, HA validation, documentation | Per agreed project plan |

### 4.2 Key Milestones

| Milestone | Target Date | Success Criteria |
|-----------|-------------|------------------|
| M1: Gateway Deployed | Per project plan | Health checks passing, auth functional |
| M2: Pipeline Operational | Per project plan | SIEM receiving events, dashboard live |
| M3: Detection Rules Live | Per project plan | Rules active, alerts configured |
| M4: Pilot Started | Per project plan | Pilot users onboarded and active |
| M5: Pilot Complete | Per project plan | Pilot metrics collected, issues resolved |
| M6: Hardening Complete | Per project plan | Security review passed, docs complete |
| M7: Engagement Complete | Per project plan | All deliverables accepted |

### 4.3 Phase-Gate Requirement

If the engagement includes a customer pilot, it must follow the phase model in `docs/PILOT_EXECUTION_MODEL.md`:

- Qualify
- Charter
- Implement
- Validate Customer Controls
- Decide
- Transition

The delivery team must not skip from implementation output to rollout language without the customer-control validation and decision phases.

---

## 5. Roles and Responsibilities

### 5.1 Project Team

| Role | Name | Responsibilities |
|------|------|------------------|
| Technical Lead | [PROJECT_TECHNICAL_LEAD] | Implementation, integration, technical decisions |
| Security Engineer | [PROJECT_SECURITY_ENGINEER] | Detection rules, runbook development, security review |
| DevOps Engineer | [PROJECT_DEVOPS_ENGINEER] | Deployment automation, pipeline setup |
| Project Manager | [PROJECT_PM] | Project coordination, status reporting |

### 5.2 Customer Team

| Role | Name | Responsibilities |
|------|------|------------------|
| Executive Sponsor | [CUSTOMER_SPONSOR] | Strategic alignment, escalation, acceptance |
| Platform Lead | [CUSTOMER_PLATFORM_LEAD] | Environment provisioning, production access |
| Security Lead | [CUSTOMER_SECURITY_LEAD] | Security review, policy approval, IR alignment |
| SIEM Team Lead | [CUSTOMER_SIEM_LEAD] | SIEM integration support, dashboard creation |
| Network Team Lead | [CUSTOMER_NETWORK_LEAD] | Egress implementation (customer-operated) |
| Pilot User Coordinator | [CUSTOMER_PILOT_COORD] | Pilot user identification and communication |

### 5.3 RACI Matrix

| Activity | Project Tech Lead | Project Security | Customer Platform | Customer Security | Customer Network |
|----------|---------------|--------------|-------------------|-------------------|------------------|
| Gateway Deployment | R/A | C | A/C | I | I |
| SIEM Integration | R | C | C | C | I | A/C |
| Detection Rules | C | R/A | I | A/C | I |
| Network Design | R | C | C | C | A |
| Network Implementation | C | I | I | I | R/A |
| Pilot Execution | R | C | C | I | I |
| Runbook Development | R | R/A | I | A/C | I |

---

## 6. Customer-Provided Inputs and Access

### 6.1 Required Inputs

| Input | Purpose | Timing | Owner |
|-------|---------|--------|-------|
| Approved provider list | Model allowlist configuration | Kickoff | [CUSTOMER_SECURITY_LEAD] |
| Environment specs | Gateway sizing and deployment | Kickoff | [CUSTOMER_PLATFORM_LEAD] |
| SIEM documentation | Integration planning | Integration phase | [CUSTOMER_SIEM_LEAD] |
| Network architecture | Egress design | Design phase | [CUSTOMER_NETWORK_LEAD] |
| Pilot user list | Onboarding planning | Pilot preparation | [CUSTOMER_PILOT_COORD] |
| Provider API keys | Gateway configuration | Setup phase | [CUSTOMER_SECURITY_LEAD] |

### 6.2 System Access

| System | Access Level | Purpose | Timing |
|--------|--------------|---------|--------|
| Gateway environment | Admin/Root | Deployment and configuration | Setup phase |
| SIEM platform | Write/API | Event ingestion, dashboard creation | Integration phase |
| Network devices | Read/Design | Egress pattern design | Design phase |
| Workspace admin | Admin | Workspace configuration | Design phase |

### 6.3 Customer Infrastructure Dependencies

| Infrastructure | Customer Action | Project Support | Target Window |
|----------------|-----------------|-------------|---------------|
| Egress controls | Implement default-deny rules | Design and validate | Per project plan |
| SWG/CASB config | Configure policies per design | Design guidance | Per project plan |
| MDM deployment | Deploy tool configs | Provide config specs | Per project plan |

---

## 7. Data Handling and Security

### 7.1 Data Classification

| Data Type | Classification | Handling |
|-----------|----------------|----------|
| Gateway logs | Internal | Standard log handling |
| API keys | Confidential | Encrypted, access-controlled, rotated post-engagement |
| SIEM data | Internal/Confidential | Standard SIEM handling |
| User data | Confidential | Per [CUSTOMER] data handling policies |

### 7.2 Metadata-Only Default

Standard SIEM integration transmits metadata only:
- Principal identity
- Model/provider used
- Timestamp, token counts, cost
- Policy outcome (allowed/blocked)

**No conversation content or prompts are transmitted by default.**

### 7.3 Transcript Ingestion (If Required)

If prompt/response content ingestion is required:
- Must be explicitly scoped in writing
- Separate restricted storage with enhanced controls
- Defined retention period per customer policy and data classification
- Access logging and approval workflows
- Separate from primary SIEM feed

### 7.4 API Key Handling

- API keys stored in encrypted configuration
- Keys rotated upon engagement completion
- Customer retains ownership of all provider keys
- No keys stored in Project systems post-engagement

---

## 8. Assumptions and Dependencies

### 8.1 Assumptions

1. **Environment Ready**: [CUSTOMER] provides suitable environment (Docker/K8s) at kickoff
2. **Provider Access**: [CUSTOMER] has or obtains AI provider API keys during setup
3. **SIEM Ready**: SIEM platform operational and accessible during integration
4. **Network Team Available**: Network team available for egress implementation during design/validation phases
5. **Pilot Users Committed**: Pilot users identified and committed before pilot execution
6. **No Major Blockers**: Customer change control does not block agreed timeline
7. **Customer Ownership Accepted**: Customer accepts the shared-responsibility boundary in writing before pilot language is used externally

### 8.2 External Dependencies

| Dependency | Impact if Delayed | Mitigation |
|------------|-------------------|------------|
| Environment provisioning | Gateway deployment delay | Remote environment option |
| API keys | Gateway non-functional | Offline mode for testing |
| SIEM access | Pipeline delay | Local log analysis interim |
| Network team availability | Egress validation delay | Documented test cases for later validation |
| Missing control owners | Pilot cannot close credibly | Reframe as implementation only |

---

## 9. Change Control

### 9.1 Change Request Process

1. Either party may submit written change request
2. Project assesses impact according to the engagement change-management SLA
3. Joint review of impact assessment
4. Written approval required for changes affecting timeline or cost

### 9.2 Common Change Types

| Change Type | Impact Assessment | Approval Authority |
|-------------|-------------------|-------------------|
| Provider addition | Incremental scope/cost/timeline impact | Both parties |
| Environment change | Variable | Both parties |
| SIEM platform change | Incremental scope/cost/timeline impact | Both parties |
| Guardrail package expansion | Incremental scope/cost/timeline impact | Both parties |
| Scope expansion | Additional cost/time | Both parties |
| Timeline compression | Resource/cost increase | Both parties |

---

## 10. Exit Criteria

This engagement will be considered complete when:

- [ ] All deliverables (D1–D9) produced and accepted
- [ ] Gateway operational and passing health checks
- [ ] Pilot completed with documented metrics and explicit closeout decision
- [ ] Knowledge transfer to [CUSTOMER] operations team
- [ ] Final invoice submitted and paid

Completion of this engagement does not, by itself, authorize the phrases `enterprise-ready`, `bypass prevention complete`, or `production rollout approved`. Those statements require current evidence and customer sign-off against the pilot execution model.

### 10.1 Definition of Done (per deliverable)

- Deployed: Functional, tested, documented
- Configuration: Tested, version-controlled, reviewed
- Documentation: Complete, reviewed, accepted
- Runbooks: Tested, validated with operations team

### 10.2 Early Termination

Either party may terminate with [NOTICE_PERIOD] written notice. Customer pays for:
- All completed and accepted deliverables
- Work in progress (prorated)
- Non-cancelable commitments

---

## 11. Post-Engagement Support

### 11.1 Warranty Period

[ WARRANTY_PERIOD ] warranty on deliverables for defects in implementation.

### 11.2 Transition to Managed Ops

Optional transition to Managed AI Security Operations (separate SOW):
- Knowledge transfer session
- Runbook handoff
- Monitoring transition
- Ongoing support activation

---

## 12. Signatures

By signing below, the parties agree to the terms of this Statement of Work.

**Project Maintainer:**

| Role | Name | Signature | Date |
|------|------|-----------|------|
| Technical Lead | [PROJECT_TECHNICAL_LEAD] | | |
| Project Manager | [PROJECT_PM] | | |
| Practice Lead | [PROJECT_PRACTICE_LEAD] | | |

**[CUSTOMER]:**

| Role | Name | Signature | Date |
|------|------|-----------|------|
| Executive Sponsor | [CUSTOMER_SPONSOR] | | |
| Platform Lead | [CUSTOMER_PLATFORM_LEAD] | | |

---

## Appendix A: Technical Specifications

### Gateway Requirements

| Component | Specification |
|-----------|---------------|
| Platform | Docker or Kubernetes |
| Database | PostgreSQL 15+ |
| Network | Outbound HTTPS to providers |
| Storage | 50GB minimum for logs |

### Supported Providers

| Provider | API-Key Mode | Subscription Mode |
|----------|--------------|-------------------|
| OpenAI | ✓ | Via workspace |
| Anthropic | ✓ | Via gateway |
| Google (Gemini) | ✓ | Via workspace |

## Appendix B: Glossary

See `SOW_AI_USAGE_EXPOSURE_ASSESSMENT.md` Appendix A, plus:

| Term | Definition |
|------|------------|
| HA | High Availability |
| MDM | Mobile Device Management |
| OTEL | OpenTelemetry |
| SWG | Secure Web Gateway |
| CASB | Cloud Access Security Broker |

---

*Template Version 1.0 — Project Maintainer*
