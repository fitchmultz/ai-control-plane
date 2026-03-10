# Documentation (Start Here)

This directory is the canonical documentation set for the AI Control Plane reference implementation.

## Fastest Useful Paths

| You are trying to do... | Read this first | Then this |
|---|---|---|
| Understand what the repo proves | `../README.md` | `technical-architecture.md` |
| Run the validated baseline | `DEPLOYMENT.md` | `release/VALIDATION_CHECKLIST.md` |
| Inspect the typed operator interface | `tooling/ACPCTL.md` | `RUNBOOK.md` |
| Review enterprise positioning and scope boundaries | `ENTERPRISE_STRATEGY.md` | `GO_TO_MARKET_SCOPE.md` |
| Review proof/evidence discipline | `evidence/README.md` | `release/READINESS_EVIDENCE_WORKFLOW.md` |
| Review deployment extensions | `deployment/KUBERNETES_HELM.md` or `deployment/TERRAFORM.md` | `DEPLOYMENT.md` |

## Deployment Tracks

| Track | Status | Use When | Start Here |
|---|---|---|---|
| Linux host (Docker-first) | **Default** | Default track for this reference implementation and its validated baseline | [`DEPLOYMENT.md`](DEPLOYMENT.md) |
| Portainer management | Optional | You already use Portainer to operate Docker environments | [`deployment/PORTAINER.md`](deployment/PORTAINER.md) |
| Kubernetes (Helm) | Optional | You already run Kubernetes and need cluster-native ops | [`deployment/KUBERNETES_HELM.md`](deployment/KUBERNETES_HELM.md) |
| Terraform | Optional | You need cloud infra provisioning + cluster bootstrap | [`deployment/TERRAFORM.md`](deployment/TERRAFORM.md) |

## Choose your path

### Security leadership / CTO

- **Strategy and decisions**: `ENTERPRISE_STRATEGY.md`
- **Go-to-market scope**: `GO_TO_MARKET_SCOPE.md`
- **Pilot package**: `ENTERPRISE_PILOT_PACKAGE.md`
- **Pilot execution model**: `PILOT_EXECUTION_MODEL.md`
- **Pilot control ownership**: `PILOT_CONTROL_OWNERSHIP_MATRIX.md`
- **Pilot customer validation checklist**: `PILOT_CUSTOMER_VALIDATION_CHECKLIST.md`
- **Shared responsibility model**: `SHARED_RESPONSIBILITY_MODEL.md`
- **Buyer objection handling**: `ENTERPRISE_BUYER_OBJECTIONS.md`
- **Service offerings and SOW templates**: `SERVICE_OFFERINGS.md` and `templates/`
- **Executive summary and presenter guidance**: `presentation/EXECUTIVE_ONE_PAGER.md` and `presentation/PRESENTATION_GUIDE.md`
- **Current validated scope (local host-first reference environment)**: `LOCAL_DEMO_PLAN.md`
- **Managed browser/workspace proof track**: `BROWSER_WORKSPACE_PROOF_TRACK.md`
- **Local performance baseline**: `PERFORMANCE_BASELINE.md`
- **Cloud/environment validation boundary**: `GO_TO_MARKET_SCOPE.md`

### Engineers running the demo

- **Demo quickstart**: `../demo/README.md`
- **Repository reviewer path**: `../README.md#reviewer-path`
- **Validation checklist**: `release/VALIDATION_CHECKLIST.md`
- **Deployment/topology**: `DEPLOYMENT.md`
- **Typed operator CLI**: `tooling/ACPCTL.md`
- **Host-first production workflow (declarative)**: `DEPLOYMENT.md#43-declarative-host-first-deployment-recommended-for-production`
- **Runtime/capacity validation and sizing**: `DEPLOYMENT.md#runtime-and-capacity-validation-public-snapshot`
- **Operations runbook**: `RUNBOOK.md`
- **Database reference**: `DATABASE.md`

