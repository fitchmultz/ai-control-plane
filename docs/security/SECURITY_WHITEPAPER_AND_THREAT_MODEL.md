# Security Whitepaper and Threat Model

**Document ID**: `acp-security-whitepaper-threat-model`  
**Version**: 1.0.0  
**Last Updated**: 2026-03-19  
**Status**: Canonical  
**Claim Boundary**: Host-first validated reference surface; customer-environment controls remain required for identity, network, SIEM operations, retention, and workspace governance.

---

## Executive Summary

AI Control Plane is a **host-first Docker reference implementation** for enterprise AI governance.
It combines a LiteLLM gateway, PostgreSQL-backed attribution and policy state, typed operator workflows, detection content, and buyer-safe evidence artifacts.

This document explains:

- the architecture being secured
- the protected assets and trust boundaries
- the attacker capabilities and abuse paths that matter most
- the mitigations ACP provides today
- the residual risks and customer-owned dependencies that remain
- the control-ownership split required for truthful deployment and procurement conversations

This is a **security whitepaper and threat model**, not a certification, penetration-test report, or formal attestation.
It should be paired with the [Compliance Crosswalk](../COMPLIANCE_CROSSWALK.md), [Shared Responsibility Model](../SHARED_RESPONSIBILITY_MODEL.md), and [Go-To-Market Scope](../GO_TO_MARKET_SCOPE.md).

## Scope and validated boundary

This artifact stays inside the repository's current support and truth boundary:

- **Validated now:** host-first Docker reference implementation, typed operator workflows, detection/SIEM validation, readiness evidence, pilot closeout artifacts, disaster-recovery drills, customer-operated HA failover-drill evidence tooling, and an AWS-first incubating cloud deployment package validated through explicit Terraform fmt, validate, and validation-only plan workflows plus AWS hardening guidance and a basic cost-estimation model.
- **Conditionally ready:** customer pilots on controlled Linux hosts where the customer validates identity, network, SIEM, retention, secrets handling, and workspace/browser governance in the target environment.
- **Not yet validated:** Azure/GCP cloud deployment claims, AWS applied/runtime cloud-operation evidence beyond the explicit validation package, multi-tenant managed-service claims, and universal bypass prevention.

## Architecture overview

### System summary

AI Control Plane is centered on a LiteLLM gateway that enforces approved-model usage, attribution, budgets, rate limits, and optional guardrails.
A PostgreSQL database stores spend logs, verification tokens, and budget state.
Optional overlays add TLS ingress, deterministic DLP, and a browser-based UI.

### Component view

```text
                       Customer-owned DNS / LB / network controls
                                         |
                                         v
                              [ Optional Caddy TLS ]
                                         |
                                         v
Browser / API clients --> [ LiteLLM Gateway ] <--> [ PostgreSQL ]
        |                         |   |   |
        |                         |   |   +--> Spend logs / keys / budgets
        |                         |   +------> Detection and evidence pipeline
        |                         +----------> Optional Presidio DLP sidecars
        |
        +--> Optional LibreChat UI --> trusted user context --> LiteLLM
```

### Operator and evidence layer

```text
Operators --> make / acpctl --> deploy, validate, health, smoke, doctor,
                               key lifecycle, backup/restore, readiness evidence,
                               release bundles, pilot closeout, failover-drill evidence
```

## Protected assets

| Asset | Why it matters | Default sensitivity |
| --- | --- | --- |
| Virtual keys and gateway credentials | Control access to routed model usage and attribution scope | High |
| `/etc/ai-control-plane/secrets.env` | Canonical host-production secret source | High |
| PostgreSQL usage and policy tables | Holds attribution, spend, budgets, and control evidence | High |
| Detection outputs and webhook events | Drive monitoring, response, and governance workflows | Medium/High |
| Normalized evidence feed | Supports SIEM ingestion and downstream investigations | Medium |
| Deployment and runtime configuration | Defines exposure, overlays, policies, and trust boundaries | High |
| Operator workflows and generated artifacts | Provide release, readiness, and recovery evidence | Medium |
| Customer identity context forwarded through trusted surfaces | Enables attribution without per-user upstream keys | High |

## Trust boundaries

### Boundary map

```text
[Trust 0] Browser / API client / external caller
   Untrusted until authenticated and re-bound by a trusted server surface

[Trust 1] LibreChat / trusted application session layer
   Authenticates users and forwards trusted user context

[Trust 2] LiteLLM Gateway and typed ACP operator surface
   Enforces policy, attribution, budgets, detections, and workflow orchestration

[Trust 3] Host runtime and PostgreSQL state
   Stores enforcement state, spend logs, budgets, and evidence artifacts

[Trust 4] Upstream model providers / customer SIEM / customer infra
   External dependencies outside ACP direct control
```

