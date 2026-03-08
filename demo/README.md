# AI Control Plane - Demo Environment

This directory contains the **demo/lab environment** for the AI Control Plane.

## Purpose

This is a **reference implementation** demonstrating centralized AI governance via a LiteLLM gateway. It is designed for:

- Local development and testing
- Architecture validation
- Customer demos and proofs-of-concept
- Foundation for production deployments

**This is NOT production infrastructure.**

## Architecture Overview

The demo environment consists of services that can run on either a single machine or a remote Docker host:

- **LiteLLM Proxy**: Central API gateway (port 4000)
- **PostgreSQL**: Persistent storage for keys, budgets, logs (port 5432, internal by default)

Configuration is defined in:
- `docker-compose.yml` - Service orchestration
- `config/litellm.yaml` - LiteLLM routing and authentication rules

## Operator Interface Priority

- Primary day-to-day operator entrypoint: `make` targets
- Primary typed workflow interface: `./scripts/acpctl.sh` for migrated command paths (current command: `./scripts/acpctl.sh ci should-run-runtime`)
- Secondary compatibility/inspection paths: direct shell scripts and LiteLLM WebUI

## Quick Start

### Single Machine Mode (Local Demo)

1. **Set up environment variables:**

   ```bash
   cp demo/.env.example demo/.env
   # Edit demo/.env and add your API keys (optional for basic demo)
   ```

2. **Start the services:**

   ```bash
   # Primary operator path (from project root) - LiteLLM core only
   make up-core

   # Full standard package (includes managed LibreChat stack)
   # Requires LibreChat secrets in demo/.env
   make up

   # Secondary direct compose path
   docker compose up -d
   ```

   **Note:** OTEL collector is optional and opt-in. It is primarily for direct/bypass client telemetry and optional correlation exports. If traffic is fully gateway-routed, LiteLLM/PostgreSQL evidence is typically sufficient.
   To start OTEL collector, use: `make up-production`

3. **Verify services are running:**

   ```bash
   # Primary operator path (from project root)
   make health

   # Secondary direct compose path
   docker compose ps
   ```

Services will be available at:
- LiteLLM Gateway: `http://127.0.0.1:4000`
- LiteLLM WebUI (optional): `http://127.0.0.1:4000/ui`
- PostgreSQL: internal (not published by default; use `make db-shell` / `make db-status`)

### Optional: Accessing the LiteLLM WebUI

The LiteLLM proxy includes a built-in admin dashboard for managing virtual keys, budgets, and monitoring usage.

**Local Mode (single machine):**
- URL: `http://127.0.0.1:4000/ui`

**Remote Mode (optional remote client):**
- URL: `https://gateway.example.com/ui`

**Default Credentials:**
- Username: `admin`
- Password: The value of `LITELLM_MASTER_KEY` from `demo/.env`

**Find your master key:**
```bash
cat demo/.env | grep LITELLM_MASTER_KEY
```

### Remote Gateway Host Mode (Optional)

In this mode, services run on a gateway host and client machines connect over the network.

**Network Configuration:**
- **Gateway Host** (where services run): `GATEWAY_HOST`
  - LiteLLM Gateway: `https://gateway.example.com`
  - LiteLLM WebUI (optional): `https://gateway.example.com/ui`
  - PostgreSQL: internal (not published by default; publish 5432 only if needed)
- **Client Machine** (where AI tools run): `CLIENT_HOST`
  - Claude Code, Codex CLI, and other tools connect to the remote gateway

For detailed remote setup instructions, see the [Local Demo Implementation Plan](../docs/LOCAL_DEMO_PLAN.md).

## Offline Demo Mode

The offline demo mode allows you to run the AI Control Plane without requiring external provider API keys (OpenAI, Anthropic, etc.). All LLM requests are routed to a local mock upstream service that returns deterministic responses.

### When to Use Offline Mode

**Use offline mode for:**
- Air-gapped or restricted network environments
- Cost-free demonstrations and testing
- CI/CD pipelines and automated testing
- Development and debugging without provider rate limits
- Training and workshops without API key setup