### Operators / Governance

- **Detections and governance mechanics**: `security/DETECTION.md`
- **SIEM integration**: `security/SIEM_INTEGRATION.md`
- **Budgets and rate limits**: `policy/BUDGETS_AND_RATE_LIMITS.md`
- **Approved models policy**: `policy/APPROVED_MODELS.md`
- **Financial governance and chargeback**: `policy/FINANCIAL_GOVERNANCE_AND_CHARGEBACK.md`

## What Reviewers Usually Miss

- `tooling/ACPCTL.md` explains why the repo favors typed operator flows over ad-hoc script sprawl.
- `technical-architecture.md` is the shortest explanation of how the repo is structured internally.
- `release/READINESS_EVIDENCE_WORKFLOW.md` and `release/VALIDATION_CHECKLIST.md` show that validation is treated as a product surface, not an afterthought.
- `ARTIFACTS.md` defines what is generated locally and intentionally kept out of version control.
## Index

### Core docs (top-level)

- `ENTERPRISE_STRATEGY.md` — CTO-facing strategy and decision memo
- `GO_TO_MARKET_SCOPE.md` — readiness scopes: validated baseline, conditional pilot readiness, and out-of-scope claims
- `PILOT_EXECUTION_MODEL.md` — strict pilot phase model, hard-stop rules, and closeout decision gates
- `PILOT_SPONSOR_ONE_PAGER.md` — sponsor/procurement view of pilot proof, prerequisites, and hard stops
- `PILOT_CONTROL_OWNERSHIP_MATRIX.md` — customer/provider/shared control ownership for pilots
- `PILOT_CUSTOMER_VALIDATION_CHECKLIST.md` — customer-environment validation tracker for pilot closeout
- `PILOT_CLOSEOUT_EXAMPLES.md` — decision-grade expand/remediate/no-go pilot closeout examples
- `SHARED_RESPONSIBILITY_MODEL.md` — default operating boundary for implementation and managed-service engagements
- `BROWSER_WORKSPACE_PROOF_TRACK.md` — browser/workspace governance proof track and customer validation boundary
- `ENTERPRISE_PILOT_PACKAGE.md` — pilot scope, customer prerequisites, success criteria, and exit gates
- `ENTERPRISE_BUYER_OBJECTIONS.md` — direct answers to common enterprise buyer objections with proof boundaries
- `SERVICE_OFFERINGS.md` — productized service catalog with deliverables, prerequisites, and RACI
  - Includes guardrails customization packaging across pre-call, in-call, and post-call lifecycle controls
- `LOCAL_DEMO_PLAN.md` — single-server local demo implementation plan
- `DEPLOYMENT.md` — deployment tracks, topology, and networking
- `technical-architecture.md` — concise architecture, control/data flow, and design trade-offs
- `RUNBOOK.md` — operational runbook and troubleshooting
- `DATABASE.md` — database schema notes and verification queries
- `ARTIFACTS.md` — generated artifact policy and regeneration commands
- `PERFORMANCE_BASELINE.md` — local reference-host benchmark workflow and interpretation rules
- `release/PRESENTATION_READINESS_TRACKER.md` — Canonical instructions for generating current readiness evidence
- `release/READINESS_EVIDENCE_WORKFLOW.md` — canonical commands and artifact layout for regenerating proof
- `release/go_no_go_decision.md` — Canonical instructions for using generated decision records
- `release/VALIDATION_CHECKLIST.md` — Exact validation command sequence for the current baseline
- `evidence/README.md` — Evidence materials mapping claims to demo/workshop/cookbook/operational artifacts

### Presentation assets (docs/presentation/)

- `presentation/EXECUTIVE_ONE_PAGER.md` — Canonical executive summary artifact for leadership audiences
- `presentation/PRESENTATION_GUIDE.md` — concise presenter framing and pre-read checklist

### Tooling setup (docs/tooling/)

