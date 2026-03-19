# CVE Review Log

This file is the lightweight dated review record for open CVEs governed by AI Control Plane.

Use it together with [CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md](CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md), [KNOWN_LIMITATIONS.md](../KNOWN_LIMITATIONS.md), and [`demo/config/supply_chain_vulnerability_policy.json`](../../demo/config/supply_chain_vulnerability_policy.json).

## Review rules

- Add one entry for each quarterly review or material off-cycle review.
- Update this file in the same change set as any status, expiry, or mitigation changes.
- Record the commands or evidence sources used for the review.

## 2026-03-19 — Quarterly governance review

- **Reviewers:** `platform-security`, `release-owner`
- **Evidence commands validated in this cycle:** `make supply-chain-gate`, `make supply-chain-allowlist-expiry-check`
- **Open CVEs reviewed:** `CVE-2026-0861`, `CVE-2026-26278`, `CVE-2026-26960`, `CVE-2026-26996`
- **Outcome summary:**
  - `CVE-2026-0861` remains a temporary accepted risk pending patched Presidio base images from Microsoft.
  - `CVE-2026-26278` remains a temporary accepted risk pending an upstream LibreChat dependency refresh.
  - `CVE-2026-26960` remains a temporary accepted risk pending an upstream LiteLLM dependency refresh.
  - `CVE-2026-26996` remains a temporary accepted risk pending the upstream minimatch patch rollup and digest refresh.
- **Required next action:** remove allowlist entries as patched digests land; renew only with updated `expires_on`, `last_reviewed_on`, and fresh justification.
- **Next review due:** on or before `2026-06-19`, or sooner if exploitability changes, a vendor patch lands, or an expiry warning triggers.
- **Canonical records updated in this cycle:**
  - [KNOWN_LIMITATIONS.md](../KNOWN_LIMITATIONS.md)
  - [`demo/config/supply_chain_vulnerability_policy.json`](../../demo/config/supply_chain_vulnerability_policy.json)
  - [CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md](CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md)
