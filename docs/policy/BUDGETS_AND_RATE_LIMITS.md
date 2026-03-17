# Budgets and Rate Limits

This document defines budget and rate-limit operations for the public AI Control Plane snapshot.

## Overview

Budget and rate-limit controls exist to:

1. cap spend
2. prevent quota exhaustion
3. enforce fair usage across keys/workloads

Controls are configured at multiple layers in LiteLLM:

- global proxy settings
- model-level limits
- key-level limits

## Configuration Layers

### 1) Global proxy controls

Defined in `demo/config/litellm.yaml` under `litellm_settings`.

Typical fields:
- `max_budget`
- `budget_duration`
- `max_parallel_requests`

### 2) Model-level limits

Defined per `model_list` entry in `demo/config/litellm.yaml`.

Typical fields:
- `rpm`
- `tpm`

### 3) Key-level limits

Set when generating a key.

Canonical operator commands:

```bash
# Standard key
make key-gen ALIAS=my-key BUDGET=10.00

# Role-shaped presets
make key-gen-dev ALIAS=my-dev-key
make key-gen-lead ALIAS=my-lead-key

# Inventory, inspection, and rotation
make key-list
make key-inspect ALIAS=my-key REPORT_MONTH=2026-02
make key-rotate ALIAS=my-key DRY_RUN=1

# Revoke a key
make key-revoke ALIAS=<alias>

# acpctl equivalents
./scripts/acpctl.sh key gen my-key --budget 10.00
./scripts/acpctl.sh key gen-dev my-dev-key
./scripts/acpctl.sh key gen-lead my-lead-key
./scripts/acpctl.sh key list
./scripts/acpctl.sh key inspect my-key --month 2026-02
./scripts/acpctl.sh key rotate my-key --dry-run
./scripts/acpctl.sh key revoke my-key
```

## Monitoring and Validation

### Budget/usage visibility

```bash
make db-status
make key-inspect ALIAS=my-key REPORT_MONTH=2026-02
```

### Detection validation

```bash
make validate-detections
```

### SIEM sync validation

```bash
make validate-siem-queries
```

## Recommended Operating Cadence

### Daily

- `make health`
- `make db-status`

### Weekly

- `make validate-detections`
- `make validate-siem-queries`

### Before release/handoff

- `make ci-pr`
- `make ci`

## Testing Workflow

Use the canonical scenario surface:

```bash
# Run a specific scenario
make demo-scenario SCENARIO=4

# Run all scenarios
make demo-all
```

## Detection Rules Related to Spend Risk

Refer to `demo/config/detection_rules.yaml` for thresholds and remediation text.

Common budget-related controls include:
- budget exhaustion warning
- budget threshold alert

Run with canonical commands:

```bash
make detection
# alias of validate-detections
```

## Troubleshooting

### Key creation fails

- Verify services are healthy: `make health`
- Confirm master key/env setup in `demo/.env`
- Validate config: `make validate-config`

### Detection validation fails

- Verify PostgreSQL and gateway availability (`make health`)
- Check rule syntax in `demo/config/detection_rules.yaml`

### Budget data looks stale

- Confirm requests are actually flowing through gateway
- Re-run `make db-status`
- Inspect gateway logs with `make logs`

## Public Snapshot Boundary

Legacy script-path workflows and approval-queue script flows are intentionally not documented in this snapshot.

Use only `make` and `./scripts/acpctl.sh` entrypoints.