### Boundary details

| Boundary | What crosses it | Main risk | Primary mitigation |
| --- | --- | --- | --- |
| Browser/client -> LibreChat or gateway | Prompts, user identity claims, API tokens | Spoofed identity, misuse, prompt abuse | Trusted auth pattern, shared responsibility boundary, canonical auth architecture |
| LibreChat -> LiteLLM | Shared service key plus trusted `user` context | Identity confusion or broken attribution | Documented trust model, deterministic attribution precedence, optional enterprise strictness |
| Gateway -> PostgreSQL | Spend logs, key metadata, budgets, audit-relevant state | Tampering, loss of attribution, recovery risk | Single canonical DB service, backup/restore workflows, typed status/doctor surfaces |
| Gateway -> upstream providers | Approved requests, model/provider metadata | Data exfiltration, provider drift, cost abuse | Approved-model policy, budgets/rate limits, optional DLP guardrails |
| ACP -> customer SIEM / webhook receivers | Metadata-first events and detections | Incomplete ingestion, alert-routing failure | Normalized schema, query validation, webhook catalog, customer-owned retention/routing |
| Host runtime -> customer network perimeter | TLS ingress, outbound model/API connectivity | Bypass, exposure, misrouted traffic | Customer-owned DNS/LB/SWG/CASB/firewall controls plus ACP exposure rules |

## Threat actors and attacker capabilities

| Actor | Capability | Example abuse goal |
| --- | --- | --- |
| Normal user attempting policy bypass | Uses direct SaaS endpoint, personal API key, or unmanaged device | Avoid gateway controls and attribution |
| Compromised or careless internal user | Uses a valid ACP key or trusted UI session improperly | Overspend, access unapproved models, misuse approved access |
| Prompt attacker / model abuse actor | Crafts prompt injection, sensitive-data solicitation, or tool-abuse prompts | Extract secrets or cause unsafe model behavior |
| Stolen key / credential holder | Reuses leaked gateway or virtual key material | Unattributed or malicious routed usage |
| Malicious or mistaken operator | Misconfigures deployment, exposes services, weakens policy, or mishandles secrets | Expand attack surface or degrade auditability |
| Supply-chain adversary / vulnerable dependency | Exploits known vulnerable packages or image layers | Gain foothold or degrade trust in runtime components |
| Infrastructure or availability attacker | Disrupts host, disk, database, Docker runtime, or customer network path | Cause outage or force recovery event |

## Threat scenarios and abuse paths

### 1. Direct bypass of inline gateway enforcement

**Scenario:** a user sends prompts directly to vendor SaaS or uses a personal API key instead of routed ACP paths.

**Why it matters:** routed controls can be strong while direct usage remains outside inline enforcement.

**Mitigations provided by ACP:**

- documented enforce-vs-detect boundary
- normalized evidence patterns and SIEM integration for detective coverage
- browser/workspace governance guidance and pilot ownership mapping
- compliance crosswalk and buyer-objection guidance that prevent overstated claims

**Customer-owned controls still required:**

- SWG, CASB, firewall, DNS, proxy, and endpoint controls
- workspace administration and browser governance
- managed-device policy

**Residual risk:** ACP alone does not guarantee bypass prevention.

### 2. Identity spoofing or broken user attribution

**Scenario:** an attacker or broken integration injects untrusted user identity values, causing inaccurate attribution or impersonation.

**Mitigations provided by ACP:**

- documented trust boundary between browser claims and server-authenticated context
- trusted `user` propagation model via LibreChat -> LiteLLM
- deterministic attribution precedence
- enterprise-enhanced option for stricter user-field enforcement
- role-shaped key lifecycle and inspection workflows

**Evidence anchors:**

- [ENTERPRISE_AUTH_ARCHITECTURE.md](ENTERPRISE_AUTH_ARCHITECTURE.md)
- [../policy/ROLE_BASED_ACCESS_CONTROL.md](../policy/ROLE_BASED_ACCESS_CONTROL.md)

**Residual risk:** customer IdP assurance, MFA posture, and directory hygiene remain external to ACP.

### 3. Abuse of approved routed access

**Scenario:** a legitimate user or stolen key uses the routed path excessively or outside policy intent.

**Mitigations provided by ACP:**

- approved-model allowlists
- budget and rate-limit controls
- key rotation, inspection, and revocation workflows
- request attribution in spend logs
- operational health and status checks

**Residual risk:** ACP can constrain and attribute routed usage, but policy intent and spend tolerances still require customer governance.

### 4. Prompt injection, sensitive-data leakage, or unsafe content flow

