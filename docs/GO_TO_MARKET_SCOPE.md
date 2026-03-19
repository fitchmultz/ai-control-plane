# Go-To-Market Scope

This document separates what is currently validated from what is only planned. Do not collapse these scopes into a single “enterprise ready” claim.

---

## Readiness Scopes

| Scope | Status | What can be claimed now | What still must happen |
|---|---|---|---|
| Local host-first reference environment | ✅ Ready | Linux host Docker deployment, gateway controls, typed operator workflows, local CI, offline demo, generated readiness-evidence workflow, pilot closeout bundle workflow, and runnable benchmark profiles | Refresh evidence before external reuse |
| Customer pilot on controlled Linux host | ⚠️ Conditionally ready | Architecture, deployment pattern, runbooks, SIEM integration pattern, budgets/chargeback model, workshop/demo flow, named control-owner matrix, and decision-grade pilot packet | Re-run evidence in the customer-like environment; validate identity, SIEM, secrets, retention, network controls, and customer-owned browser/workspace governance |
| Cloud production / AWS-specific enforcement claims | ⏳ Not yet validated | Architecture and validation plan only | Complete AWS lab or customer-cloud validation for egress controls, cloud operations, and production hardening |

Generated evidence artifacts are intentionally not committed. See [docs/ARTIFACTS.md](ARTIFACTS.md).
Customer-owned pilot obligations are documented in [PILOT_CONTROL_OWNERSHIP_MATRIX.md](PILOT_CONTROL_OWNERSHIP_MATRIX.md).
Customer-environment validation steps are tracked in [PILOT_CUSTOMER_VALIDATION_CHECKLIST.md](PILOT_CUSTOMER_VALIDATION_CHECKLIST.md).
The strict pilot phase gate is documented in [PILOT_EXECUTION_MODEL.md](PILOT_EXECUTION_MODEL.md).

---

## Minimum Bar Before Customer-Facing Commitments

### For reference-environment demonstrations

- `make ci` passes
- Release bundle builds and verifies
- Demo and operator documentation align to the current implementation

### For named customer pilots

- Re-run the validation set in the target environment
- Validate secrets handling, identity integration, SIEM ingestion, and retention settings
- Complete the customer validation checklist with named owners and evidence links
- Confirm enforce-vs-detect boundaries in writing with the customer team
- Follow the phase-gated pilot execution model through closeout decision

### For cloud-production positioning

- Validate egress controls in AWS lab or the customer cloud environment
- Validate operational ownership, backups, upgrades, and failure handling in that environment
- Refresh readiness artifacts immediately before the external decision

---

## Target Profile

- Public sector: baseline rigor by default
- Commercial: profile-specific adjustments without lowering the validation bar

---

## Deployment Strategy

- Primary validated path: host-first Linux deployment
- Incubating deployment assets for Kubernetes/Helm remain in-repo for explicit internal exploration only
- Cloud positioning is gated on environment-specific proof, not architecture intent alone

---

## Explicitly Out Until Additional Validation Exists

- Kubernetes-first production positioning as the default recommendation
- Customer commitments on egress-blocking effectiveness outside validated environments
- Multi-tenant managed-service claims
- completed third-party validation claims before an outside assessment exists
- “Blocking all AI usage” claims

## Claim Discipline Rules

Customer-facing material should not use these phrases unless the named conditions are actually true:

| Phrase | Required condition |
|---|---|
| enterprise-ready | named environment, current evidence, and current owner signoff |
| managed service | explicit shared-responsibility split and staffed operating boundary |
| browser governance | named workspace/browser/endpoint owners plus validation evidence |
| bypass prevention | customer-owned network and endpoint controls validated |
| rollout-ready pilot | pilot execution model completed through decision phase |
