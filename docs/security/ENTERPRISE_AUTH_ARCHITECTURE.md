# Enterprise Authentication Architecture (LibreChat + LiteLLM)

**Document ID**: `acp-enterprise-auth-architecture`  
**Version**: 1.0.0  
**Last Updated**: 2026-02-11  
**Status**: Canonical

---

## Purpose

This document defines the production-usable, license-aware authentication architecture for deploying LibreChat + LiteLLM in enterprise environments. It establishes clear boundaries between open-source (OSS) capabilities and enterprise-licensed features, ensuring organizations can deploy confidently with appropriate identity and governance controls.

### Goals

- Provide two well-defined authentication profiles for different organizational needs
- Establish deterministic identity claim mapping for governance attribution
- Enable scalable user attribution without requiring per-user API keys
- Define trust boundaries and security invariants
- Document migration paths between deployment modes

### Non-Goals

- This document does NOT replace vendor-specific authentication documentation
- This document does NOT mandate specific identity providers (IdPs)
- This document does NOT cover multi-tenant deployment architectures
- This document does NOT prescribe specific network topologies
- This document does NOT implement customer egress/SWG/CASB controls; those are mandatory customer-side network responsibilities for bypass prevention
- This document does NOT prove customer browser/workspace enforcement; that proof belongs in the pilot track and customer validation record

---

## Profile Overview

The AI Control Plane supports **exactly two** authentication profiles:

| Profile | Purpose | License Requirements |
|---------|---------|---------------------|
| **OSS-First** (Default) | Cost-effective, scalable deployment using shared service keys with trusted user context propagation | OSS-only |
| **Enterprise-Enhanced** | Same baseline with additional enterprise identity controls and stricter policy enforcement | May use LiteLLM Enterprise features |

### Profile Selection Guidance

```
┌─────────────────────────────────────────────────────────────────┐
│  Profile Selection Decision Tree                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Do you require enterprise SSO integration                       │
│  (SAML, OIDC with advanced policy)?                             │
│       │                                                          │
│       ├── NO → Use OSS-First Profile                             │
│       │                                                          │
│       └── YES → Do you have LiteLLM Enterprise license?         │
│                 │                                                │
│                 ├── NO → Use OSS-First + LibreChat OIDC/LDAP    │
│                 │                                                │
│                 └── YES → Use Enterprise-Enhanced Profile       │
│                           (enterprise audit logs, admin UI)     │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Profile 1: OSS-First (Default)

### Characteristics

The OSS-First profile provides enterprise-grade authentication using only open-source components:

- **User authentication surface**: LibreChat handles OIDC, LDAP, or local authentication
- **Gateway authentication**: Shared virtual key with trusted server-side user context
- **Attribution model**: Spend logs capture user identity via trusted context propagation
- **Scalability**: No per-user key issuance required

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Trust Boundaries                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────────────────┐ │
│  │   Browser    │────▶│  LibreChat   │────▶│      LiteLLM Gateway     │ │
│  │  (Untrusted) │     │   (Trust 1)  │     │       (Trust 2)          │ │
│  └──────────────┘     └──────────────┘     └──────────────────────────┘ │
│         │                   │                           │               │
│         │                   │ User context forwarded    │               │
│         │                   │ via OpenAI-compatible     │               │
│         │                   │ "user" field              │               │
│         │                   ▼                           ▼               │
│         │            ┌──────────────┐            ┌──────────────┐       │
│         │            │  MongoDB     │            │  PostgreSQL  │       │
│         │            │  (Chat data) │            │  (Spend logs)│       │
│         │            └──────────────┘            └──────────────┘       │
│         │                                                               │
│         ▼                                                               │
│    ═══════════════════════════════════════════════════════════════     │
│    WARNING: Browser-originated identity claims are UNTRUSTED          │
│    unless re-bound by LibreChat server authentication context.        │
│    ═══════════════════════════════════════════════════════════════     │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Identity Claim Flow

```
1. User authenticates to LibreChat (OIDC/LDAP/Local)
        │
        ▼
