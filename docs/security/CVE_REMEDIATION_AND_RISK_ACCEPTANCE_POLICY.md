# CVE Remediation and Risk Acceptance Policy

This document is the canonical AI Control Plane process for how open CVEs are triaged, remediated, temporarily accepted, reviewed, evidenced, and described to buyers.

It governs vulnerabilities; it does **not** claim zero vulnerabilities, instant upstream patch availability, or blanket compliance by itself.

## Purpose and scope

This policy applies to CVEs affecting the repository's supported host-first surface, including tracked images, pinned dependencies, deployment assets, and operator workflows that ship with the public reference implementation.

Use it together with these canonical artifacts:

| Artifact | Role |
| --- | --- |
| [KNOWN_LIMITATIONS.md](../KNOWN_LIMITATIONS.md) | Human-readable register for current material findings and operator-facing status |
| [`demo/config/supply_chain_vulnerability_policy.json`](../../demo/config/supply_chain_vulnerability_policy.json) | Machine-readable accepted-risk inventory with expiry windows |
| [CVE_REVIEW_LOG.md](CVE_REVIEW_LOG.md) | Dated quarterly and off-cycle review record |
| [release/GO_NO_GO.md](../release/GO_NO_GO.md) | Severity rubric, release stop rule, and presentation decision contract |

## Severity and track selection

Severity follows the release rubric in [release/GO_NO_GO.md](../release/GO_NO_GO.md). Scanner severity is an input, not the only decision factor.

Every new CVE is assessed using:

- scanner severity and CVSS context
- whether the affected component is in the validated support boundary
- exposure path, required privileges, and exploit preconditions
- compensating controls already present in the runtime
- whether the issue is awaiting an upstream/vendor patch
- whether the issue changes release readiness or support claims

Each CVE is then placed on one track:

1. **Remediate** — patch, rebuild, re-pin, or remove the affected component.
2. **Time-bounded accepted risk** — temporary exception with owner, ticket, mitigation, expiry, and review record.
3. **Release stop / no-go** — blocker or actively unacceptable exposure that must be fixed before release progression.

## Intake and triage workflow

1. **Discovery sources**
   - `make supply-chain-gate`
   - `make supply-chain-allowlist-expiry-check`
   - hardened image scans
   - upstream/vendor advisories
   - operator or customer-reported findings
2. **Initial record within two business days**
   - CVE identifier
   - affected package and image or dependency
   - source of the finding
   - current severity and exploitability notes
   - named owner
   - remediation or exception ticket
3. **Context decision within five business days**
   - decide whether the finding is a remediation candidate, a temporary accepted risk, or a release stop
   - document mitigation and due date in [KNOWN_LIMITATIONS.md](../KNOWN_LIMITATIONS.md)
   - add or update the machine-readable allowlist entry when an exception is required
4. **Public truth check before release claims**
   - if the finding changes support, security, or procurement claims, update the relevant docs in the same change set

## Remediation track

Use the remediation track when a fix is available or when the risk is not acceptable to carry.

### Expected timelines

- **Blocker / release-stop findings:** fix before release progression; do not treat as routine accepted risk.
- **Major findings:** target removal through a patched digest, dependency update, or supported configuration change within 30 days when a fix is available. If the fix is upstream-blocked, convert to a time-bounded accepted risk instead of leaving the status implicit.
- **Minor findings:** target removal in the next scheduled maintenance window, normally within 90 days.

### Escalation rules

Escalate immediately when any of the following becomes true:

- a blocked vendor fix remains unresolved near the exception expiry
- exploit maturity materially increases
- the affected component becomes internet reachable or otherwise more exposed
- the finding would make current buyer-facing language misleading

### Definition of remediated

A CVE is treated as remediated only when all of the following are true:

- the vulnerable digest or dependency is no longer part of the supported tracked surface
- `make supply-chain-gate` passes without requiring the exception for that finding
- the corresponding exception entry is removed from `demo/config/supply_chain_vulnerability_policy.json`
- the human-readable status is updated in [KNOWN_LIMITATIONS.md](../KNOWN_LIMITATIONS.md)

## Time-bounded accepted-risk track

Use this track only when the repo can truthfully explain why the issue is temporarily carried and what causes the exception to end.

### Required record set

Every accepted-risk CVE must have all of the following:

1. A machine-readable entry in [`demo/config/supply_chain_vulnerability_policy.json`](../../demo/config/supply_chain_vulnerability_policy.json) with:
   - `id`
   - `package`
   - `image`
   - `expires_on`
   - `justification`
   - `owner`
   - `ticket`
   - `last_reviewed_on`
   - `remediation_plan`