**Scenario:** a user prompt or external content attempts prompt injection, sensitive-data extraction, or policy evasion.

**Mitigations provided by ACP:**

- Presidio-backed deterministic DLP path for supported environments
- detection content covering DLP blocks and prompt-injection signals
- metadata-first evidence pipeline by default
- explicit recommendation to isolate transcript-bearing exports when used

**Evidence anchors:**

- [DETECTION.md](DETECTION.md)
- [SIEM_INTEGRATION.md](SIEM_INTEGRATION.md)

**Residual risk:** DLP enforcement depends on runtime support and configuration; offline and non-supported environments may only provide rehearsal-level proof.

### 5. Stolen virtual key or operator secret

**Scenario:** attacker obtains a virtual key, service key, or host-side secret file.

**Mitigations provided by ACP:**

- typed key inventory, rotation, and revocation workflows
- canonical secret-source contract
- host-production config validation
- private local artifact handling defaults
- hardened container/runtime posture

**Residual risk:** customer secret distribution, storage, and host access controls remain decisive.

### 6. Detection or SIEM blind spot

**Scenario:** detections are configured but ingestion, routing, or analyst workflow fails in the customer environment.

**Mitigations provided by ACP:**

- normalized schema and query pack
- detection-rule validation
- SIEM schema/query validation
- webhook event contract
- readiness evidence for current repo state

**Residual risk:** ingestion plumbing, retention, and SOC response remain customer-owned.

### 7. Supply-chain exploitation or dependency vulnerability

**Scenario:** a vulnerable dependency or image layer is exploited.

**Mitigations provided by ACP:**

- hardened images
- `no-new-privileges`
- dropped capabilities
- supply-chain policy and security gates
- canonical CVE remediation and risk-acceptance policy
- tracked known limitations and dated review disclosures

**Residual risk:** open CVEs still exist and are governed, not eliminated.

### 8. Host, disk, database, or runtime failure

**Scenario:** the single host, local storage, Docker runtime, or embedded database fails.

**Mitigations provided by ACP:**

- backup timer and retention workflows
- restore drill
- off-host recovery validation
- customer-operated HA failover-drill evidence surface
- explicit failure-domain documentation

**Residual risk:** the primary validated topology remains single-node and does not provide automatic failover.

## Mitigations and control surfaces

### Policy enforcement

- approved-model allowlists
- virtual-key attribution
- budgets, rate limits, and spend controls
- role-shaped key issuance patterns

### Detection and response

- DR-001 through DR-010 detection pack
- auto-response patterns for selected detections
- webhook notifications and SIEM-ready normalized events
- typed diagnostics through `health`, `status`, `doctor`, and `smoke`

### Identity and access

- trusted identity propagation model
- OSS-first and enterprise-enhanced auth profiles
- documented RBAC role model
- key rotation and revocation workflows

### Data handling and evidence

- metadata-first evidence pipeline by default
- prompt/response content excluded by default
- readiness evidence, release bundles, and pilot closeout artifacts
- private local artifact permissions and inventories

### Deployment and recovery

- host-first deployment contract
- TLS overlay for remote ingress
- production config validation
- backup, restore, off-host drill, and failover-drill workflows
- upgrade planning and rollback artifacts

## Detection coverage highlights

| Threat area | Coverage example | Response path |
| --- | --- | --- |
| Unapproved model use | DR-001 | Investigate, optionally suspend compromised key |
| Token spikes or rapid request bursts | DR-002, DR-005 | Review abuse or workload anomaly |
| Elevated error rates / auth failures | DR-003, DR-006 | Triage service or credential issue; suspend key where configured |
| Budget exhaustion | DR-004 | Prevent overspend and trigger review |
| DLP and prompt-injection signals | DR-007, DR-010 | Investigate attempted exfiltration or unsafe request pattern |

## Residual risks and explicit limitations

These are disclosed intentionally and should be part of buyer and operator review:

1. **Single-node topology remains the primary validated baseline.** Host, disk, database, or Docker failure can still cause a full outage.
2. **Automatic failover is not part of the supported contract.** ACP now ships failover-drill evidence tooling, not automatic HA orchestration.
3. **Bypass prevention is customer-dependent.** Without customer network and endpoint controls, direct vendor usage may remain possible.
4. **Broad cloud-production claims are still not validated.** This whitepaper recognizes the AWS-first incubating Terraform validation package, but it remains grounded in the host-first support boundary and does not claim named-account AWS runtime proof or Azure/GCP validation.
5. **Open CVEs remain under governance.** Current accepted-risk status, expiry windows, and quarterly review history live in [CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md](CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md), [CVE_REVIEW_LOG.md](CVE_REVIEW_LOG.md), [KNOWN_LIMITATIONS.md](../KNOWN_LIMITATIONS.md), and [`demo/config/supply_chain_vulnerability_policy.json`](../../demo/config/supply_chain_vulnerability_policy.json).
6. **Metadata-first evidence is intentional but incomplete for transcript-centric investigations.** If customers ingest transcript-bearing exports, they need additional controls and retention discipline.
7. **Offline and lab modes are not equivalent to production proof for every guardrail feature.** Especially for DLP and exact token-accounting behavior.

