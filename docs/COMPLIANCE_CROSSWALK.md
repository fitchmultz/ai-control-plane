# Compliance Crosswalk

This document is a **control-mapping reference**, not a certification, audit report, or attestation.

Use it to scope ownership, due-diligence questions, and evidence planning for customer pilots and production evaluations.
Do **not** present this repository as automatically compliant, certified, or auditor-approved because of this mapping alone.

## Scope and claim boundary

This crosswalk stays inside the current validated claim boundary:

- **Validated now:** host-first Docker reference implementation, typed operator workflows, readiness evidence, pilot closeout artifacts, detection and SIEM contract validation, backup and recovery workflows, and the active-passive failover drill evidence surface.
- **Conditionally ready:** customer pilots on controlled Linux hosts where customer-owned identity, network, SIEM, retention, and workspace controls are validated.
- **Not yet validated:** AWS/cloud-production enforcement claims, multi-tenant managed-service claims, and universal browser-bypass prevention.

## How to read this document

Ownership uses the repository's canonical three-column model:

- **Customer** — controls the customer must implement and operate in their own environment
- **Shared** — controls that require both ACP and customer processes/evidence
- **Provider (ACP)** — capabilities and evidence this repository provides directly

A mapped control means ACP can contribute evidence or implementation support for that control area. It does **not** mean ACP alone satisfies the control.

## Evidence anchors available now

Use these canonical commands and docs when building evidence packets:

- `make ci`
- `make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env`
- `make validate-detections`
- `make validate-siem-queries`
- `make security-gate`
- `make readiness-evidence`
- `make release-bundle`
- `make db-off-host-drill OFF_HOST_RECOVERY_MANIFEST=...`
- `make ha-failover-drill HA_FAILOVER_MANIFEST=...`

Primary supporting documents:

- [GO_TO_MARKET_SCOPE.md](GO_TO_MARKET_SCOPE.md)
- [SHARED_RESPONSIBILITY_MODEL.md](SHARED_RESPONSIBILITY_MODEL.md)
- [PILOT_CONTROL_OWNERSHIP_MATRIX.md](PILOT_CONTROL_OWNERSHIP_MATRIX.md)
- [SECURITY_GOVERNANCE.md](SECURITY_GOVERNANCE.md)
- [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md)

---

## SOC 2 Type II crosswalk

Focus areas below use the Trust Services Criteria families most likely to come up in buyer diligence.