2. A human-readable status row in [KNOWN_LIMITATIONS.md](../KNOWN_LIMITATIONS.md) that explains impact, mitigation, owner, due date, and status.
3. A dated review outcome in [CVE_REVIEW_LOG.md](CVE_REVIEW_LOG.md).

### What “time-bounded” means here

- Every exception must have an explicit `expires_on` date.
- Open-ended allowlists are not allowed.
- New or renewed accepted-risk records for non-blocker findings may not extend more than **180 days** from the review date that approved them.
- If a finding cannot be defended inside that window, the next action is remediation, support-boundary reduction, or release-stop escalation — not silent renewal.

### Renewal and expiry handling

- At **30 days before expiry**, the owner must either land a fix or prepare a renewed decision with updated justification.
- At **7 days before expiry**, unresolved entries are treated as release-risk items and must be reviewed before any buyer-facing milestone.
- Renewal requires updating both `expires_on` and `last_reviewed_on`, plus a fresh note in [CVE_REVIEW_LOG.md](CVE_REVIEW_LOG.md).

## Quarterly review cadence

The minimum review cadence is **quarterly**, with off-cycle review whenever risk changes materially.

### Review participants

- `platform-security` as the default risk owner
- the current release or readiness owner
- additional domain owners when the affected component changes exposure or support claims

### Minimum review output

A quarterly or off-cycle review is complete only when the repository shows all of the following in the same review cycle:

- current state in [`demo/config/supply_chain_vulnerability_policy.json`](../../demo/config/supply_chain_vulnerability_policy.json)
- current human-readable status in [KNOWN_LIMITATIONS.md](../KNOWN_LIMITATIONS.md)
- a dated entry in [CVE_REVIEW_LOG.md](CVE_REVIEW_LOG.md)

### Off-cycle triggers

Review early when:

- an upstream patch becomes available
- exploit maturity or severity changes
- the deployment exposure changes
- an allowlist entry approaches expiry
- buyer diligence asks for updated status

## Evidence requirements

The minimum auditable evidence set for governed CVEs is:

- `make supply-chain-gate` exit status
- `make supply-chain-allowlist-expiry-check` exit status
- current [`demo/config/supply_chain_vulnerability_policy.json`](../../demo/config/supply_chain_vulnerability_policy.json)
- current [KNOWN_LIMITATIONS.md](../KNOWN_LIMITATIONS.md)
- current [CVE_REVIEW_LOG.md](CVE_REVIEW_LOG.md)
- release decision usage via [release/GO_NO_GO.md](../release/GO_NO_GO.md)

Optional local artifact support:

- `make supply-chain-report` for `demo/logs/supply-chain/summary.json`
- readiness or release evidence bundles when the current delivery cycle needs archived proof

## Buyer-safe status communication

Use this wording as the default diligence response when buyers ask about open CVEs:

> AI Control Plane discloses current open supply-chain findings in a public register and machine-readable exception policy. Each accepted-risk CVE is time-bounded, has a named owner and ticket, carries a mitigation and remediation plan, and is re-reviewed at least quarterly or sooner when the risk changes. The correct claim is that open vulnerabilities are governed and disclosed, not that they are universally absent or instantly remediated.

### Claims this policy supports

- open CVEs are disclosed, tracked, and time-bounded
- exceptions have owners, tickets, due dates, and review history
- buyer-facing status language is governed by repository evidence

### Claims this policy does not support

- zero-CVE posture
- guaranteed same-day upstream remediation
- automatic compliance certification
- universal risk elimination in customer environments

## Cross-references

- [KNOWN_LIMITATIONS.md](../KNOWN_LIMITATIONS.md)
- [`demo/config/supply_chain_vulnerability_policy.json`](../../demo/config/supply_chain_vulnerability_policy.json)
- [CVE_REVIEW_LOG.md](CVE_REVIEW_LOG.md)
- [release/GO_NO_GO.md](../release/GO_NO_GO.md)
- [SECURITY_WHITEPAPER_AND_THREAT_MODEL.md](SECURITY_WHITEPAPER_AND_THREAT_MODEL.md)
- [../COMPLIANCE_CROSSWALK.md](../COMPLIANCE_CROSSWALK.md)
- [../SECURITY_GOVERNANCE.md](../SECURITY_GOVERNANCE.md)