## Control ownership

| Domain | Customer | Shared | Provider (ACP) |
| --- | --- | --- | --- |
| Network perimeter, egress, and bypass prevention | Own firewall, DNS, SWG, CASB, proxy, and managed-device controls | Validate required AI endpoints and rollout sequencing | Provide endpoint inventory, architecture boundary, and operator guidance |
| Identity and access policy | Own IdP, MFA, lifecycle policy, device trust, and workspace admin settings | Map claims/groups to ACP-facing roles and trusted attribution model | Implement supported auth pattern, key lifecycle, and role guidance |
| Application-layer policy enforcement | Approve policy intent and rollout decisions | Review enforcement outcomes and exceptions | Provide approved-model, budget, rate-limit, attribution, and detection surfaces |
| SIEM onboarding and incident routing | Own retention, routing, analyst workflow, and case management | Tune fields, queries, and escalation handoffs | Provide normalized schema, query pack, webhook catalog, and detection content |
| Backup, recovery, and continuity | Own off-host retention, replacement-host readiness, and external traffic cutover | Run restore and failover drills with named operators | Provide typed drill workflows, evidence bundles, and truthful topology limits |
| Supply-chain and risk acceptance | Own enterprise risk acceptance and compensating-control process | Review open findings and mitigation status together | Publish current findings, gates, and risk-boundary documentation |

## Compliance and governance tie-in

For detailed SOC 2, ISO 27001, and NIST-style control mapping, use the canonical [Compliance Crosswalk](../COMPLIANCE_CROSSWALK.md).
This whitepaper explains the architecture and threat logic behind those mappings; the crosswalk explains how ownership and evidence align to external control families.

## External review readiness

This whitepaper is a key input to outside review, but it is not a substitute for outside review.
Use [EXTERNAL_REVIEW_READINESS.md](EXTERNAL_REVIEW_READINESS.md) as the canonical pre-review packet and claim-boundary guide when preparing a third-party assessor or answering diligence questions about independent validation status.

## Recommended validation and evidence workflow

For a current buyer-safe packet, pair this whitepaper with:

- `make ci`
- `make validate-detections`
- `make validate-siem-queries`
- `make security-gate`
- `make readiness-evidence`
- `make release-bundle`

For recovery and availability review when applicable:

- `make dr-drill`
- `make db-off-host-drill OFF_HOST_RECOVERY_MANIFEST=...`
- `make ha-failover-drill HA_FAILOVER_MANIFEST=...`

## References

- [../COMPLIANCE_CROSSWALK.md](../COMPLIANCE_CROSSWALK.md)
- [../GO_TO_MARKET_SCOPE.md](../GO_TO_MARKET_SCOPE.md)
- [../SHARED_RESPONSIBILITY_MODEL.md](../SHARED_RESPONSIBILITY_MODEL.md)
- [../KNOWN_LIMITATIONS.md](../KNOWN_LIMITATIONS.md)
- [CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md](CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md)
- [CVE_REVIEW_LOG.md](CVE_REVIEW_LOG.md)
- [../technical-architecture.md](../technical-architecture.md)
- [ENTERPRISE_AUTH_ARCHITECTURE.md](ENTERPRISE_AUTH_ARCHITECTURE.md)
- [DETECTION.md](DETECTION.md)
- [SIEM_INTEGRATION.md](SIEM_INTEGRATION.md)
- [WEBHOOK_EVENTS.md](WEBHOOK_EVENTS.md)
- [../deployment/SINGLE_TENANT_PRODUCTION_CONTRACT.md](../deployment/SINGLE_TENANT_PRODUCTION_CONTRACT.md)
- [../deployment/DISASTER_RECOVERY.md](../deployment/DISASTER_RECOVERY.md)
- [../deployment/HA_FAILOVER_TOPOLOGY.md](../deployment/HA_FAILOVER_TOPOLOGY.md)
- [../deployment/HA_FAILOVER_RUNBOOK.md](../deployment/HA_FAILOVER_RUNBOOK.md)
- [../ENTERPRISE_BUYER_OBJECTIONS.md](../ENTERPRISE_BUYER_OBJECTIONS.md)