- `tooling/ACPCTL.md` - Typed CLI core (`acpctl`) migration guide and contracts
- `tooling/LIBRECHAT.md` - Managed web UI for governed browser chat
- `tooling/CODEX.md`
- `tooling/CURSOR.md`
- `tooling/OPENCODE.md`
- `tooling/COPILOT.md` - Supplemental VS Code/Copilot proxy + telemetry guidance (visibility-first)
- `tooling/CLAUDE_CODE_TESTING.md`
- `tooling/CLAUDE_CODE_QUICKREF.md`
- `tooling/TOOLING_REFERENCE_LINKS.md` - Authoritative upstream references for OpenCode, Claude Code, Codex, Cursor/Copilot, LiteLLM, and OpenWebUI

### Policy (docs/policy/)

- `policy/APPROVED_MODELS.md` — Approved model list and MODELS array contract
- `policy/BUDGETS_AND_RATE_LIMITS.md` — Budget enforcement and rate limiting
- `policy/FINANCIAL_GOVERNANCE_AND_CHARGEBACK.md` — Showback/chargeback workflows and attribution
- `policy/THIRD_PARTY_LICENSE_MATRIX.md` — Third-party license policy and compliance boundaries
- `policy/THIRD_PARTY_LICENSE_MATRIX.json` — Machine-readable license policy (source of truth)

### Security & detections (docs/security/)

- `security/ENTERPRISE_AUTH_ARCHITECTURE.md` — License-aware auth architecture (LibreChat + LiteLLM)
- `security/DETECTION.md` — Detection rules, SIEM integration, and guardrail capability map (LiteLLM native vs Presidio)
- `DETECTION_RULES.md` — Detection rule authoring guide (SQL patterns, validation, troubleshooting)
- `security/SIEM_INTEGRATION.md` — SIEM integration architecture and configuration
### Observability (docs/observability/)

- `observability/OTEL_SETUP.md`

### Demos (docs/demos/)

- `demos/API_KEY_GOVERNANCE_DEMO.md` — API-key enforcement and preventive controls
- `demos/SaaS_SUBSCRIPTION_GOVERNANCE_DEMO.md` — SaaS/subscription route-based governance (enforce when routed, detect/respond on bypass)
- `demos/NETWORK_ENDPOINT_ENFORCEMENT_DEMO.md` — network and endpoint enforcement demonstration

### Templates (docs/templates/)

- `templates/README.md` — how to use SOW templates
- `templates/PILOT_CHARTER.md` — pilot kickoff scope, owners, and exit criteria
- `templates/PILOT_ACCEPTANCE_MEMO.md` — pilot checkpoint/closeout decision record
- `templates/PILOT_OPERATOR_HANDOFF_CHECKLIST.md` — pilot operations handoff checklist
- `templates/SOW_AI_USAGE_EXPOSURE_ASSESSMENT.md` — assessment engagement template
- `templates/SOW_AI_CONTROL_PLANE_IMPLEMENTATION.md` — implementation engagement template
- `templates/SOW_MANAGED_AI_SECURITY_OPERATIONS.md` — managed operations template
- `templates/SOW_VENDOR_WORKSPACE_GOVERNANCE.md` — workspace governance template

### Deployment supplements (docs/deployment/)

- `deployment/PORTAINER.md` — Optional Portainer operations on top of host-first Docker deployment
- `deployment/KUBERNETES_HELM.md` — Optional Kubernetes/Helm deployment track
- `deployment/TERRAFORM.md` — Optional Terraform cloud provisioning track
- `deployment/PRODUCTION_HANDOFF_RUNBOOK.md` — Production deployment handoff procedures
- `deployment/SINGLE_TENANT_PRODUCTION_CONTRACT.md` — Production deployment contract and validation
- `deployment/THIRD_PARTY_LICENSE_SUMMARY.md` — Third-party license compliance report (generated)
- `deployment/TLS_SETUP.md`
