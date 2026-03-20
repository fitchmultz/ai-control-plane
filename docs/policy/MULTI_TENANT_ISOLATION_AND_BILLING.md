# Multi-Tenant Isolation And Billing Design

This document captures the ACP design-only package for future multi-tenant expansion.

## Status

- **Current status:** incubating, design-only
- **Validated today:** single-tenant host-first runtime
- **Not validated today:** shared-runtime multi-tenant managed service

Use this document to understand the target boundary without implying it already exists in production.

## Why this package exists

Roadmap item #28 required a credible answer to four hard questions before any managed-service claim:

1. How are organizations and workspaces isolated?
2. How is row-level access enforced on shared data?
3. How are reports kept tenant-safe?
4. How are internal chargeback and provider billing kept separate?

The design package answers those questions with tracked config, typed validation, and operator inspection commands.

## Canonical tracked assets

- `demo/config/tenant_design.yaml`
- `docs/contracts/config/tenant_design.schema.json`
- `docs/adr/0002-multi-tenant-isolation-design.md`
- `./scripts/acpctl.sh tenant inspect`
- `./scripts/acpctl.sh tenant validate`
- `make validate-tenant`
- `make tenant-inspect`

## Isolation model

### Organization boundary

An organization is the top-level customer tenancy boundary.

It owns:
- customer invoice boundary
- provider-facing bill-to identity
- cross-workspace policy umbrella inside that customer only

It does **not** imply shared reporting or allocation across customers.

### Workspace boundary

A workspace is the mandatory write and internal chargeback boundary.

It owns:
- workspace-scoped key namespace prefix
- workspace-scoped role bindings
- allowed model set for that workspace
- workspace cost center for internal showback/chargeback

## Row-level access design

The design package requires every future shared-runtime query path to carry these predicates:

- `organization_id`
- `workspace_id`
- `key_namespace_prefix`

Interpretation:
- `organization_id` prevents cross-customer reads
- `workspace_id` prevents cross-workspace leakage inside one customer
- `key_namespace_prefix` keeps key attribution and report rows aligned to the same workspace boundary

## Reporting design

Tenant-safe reporting is metadata-only by default.

Allowed aggregation levels:
- workspace
- organization

Forbidden by design:
- cross-organization reporting
- content-bearing reports
- silent fallback from workspace scope to global scope

## Financial boundaries

### Internal chargeback

Internal chargeback stays at the **workspace** boundary.

Use it for:
- business-unit showback
- departmental allocation
- workspace budget ownership

### Provider billing

Provider billing stays at the **organization** boundary.

Use it for:
- customer invoice recipient
- pass-through usage charges
- separate provider platform fee

This separation avoids the common failure mode where internal workspace allocation gets confused with the external customer invoice.

## RBAC design boundary

The current runtime RBAC model is global and single-tenant.

The design package tightens the future contract by requiring:
- workspace-scoped bindings only
- no implicit cross-tenant role inheritance
- workspace model allowlists that must remain reachable from the tracked RBAC role catalog

## Operating rule

Until runtime implementation exists, use one ACP deployment per customer boundary for any environment that requires strong tenant isolation.

## Verification

```bash
make validate-tenant
make tenant-inspect
./scripts/acpctl.sh validate tenant
./scripts/acpctl.sh tenant inspect --format json
```

## Claim boundary

This package is evidence of design discipline, not evidence of shipped multi-tenant enforcement.