| SOC 2 criteria | Control intent | Customer | Shared | Provider (ACP) | Evidence anchors | Notes / limits |
| --- | --- | --- | --- | --- | --- | --- |
| CC6.1, CC6.2, CC6.3 | Logical access, authorization, least privilege | Own IdP, MFA, joiner/mover/leaver policy, device trust, and workspace administration | Map enterprise identity claims and role expectations into the deployment | Provides role-shaped key workflows, trusted attribution model guidance, and documented enterprise auth architecture | [policy/ROLE_BASED_ACCESS_CONTROL.md](policy/ROLE_BASED_ACCESS_CONTROL.md), [security/ENTERPRISE_AUTH_ARCHITECTURE.md](security/ENTERPRISE_AUTH_ARCHITECTURE.md), `make validate-config` | ACP supports control implementation patterns; it is not an IdP or workforce IAM system |
| CC6.6 | Restrict privileged access and administrative actions | Own host sudo policy, SSH key custody, and admin approval workflow | Validate who can run host deployment, backup, restore, and key operations | Provides typed operator workflows for host deploy, key rotation, backup, restore, and readiness evidence | [DEPLOYMENT.md](DEPLOYMENT.md), [deployment/PRODUCTION_HANDOFF_RUNBOOK.md](deployment/PRODUCTION_HANDOFF_RUNBOOK.md), `make ci` | Privileged host access remains customer-controlled |
| CC7.2, CC7.3 | Monitor system components and detect anomalies | Own SIEM retention, alert routing, case management, and SOC response | Tune detections, thresholds, and escalation handoffs during pilot and production rollout | Provides normalized evidence schema, SIEM mappings, webhook patterns, and typed validation for detections and SIEM contracts | [security/SIEM_INTEGRATION.md](security/SIEM_INTEGRATION.md), `make validate-detections`, `make validate-siem-queries` | Customer ingestion and analyst workflow are still environment-dependent |
| CC7.4 | Respond to detected incidents | Own incident command, customer-side containment, and business-risk decisions | Coordinate cross-boundary investigations using shared evidence | Provides attribution data, policy outcomes, key rotation/revocation, and runbooks for application-layer response | [SHARED_RESPONSIBILITY_MODEL.md](SHARED_RESPONSIBILITY_MODEL.md), [RUNBOOK.md](RUNBOOK.md), `make key-revoke ALIAS=<alias>` | Customer owns cloud, network, IdP, and vendor-side incidents by default |
| CC8.1 | Change management for deployed services and controls | Own change windows, host inventory approvals, and target-environment rollout signoff | Validate upgrade, rollback, and production config changes before release | Provides typed upgrade planning, rollback artifacts, config validation, release bundles, and readiness evidence | [deployment/UPGRADE_MIGRATION.md](deployment/UPGRADE_MIGRATION.md), [release/READINESS_EVIDENCE_WORKFLOW.md](release/READINESS_EVIDENCE_WORKFLOW.md), `make release-bundle` | Change approval governance remains customer-specific |
| CC9.2 | Risk mitigation for business disruption and vendor dependence | Own business continuity requirements, off-host backup policy, and external traffic-management decisions | Rehearse recovery and failover evidence with named owners | Provides backup timer workflow, scratch-restore drills, off-host recovery validation, HA failover drill surface, and explicit topology truth | [deployment/DISASTER_RECOVERY.md](deployment/DISASTER_RECOVERY.md), [deployment/HA_FAILOVER_RUNBOOK.md](deployment/HA_FAILOVER_RUNBOOK.md), `make dr-drill`, `make db-off-host-drill`, `make ha-failover-drill` | Single-node baseline and no automatic failover remain explicit limitations |
| A1.2, A1.3 | Availability monitoring and recovery preparedness | Own production hosting capacity, DNS/LB/VIP control, and customer SLAs | Validate target-environment recovery and failover exercises | Provides health, smoke, doctor, release, backup, and failover drill command surfaces | [DEPLOYMENT.md](DEPLOYMENT.md), [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md), `make health`, `make prod-smoke` | ACP does not publish a universal SLA or automatic HA guarantee |
| C1.1, C1.2 | Confidentiality of governed metadata and operator artifacts | Own data classification, retention, and encryption policies in the customer environment | Align metadata-only evidence handling to customer requirements | Provides metadata-first evidence design, secrets-handling guidance, and local private artifact generation | [security/SIEM_INTEGRATION.md](security/SIEM_INTEGRATION.md), [ARTIFACTS.md](ARTIFACTS.md), [deployment/SINGLE_TENANT_PRODUCTION_CONTRACT.md](deployment/SINGLE_TENANT_PRODUCTION_CONTRACT.md) | Prompt/response content is excluded by default, but customer exports may still need separate restricted handling |

---

## ISO/IEC 27001:2022 crosswalk

The mapping below uses Annex A control themes buyers commonly ask about during architecture and procurement review.

