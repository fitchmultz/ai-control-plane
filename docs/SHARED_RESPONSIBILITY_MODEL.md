# Shared Responsibility Model

This document sets the default operating boundary for implementation-only engagements and managed-service engagements.

## Responsibility Baseline

| Domain | Customer | Shared | Provider |
| --- | --- | --- | --- |
| Cloud account, subscriptions, and tenant ownership | Owns the AWS/Azure/GCP account, billing, and tenant-level governance | Reviews architecture choices and change windows | Advises on required platform prerequisites |
| Network perimeter and egress controls | Owns firewall, DNS, SWG, CASB, proxies, and device controls | Validates required endpoint lists and rollout sequencing | Supplies sanctioned endpoint inventory and enforcement guidance |
| Identity provider and enterprise access policy | Owns IdP, MFA, device trust, and user lifecycle policy | Maps required claims/groups to platform roles | Implements supported claim mapping and tests attribution |
| AI Control Plane application behavior | Approves change windows and desired policies | Reviews release and incident outcomes | Owns defects, fixes, release quality, and operator workflows in the delivered application |
| SIEM onboarding and evidence retention | Owns retention policy, case management, and downstream alert handling | Tunes fields, queries, and escalation handoffs | Provides normalized feed, detection logic, and runbooks |
| FinOps allocation and chargeback policy | Owns cost-center taxonomy and budget policy | Confirms report consumers and chargeback cadence | Implements attribution mappings and reporting workflow |
| Future multi-tenant managed-service boundary | Owns tenant legal/entity definition, customer invoice recipient, and customer-facing risk acceptance | Reviews workspace isolation model and escalation boundaries | Supplies the design-only tenant package today; would own runtime tenant enforcement only after future validation |
| Day-2 operations under managed-service agreement | Provides access, approvals, and dependency ownership | Participates in incidents crossing customer-controlled systems | Owns agreed operational tasks, application incidents, and maintenance within the SOW |
| Third-party outages and infrastructure faults | Owns vendor relationships for customer-controlled infrastructure | Coordinates impact assessment and communication | Owns application-side mitigation only when the issue is within the delivered platform |

## Engagement Modes

### Implementation Only

Use this mode when the customer buys deployment and handoff, not ongoing operations.

- Customer owns steady-state operations after handoff.
- Provider owns implementation quality and remediation of defects discovered during warranty or explicitly contracted support windows.
- Escalations involving cloud/network/IdP remain customer-owned.

### Managed Operations

Use this mode when the SOW includes ongoing maintenance and operational response.

- Provider owns the AI Control Plane application layer and agreed operational tasks.
- Customer still owns underlying cloud tenancy, network perimeter, IdP, and vendor workspace administration unless the SOW says otherwise.
- Shared incident handling is required whenever evidence points to both application and customer-controlled infrastructure.

## Incident Ownership Model

Use this table when the buyer asks "who owns the incident?"

| Incident type | Provider default role | Customer default role | Decision owner |
|---|---|---|---|
| Application defect in the delivered control plane | diagnose, fix, release, communicate | provide access and approvals | Provider for remediation; customer for rollout approval |
| Misconfiguration in delivered policy layer | investigate, propose correction, implement if in scope | approve policy intent and business exceptions | Shared |
| SIEM ingestion failure inside customer environment | help isolate field/path issue and provide expected payloads | repair ingestion plumbing, retention, routing | Customer |
| Cloud, DNS, firewall, SWG, CASB, or proxy issue | identify likely dependency and provide evidence | repair customer-controlled infrastructure | Customer |
| IdP, MFA, device-posture, or claim-mapping issue | validate expected mapping and app-side behavior | repair identity policy and lifecycle process | Customer |
| Vendor workspace admin or export issue | provide guidance on required evidence and parser expectations | repair workspace settings, admin access, or vendor escalation | Customer |
| Cross-boundary major incident | coordinate timeline, evidence, and application actions | coordinate customer-side containment and executive decisions | Shared, with customer owning business-risk decisions |

## Managed-Service Exclusions By Default

Unless the SOW says otherwise, managed operations does not include:

- owning the customer's cloud account
- owning firewall, SWG, CASB, DNS, or proxy changes
- owning the customer's IdP or device management program
- owning vendor workspace licensing or enterprise admin changes
- making business-risk acceptance decisions for unresolved governance gaps

## Contract Rule

The SOW can tighten or expand provider scope, but it should never blur these baseline ownership lines. The fastest way to lose enterprise trust is to imply ownership of systems the provider does not actually control.
