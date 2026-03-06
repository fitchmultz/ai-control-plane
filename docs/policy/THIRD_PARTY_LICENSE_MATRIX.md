# Third-Party License Policy Matrix

**Policy ID**: `acp-third-party-license-boundary`  
**Version**: 1.0.0  
**Last Updated**: 2026-02-09

## Overview

This document defines the third-party license policy for the AI Control Plane project. It establishes clear boundaries between:

- **Allowed Components**: OSS-licensed components safe for customer packaging
- **Restricted Components**: Commercially-licensed components requiring explicit approval
- **Conditional Components**: Components with usage restrictions or requirements

## Purpose

Prevent accidental inclusion of restricted commercial artifacts in OSS/customer packaging, ensuring compliance with upstream license terms and avoiding legal/commercial missteps during sales and deployment.

## Policy Scope

### Included Paths (Packaging-Sensitive)

The following paths are scanned for license boundary enforcement:

- `Makefile` - Build orchestration
- `mk/**/*.mk` - Build module definitions
- `scripts/**/*.sh` - Shell entrypoints and automation
- `cmd/acpctl/**/*.go` - CLI command surface
- `internal/**/*.go` - Typed implementation modules
- `demo/docker-compose*.yml` - Service orchestration
- `demo/config/**/*.yaml` - Gateway configuration
- `deploy/**/*` - Deployment configurations

### Excluded Paths

- `.ralph/**` - Task queue and cache
- `.git/**` - Version control
- `demo/logs/**` - Application logs
- `demo/backups/**` - Database backups
- `handoff-packet/**` - Runtime/export artifacts
- Build artifacts and dependencies

## Allowed Components

### LiteLLM (OSS Edition)

| Attribute | Value |
|-----------|-------|
| **License** | MIT |
| **Boundary** | Non-enterprise code only |
| **Source** | [LICENSE](https://raw.githubusercontent.com/BerriAI/litellm/main/LICENSE) |
| **Usage** | Gateway proxy, virtual keys, budgets, model routing |

The LiteLLM repository contains both MIT-licensed OSS code and commercially-licensed enterprise code. **Only the OSS components** may be used in customer packages without an explicit enterprise agreement.

### LibreChat

| Attribute | Value |
|-----------|-------|
| **License** | MIT |
| **Source** | [LICENSE](https://raw.githubusercontent.com/danny-avila/LibreChat/main/LICENSE) |
| **Usage** | Optional chat UI component |

### PostgreSQL

| Attribute | Value |
|-----------|-------|
| **License** | PostgreSQL License (MIT-like) |
| **Source** | [LICENSE](https://www.postgresql.org/about/licence/) |
| **Usage** | Database persistence, spend logs, key metadata |

### Caddy

| Attribute | Value |
|-----------|-------|
| **License** | Apache-2.0 |
| **Source** | [LICENSE](https://github.com/caddyserver/caddy/blob/master/LICENSE) |
| **Usage** | TLS termination, reverse proxy, HTTPS |

## Restricted Components

### LiteLLM Enterprise

| Attribute | Value |
|-----------|-------|
| **Severity** | Blocking |
| **License** | Commercial (requires enterprise agreement) |
| **Source** | [LICENSE.md](https://raw.githubusercontent.com/BerriAI/litellm/main/enterprise/LICENSE.md) |

**Description**: LiteLLM Enterprise components require a commercial license for production use. These include SSO integration, enterprise audit logs, enterprise admin UI, and other premium features.

**Detection Patterns**:

- Path patterns: `litellm/enterprise/`, `litellm-enterprise/`, `BerriAI/litellm/.*/enterprise/`
- Content patterns: `litellm-enterprise`, `from litellm.enterprise`, `import litellm.enterprise`
- Package references: `litellm-enterprise`, `litellm[enterprise]`

**Remediation**: Use LiteLLM OSS edition only. If enterprise features are required, contact BerriAI for a commercial license agreement.

### Authentication Feature Matrix

Per the [Enterprise Authentication Architecture](../security/ENTERPRISE_AUTH_ARCHITECTURE.md), authentication capabilities have the following license boundaries:

| Feature | OSS (MIT) | Enterprise (Commercial) | Notes |
|---------|-----------|------------------------|-------|
| **API Key Authentication** | ✅ | ✅ | Virtual keys, budgets, basic auth |
| **Shared-Key Attribution** | ✅ | ✅ | Default scalable attribution model |
| **LibreChat OIDC/LDAP/SAML** | ✅ | ✅ | Via LibreChat (MIT licensed) |
| **LiteLLM Enterprise SSO** | ❌ | ✅ | Native SAML/OIDC in LiteLLM proxy |
| **Enterprise Audit Logs** | ❌ | ✅ | Enhanced audit with session correlation |
| **Enterprise Admin UI** | ❌ | ✅ | Administrative web interface |
| **Advanced enforced_params** | ⚠️ Basic | ✅ | Strict claim validation modes |

**OSS-First Profile**: All auth requirements can be met using OSS-only components via LibreChat authentication + LiteLLM shared-key attribution.

**Enterprise-Enhanced Profile**: May use LiteLLM Enterprise features for advanced identity controls, but must document active license coverage.

## Override Process

In exceptional cases, restricted components may be approved with explicit override documentation.

### Required Override Fields

| Field | Description |
|-------|-------------|
| `id` | Unique override identifier |
| `approver` | Person or process approving |
| `rationale` | Business justification |
| `expires_at` | ISO 8601 expiration timestamp (must be finite) |
| `scope` | Specific component and path scope |
| `reviewed_by_legal` | Legal review completion status |

### Override Behavior

- Expired overrides cause CI failure (`fail_on_expired`)
- All overrides are logged in generated license reports
- Overrides should be reviewed quarterly

## Verification

### Automated Checks

The license boundary guard runs automatically in CI:

```bash
# Check license compliance
make license-check

# Regenerate license report
make license-report-update
```

### Compliance Report

A generated compliance report is included in deployment handoff assets:

- **Location**: `docs/deployment/THIRD_PARTY_LICENSE_SUMMARY.md`
- **Content**: Policy version, allowed/restricted matrix, scan results, overrides applied
- **Determinism**: Stable ordering for reliable diff checking

## References

- [LiteLLM License](https://raw.githubusercontent.com/BerriAI/litellm/main/LICENSE)
- [LiteLLM Enterprise License](https://raw.githubusercontent.com/BerriAI/litellm/main/enterprise/LICENSE.md)
- [LibreChat License](https://raw.githubusercontent.com/danny-avila/LibreChat/main/LICENSE)
- [Production Handoff Runbook](../deployment/PRODUCTION_HANDOFF_RUNBOOK.md)

## Updates

This policy should be reviewed:

1. **Quarterly** as part of regular compliance review
2. **Immediately** when upstream license changes are detected
3. **Before major releases** to ensure packaging compliance

---

*This policy is enforced by `make license-check` (see `mk/security.mk` for implementation details).*