**Do NOT use offline mode for:**
- Production deployments (responses are mock templates, not actual AI)
- Evaluating AI model quality or capabilities
- Load testing with realistic response characteristics

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Offline Demo Mode                         │
│                                                              │
│  ┌──────────────┐      ┌──────────────┐      ┌──────────┐  │
│  │   Client     │─────▶│   LiteLLM    │─────▶│  Mock    │  │
│  │   (curl,     │      │   Gateway    │      │ Upstream │  │
│  │    Claude    │◀─────│   (port 4000)│◀─────│ (port    │  │
│  │    Code,     │      │              │      │  8080)   │  │
│  │    etc.)     │      └──────┬───────┘      └──────────┘  │
│  └──────────────┘             │                             │
│                               ▼                             │
│                        ┌──────────────┐                    │
│                        │  PostgreSQL  │                    │
│                        │  (port 5432) │                    │
│                        └──────────────┘                    │
└─────────────────────────────────────────────────────────────┘
```

### Quick Start

```bash
# 1. Start services in offline mode (no API keys required)
make up-offline

# 2. Verify services are running
make health-offline

# 3. Run quick smoke test
make demo-offline-test

# 4. Run comprehensive demo scenarios
make demo-offline

# 5. Stop services when done
make down-offline
```

### Available Models

In offline mode, two mock models are available:

| Model | Description | Purpose |
|-------|-------------|---------|
| `mock-gpt` | Mock OpenAI GPT-style model | Testing OpenAI-compatible routing |
| `mock-claude` | Mock Anthropic Claude-style model | Testing Anthropic-compatible routing |

### Testing with curl

```bash
# List available models
curl -H "Authorization: Bearer $LITELLM_MASTER_KEY" \
  http://127.0.0.1:4000/v1/models

# Generate a virtual key
curl -X POST http://127.0.0.1:4000/key/generate \
  -H "Authorization: Bearer $LITELLM_MASTER_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "key_alias": "test-key",
    "max_budget": 10.00,
    "models": ["mock-gpt", "mock-claude"]
  }'

# Chat completion with mock-gpt
curl -X POST http://127.0.0.1:4000/v1/chat/completions \
  -H "Authorization: Bearer $LITELLM_VIRTUAL_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mock-gpt",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### Mock Response Format

Mock responses follow the OpenAI format with estimated token counts:

```json
{
  "id": "chatcmpl-mock-abc123",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "mock-gpt",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "This is a mock response from the mock-gpt offline demo model. Your message was: \"Hello!\""
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 25,
    "total_tokens": 35
  }
}
```

### Token Estimation

The mock server estimates token counts using a character-based heuristic (~4 characters per token). This provides sufficient accuracy for:

- Budget tracking and enforcement
- Rate limiting (TPM - tokens per minute)
- Usage analytics and reporting

Note: This is not 100% accurate compared to actual provider tokenizers, but is sufficient for demo purposes.

### Environment Variables

The mock upstream service supports these environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `MOCK_LATENCY_MS` | 100 | Simulated response latency in milliseconds |
| `MOCK_TOKENS_PER_CHAR` | 0.25 | Token estimation ratio (~4 chars per token) |
| `MOCK_RESPONSE_TEMPLATE` | See code | Template for mock responses |

To customize, edit `demo/docker-compose.offline.yml`:

```yaml
services:
  mock-upstream:
    environment:
      - MOCK_LATENCY_MS=200  # Slower responses for testing timeouts
      - MOCK_TOKENS_PER_CHAR=0.3
```

### Troubleshooting Offline Mode

**Services fail to start:**
```bash
# Check container logs
make logs-offline

# Verify Docker is running
docker ps
```

**Mock upstream not responding:**
```bash
# Check mock container health
docker exec ai-control-plane-mock-upstream python -c \
  "import urllib.request; print(urllib.request.urlopen('http://localhost:8080/health').read())"
```