2. LibreChat creates/validates user session
        │
        ▼
3. User makes chat request
        │
        ▼
4. LibreChat forwards to LiteLLM with:
   - Authorization: Bearer <shared-service-key>
   - Request body includes: {"user": "<authenticated-user-id>"}
        │
        ▼
5. LiteLLM logs to "LiteLLM_SpendLogs".user
        │
        ▼
6. Export pipeline resolves principal.id from spend logs
```

### Required Configuration

#### LibreChat Configuration

```yaml
# demo/config/librechat/librechat.yaml
endpoints:
  custom:
    - name: "LiteLLM"
      apiKey: "${LIBRECHAT_LITELLM_API_KEY}"  # Shared service key
      baseURL: "http://litellm:4000/v1"
      models:
        default:
          - "claude-haiku-4-5"
          - "openai-gpt5.2"
        fetch: false
      # CRITICAL: Preserve "user" field for attribution
      # Do NOT add dropParams that includes "user"
```

#### LiteLLM Configuration

```yaml
# demo/config/litellm.yaml (OSS-First baseline)
general_settings:
  # Required for subscription-through-gateway flows
  forward_client_headers_to_llm_api: true

litellm_settings:
  # Reduce risk of key leakage in logs
  redact_user_api_key_info: true
  master_key: os.environ/LITELLM_MASTER_KEY
```

### Attribution Precedence (Canonical)

Gateway exports resolve `principal.id` with deterministic precedence:

```sql
CASE
  -- 1. Valid LiteLLM_SpendLogs.user (preferred)
  WHEN NULLIF(BTRIM(s."user"), '') IS NOT NULL
    AND LOWER(BTRIM(s."user")) <> 'unknown'
    AND BTRIM(s."user") !~ '\s'
    THEN BTRIM(s."user")
  
  -- 2. Valid LiteLLM_VerificationToken.user_id (fallback)
  WHEN NULLIF(BTRIM(v.user_id), '') IS NOT NULL
    AND LOWER(BTRIM(v.user_id)) <> 'unknown'
    AND BTRIM(v.user_id) !~ '\s'
    THEN BTRIM(v.user_id)
  
  -- 3. Valid key_alias (service identification)
  WHEN NULLIF(BTRIM(v.key_alias), '') IS NOT NULL
    AND LOWER(BTRIM(v.key_alias)) <> 'unknown'
    AND BTRIM(v.key_alias) !~ '\s'
    THEN BTRIM(v.key_alias)
  
  -- 4. Fail-safe to unknown
  ELSE 'unknown'
