# Multi-Tenant Design Example

This example points to the tracked ACP design-only tenant package.

## Canonical source

- `demo/config/tenant_design.yaml`

## What it demonstrates

- organization boundary for provider billing
- workspace boundary for key namespaces and internal chargeback
- row-level access predicate requirements
- tenant-safe metadata-only reporting rules

## Validate it

```bash
make validate-tenant
make tenant-inspect
./scripts/acpctl.sh tenant inspect --format json
```

## Important

This is a design package only. It does not mean ACP currently supports shared-runtime multi-tenant operation.
