# CVE Review Log

This file is the lightweight dated review record for open CVEs governed by AI Control Plane.

Use it together with [CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md](CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md), [KNOWN_LIMITATIONS.md](../KNOWN_LIMITATIONS.md), and [`demo/config/supply_chain_vulnerability_policy.json`](../../demo/config/supply_chain_vulnerability_policy.json).

## Review rules

- Add one entry for each quarterly review or material off-cycle review.
- Update this file in the same change set as any status, expiry, or mitigation changes.
- Record the commands or evidence sources used for the review.

## 2026-05-07 — Off-cycle Presidio expiry review

- **Reviewers:** `platform-security`, `release-owner`
- **Evidence commands validated in this cycle:** `docker buildx imagetools inspect mcr.microsoft.com/presidio-analyzer:latest`, `docker buildx imagetools inspect mcr.microsoft.com/presidio-anonymizer:latest`, `docker buildx imagetools inspect mcr.microsoft.com/presidio-analyzer:2.2.362`, `docker buildx imagetools inspect mcr.microsoft.com/presidio-anonymizer:2.2.362`, `trivy image --scanners vuln mcr.microsoft.com/presidio-analyzer:2.2.362`, `trivy image --scanners vuln mcr.microsoft.com/presidio-anonymizer:2.2.362`
- **Open CVEs reviewed:** `CVE-2026-0861`
- **Outcome summary:**
  - Official Microsoft Presidio `latest` currently resolves to `2.2.362`.
  - Trivy still reports `CVE-2026-0861` against `libc-bin` and `libc6` in both `2.2.362` analyzer and anonymizer images.
  - `2.2.362` also reports additional high/critical findings, so ACP remains pinned to the previously reviewed `2.2.361` digests instead of taking a noisier image refresh.
- **Required next action:** keep the Presidio exception time-bounded and adopt the next Microsoft Presidio digest that removes `CVE-2026-0861` without introducing a broader vulnerability set.
- **Next review due:** on or before `2026-06-19`, or sooner if exploitability changes or a cleaner vendor image lands.
- **Canonical records updated in this cycle:**
  - [KNOWN_LIMITATIONS.md](../KNOWN_LIMITATIONS.md)
  - [`demo/config/supply_chain_vulnerability_policy.json`](../../demo/config/supply_chain_vulnerability_policy.json)
  - [release/GO_NO_GO.md](../release/GO_NO_GO.md)

## 2026-04-26 — Off-cycle hardened image remediation review

- **Reviewers:** `platform-security`, `release-owner`
- **Evidence commands validated in this cycle:** `make hardened-images-scan`, `docker buildx imagetools inspect ghcr.io/fitchmultz/acp/litellm-hardened:20260426`, `docker buildx imagetools inspect ghcr.io/fitchmultz/acp/librechat-hardened:20260426`
- **Open CVEs reviewed:** `CVE-2026-0861`
- **Remediated CVEs closed in this cycle:** `CVE-2026-26278`, `CVE-2026-26960`, `CVE-2026-26996`
- **Outcome summary:**
  - `CVE-2026-0861` remains a temporary accepted risk pending patched Presidio base images from Microsoft.
  - `CVE-2026-26278` was remediated in refreshed hardened LibreChat image `ghcr.io/fitchmultz/acp/librechat-hardened:20260426`.
  - `CVE-2026-26960` was remediated in refreshed hardened LiteLLM image `ghcr.io/fitchmultz/acp/litellm-hardened:20260426`.
  - `CVE-2026-26996` was remediated by the hardened dependency rollup and no longer requires temporary allowlist entries.
- **Required next action:** keep Presidio CVE expiry windows current and remove remaining allowlist entries as patched Presidio digests land.
- **Next review due:** on or before `2026-06-19`, or sooner if exploitability changes, a vendor patch lands, or an expiry warning triggers.
- **Canonical records updated in this cycle:**
  - [KNOWN_LIMITATIONS.md](../KNOWN_LIMITATIONS.md)
  - [`demo/config/supply_chain_vulnerability_policy.json`](../../demo/config/supply_chain_vulnerability_policy.json)
  - [release/GO_NO_GO.md](../release/GO_NO_GO.md)
  - [CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md](CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md)

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
