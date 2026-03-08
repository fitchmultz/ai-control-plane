# Role-Based Access Control (RBAC)

This document describes the RBAC model and the public operator surface for role-aware key lifecycle operations.

## Scope

RBAC in this snapshot is focused on:
- role-shaped key issuance patterns
- model/budget policy mapping in configuration
- deterministic operator entrypoints (`make` / `acpctl`)

## Canonical Configuration

Role policy source:
- `demo/config/roles.yaml`

Model policy source:
- `demo/config/litellm.yaml`

Use these files as the authoritative policy definition for role/mode constraints.

## Role Model (Policy Intent)

Typical roles represented in configuration:
- `admin`
- `team-lead`
- `developer`
- `auditor`

Expected policy dimensions:
- model access scope
- budget ceilings
- approval authority (if implemented in downstream/private workflows)

## Canonical Operator Commands

Use these entrypoints for role-shaped key lifecycle actions:

```bash
# Standard key issuance
make key-gen ALIAS=my-key BUDGET=10.00

# Developer preset
make key-gen-dev ALIAS=my-dev-key

# Team-lead preset
make key-gen-lead ALIAS=my-lead-key

# Key revocation
make key-revoke ALIAS=<alias>

# acpctl equivalents
./scripts/acpctl.sh key gen my-key --budget 10.00
./scripts/acpctl.sh key gen-dev my-dev-key
./scripts/acpctl.sh key gen-lead my-lead-key
./scripts/acpctl.sh key revoke my-key
```

## Validation Workflow

```bash
# Config and policy validation
make validate-config
make validate-detections
make validate-siem-queries

# Fast deterministic gate
make ci-pr
```

## Operational Guidance

1. Keep `roles.yaml` and model allowlists aligned.
2. Prefer preset commands (`key-gen-dev`, `key-gen-lead`) for standard role workflows.
3. Use `key-revoke` for immediate containment in incident response.
4. Validate rule and SIEM consistency after policy changes.

## Incident Response Tie-In

For suspected key compromise:

```bash
make key-revoke ALIAS=<alias>
make validate-detections
make db-status
```

## Public Snapshot Boundary

Legacy command paths and legacy approval-queue scripts are not canonical in this repository snapshot.

Not documented as operator entrypoints:
- retired legacy script-path workflows

Use only:
- `make <target>`
- `./scripts/acpctl.sh <group> <subcommand>`
