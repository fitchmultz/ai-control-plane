# Presentation Readiness Tracker

This file defines the canonical tracker format and the source-of-truth workflow for current presentation readiness.

The current tracker is no longer maintained here as a stale committed snapshot. Generate it locally with:

```bash
make readiness-evidence
make readiness-evidence-verify
```

The generated tracker for the latest run lives at:

- `demo/logs/evidence/readiness-<TIMESTAMP>/presentation-readiness-tracker.md`
- `demo/logs/evidence/latest-run.txt`
- `demo/logs/evidence/latest-success.txt`

## Gate Definitions

| Gate | What it proves | Canonical command surface |
| --- | --- | --- |
| Local CI Gate | The host-first validated baseline is green | `make ci` |
| Production CI Gate | Customer-like host invariants pass when secrets are available | `make readiness-evidence READINESS_INCLUDE_PRODUCTION=1 SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env` |
| Active-Passive Failover Drill Evidence | Customer-operated HA failover proof exists with replication, fencing, promotion, cutover, and postcheck evidence | `make ha-failover-drill HA_FAILOVER_MANIFEST=demo/logs/recovery-inputs/ha_failover_drill.yaml` |
| Supply Chain Gate | The supply-chain contract passes | `make supply-chain-gate` |
| Allowlist Expiry Gate | Time-bound exceptions remain within policy | `make supply-chain-allowlist-expiry-check` |
| Release Bundle Gate | Deployment bundle builds and verifies | `make release-bundle`; `make release-bundle-verify` |
| Evidence Completeness Gate | The run directory contains an auditable summary and inventory | `make readiness-evidence-verify` |

## External Use Rule

Do not use this repository externally as "currently certified" unless a fresh generated tracker exists for the current decision window.

## Related Documents

- [READINESS_EVIDENCE_WORKFLOW.md](READINESS_EVIDENCE_WORKFLOW.md)
- [GO_NO_GO.md](GO_NO_GO.md)
- [go_no_go_decision.md](go_no_go_decision.md)