**LiteLLM can't connect to mock upstream:**
- Ensure both containers are on the same Docker network
- Check `demo/config/litellm-offline.yaml` has correct `api_base`
- Verify mock-upstream health before starting litellm

### Switching Between Online and Offline Modes

```bash
# Switch from online to offline
make down
make up-offline

# Switch from offline to online
make down-offline
make up-core

# Note: Database volumes are preserved between switches
# but model configurations are different
```

### Limitations

**Important:** Offline mode is for demonstration purposes only.

1. **Deterministic Responses**: Mock responses are templates, not AI-generated content
2. **No Context Awareness**: Each request is independent (no conversation memory)
3. **Estimated Tokens**: Character-based estimation, not real tokenizers
4. **Limited Model Behavior**: No temperature, top_p, or other sampling variations
5. **No Multi-modal**: Text-only, no image/audio support

Production deployments must use real providers with actual API keys.

## Environment Variables Reference

### Required Variables

| Variable | Purpose | Default Value |
|----------|---------|---------------|
| `POSTGRES_USER` | PostgreSQL database user | `litellm` |
| `POSTGRES_PASSWORD` | PostgreSQL database password | `litellm` |
| `POSTGRES_DB` | PostgreSQL database name | `litellm` |
| `LITELLM_MASTER_KEY` | Admin key for generating per-tool virtual keys | `sk-litellm-master-change-me` |
| `LITELLM_SALT_KEY` | Persistent salt for key encryption | `sk-litellm-salt-change-me` |
| `DATABASE_URL` | PostgreSQL connection string | `postgresql://litellm:litellm@postgres:5432/litellm` |

### Optional Provider Keys

For API-key mode demos, you may add keys from these providers:

| Variable | Provider | Where to Obtain |
|----------|----------|-----------------|
| `OPENAI_API_KEY` | OpenAI | <https://platform.openai.com/api-keys> |
| `ANTHROPIC_API_KEY` | Anthropic | <https://console.anthropic.com/settings/keys> |
| `GEMINI_API_KEY` | Google Gemini | <https://makersuite.google.com/app/apikey> |