END AS principal_id
```

### Identity Source Tracking

| Source | Meaning | Use Case |
|--------|---------|----------|
| `spendlogs_user` | Identity from SpendLogs.user (highest trust) | Normal steady-state operation |
| `token_user_id` | Fallback to VerificationToken.user_id | Legacy or direct API access |
| `key_alias` | Service-level identification | No user context available |
| `unknown` | Identity missing/invalid | Investigation required |

### Security Invariants

1. **Shared-key baseline**: Default attribution uses shared service key + trusted user context
2. **Browser identity untrusted**: Client-side claims require server-side re-binding
3. **Per-user keys optional**: Not required for baseline attribution; available as hardening
4. **Fail-safe resolution**: Missing/invalid identity defaults to `unknown` with explicit reason

---

## Profile 2: Enterprise-Enhanced

### Characteristics

The Enterprise-Enhanced profile extends OSS-First with licensed LiteLLM Enterprise features:

- **All OSS-First capabilities** (shared-key attribution, LibreChat auth integration)
- **Enterprise identity controls**: SSO enforcement, advanced audit logging
- **Stricter policy gates**: Required claims validation, enhanced RBAC
- **Administrative UI**: Enterprise admin dashboard for key management

### License Requirements

| Component | OSS | Enterprise |
|-----------|-----|------------|
| LiteLLM Gateway | ✅ MIT | ✅ MIT |
| Virtual Keys & Budgets | ✅ Included | ✅ Included |
| Basic Auth (API keys) | ✅ Included | ✅ Included |
| OIDC/LDAP (LibreChat) | ✅ MIT | ✅ MIT |
| **LiteLLM Enterprise SSO** | ❌ | ✅ Licensed |
| **Enterprise Audit Logs** | ❌ | ✅ Licensed |
| **Enterprise Admin UI** | ❌ | ✅ Licensed |
| **Enforced Params** | ⚠️ Basic | ✅ Advanced |

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     Enterprise-Enhanced Architecture                     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────────────────┐ │
│  │   Browser    │────▶│  LibreChat   │────▶│      LiteLLM Gateway     │ │
│  │              │     │  (OIDC/SAML) │     │    (Enterprise Enabled)  │ │
│  └──────────────┘     └──────────────┘     └──────────────────────────┘ │
│                              │                           │              │
│                              │                           │              │
│                              ▼                           ▼              │
│                       ┌──────────────┐            ┌──────────────┐      │
│                       │   IdP        │            │  Enterprise  │      │
│                       │ (Corporate)  │            │  Audit Logs  │      │
│                       └──────────────┘            └──────────────┘      │
│                                                                          │
│  Additional Enterprise Components:                                       │
│  - Enhanced audit logging with user session correlation                  │
│  - Enterprise admin UI for key lifecycle management                      │
│  - Advanced enforced_params for mandatory claim validation               │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Strict Attribution Mode (Optional)

Enterprise deployments may enforce stricter identity requirements:

```yaml
# LiteLLM Enterprise configuration example
general_settings:
  # Enforce user parameter presence (reject requests without attribution)
  enforced_params:
    - user
  
  # Additional enterprise settings
  enable_sso: true
  sso_config:
    # Enterprise SSO configuration
```

### Identity Claim Flow

```
1. User authenticates to LibreChat (OIDC/LDAP/SAML)
        │
        ▼
2. LibreChat validates the server-side session and role context
        │
        ▼
3. LibreChat forwards request to LiteLLM with:
   - Authorization: Bearer <shared-service-key>
   - Request body includes: {"user": "<authenticated-user-id>"}
        │
        ▼
4. LiteLLM enterprise policy checks run:
   - If strict mode enabled and user missing/invalid -> HTTP 400 reject
   - Otherwise continue and log attribution metadata
        │
        ▼
5. Spend logs + enterprise audit logs capture request and identity metadata
        │
        ▼
6. Export pipeline resolves principal.id / principal.role for governance outputs
```

### Denied-Auth Behavior

When strict mode is enabled and required claims are missing:

| Scenario | OSS-First | Enterprise-Enhanced (Strict) |
|----------|-----------|------------------------------|
| Missing `user` claim | Logs `unknown`, continues | Rejects request (HTTP 400) |
| Invalid `user` format | Falls back to next trusted source; logs fallback reason | Rejects request (HTTP 400) |
| Unauthenticated request | Rejects (401) | Rejects (401) |

### Migration Path: OSS-First → Enterprise-Enhanced

```
Phase 1: Deploy OSS-First Profile
├── Set up LibreChat with OIDC/LDAP
├── Configure shared-key attribution
└── Validate identity resolution in spend logs

Phase 2: Enable Enterprise Features
├── Acquire LiteLLM Enterprise license
├── Enable enterprise audit logging
├── Configure enterprise admin UI
└── Validate enhanced logging correlation

