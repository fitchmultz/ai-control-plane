# Known Limitations

Track unresolved non-blocking issues required for transparent go/no-go decisions.

## Purpose

This document records Major and Minor findings that do not block presentation but must be tracked for transparency and follow-up. Blocker findings do not belong here; Blockers must be fixed before presentation.

For CVEs, this file is the human-readable register. The canonical machine-readable exception inventory lives in [`demo/config/supply_chain_vulnerability_policy.json`](../demo/config/supply_chain_vulnerability_policy.json), and the governing process lives in [security/CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md](security/CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md).

## Entry Requirements

Each open Major/Minor finding must include:
- Owner (accountable for resolution)
- Mitigation (current workaround or risk reduction)
- Due Date (target for resolution or exception expiry)
- Status (Open/In Progress/Closed)
- Evidence Links (logs, tickets, docs)

For CVEs, the evidence set must also point to the machine-readable allowlist record and the dated review log when an accepted-risk exception is active.

## Active Findings

| Severity | Finding | Impact | Mitigation | Owner | Due Date | Status | Evidence Links |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Major | CVE-2026-0861 Supply-Chain Risk | Presidio images contain an unpatched glibc vulnerability (MEDIUM). Exploitation requires a local attacker plus an application bug chain. | Containers are hardened with `no-new-privileges` and dropped capabilities. Vendor patch remains blocked on Microsoft-published Presidio images. | platform-security | 2026-05-15 | Open | [supply_chain_vulnerability_policy.json](../demo/config/supply_chain_vulnerability_policy.json), [security/CVE_REVIEW_LOG.md](security/CVE_REVIEW_LOG.md) |
| Major | CVE-2026-26278 Supply-Chain Risk | `fast-xml-parser` in the hardened LibreChat image carries a DoS issue via XML entity expansion. | XML processing remains limited to controlled inputs; container is hardened while upstream patching is pending. | platform-security | 2026-05-15 | Open | [supply_chain_vulnerability_policy.json](../demo/config/supply_chain_vulnerability_policy.json), [security/CVE_REVIEW_LOG.md](security/CVE_REVIEW_LOG.md) |
| Major | CVE-2026-26960 Supply-Chain Risk | `tar` in the hardened LiteLLM image carries a symlink-chain file read/write issue. | LiteLLM runtime does not process untrusted archives in the supported path; container remains hardened while upstream patching is pending. | platform-security | 2026-05-15 | Open | [supply_chain_vulnerability_policy.json](../demo/config/supply_chain_vulnerability_policy.json), [security/CVE_REVIEW_LOG.md](security/CVE_REVIEW_LOG.md) |
| Major | CVE-2026-26996 Supply-Chain Risk (Temporary Allowlist) | `minimatch` ReDoS vulnerability (HIGH) remains in hardened LibreChat and LiteLLM images until the upstream patch rollup is available. | Temporary time-bounded allowlist entries expire on `2026-07-31` with explicit remediation plan, hardening, and ticketed follow-up. | platform-security | 2026-07-31 | In Progress | [supply_chain_vulnerability_policy.json](../demo/config/supply_chain_vulnerability_policy.json), [security/CVE_REVIEW_LOG.md](security/CVE_REVIEW_LOG.md) |
| Major | Single-Node Topology / No Automatic Failover | The supported host-first deployment converges one host only. Gateway, database, overlays, and local backup artifacts can share the same host/storage failure domain. Host or disk loss can cause a full outage until recovery completes. | Treat the current contract as backup-and-recovery, not HA. Keep off-host backup copies, document customer-owned DNS/LB failover, and use [deployment/HA_FAILOVER_TOPOLOGY.md](deployment/HA_FAILOVER_TOPOLOGY.md) when scoping availability requirements. | platform | 2026-06-30 | Open | [deployment/HA_FAILOVER_TOPOLOGY.md](deployment/HA_FAILOVER_TOPOLOGY.md), [DEPLOYMENT.md](DEPLOYMENT.md) |
| Minor | Port 4000 Conflict | Gateway fail to start if port 4000 is occupied by other slots/services. | Stop conflicting services or use `LITELLM_HOST_PORT` override. | SRE | 2026-06-01 | Open | [README.md](../README.md#installation) |
| Minor | Offline Token Estimation | Token counts in offline mode are estimated, not precise. | Use real providers for precise token usage validation. | Dev | 2026-03-15 | Open | [README.md](../README.md#offline-demo-mode) |
| Minor | Presidio Service Footprint | Deterministic DLP relies on two additional services (Presidio analyzer/anonymizer), which increases runtime surface area compared to native LiteLLM-only guardrails. | Keep Presidio scoped to deterministic/custom-entity requirements; use native LiteLLM guardrails for lightweight coverage where appropriate. | Security | 2026-04-01 | Open | [DEPLOYMENT.md](DEPLOYMENT.md#10-dlp-and-guardrails) |
| Minor | DLP Offline Mode | Inline guardrail attachment requires LiteLLM guardrail support in the running tier. In offline/lab modes without required feature support, guardrail config exists but live blocking cannot be fully validated. | Treat offline as configuration/evidence rehearsal and validate live blocking in production-capable environments. | Dev | 2026-06-01 | Open | `demo/logs/evidence/19_dress_rehearsal.log` *(generated locally; see [ARTIFACTS.md](ARTIFACTS.md))* |

## Closed Findings

| Severity | Finding | Resolution | Closed Date | Evidence Links |
| --- | --- | --- | --- | --- |
| Minor | Key Generation Model Mismatch | `make key-gen` and demo scenarios now auto-detect offline mode via `ACP_OFFLINE_MODE=1`, resolving models from `demo/config/litellm-offline.yaml` (`mock-gpt`, `mock-claude`) in offline runs. Set `ACP_OFFLINE_MODE=1` before key generation in offline demos. | 2026-02-18 | `make key-gen`, `make demo-scenario SCENARIO=8`, [APPROVED_MODELS.md](policy/APPROVED_MODELS.md) |

## Process Rules

1. **Blocker findings do not belong here** — Blockers must be fixed before presentation notification.
2. **Major/Minor entries must be updated** whenever status changes.
3. **Presentation readiness review** must reference this file directly.
4. **Closed findings** move to the Closed Findings section with resolution summary.
5. **CVE exceptions must stay time-bounded** and align to the live machine-readable policy plus the dated review log.