**Note**: For demo/testing purposes, the gateway can operate without provider keys using [offline demo mode](#offline-demo-mode).

See `demo/config/litellm.yaml` for configured model aliases and routing rules.

## Tool Onboarding

Configure AI tools to use the gateway with one-command setup:

```bash
# Onboard Claude Code
make onboard TOOL=claude MODE=api-key

# Onboard Codex CLI  
make onboard TOOL=codex MODE=api-key

# Onboard OpenCode
make onboard TOOL=opencode MODE=gateway

# Onboard Cursor IDE
make onboard TOOL=cursor

# With connectivity verification
make onboard TOOL=claude MODE=api-key VERIFY=1

# For remote Docker host
make onboard TOOL=claude MODE=api-key HOST=192.168.1.122
```

The onboarding workflow is exposed through `make onboard` (primary) and implemented by thin scripts that generate virtual keys, display environment variable configuration, and optionally verify connectivity. See `make onboard-help` for complete usage.

## Secondary Direct Script Entry Points

- `./scripts/acpctl.sh bridge ...` is transitional; use `./scripts/acpctl.sh bridge --help` for the current supported surface
- `./scripts/acpctl.sh bridge onboard` - One-command tool onboarding (Claude, Codex, OpenCode, Cursor)
- `make key-gen` - Generate LiteLLM virtual keys for testing authentication
- `make health` - Comprehensive health check for all services
- `make db-backup` - Backup the PostgreSQL database
- `make db-restore` - Restore the PostgreSQL database from a backup
- `make db-status` - Display database status and statistics
- `make validate-detections` - Execute SIEM-style detection rules against usage logs
- `make validate-compose-healthchecks` - Validate Docker Compose healthcheck syntax

## Docker Compose Healthcheck Validation

The demo environment includes a validator that checks Docker Compose healthcheck configurations to prevent orchestration issues. Healthchecks ensure services report their status correctly to Docker, which is critical for dependent service startup and monitoring.

### Healthcheck Conventions

Healthchecks in this project follow Docker's standard format with these conventions:

- **test must be a JSON array** (not a string)
- **test[0] must be `"CMD"` or `"CMD-SHELL"`**
- **CMD requires at least 2 elements**: `["CMD", "binary", "arg1", ...]`
- **CMD-SHELL requires exactly 2 elements**: `["CMD-SHELL", "shell command"]`
- **interval, timeout, start_period**: Go-style duration (e.g., `"5s"`, `"1m30s"`)
- **retries**: positive integer
- **disable**: boolean (use `true` to disable healthcheck)

### Example Healthchecks

```yaml
# CMD form - execute a binary with arguments
healthcheck:
  test: ["CMD", "wget", "--spider", "-q", "http://localhost:4318/health"]
  interval: 30s
  timeout: 10s
  retries: 3

# CMD-SHELL form - execute a shell command
healthcheck:
  test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-litellm} -d ${POSTGRES_DB:-litellm}"]
  interval: 5s
  timeout: 5s
  retries: 5
```

### Running Validation

```bash
# Validate compose healthcheck configuration (run by make lint)
make validate-compose-healthchecks

# Typed entrypoint help
./scripts/acpctl.sh validate compose-healthchecks --help
```

Validation runs automatically during `make lint` and `make ci`.

## Detection Rules

The demo environment includes SIEM-style detection rules for security monitoring and anomaly detection. These rules analyze LiteLLM-managed usage/cost logs (e.g., `"LiteLLM_SpendLogs"`) to identify:

- Non-approved model access (policy violations)
- Token usage spikes (anomaly detection)
- High error/block rates (availability issues)
- Budget exhaustion warnings (cost management)
- Rapid request rates (potential abuse)
- Failed authentication attempts (security incidents)

### Running Detection

```bash
# Run all detection rules
make detection

# Run with options
make validate-detections
make validate-detections
```

#### Previewing Detection Rules (Dry-Run)

Before running detection rules, you can preview which rules would execute and view their SQL queries:

```bash
# Preview all enabled rules
make validate-detections

# Preview high-severity rules only
make validate-detections

# Preview a specific rule
make validate-detections

# Preview as JSON (for programmatic use)
make validate-detections
```

The `--dry-run` flag:
- Lists selected rules (based on filters)
- Shows rule metadata (name, severity, category, description)
- Displays SQL queries that would be executed
- Shows rule parameters (if any)
- Does not require Docker/PostgreSQL/jq/evidence files
- Exits with code 0

This is useful for:
- Validating detection rule configuration changes
- Reviewing SQL queries before execution
- CI/CD validation of rule syntax
- Understanding what rules are active without running queries

For complete documentation, see [docs/security/DETECTION.md](../docs/security/DETECTION.md).

## SIEM Integration Demo

Demonstrate the **normalized evidence pipeline** for unified AI governance across API key and subscription modes.

### Quick Start

```bash
# Run interactive SIEM integration demo
make validate-detections

# View normalized evidence schema
cat demo/config/normalized_schema.yaml

# View sample telemetry data
cat demo/logs/otel/telemetry.jsonl

# View SIEM query examples
make validate-siem-queries

# Run the complete evidence pipeline
make release-bundle

# View unified evidence feed
make release-bundle-verify
```

### What's Included

1. **Normalized Evidence Schema** - Unified format for all AI activity:
   - `principal.id` - Who initiated the request
   - `ai.model.id` - Model and provider identification
   - `ai.tokens.*` / `ai.cost.*` - Usage and cost metrics
   - `policy.action` - Enforcement outcomes (allowed/blocked/error)
   - `correlation.*` - Trace IDs for investigations

2. **Three Evidence Sources** - Complete visibility across usage modes:
   - **Gateway logs** (API-key mode): PostgreSQL audit logs with full enforcement
   - **OTEL telemetry** (direct/bypass mode): OpenTelemetry exports for direct vendor auth
   - **Compliance exports** (SaaS governance): Vendor audit logs (OpenAI Enterprise)

3. **Sample Telemetry Data** - 25+ normalized OTEL records showing:
   - Normal approved usage
   - Policy violations (non-approved models)
   - Authentication failures
   - Budget alerts and token spikes

4. **SIEM Query Library** - Detection rules in multiple formats:
   - Splunk SPL
   - ELK/Kibana KQL
   - Microsoft Sentinel KQL
   - Sigma (generic format)

### Demo Scenario

The demo shows how to unify governance across three tracks with three evidence sources:

| Mode | Authentication | Enforcement | Telemetry Source |
|------|---------------|-------------|------------------|
| **API Key** | Virtual keys | Inline blocking | PostgreSQL audit log |
| **Subscription-Backed** | OAuth + virtual key | Inline blocking | PostgreSQL audit log |
| **Direct Subscription** | OAuth (direct) | Detect + respond only | OTEL collector |
| **SaaS Compliance** | Enterprise workspace | Detect + respond only | Compliance API exports |

All sources normalize to the same schema for unified SIEM ingestion.

**Note:** Both Claude Code and Codex CLI can route through the gateway in subscription mode using LiteLLM's subscription provider support. OTEL is primarily for direct-to-vendor (bypass) scenarios. See [SaaS_SUBSCRIPTION_GOVERNANCE_DEMO.md](../docs/demos/SaaS_SUBSCRIPTION_GOVERNANCE_DEMO.md) for details.

**Boundary:** Vendor-hosted web interfaces (ChatGPT web, Claude web) are not forcibly gateway-routed by default. Use LibreChat when a managed browser path must stay on governed routing.

### Evidence Pipeline Commands

```bash
# Validate detection and SIEM contracts before export
make validate-detections
make validate-siem-queries
make db-status

# Build and verify the evidence/release bundle
make release-bundle
make release-bundle-verify

# Review the current readiness tracker for freshness
sed -n '1,120p' docs/release/PRESENTATION_READINESS_TRACKER.md
```

**Output locations:**
| Source | File Path |
|--------|-----------|
| Gateway | `demo/logs/gateway/gateway_events.jsonl` |
| OTEL | `demo/logs/otel/telemetry.jsonl` |
| Compliance | `demo/logs/compliance/compliance_events.jsonl` |
| **Unified** | `demo/logs/normalized/evidence.jsonl` |

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         AI Control Plane Sources                         │
├──────────────────────────┬─────────────────────┬────────────────────────┤
│  API Key Mode            │ Subscription Mode   │ SaaS Compliance        │
│  (LiteLLM Gateway)       │ (Routed or direct) │ (OpenAI Enterprise)    │
│                          │                     │                        │
│  • Full enforcement      │ • Routed: full enforcement │ • Audit exports  │
│  • PostgreSQL logs       │ • Direct: OTEL export     │ • User attribution│
└──────────┬───────────────┴──────────┬──────────┴──────────┬─────────────┘
           │                          │                     │
           └──────────────────────────┼─────────────────────┘
                                      │
                                      v
                    ┌─────────────────────────────────┐
                    │  Unified Evidence Pipeline      │
                    │  (export + merge + validate)    │
                    └─────────────────┬───────────────┘
                                      │
                                      v
                    ┌─────────────────────────────────┐
                    │  Normalized Evidence Schema     │
                    │  (principal, model, tokens,     │
                    │   cost, policy, correlation)    │
                    └─────────────────┬───────────────┘
                                      │
                    ┌─────────────────┼─────────────────┐
                    v                 v                 v
               ┌─────────┐      ┌─────────┐      ┌─────────┐
               │  Splunk │      │   ELK   │      │Sentinel │
               └─────────┘      └─────────┘      └─────────┘
```

### Complete SaaS/Subscription Demo Walkthrough

For a comprehensive step-by-step walkthrough of the SaaS/subscription governance scenario, see:
**[docs/demos/SaaS_SUBSCRIPTION_GOVERNANCE_DEMO.md](../docs/demos/SaaS_SUBSCRIPTION_GOVERNANCE_DEMO.md)**

This walkthrough covers:
- Claude Code subscription-through-gateway path
- Codex subscription-through-gateway path
- Direct subscription telemetry path (bypass detection)
- Compliance export ingestion
- Evidence pipeline execution
- SIEM integration demonstration
- Executive reporting

For comprehensive SIEM integration documentation, see [docs/security/SIEM_INTEGRATION.md](../docs/security/SIEM_INTEGRATION.md).

## Budgets and Rate Limits

The demo environment includes budget enforcement and rate limiting capabilities to control spend and protect upstream API quotas. These are core control plane features that apply at multiple levels:

### Configuration Levels

1. **Global Proxy Level** - Applies to all traffic through the gateway
   - `max_budget`: Global spend limit (demo: $100.00)
   - `budget_duration`: Reset period (demo: 30 days)
   - `max_parallel_requests`: Concurrent request limit (demo: 100)

2. **Per-Model Level** - Limits per model (GPT, Claude, etc.)
   - `rpm`: Requests per minute
   - `tpm`: Tokens per minute

3. **Per-Key Level** - Limits per virtual key
   - `max_budget`: Budget limit
   - `rpm_limit`: Requests per minute
   - `tpm_limit`: Tokens per minute

### Generating Keys with Budget Controls

Use canonical key lifecycle entrypoints:

```bash
# Standard key
make key-gen ALIAS=my-key BUDGET=10.00

# Role-shaped presets
make key-gen-dev ALIAS=my-dev-key
make key-gen-lead ALIAS=my-lead-key

# Revoke a key
make key-revoke ALIAS=<alias>

# acpctl equivalents
./scripts/acpctl.sh key gen ALIAS=my-key BUDGET=10.00
./scripts/acpctl.sh key gen-dev ALIAS=my-dev-key
./scripts/acpctl.sh key gen-lead ALIAS=my-lead-key
./scripts/acpctl.sh key revoke ALIAS=<alias>
```

> The approval-queue/request workflow from older private iterations is not part of this public snapshot command surface.

### Viewing Budget Usage

Check current budget status for all keys:

```bash
# Show budget usage
make db-status
```

Output includes:
- Virtual keys (aliases, max budgets, expiry)
- Budget usage summary
- Recent usage (“audit log”) activity

### Testing Budget Enforcement

Run the budget enforcement demo scenario to verify limits work:

```bash
make demo-scenario SCENARIO=4
```

This scenario:
1. Generates a key with $0.10 budget and 10 RPM limit
2. Makes successful requests within limits
3. Exceeds budget and verifies error response
4. Exceeds RPM limit and verifies rate limit error
5. Shows budget status and detection rule alerts

### Detection Rules

Two detection rules monitor budget-related events:

- **DR-004**: Budget Exhaustion Warning (spend >= 100% of budget)
- **DR-007**: Budget Threshold Alert (spend >= 80% of budget)

Run detection rules to check for budget alerts:

```bash
make detection
```

For comprehensive documentation, see [docs/policy/BUDGETS_AND_RATE_LIMITS.md](../docs/policy/BUDGETS_AND_RATE_LIMITS.md).

## Logging

Logs are automatically rotated by Docker's built-in log driver to prevent indefinite disk usage growth. For detailed information about log configuration, viewing logs, and cleanup, see [LOGGING.md](LOGGING.md).

**Quick log commands:**
- `make logs` - Follow all service logs in real-time
- `make clean` - Destructive cleanup (stops services and removes volumes/log artifacts)
- `docker compose logs litellm` - View LiteLLM-specific logs
- `docker compose logs --tail=100 litellm` - Last 100 lines of LiteLLM logs

### Secondary Database Scripts

The demo environment includes comprehensive database management scripts:

```bash
# Check database status (table counts, virtual keys, budget usage)
make db-status
# Secondary: make db-status

# Backup the database (creates timestamped .sql.gz file)
make db-backup
# Secondary: ./scripts/acpctl.sh db backup [backup-name]

# Restore from a backup (latest backup auto-detected)
make db-restore
# Secondary: ./scripts/acpctl.sh db restore [backup-file]

# Manual restore path when needed
gunzip < demo/backups/litellm-backup-20240128.sql.gz \
  | docker exec -i $(docker compose -f demo/docker-compose.yml ps -q postgres) \
      psql -U litellm -d litellm
```

**Important:** Always back up before running `make clean`, as it removes all volumes including database data. In interactive shells, `make clean` prompts for confirmation (`Continue? [y/N]`); for scripts/automation, use `make clean-force`. Validate backup archives before destructive restore operations (for example `gzip -t <backup.sql.gz`).

For complete database documentation, see [docs/DATABASE.md](../docs/DATABASE.md).

## Troubleshooting

1. **Services fail to start**: Check `docker-compose logs` for detailed error messages
2. **Database connection errors**: Ensure the postgres service is healthy: `docker-compose ps postgres`
3. **Authentication failures**: Verify `LITELLM_MASTER_KEY` matches between `.env` and client requests
4. **Remote host connectivity**: Ensure firewall allows port 443 to the TLS gateway from the client machine (Postgres 5432 only if you intentionally publish it)
5. **Log files growing too large**: Logs are automatically rotated by Docker. See [LOGGING.md](LOGGING.md) for details.

For comprehensive troubleshooting and database-specific issues, see:
- [Deployment Guide](../docs/DEPLOYMENT.md) - General deployment troubleshooting
- [Database Documentation](../docs/DATABASE.md) - Database-specific troubleshooting
- [Logging Guide](LOGGING.md) - Log configuration and management

## Security Notice

**IMPORTANT**: Never commit `.env` to version control. The `.gitignore` at the repository root explicitly excludes `demo/.env` to prevent accidental exposure of credentials.

If you have previously committed API keys or secrets:
1. Rotate the compromised keys immediately through your provider's portal
2. Remove the sensitive data from git history using `git filter-repo` or BFG Repo-Cleaner
3. Consider enabling GitHub secret scanning or pre-commit hooks for future protection

See `demo/.env.example` for detailed security guidelines.

## Development Notes

- For production deployments, use a proper secrets management system (e.g., HashiCorp Vault, AWS Secrets Manager) instead of environment files
- The `demo/` directory contains other tracked files (configurations, scripts) - only `.env` is gitignored
- Docker volumes persist database data; use `docker-compose down -v` to clean up completely

## Enabling HTTPS/TLS

The demo environment supports **optional** TLS/HTTPS through Caddy reverse proxy.

**Why enable TLS?**
- Protect OAuth tokens in transit (subscription mode)
- Required for production deployments
- Recommended for remote Docker host deployments

**Quick Start (Local Demo with Self-Signed Certs):**

```bash
# Start services with TLS enabled
make up-tls

# Verify HTTPS is working
curl -k https://localhost/health
```

**Configuration Modes:**
- **Self-signed certificates** (default): Suitable for local development
- **Let's Encrypt**: For production deployments with valid domain names

**Key Files:**
- `demo/docker-compose.tls.yml` - TLS-enabled Docker Compose override
- `demo/config/caddy/Caddyfile.dev` - Self-signed certificate configuration
- `demo/config/caddy/Caddyfile.prod` - Let's Encrypt configuration
- `docs/deployment/TLS_SETUP.md` - Comprehensive TLS setup guide

**Makefile Targets:**
- `make up-tls` - Start services with TLS
- `make down-tls` - Stop TLS services
- `make tls-health` - Verify TLS endpoint

**Important Notes:**
- TLS is **opt-in** - default HTTP mode (`make up`) remains unchanged
- OAuth tokens are protected in transit when TLS is enabled
- Caddy's default logging does NOT log Authorization headers (safe by default)
- Self-signed certificates will show security warnings (normal for local dev)

For complete documentation, including client configuration updates and troubleshooting, see [docs/deployment/TLS_SETUP.md](../docs/deployment/TLS_SETUP.md).

---

## Migration Note for Existing Users

If you have an existing checkout with the old `local/` directory:

```bash
# After pulling the latest changes
mv local/.env demo/.env
```

The `local/` directory has been renamed to `demo/` to better reflect its purpose as a demo/lab environment.

## Demo State Management

For consistent demos and testing, use the snapshot workflow to save and restore demo states. This ensures deterministic initial state between demo runs.

### Quick Reference

```bash
# Create a snapshot of current state
make demo-snapshot NAME=post-scenario-1

# Create snapshot including log artifacts
make demo-snapshot NAME=debug-state WITH_LOGS=1

# Restore from a snapshot
make demo-restore NAME=baseline

# Quick reset to baseline (clean logs + restore)
make demo-reset

# Check current state (services, DB, logs, snapshots)
make demo-status

# List available snapshots
make demo-snapshots
```

### Snapshot Workflow for Demos

**Before your first demo run:**

1. Start with a fresh environment:
   ```bash
   make clean  # prompts for confirmation; for scripts use: make clean-force
   make up
   ```

2. Create baseline snapshot:
   ```bash
   make demo-snapshot NAME=baseline
   ```

**During demo development:**

3. Run scenarios, then capture interesting states:
   ```bash
   make demo-all
   make demo-snapshot NAME=after-full-demo
   ```

**Before each demo presentation:**

4. Reset to known clean state:
   ```bash
   make demo-reset
   ```
   If no baseline exists, you'll be prompted to create one from the current state.

**After demos that modify state:**

5. Quickly restore and try again:
   ```bash
   make demo-restore NAME=baseline
   make demo-scenario SCENARIO=3
   ```

### Snapshot Storage

Snapshots are stored in `demo/backups/snapshots/<name>/`:
- `db.sql.gz` - PostgreSQL database dump
- `metadata.json` - Snapshot info (timestamp, description, log inclusion)
- `logs/` - Optional captured log artifacts (if created with `WITH_LOGS=1`)

**Security Note:** Snapshots may contain API keys, tokens, or audit data. They are gitignored by default and should never be committed.

## Automatic Resource Cleanup

Demo scenarios automatically clean up created resources when they exit:

### What Gets Cleaned Up

- **Virtual Keys**: All test keys created during scenario runs (e.g., `scenario-1-test-key`, `scenario-2-claude`)
- **Temp Files**: Any temporary files created by scenarios

### How It Works

Cleanup uses bash `trap` handlers that run on script exit (both normal exit and error conditions):

1. When a scenario starts, it registers resources for cleanup
2. When the scenario exits (success or failure), cleanup runs automatically
3. Virtual keys are revoked via the LiteLLM API

### Disabling Cleanup (Debugging)

The legacy `--no-cleanup` script flags are no longer part of the canonical interface.
For debugging, run the scenario, inspect state, and defer reset until after inspection:

```bash
# Run a scenario
make demo-scenario SCENARIO=1

# Inspect state
make db-status

# Reset to clean baseline when finished
make demo-reset
```

When cleanup is disabled, keys remain in the database and can be inspected:

```bash
# View remaining keys
make db-status

# Manually revoke a specific key
make key-revoke ALIAS=<alias>

# Reset to clean state
make demo-reset
```

### Manual Cleanup Commands

If you need to clean up manually:

```bash
# Revoke all scenario keys at once
for key in scenario-1-test-key scenario-2-claude scenario-3-codex \
           scenario-4-budget-test governance-demo-key scenario-6-dlp-test \
           scenario-8-compromised-key; do
    make key-revoke ALIAS="$key"
done

# Or restore database to baseline
make demo-restore NAME=baseline
```

### Troubleshooting

**"No baseline snapshot found"**
- Create one first: `make demo-snapshot NAME=baseline`
- Or let `make demo-reset` create it interactively from current state

**"Database connection failed during restore"**
- Ensure services are running: `make health`

**"Snapshot restore cancelled"**
- You must type the exact snapshot name to confirm. This prevents accidental data loss.
- Use `--force` flag in scripts for non-interactive environments (CI/CD).