Phase 3: Enable Strict Mode (Optional)
├── Configure enforced_params
├── Test denied-auth scenarios
├── Update monitoring for 400/rejection rates
└── Document policy exception procedures
```

---

## Claim Mapping Reference

### Identity Claims to Governance Metadata

| Source Claim | Governance Field | Description |
|--------------|------------------|-------------|
| `LiteLLM_SpendLogs.user` | `principal.id` | Primary user identifier |
| `VerificationToken.user_id` | `principal.id` (fallback) | Secondary user identifier |
| `VerificationToken.key_alias` | `principal.id` (service) | Service account identifier |
| Resolved source | `principal.identity_source` | How identity was resolved |
| Fallback reason | `principal.identity_reason` | Why fallback occurred |

### Role Mapping Contract

Role attribution follows deterministic precedence:

```
1. Explicit user_role_assignments match (roles.yaml)
2. default_role fallback (roles.yaml)
3. unknown fail-safe (if config missing)
```

Output fields:
- `principal.role`: Resolved RBAC role (`admin`, `team-lead`, `developer`, `auditor`)
- `principal.role_source`: How role was resolved (`user_role_assignment`, `default_role`, `unknown`)

### RBAC Roles Reference

| Role | Model Access | Budget Ceiling | Can Approve |
|------|-------------|----------------|-------------|
| `admin` | All models | $500.00 | Yes (any) |
| `team-lead` | Standard + Sonnet | $100.00 | Yes (up to $50) |
| `developer` | Standard only | $25.00 | No |
| `auditor` | None (read-only) | $0.00 | No |

---

## Security Assumptions and Invariants

### Trust Boundaries

| Boundary | Trust Level | Enforcement |
|----------|-------------|-------------|
| Browser → LibreChat | Untrusted → Trusted | LibreChat authentication layer |
| LibreChat → LiteLLM | Trusted | Network isolation + shared keys |
| LiteLLM → Providers | Trusted | API key authentication |

### Non-Negotiable Invariants

1. **Default attribution is shared service key + trusted server-authenticated user context**
2. **Browser-originated identity is untrusted unless re-bound by LibreChat server auth context**
3. **Per-user virtual keys are optional hardening, not baseline architecture**
4. **Missing/invalid identity always fails safe to `unknown` with explicit reason**
5. **OSS profile must not rely on LiteLLM enterprise components**

### Broken-Claim Behavior

When identity claims are malformed or missing:

| Condition | Behavior | Logged Reason |
|-----------|----------|---------------|
| Empty/whitespace identity | Fallback to next source | `identity_fallback_*` |
| Contains whitespace | Fallback to next source | `identity_fallback_*` |
| Literal "unknown" | Fallback to next source | `identity_fallback_*` |
| All sources invalid | Fail-safe to `unknown` | `identity_missing_or_invalid` |

---

## Configuration Templates

### OSS-First Profile Template

```yaml
# LiteLLM configuration
general_settings:
  database_url: os.environ/DATABASE_URL
  forward_client_headers_to_llm_api: true

litellm_settings:
  redact_user_api_key_info: true
  master_key: os.environ/LITELLM_MASTER_KEY
  max_budget: 100.0
  budget_duration: 30d

# LibreChat configuration
endpoints:
  custom:
    - name: "LiteLLM"
      apiKey: "${LIBRECHAT_LITELLM_API_KEY}"
      baseURL: "http://litellm:4000/v1"
      models:
        default:
          - "claude-haiku-4-5"
        fetch: false
```

### Enterprise-Enhanced Profile Template

```yaml
# LiteLLM Enterprise configuration
general_settings:
  database_url: os.environ/DATABASE_URL
  forward_client_headers_to_llm_api: true
  
  # Enterprise strict mode (optional)
  enforced_params:
    - user
  
  # Enterprise features (requires license)
  enable_sso: true
  admin_ui: true

litellm_settings:
  redact_user_api_key_info: true
  master_key: os.environ/LITELLM_MASTER_KEY
  
  # Enhanced audit logging
  detailed_audit_logs: true
  audit_log_destination: "postgresql"

# LibreChat configuration (same attribution baseline as OSS profile)
endpoints:
  custom:
    - name: "LiteLLM"
      apiKey: "${LIBRECHAT_LITELLM_API_KEY}"
      baseURL: "http://litellm:4000/v1"
      models:
        default:
          - "claude-haiku-4-5"
          - "openai-gpt5.2"
        fetch: false
      # Keep "user" parameter for trusted attribution
