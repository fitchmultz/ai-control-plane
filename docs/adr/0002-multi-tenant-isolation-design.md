# 0002: Multi-tenant isolation design package before runtime claims

- Status: Accepted
- Date: 2026-03-19

## Context

The roadmap required a credible multi-tenant isolation and service-provider billing path before any managed-service claim is made.

The current validated repository surface is still single-tenant and host-first. LiteLLM owns the runtime schema, so the repo cannot truthfully claim shared-runtime tenant isolation until the product has tenant-aware key namespaces, query predicates, reporting boundaries, and operating evidence.

## Decision

Adopt a tracked, incubating, design-only package for future multi-tenant support with these hard boundaries:

- organization is the service-provider customer invoicing boundary
- workspace is the write, reporting, and internal chargeback boundary
- row-level access must require `organization_id`, `workspace_id`, and `key_namespace_prefix`
- tenant-safe reporting stays metadata-only by default
- cross-organization allocation and cross-organization reporting are forbidden in the design package
- all tenant-facing bindings stay workspace-scoped; no global cross-tenant role inheritance is implied

The package is tracked in `demo/config/tenant_design.yaml`, validated by `acpctl tenant validate`, and documented in [policy/MULTI_TENANT_ISOLATION_AND_BILLING.md](../policy/MULTI_TENANT_ISOLATION_AND_BILLING.md).

## Consequences

- The repository now has a concrete, inspectable, design-only contract for organization/workspace isolation and provider billing boundaries.
- Public docs can point to a real design package without overstating runtime capability.
- Multi-tenant managed-service claims remain out until runtime enforcement and environment validation exist.
- Future implementation work should cut over directly to this contract instead of inventing parallel tenant models.

## Alternatives Considered

- Claiming managed-service readiness based on strategy docs alone
- Delaying all tenant work until a runtime build existed
- Using organization-wide write/report scopes that would increase leakage risk