| ISO 27001:2022 control | Control intent | Customer | Shared | Provider (ACP) | Evidence anchors | Notes / limits |
| --- | --- | --- | --- | --- | --- | --- |
| A.5.7, A.5.8 | Threat intelligence and project security planning | Own enterprise risk register, control acceptance, and project governance | Use ACP findings, roadmap, and known limitations in risk review | Provides documented architecture boundaries, roadmap discipline, and known limitations register | [ENTERPRISE_STRATEGY.md](ENTERPRISE_STRATEGY.md), [ROADMAP.md](ROADMAP.md), [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md) | This is planning support, not a formal ISMS program |
| A.5.23 | Information security for use of cloud services | Own tenant configuration, cloud service approvals, and cloud-native controls | Validate customer cloud control stack during pilots or future AWS work | Provides host-first control surface and explicitly marks cloud-production claims as not yet validated | [GO_TO_MARKET_SCOPE.md](GO_TO_MARKET_SCOPE.md), [DEPLOYMENT.md](DEPLOYMENT.md) | AWS/cloud path is still a roadmap item, not a validated support claim |
| A.5.30 | ICT readiness for business continuity | Own business continuity objectives and recovery time commitments | Run recovery and failover drills with customer operators | Provides recovery/failover runbooks, off-host drill validation, and evidence bundles | [deployment/DISASTER_RECOVERY.md](deployment/DISASTER_RECOVERY.md), [deployment/HA_FAILOVER_RUNBOOK.md](deployment/HA_FAILOVER_RUNBOOK.md) | ACP helps prove execution paths; customer still owns continuity policy and external dependencies |
| A.6.3 | Information security awareness, education, and training | Own workforce training, acceptable-use policy, and admin training program | Use ACP runbooks and pilot materials for operator onboarding | Provides operator docs, buyer-safe runbooks, and pilot handoff materials | [RUNBOOK.md](RUNBOOK.md), [ENTERPRISE_PILOT_PACKAGE.md](ENTERPRISE_PILOT_PACKAGE.md), [templates/PILOT_OPERATOR_HANDOFF_CHECKLIST.md](templates/PILOT_OPERATOR_HANDOFF_CHECKLIST.md) | ACP is not a substitute for a customer training program |
| A.8.2 | Privileged access rights | Own host, cloud, and directory-level privileged-access controls | Validate ACP administrative role boundaries and workflow approvals | Provides role-aware key lifecycle, host deployment surfaces, and documented admin boundaries | [policy/ROLE_BASED_ACCESS_CONTROL.md](policy/ROLE_BASED_ACCESS_CONTROL.md), [SHARED_RESPONSIBILITY_MODEL.md](SHARED_RESPONSIBILITY_MODEL.md), `make key-list` | ACP administrative access does not replace enterprise PAM |
| A.8.9, A.8.10 | Configuration management and secure deletion | Own base-host lifecycle, disk handling, and infrastructure-level disposal | Validate tracked config and release changes before deployment | Provides typed config validation, deterministic deployment assets, and rollback/recovery paths | [deployment/SINGLE_TENANT_PRODUCTION_CONTRACT.md](deployment/SINGLE_TENANT_PRODUCTION_CONTRACT.md), `make validate-config`, `make upgrade-check` | Secure media disposal and host decommissioning remain customer-run |
| A.8.15 | Logging | Own retention, storage, and downstream access policy for logs | Align normalized evidence with customer SIEM schema and retention requirements | Provides normalized evidence model, policy outcome fields, and metadata-first telemetry guidance | [security/SIEM_INTEGRATION.md](security/SIEM_INTEGRATION.md), `make validate-siem-queries` | Logging coverage is meaningful only when customer ingestion is operational |
| A.8.16 | Monitoring activities | Own 24x7 monitoring process and escalation rules | Use ACP detections plus customer telemetry to monitor governance posture | Provides detection rules, webhook events, health, doctor, and readiness evidence workflows | [security/WEBHOOK_EVENTS.md](security/WEBHOOK_EVENTS.md), [security/DETECTION.md](security/DETECTION.md), `make validate-detections` | Customer monitoring scope must include systems outside ACP |
| A.8.20 | Network security | Own firewall, DNS, SWG, CASB, proxy, and endpoint controls | Validate allowed endpoints and bypass-prevention boundaries in writing | Provides network contract artifacts and explicitly documented enforce-vs-detect boundaries | [deployment/network_firewall_contract.md](deployment/network_firewall_contract.md), [ENTERPRISE_BUYER_OBJECTIONS.md](ENTERPRISE_BUYER_OBJECTIONS.md) | ACP does not claim universal browser-bypass prevention without customer controls |

---

## NIST-style controls crosswalk

This mapping uses NIST SP 800-53 Rev. 5 control families as the primary shorthand. It is also useful as a starting point for NIST CSF 2.0 discussions, especially across Govern, Protect, Detect, Respond, and Recover functions.

