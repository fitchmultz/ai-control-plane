# Approved Models Policy

## Purpose

This document defines the approved-model contract for the AI Control Plane public snapshot.

The goal is to keep model access deterministic across:
- gateway routing
- detection logic
- SIEM query mappings
- managed UI configuration

## Canonical Source of Truth

The canonical allowlist lives in:

- `demo/config/litellm.yaml` → `model_list[].model_name`

Only these `model_name` aliases are considered approved for operator-facing workflows.

## Downstream Surfaces That Must Stay in Sync

When model aliases change, validate these surfaces:

1. `demo/config/litellm.yaml`
2. `demo/config/librechat/librechat.yaml` (if LibreChat is enabled)
3. `demo/config/siem_queries.yaml`
4. `demo/config/detection_rules.yaml` (especially DR-001 model-policy checks)

## Operator Entry Points (Canonical)

Use Make or `acpctl` as the only documented operator interfaces.

```bash
# Validate SIEM query/rule synchronization
make validate-siem-queries

# Validate detection output contract
make validate-detections

# Run fast deterministic gate before commit
make ci-pr

# acpctl equivalents
./scripts/acpctl.sh validate siem-queries
./scripts/acpctl.sh validate detections
```

## Update Procedure

When you add/remove/rename a model alias:

1. Update `demo/config/litellm.yaml`.
2. Update any explicit allowlists in `demo/config/librechat/librechat.yaml`.
3. Re-run validation gates:

```bash
make validate-siem-queries
make validate-detections
make ci-pr
```

4. If schema/mapping changes are involved, also run:

```bash
make validate-siem-schema
```

## Security and Correctness Invariants

1. Approved-model checks must use aliases from `litellm.yaml`, not ad hoc hardcoded lists.
2. Detection and SIEM mappings must stay synchronized (`validate-siem-queries` must pass).
3. Public docs must not reference retired legacy script-path command patterns.

## Troubleshooting

### `make validate-siem-queries` fails

- Ensure every `rule_id` in `demo/config/detection_rules.yaml` exists in `demo/config/siem_queries.yaml`.
- Ensure enabled rules include required platform query sections.

### `make validate-detections` fails

- Confirm services are healthy (`make health`).
- Verify `demo/config/detection_rules.yaml` syntax and required fields.

### LibreChat model list drift

- Ensure LibreChat model defaults match approved aliases from `litellm.yaml`.
- Keep `fetch: false` when you need deterministic allowlists.

## Public Snapshot Boundary

Legacy script-path workflows were retired for this snapshot.

Use only:
- `make <target>`
- `./scripts/acpctl.sh <group> <subcommand>`

for documentation and operator runbooks.