```

---

## Operational Guidance

### Verifying Attribution

```bash
# Check identity resolution quality
make db-status

# Query attribution distribution
docker compose -f demo/docker-compose.yml exec -T postgres \
  psql -U litellm -d litellm -c "
SELECT 
  principal_identity_source,
  COUNT(*) as requests,
  ROUND(100.0 * COUNT(*) / SUM(COUNT(*)) OVER (), 2) as pct
FROM (
  SELECT
    CASE
      WHEN NULLIF(BTRIM(s.\"user\"), '') IS NOT NULL
        AND LOWER(BTRIM(s.\"user\")) <> 'unknown'
        AND BTRIM(s.\"user\") !~ '\\s'
        THEN 'spendlogs_user'
      WHEN NULLIF(BTRIM(v.user_id), '') IS NOT NULL
        AND LOWER(BTRIM(v.user_id)) <> 'unknown'
        AND BTRIM(v.user_id) !~ '\\s'
        THEN 'token_user_id'
      WHEN NULLIF(BTRIM(v.key_alias), '') IS NOT NULL
        AND LOWER(BTRIM(v.key_alias)) <> 'unknown'
        AND BTRIM(v.key_alias) !~ '\\s'
        THEN 'key_alias'
      ELSE 'unknown'
    END as principal_identity_source
  FROM \"LiteLLM_SpendLogs\" s
  LEFT JOIN \"LiteLLM_VerificationToken\" v ON s.api_key = v.token
  WHERE s.\"startTime\" > NOW() - INTERVAL '24 hours'
) sub
GROUP BY 1
ORDER BY requests DESC;"
```

### Monitoring Identity Quality

| Metric | Target | Alert Threshold |
|--------|--------|-----------------|
| `spendlogs_user` percentage | >95% | <90% |
| `unknown` identity rate | <1% | >5% |
| `identity_missing_or_invalid` events | 0 | >0 (per hour) |

### Detection Rules for Auth Issues

| Rule ID | Name | Purpose |
|---------|------|---------|
| DR-006 | Failed Authentication Attempts | Detect brute force / credential issues |
| DR-008 | DLP Block Event | Detect content policy violations |
| (Custom) | High Unknown Identity Rate | Detect broken claim propagation |

---

## Testing and Validation

### Contract Tests

The following contract tests validate auth profile behavior:

| Test | Validates |
|------|-----------|
| `auth_profile_contract_test.sh` | Profile definitions and license boundaries |
| `export_gateway_identity_contract_test.sh` | Identity resolution precedence |
| `validate_detections_test.sh` | Detection rule claim precedence |

### Running Tests

```bash
# Run all script tests
make script-tests

# Run specific auth contract tests
make script-tests
make script-tests

# Run CI gate (includes all validation)
make ci
```

---

## References

- [LibreChat Tooling Guide](../tooling/LIBRECHAT.md)
- [LiteLLM Configuration](../../demo/config/litellm.yaml)
- [RBAC Configuration](../../demo/config/roles.yaml)
- [Detection Rules](DETECTION.md)
- [SIEM Integration](SIEM_INTEGRATION.md)
- [Third-Party License Matrix](../policy/THIRD_PARTY_LICENSE_MATRIX.md)
- [Production Deployment Contract](../deployment/SINGLE_TENANT_PRODUCTION_CONTRACT.md)

---

## Document History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-02-11 | Initial publication with OSS-First and Enterprise-Enhanced profiles |

---

*This architecture is enforced by repository contract tests. Run `make script-tests` and `make ci-pr` for validation details.*


## Related Pilot Packaging

- [Browser and Workspace Proof Track](../BROWSER_WORKSPACE_PROOF_TRACK.md)
- [Pilot Control Ownership Matrix](../PILOT_CONTROL_OWNERSHIP_MATRIX.md)
