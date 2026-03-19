# Go/No-Go Criteria and Severity Rubric

Define objective presentation-readiness gates so release decisions are evidence-based and repeatable.

## Decision Policy

| Decision | Criteria |
| --- | --- |
| **Go** | All mandatory gates pass and there are zero open Blocker findings. |
| **Conditional Go** | All mandatory gates pass, zero Blockers, and any open Major/Minor findings are documented in `docs/KNOWN_LIMITATIONS.md` with owner and due date. |
| **No-Go** | Any mandatory gate fails or any Blocker finding remains open. |

## Severity Rubric

### Blocker (Must Fix Before Presentation)

A finding that invalidates safety, security, core functionality, or confidence in the validation evidence.

**Required action:**
- Fix before presentation notification.
- Re-run impacted validation evidence.

**Examples:**
- `make ci` fails.
- Security validation fails.
- Required evidence artifacts are missing or contradictory.

### Major (Must Mitigate + Track)

A high-impact issue that does not invalidate the presentation but requires explicit mitigation and near-term follow-up.

**Required action:**
- Add to `docs/KNOWN_LIMITATIONS.md` with mitigation, owner, and due date.
- Explain mitigation in presentation risk notes.

### Minor (Can Defer)

A low-impact issue with limited risk to presentation outcomes.

**Required action:**
- Track in `docs/KNOWN_LIMITATIONS.md` with owner, due date, and status.

## Mandatory Pass Gates (Objective)

All gates below must pass before presentation notification:

1. **Local CI Gate**: `make ci` exits 0.
2. **Security Validation Gate**: `make supply-chain-gate` and `make supply-chain-allowlist-expiry-check` both exit 0.
3. **Evidence Package Completeness**: Required logs/artifacts are present and referenced.
4. **Peer Review Complete**: Findings and sign-off status have been recorded.

## Stop Rule

If any open **Blocker** exists, stop release progression immediately and mark decision as **No-Go** until resolved.

## Timebox Rule for Non-Blockers

If a Major/Minor issue cannot be resolved within the agreed pre-presentation timebox, it must be documented in `docs/KNOWN_LIMITATIONS.md` with owner, mitigation, due date, and status before any Go/Conditional Go decision.

## Known Security Risks

Document active security risk acceptances here. Link to detailed findings in `docs/KNOWN_LIMITATIONS.md` and govern them under [`docs/security/CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md`](../security/CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md).

| Risk ID | Severity | Description | Owner | Due Date | Status | Evidence |
|---------|----------|-------------|-------|----------|--------|----------|
| CVE-2026-0861 | Major | Presidio images contain unpatched glibc (MEDIUM). Exploitation requires local attacker + application bug + heap manipulation chain. | platform-security | 2026-05-15 | Accepted | [supply_chain_vulnerability_policy.json](../../demo/config/supply_chain_vulnerability_policy.json), [CVE_REVIEW_LOG.md](../security/CVE_REVIEW_LOG.md) |
| CVE-2026-26278 | Major | `fast-xml-parser` in the hardened LibreChat image carries a DoS issue via XML entity expansion. | platform-security | 2026-05-15 | Accepted | [supply_chain_vulnerability_policy.json](../../demo/config/supply_chain_vulnerability_policy.json), [CVE_REVIEW_LOG.md](../security/CVE_REVIEW_LOG.md) |
| CVE-2026-26960 | Major | `tar` in the hardened LiteLLM image carries a symlink-chain file read/write issue. | platform-security | 2026-05-15 | Accepted | [supply_chain_vulnerability_policy.json](../../demo/config/supply_chain_vulnerability_policy.json), [CVE_REVIEW_LOG.md](../security/CVE_REVIEW_LOG.md) |
| CVE-2026-26996 | Major | `minimatch` ReDoS remains temporarily allowlisted while the upstream patch rollup and digest refresh are pending. | platform-security | 2026-07-31 | Accepted | [supply_chain_vulnerability_policy.json](../../demo/config/supply_chain_vulnerability_policy.json), [CVE_REVIEW_LOG.md](../security/CVE_REVIEW_LOG.md) |

**Risk Acceptance Criteria:**
- All Major risks must satisfy the record requirements in [`docs/security/CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md`](../security/CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md)
- Risk acceptance must stay time-bounded and be reviewed at least quarterly
- Vendor dependency risks require a tracking ticket plus an explicit remediation plan for removing the exception

## Final Go/No-Go Checklist

Before presenting, confirm:

- [ ] A current readiness run has been generated with `make readiness-evidence`.
- [ ] The current readiness run has been verified with `make readiness-evidence-verify`.
- [ ] No open Blocker findings.
- [ ] `make ci` passed (timestamp + evidence link to `demo/logs/evidence/readiness-<TIMESTAMP>/make-ci.log`).
- [ ] `make ci-nightly SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env` passed (timestamp + evidence link).
- [ ] `make supply-chain-gate` and `make supply-chain-allowlist-expiry-check` passed (timestamp + evidence link).
- [ ] Evidence package complete and linked (see `demo/logs/evidence/readiness-<TIMESTAMP>/evidence-inventory.txt`).
- [ ] Release bundle generated and verified (see `demo/logs/release-bundles/` and the readiness run logs).
- [ ] Independent review completed and linked.
- [ ] All open Major/Minor findings are documented in `docs/KNOWN_LIMITATIONS.md`.

## Current Decision Reference

See [go_no_go_decision.md](go_no_go_decision.md) for how the latest generated presentation-readiness decision should be used.

See [`PRESENTATION_READINESS_TRACKER.md`](PRESENTATION_READINESS_TRACKER.md) for the canonical workflow that generates the dated gate-by-gate evidence matrix.
