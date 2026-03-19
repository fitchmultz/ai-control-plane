<!--
Purpose: Define the canonical pre-review packet for external security or architecture assessments.
Responsibilities: Collect review inputs, explain what can be referenced today, define review options, and preserve claim discipline.
Scope: Repository-authored preparation artifacts and regeneration commands only; not a completed audit, penetration test, certification, or attestation.
Usage: Use when preparing a third-party reviewer, answering buyer diligence about independent review status, or scoping roadmap item #22 work.
Invariants/Assumptions: Roadmap item #22 remains open until a real external assessment is completed and can be referenced truthfully.
-->

# External Review Readiness

This document collects the canonical artifact set for a future external security review, architecture review, or equivalent third-party assessment.

It is a **preparation package**, not the review itself.
No external assessment has been completed from this repository yet, and roadmap item `#22` remains open until that changes.

## What this package provides

- a truthful briefing set for an external reviewer
- the current control-truth, threat, and limitation documents
- the commands needed to regenerate current evidence locally
- buyer-safe language for explaining the difference between preparation and completed third-party validation

## What this package does not provide

- a completed audit, penetration test, architecture assessment, or attestation
- certification of SOC 2, ISO 27001, FedRAMP, CMMC, or equivalent programs
- proof that customer-environment controls were reviewed by an outside party

## Current status

| Question | Answer |
| --- | --- |
| Has ACP completed an external assessment yet? | No |
| Can buyers reference a third-party review today? | No |
| Is ACP prepared for an external review briefing? | Yes |
| Does this close roadmap item `#22`? | No |

## Canonical reviewer packet

Give an external reviewer these artifacts first:

| Artifact | Why it matters | Path |
| --- | --- | --- |
| Threat model and security whitepaper | Explains architecture, trust boundaries, abuse paths, mitigations, and residual risks | `docs/security/SECURITY_WHITEPAPER_AND_THREAT_MODEL.md` |
| Compliance crosswalk | Maps ACP evidence and ownership to SOC 2, ISO 27001, and NIST-style controls | `docs/COMPLIANCE_CROSSWALK.md` |
| Go-to-market scope | Defines validated, conditionally ready, and not-yet-validated claim boundaries | `docs/GO_TO_MARKET_SCOPE.md` |
| Known limitations register | Shows current material gaps and open findings | `docs/KNOWN_LIMITATIONS.md` |
| CVE governance policy | Shows how vulnerabilities are triaged, accepted temporarily, reviewed, and communicated | `docs/security/CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md` |
| Shared responsibility model | Makes customer vs ACP control ownership explicit | `docs/SHARED_RESPONSIBILITY_MODEL.md` |
| Pilot/control ownership matrix | Clarifies implementation and operating ownership lines | `docs/PILOT_CONTROL_OWNERSHIP_MATRIX.md` |
| Evidence workflow | Shows how current local readiness proof is regenerated | `docs/release/READINESS_EVIDENCE_WORKFLOW.md` |
| Evidence map | Connects repo claims to commands and source artifacts | `docs/evidence/EVIDENCE_MAP.md` |
| Go/No-Go criteria | Shows release decision rules and the still-open independent-review expectation | `docs/release/GO_NO_GO.md` |

## Regenerate current evidence before review

Run these commands before handing material to an external reviewer:

```bash
make security-gate
make release-bundle
make readiness-evidence
make readiness-evidence-verify
make pilot-closeout-bundle
```

Use the latest dated outputs under:

- `demo/logs/evidence/readiness-<TIMESTAMP>/`
- `demo/logs/release-bundles/`
- `demo/logs/pilot-closeout/pilot-closeout-<TIMESTAMP>/`

Generated evidence is local-only and intentionally not committed. See `docs/ARTIFACTS.md`.

## Review options and likely scope

### 1. External security architecture review

Best when the buyer wants an expert review of boundaries, design choices, and residual risk.

Typical scope:
- architecture and trust boundaries
- enforce-vs-detect truthfulness
- control ownership split
- key limitations and compensating controls

### 2. External security assessment or gap review

Best when the buyer wants a structured security opinion tied to current implementation and operating evidence.

Typical scope:
- threat model review
- vulnerability governance review
- evidence workflow review
- deployment hardening and operational recovery review

### 3. Penetration test

Best when the buyer wants hands-on testing of a named deployed environment.

Typical scope:
- exposed surfaces in the target environment
- auth, ingress, and runtime misconfiguration testing
- environment-specific findings and retest requirements

Important: a penetration test is environment-specific and does not replace the broader architecture or control review by itself.

## Buyer-safe language

Use this statement when asked whether ACP has been independently reviewed:

> AI Control Plane has not yet completed a third-party security or architecture assessment that we can cite as completed external validation. What we do have today is a structured external-review readiness packet: current threat model, control crosswalk, known limitations, governed CVE process, ownership boundaries, and regenerable readiness evidence. The correct claim is that ACP is prepared for external review, not that external validation is already complete.

## Claims supported now

- ACP has a structured, reviewer-ready evidence packet
- ACP can regenerate current local readiness and security evidence on demand
- ACP documents current limitations and control boundaries explicitly

## Claims not supported now

- ACP has completed an external review
- ACP has independent certification or auditor approval
- ACP has third-party validation covering every customer environment

## Roadmap item #22 completion rule

Roadmap item `#22` is complete only when a real security review, architecture review, or equivalent external assessment has been completed and can be referenced in buyer conversations.

Until then, this readiness package is preparation work only.

## Related documents

- [Security Whitepaper and Threat Model](SECURITY_WHITEPAPER_AND_THREAT_MODEL.md)
- [Compliance Crosswalk](../COMPLIANCE_CROSSWALK.md)
- [Go-To-Market Scope](../GO_TO_MARKET_SCOPE.md)
- [Known Limitations](../KNOWN_LIMITATIONS.md)
- [CVE Remediation and Risk Acceptance Policy](CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md)
- [Shared Responsibility Model](../SHARED_RESPONSIBILITY_MODEL.md)
- [Pilot Control Ownership Matrix](../PILOT_CONTROL_OWNERSHIP_MATRIX.md)
- [Readiness Evidence Workflow](../release/READINESS_EVIDENCE_WORKFLOW.md)
- [Evidence Map](../evidence/EVIDENCE_MAP.md)
- [Go/No-Go Criteria](../release/GO_NO_GO.md)
