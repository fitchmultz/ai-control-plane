# Third-Party License Summary

**Generated:** 2026-03-05T00:00:00Z
**Policy ID:** `acp-third-party-license-boundary`
**Policy Version:** `1.0.0`

## Executive Summary

This report records the current license-boundary status for packaging-sensitive project assets.

| Metric | Value |
|--------|-------|
| Status | ✅ PASS |
| Violations Found | 0 |
| Overrides Applied | 0 |

## Allowed Components

| Component | License | Boundary |
|-----------|---------|----------|
| LiteLLM (OSS) | MIT | non-enterprise-only |
| LibreChat | MIT | N/A |
| PostgreSQL | PostgreSQL License | N/A |
| Caddy | Apache-2.0 | N/A |

## Restricted Components

| Component | Severity | Reason |
|-----------|----------|--------|
| LiteLLM Enterprise | blocking | commercial-license-boundary |

## Scan Scope (Canonical)

### Included Paths

- `Makefile`
- `mk/**/*.mk`
- `scripts/**/*.sh`
- `cmd/acpctl/**/*.go`
- `internal/**/*.go`
- `demo/docker-compose*.yml`
- `demo/config/**/*.yaml`
- `deploy/**/*`

### Excluded Paths

- `.git/**`
- `.ralph/**`
- `demo/logs/**`
- `demo/backups/**`
- `handoff-packet/**`
- `**/node_modules/**`
- `**/vendor/**`
- generated/minified assets and this report file

## Verification Commands

```bash
# Enforce license boundary policy
make license-check

# Refresh this report in-repo
make license-report-update
```

## References

- Policy JSON: `docs/policy/THIRD_PARTY_LICENSE_MATRIX.json`
- Policy Doc: `docs/policy/THIRD_PARTY_LICENSE_MATRIX.md`
- Production handoff: `docs/deployment/PRODUCTION_HANDOFF_RUNBOOK.md`

---

This summary is intentionally concise to reduce drift. The policy JSON is the canonical machine-readable source.
