# Readiness Evidence Workflow

This document defines the canonical workflow for regenerating current enterprise-readiness proof.

Generated evidence is local-only and intentionally not committed. See [ARTIFACTS.md](../ARTIFACTS.md).

## What This Workflow Proves

`make readiness-evidence` proves the current state of the repository's validated baseline:

- Local CI and runtime checks pass together.
- Supply-chain policy gates pass.
- Release bundles build and verify cleanly.
- A dated, inspectable evidence pack exists for the run.

It does **not** prove customer-environment controls such as SWG/CASB enforcement, IdP policy correctness, browser management, or customer SIEM retention. Those must be validated during the pilot with customer-owned systems.

When `READINESS_INCLUDE_PRODUCTION=1` is used and a truthful `demo/logs/recovery-inputs/ha_failover_drill.yaml` manifest is present, the production-only evidence plan also archives the customer-operated active-passive failover drill proof.

## Canonical Commands

Baseline proof pack:

```bash
make readiness-evidence
make readiness-evidence-verify
```

Customer-like production proof pack when secrets are available:

```bash
make readiness-evidence READINESS_INCLUDE_PRODUCTION=1 \
  SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
make readiness-evidence-verify
```

Direct CLI form:

```bash
./scripts/acpctl.sh deploy readiness-evidence run
./scripts/acpctl.sh deploy readiness-evidence verify
```

## Generated Artifacts

Each run creates a directory under `demo/logs/evidence/readiness-<TIMESTAMP>/`.

Expected artifacts:

- `summary.json` — machine-readable run summary
- `readiness-summary.md` — concise operator summary
- `presentation-readiness-tracker.md` — generated gate tracker for the run
- `go-no-go-decision.md` — generated decision memo for the run
- `evidence-inventory.txt` — inventory of files in the run directory
- `make-*.log` — command logs for each executed gate

Pointer files under `demo/logs/evidence/`:

- `latest-run.txt` — most recent generated run
- `latest-success.txt` — most recent passing run

## External Reuse Rules

Use a readiness run externally only when all of the following are true:

- Required gates in the run are `PASS`
- Evidence is fresh for the meeting or decision window
- The customer-specific proof gaps are called out explicitly
- The generated tracker and decision memo are referenced, not stale committed snapshots

## Related Documents

- [PRESENTATION_READINESS_TRACKER.md](PRESENTATION_READINESS_TRACKER.md)
- [go_no_go_decision.md](go_no_go_decision.md)
- [GO_NO_GO.md](GO_NO_GO.md)
- [../ENTERPRISE_PILOT_PACKAGE.md](../ENTERPRISE_PILOT_PACKAGE.md)
- [../PILOT_CONTROL_OWNERSHIP_MATRIX.md](../PILOT_CONTROL_OWNERSHIP_MATRIX.md)