| NIST family / controls | Control intent | Customer | Shared | Provider (ACP) | Evidence anchors | Notes / limits |
| --- | --- | --- | --- | --- | --- | --- |
| AC-2, AC-3, AC-6 | Account management, authorization, least privilege | Own identity lifecycle, group governance, device trust, and SaaS admin settings | Map enterprise roles and approval patterns into ACP-facing workflows | Provides role-shaped key issuance, approved-model governance, and documented user-attribution patterns | [policy/ROLE_BASED_ACCESS_CONTROL.md](policy/ROLE_BASED_ACCESS_CONTROL.md), [policy/APPROVED_MODELS.md](policy/APPROVED_MODELS.md), [security/ENTERPRISE_AUTH_ARCHITECTURE.md](security/ENTERPRISE_AUTH_ARCHITECTURE.md) | ACP controls routed usage; unmanaged direct SaaS access still requires customer controls |
| IA-2, IA-5 | Identification and authentication | Own MFA policy, IdP assurance, and secret distribution process | Validate claim mapping, service-key handling, and host secret placement | Provides canonical secret-source contract, trusted context propagation model, and typed env/config validation | [deployment/SINGLE_TENANT_PRODUCTION_CONTRACT.md](deployment/SINGLE_TENANT_PRODUCTION_CONTRACT.md), `make validate-config-production`, [security/ENTERPRISE_AUTH_ARCHITECTURE.md](security/ENTERPRISE_AUTH_ARCHITECTURE.md) | Customer still owns IdP strength and secret custody outside repo workflows |
| AU-2, AU-6, AU-12 | Audit events, review, and generation | Own SIEM ingestion, retention, and review process | Tune queries and investigation playbooks for the environment | Provides normalized schema, validated detections, query packs, and metadata-only audit evidence paths | [security/SIEM_INTEGRATION.md](security/SIEM_INTEGRATION.md), `make validate-detections`, `make validate-siem-queries` | Audit completeness depends on successful customer-side ingestion and retention |
| CM-2, CM-3, CM-6 | Baseline configuration and change control | Own target-host inventory values, change approvals, and environment-specific hardening | Validate tracked config, supported overlays, and release cutovers together | Provides typed config contract checks, host-first deployment assets, release bundles, and readiness evidence | [DEPLOYMENT.md](DEPLOYMENT.md), [release/READINESS_EVIDENCE_WORKFLOW.md](release/READINESS_EVIDENCE_WORKFLOW.md), `make ci`, `make release-bundle` | ACP does not replace customer CMDB, CAB, or cloud policy engines |
| SC-7, SC-12, SC-13 | Boundary protection and cryptographic protection | Own network boundary controls, TLS termination decisions, and key-management policy | Confirm remote-ingress, localhost-only OTEL, and customer encryption requirements | Provides TLS runbook, localhost-only raw OTEL contract, and documented network exposure constraints | [deployment/TLS_SETUP.md](deployment/TLS_SETUP.md), [DEPLOYMENT.md](DEPLOYMENT.md), `make validate-config-production` | Customer controls the perimeter, certificate trust model, and external network enforcement |
| SI-4, SI-7 | System monitoring and software integrity | Own enterprise monitoring and risk acceptance for unresolved vulnerabilities | Review ACP supply-chain findings, alerting outputs, and exception records together | Provides security gate, supply-chain policy, known-limitation tracking, and deterministic validation | [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md), [SECURITY_GOVERNANCE.md](SECURITY_GOVERNANCE.md), `make security-gate` | Current open CVEs are disclosed and governed, not hidden or fully eliminated |
| IR-4, IR-5 | Incident handling and monitoring support | Own enterprise incident program, regulatory reporting, and customer-side containment | Coordinate shared investigations across application and customer infrastructure boundaries | Provides webhook events, key revocation, operator runbooks, and attributable usage evidence | [security/WEBHOOK_EVENTS.md](security/WEBHOOK_EVENTS.md), [SHARED_RESPONSIBILITY_MODEL.md](SHARED_RESPONSIBILITY_MODEL.md), `make doctor`, `make key-revoke ALIAS=<alias>` | ACP strengthens evidence and response tooling but does not replace an IR program |
| CP-9, CP-10 | Backup, recovery, and reconstitution | Own off-host retention, storage durability, and continuity targets | Run restore and failover drills with named operators and documented decisions | Provides backup timer contract, restore drill, off-host recovery validation, and HA failover drill evidence archiving | [deployment/DISASTER_RECOVERY.md](deployment/DISASTER_RECOVERY.md), [deployment/HA_FAILOVER_RUNBOOK.md](deployment/HA_FAILOVER_RUNBOOK.md), `make dr-drill`, `make db-off-host-drill`, `make ha-failover-drill` | Single-node baseline and customer-owned traffic cutover stay explicit |

---

## Control gaps and dependency notes buyers should know up front

These issues should be disclosed during diligence instead of buried in later conversations:

1. **Single-node baseline remains the primary validated topology.** ACP now ships HA failover-drill evidence tooling and runbooks, but automatic failover is still not part of the supported contract.
2. **Cloud-production claims are not yet validated.** This crosswalk is grounded in the host-first support boundary, not in a completed AWS production validation.
3. **Bypass prevention is customer-dependent.** ACP can govern routed traffic and support detective coverage, but customer network, endpoint, and workspace controls are still required for hard prevention claims.
4. **Open CVEs are governed, not magically absent.** See [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md) and the supply-chain policy for current accepted-risk handling.
5. **Metadata-first evidence reduces content risk, but customer exports may still need extra controls.** Especially when transcript-bearing vendor exports or external systems are included.

## Recommended buyer packet pairing

Use this crosswalk together with:

- [SHARED_RESPONSIBILITY_MODEL.md](SHARED_RESPONSIBILITY_MODEL.md)
- [PILOT_CONTROL_OWNERSHIP_MATRIX.md](PILOT_CONTROL_OWNERSHIP_MATRIX.md)
- [ENTERPRISE_BUYER_OBJECTIONS.md](ENTERPRISE_BUYER_OBJECTIONS.md)
- [GO_TO_MARKET_SCOPE.md](GO_TO_MARKET_SCOPE.md)
- [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md)

## What this document should never be used to claim

Do **not** use this crosswalk to claim any of the following unless separate evidence exists:

- SOC 2 certification
- ISO 27001 certification
- FedRAMP authorization or equivalence
- CMMC compliance by default
- universal cloud-production readiness
- universal browser-bypass prevention
- automatic multi-node high availability
