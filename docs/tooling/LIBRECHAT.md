# LibreChat - Managed Web UI for AI Control Plane

LibreChat provides a governed browser-based chat interface for the scoped managed-browser path. In this path, traffic routes through the LiteLLM gateway so non-coding users can interact with approved AI models while preserving the documented enforcement and attribution boundary.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Operator Workflow](#operator-workflow)
- [Rollback](#rollback)
- [Troubleshooting](#troubleshooting)
- [Security Considerations](#security-considerations)

## Overview

### Scope

LibreChat integration provides:
- **Governed browser chat**: Web UI for non-technical users
- **LiteLLM routing**: All requests flow through the gateway for policy enforcement
- **Model restrictions**: Only approved aliases from `litellm.yaml` are available
- **Spend attribution**: All usage appears in LiteLLM spend logs

### Non-scope

- **Native LiteLLM enterprise SSO setup**: Not part of this tooling guide; see [Authentication Architecture](../security/ENTERPRISE_AUTH_ARCHITECTURE.md) for profile-level integration guidance
- **Policy presets**: Per-user policies managed via LiteLLM, not LibreChat
- **Audited chat exports**: Future enhancement

### Authentication Architecture

LibreChat supports two authentication profiles for enterprise deployment:

| Profile | Auth Surface | Gateway Attribution | License |
|---------|-------------|---------------------|---------|
| **OSS-First** (Default) | LibreChat local/OIDC/LDAP/SAML | Shared key + trusted user context | OSS-only |
| **Enterprise-Enhanced** | Same + enterprise policy controls | Same + optional strict mode | LiteLLM Enterprise |

**Key Principles:**
1. **Default attribution**: Shared service key (`LIBRECHAT_LITELLM_API_KEY`) + trusted server-authenticated user context
2. **Browser identity untrusted**: Client-side claims require server-side re-binding by LibreChat
3. **Per-user keys optional**: Available as hardening, not baseline requirement

See [Enterprise Authentication Architecture](../security/ENTERPRISE_AUTH_ARCHITECTURE.md) for complete architecture, claim mapping, and trust boundaries.
For pilot packaging and customer-owned validation boundaries, see [Browser and Workspace Proof Track](../BROWSER_WORKSPACE_PROOF_TRACK.md).

### Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   User Browser  │────▶│    LibreChat    │────▶│    LiteLLM      │
│                 │     │   (UI/Proxy)    │     │   Gateway       │
└─────────────────┘     └─────────────────┘     └────────┬────────┘
                                                         │
                              ┌──────────────────────────┼──────────┐
                              │                          │          │
                              ▼                          ▼          ▼
                         ┌─────────┐               ┌──────────┐ ┌──────────┐
                         │ MongoDB │               │ PostgreSQL│ │ Upstream │
                         │ (Chat)  │               │ (Spend)   │ │ Providers│
                         └─────────┘               └──────────┘ └──────────┘
```

## Prerequisites

- Docker and Docker Compose (V2 preferred)
- Running AI Control Plane base services (`make up-core` for the host-first baseline; `make up` only after LibreChat secrets are configured)
- LiteLLM virtual key for LibreChat (generated via `make key-gen`)

## Quick Start

### 1. Generate Encryption Keys

LibreChat requires encryption keys for credential storage:

```bash
# Generate required keys
export LIBRECHAT_CREDS_KEY=$(openssl rand -hex 32)
export LIBRECHAT_CREDS_IV=$(openssl rand -hex 16)
export LIBRECHAT_MEILI_MASTER_KEY=$(openssl rand -base64 32)
export JWT_SECRET=$(openssl rand -hex 32)
export JWT_REFRESH_SECRET=$(openssl rand -hex 32)

# Add to demo/.env
echo "LIBRECHAT_CREDS_KEY=$LIBRECHAT_CREDS_KEY" >> demo/.env
echo "LIBRECHAT_CREDS_IV=$LIBRECHAT_CREDS_IV" >> demo/.env
echo "LIBRECHAT_MEILI_MASTER_KEY=$LIBRECHAT_MEILI_MASTER_KEY" >> demo/.env
echo "JWT_SECRET=$JWT_SECRET" >> demo/.env
echo "JWT_REFRESH_SECRET=$JWT_REFRESH_SECRET" >> demo/.env
```

### 2. Generate LiteLLM Virtual Key

```bash
# Create a virtual key for LibreChat
make key-gen ALIAS=librechat-managed BUDGET=10.00

# Copy the generated key and add to demo/.env
echo "LIBRECHAT_LITELLM_API_KEY=sk-..." >> demo/.env
```

### 3. Start LibreChat

```bash
# Start LibreChat only after required secrets are present in demo/.env
make up

# Verify health
make librechat-health
```

### 4. Access the UI

Open your browser to: http://127.0.0.1:3080

Create an account (local authentication) and start chatting with approved models.

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `LIBRECHAT_PUBLISH_HOST` | No | Interface to bind (default: 127.0.0.1) |
| `LIBRECHAT_PORT` | No | Port for LibreChat UI (default: 3080) |
| `LIBRECHAT_CREDS_KEY` | **Yes** | 64-char hex encryption key |
| `LIBRECHAT_CREDS_IV` | **Yes** | 32-char hex IV |
| `LIBRECHAT_MEILI_MASTER_KEY` | **Yes** | Min 16-char Meilisearch key |
| `LIBRECHAT_LITELLM_API_KEY` | **Yes** | Virtual key from `make key-gen` |
| `JWT_SECRET` | **Yes** | 64-char hex JWT signing secret |
| `JWT_REFRESH_SECRET` | **Yes** | 64-char hex JWT refresh signing secret |
| `LIBRECHAT_MONGO_URI` | No | MongoDB URI (default: internal) |

### Key Generation Commands

```bash
# Required keys
openssl rand -hex 32    # LIBRECHAT_CREDS_KEY (64 chars)
openssl rand -hex 16    # LIBRECHAT_CREDS_IV (32 chars)
openssl rand -base64 32 # LIBRECHAT_MEILI_MASTER_KEY (min 16 chars)
openssl rand -hex 32    # JWT_SECRET (64 chars)
openssl rand -hex 32    # JWT_REFRESH_SECRET (64 chars)
```

### Model Configuration

LibreChat is configured to use only approved models from `demo/config/litellm.yaml`:

```yaml
# demo/config/librechat/librechat.yaml
endpoints:
  custom:
    - name: "LiteLLM"
      apiKey: "${LIBRECHAT_LITELLM_API_KEY}"
      baseURL: "http://litellm:4000/v1"
      models:
        default:
          - "openai-gpt5.2"
          - "claude-sonnet-4-5"
          - "claude-haiku-4-5"
        fetch: false  # Prevents model drift
```

**Important**: The `fetch: false` setting prevents LibreChat from dynamically fetching model lists from the gateway, ensuring only explicitly approved models are available.

### Auth Profile Configuration

**Baseline (both profiles)**: Preserve trusted user attribution context

```yaml
# demo/config/librechat/librechat.yaml
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
      # Do not set dropParams to include "user"
      # The "user" field carries trusted attribution context
```

**Enterprise strict mode** (optional): When using LiteLLM Enterprise with `enforced_params: [user]`, requests without user attribution will be rejected. Ensure LibreChat is configured to always include the user field.

### Shared-Key Attribution Contract

LibreChat uses one shared service credential (`LIBRECHAT_LITELLM_API_KEY`) for gateway authentication while preserving per-user attribution:

1. `LiteLLM_SpendLogs.user` from LibreChat-authenticated user context (preferred)
2. `LiteLLM_VerificationToken.user_id` fallback
3. `LiteLLM_VerificationToken.key_alias` fallback
4. `unknown` fail-safe when all identity claims are missing/invalid

Operational boundaries:
- Keep LiteLLM behind LibreChat and trusted internal clients only.
- Treat browser-originated identity claims as untrusted input.
- Monitor fallback events via `principal.identity_source != "spendlogs_user"` in exported gateway evidence.

Verification query:

```bash
docker compose -f demo/docker-compose.yml exec -T postgres \
  psql -U litellm -d litellm -c "
SELECT
  CASE
    WHEN NULLIF(BTRIM(s.\"user\"), '') IS NOT NULL
      AND LOWER(BTRIM(s.\"user\")) <> 'unknown'
      AND BTRIM(s.\"user\") !~ '\\\\s'
      THEN BTRIM(s.\"user\")
    WHEN NULLIF(BTRIM(v.user_id), '') IS NOT NULL
      AND LOWER(BTRIM(v.user_id)) <> 'unknown'
      AND BTRIM(v.user_id) !~ '\\\\s'
      THEN BTRIM(v.user_id)
    WHEN NULLIF(BTRIM(v.key_alias), '') IS NOT NULL
      AND LOWER(BTRIM(v.key_alias)) <> 'unknown'
      AND BTRIM(v.key_alias) !~ '\\\\s'
      THEN BTRIM(v.key_alias)
    ELSE 'unknown'
  END AS resolved_principal,
  CASE
    WHEN NULLIF(BTRIM(s.\"user\"), '') IS NOT NULL
      AND LOWER(BTRIM(s.\"user\")) <> 'unknown'
      AND BTRIM(s.\"user\") !~ '\\\\s'
      THEN 'spendlogs_user'
    WHEN NULLIF(BTRIM(v.user_id), '') IS NOT NULL
      AND LOWER(BTRIM(v.user_id)) <> 'unknown'
      AND BTRIM(v.user_id) !~ '\\\\s'
      THEN 'token_user_id'
    WHEN NULLIF(BTRIM(v.key_alias), '') IS NOT NULL
      AND LOWER(BTRIM(v.key_alias)) <> 'unknown'
      AND BTRIM(v.key_alias) !~ '\\\\s'
      THEN 'key_alias'
    ELSE 'unknown'
  END AS identity_source,
  COUNT(*) AS requests
FROM "LiteLLM_SpendLogs" s
LEFT JOIN "LiteLLM_VerificationToken" v ON s.api_key = v.token
GROUP BY 1, 2
ORDER BY requests DESC;"
```

## Operator Workflow

### Installation

```bash
# 1. Start base services (includes LibreChat)
make up
make health

# 2. Configure environment variables
echo "LIBRECHAT_CREDS_KEY=$(openssl rand -hex 32)" >> demo/.env
echo "LIBRECHAT_CREDS_IV=$(openssl rand -hex 16)" >> demo/.env
echo "LIBRECHAT_MEILI_MASTER_KEY=$(openssl rand -base64 32)" >> demo/.env
echo "JWT_SECRET=$(openssl rand -hex 32)" >> demo/.env
echo "JWT_REFRESH_SECRET=$(openssl rand -hex 32)" >> demo/.env
echo "LIBRECHAT_LITELLM_API_KEY=sk-your-virtual-key-here" >> demo/.env

# 3. (Optional) explicitly restart LibreChat services only
make librechat-up
```

`make up` and `make librechat-up` both run `make validate-librechat-config` and fail fast if required values are missing. The default host-first proof path remains `make up-core`; LibreChat is an optional governed-browser layer on top of that baseline.

### Bootstrap

```bash
# Verify services started correctly
make librechat-health

# Check logs if needed
make logs
```

### Smoke Checks

```bash
# Run smoke test (validates health and spend log visibility)
make librechat-health

# Expected output:
# ✓ LibreChat health: OK
# ✓ Spend logs are being recorded
```

### Generate a Chat for Testing

1. Open http://127.0.0.1:3080 in your browser
2. Create an account or log in
3. Select a model from the dropdown (e.g., "claude-haiku-4-5")
4. Send a test message
5. Re-run `make librechat-health` to verify spend logging

### Verify Spend Attribution

```bash
# Check LiteLLM spend logs
make db-status

# Or query directly:
docker compose -f demo/docker-compose.yml exec postgres \
  psql -U litellm -d litellm -c 'SELECT * FROM "LiteLLM_SpendLogs" ORDER BY "startTime" DESC LIMIT 5;'
```

## Rollback

### Stop LibreChat (Preserve Data)

```bash
# Stop LibreChat services
make down

# Base services remain running
make health
```

### Full Removal

```bash
# Stop and remove containers (volumes preserved)
make down

# To remove data volumes (DESTRUCTIVE):
docker volume rm demo_librechat_mongodb_data demo_librechat_meili_data

# Remove upload and image volumes
docker volume rm demo_librechat_uploads demo_librechat_images
```

### Config Reversion

If you need to revert configuration changes:

```bash
# Restore librechat.yaml from git
git checkout demo/config/librechat/librechat.yaml

# Restart services
make down
make librechat-up
```

## Troubleshooting

### 401/403 Authentication Errors

**Symptoms**: Login fails or API requests rejected

**Solutions**:
1. Verify `LIBRECHAT_LITELLM_API_KEY` is set in `.env`
2. Ensure the virtual key is valid: `make db-status` and check `LiteLLM_VerificationToken`
3. Check key budget hasn't been exceeded
4. Verify key hasn't expired

```bash
# Check virtual key status
docker compose -f demo/docker-compose.yml exec postgres \
  psql -U litellm -d litellm -c 'SELECT "key_alias", "spend", "max_budget" FROM "LiteLLM_VerificationToken" WHERE "key_alias" = '"'"'librechat-managed'"'"';'
```

### Model Missing from Dropdown

**Symptoms**: Expected models not appearing in LibreChat model selector

**Solutions**:
1. Verify models in `demo/config/librechat/librechat.yaml` match approved models
2. Check that `fetch: false` is set (prevents dynamic model fetching)
3. Restart LibreChat to reload config:
   ```bash
   make down
   make librechat-up
   ```
4. Run contract test to diagnose:
   ```bash
   make script-tests
   ```

### No Spend Logs

**Symptoms**: Smoke test shows 0 spend logs after chat

**Solutions**:
1. Verify LiteLLM is receiving requests (check `make logs`)
2. Ensure you're using an approved model alias
3. Check that the virtual key is properly configured
4. Verify PostgreSQL is running and accessible

```bash
# Debug spend logging
docker compose -f demo/docker-compose.yml logs litellm | grep -i spend
```

### Container Exits Immediately

**Symptoms**: `make librechat-up` succeeds but container stops

**Solutions**:
1. Check logs: `make logs`
2. Verify required encryption keys are set:
   - `LIBRECHAT_CREDS_KEY` (64 hex chars)
   - `LIBRECHAT_CREDS_IV` (32 hex chars)
3. Ensure keys contain only valid hex characters (0-9, a-f)
4. Check Meilisearch master key is at least 16 characters

### Health Check Fails

**Symptoms**: `make librechat-health` returns error

**Solutions**:
1. Wait longer - LibreChat can take 60+ seconds to initialize
2. Check all services are running: `make ps`
3. Verify MongoDB is healthy:
   ```bash
   docker compose -f demo/docker-compose.yml ps librechat-mongodb
   ```
4. Check Meilisearch is healthy:
   ```bash
   docker compose -f demo/docker-compose.yml ps librechat-meilisearch
   ```

## Security Considerations

### Encryption Keys

- **Never commit** encryption keys to version control
- **Rotate annually** or when personnel change
- **Store in vault** for production (e.g., HashiCorp Vault, AWS Secrets Manager)
- **Backup safely** - loss of keys makes encrypted data unrecoverable

### Network Security

- By default, LibreChat binds to `127.0.0.1` (localhost only)
- For remote access, use a reverse proxy with TLS (see `docs/deployment/TLS_SETUP.md`)
- Change `LIBRECHAT_PUBLISH_HOST` to `0.0.0.0` only with TLS enabled

### API Key Management

- Use dedicated virtual keys for LibreChat (not master key)
- Set appropriate budgets per key
- Rotate keys periodically
- Monitor spend logs for anomalies

### Data Retention

- Chat history stored in MongoDB (librechat-mongodb_data volume)
- Uploaded files stored in librechat_uploads volume
- Review retention policies for compliance requirements

## Related Documentation

- [Approved Models Policy](../policy/APPROVED_MODELS.md)
- [Deployment Guide](../DEPLOYMENT.md)
- [LiteLLM Configuration](../../demo/config/litellm.yaml)
- [LibreChat Configuration](../../demo/config/librechat/librechat.yaml)
