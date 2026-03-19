# AI Control Plane - OpenTelemetry (OTEL) Collector Setup

## Overview

The OpenTelemetry (OTEL) collector provides centralized telemetry ingestion for AI tools that are **not routed through the gateway** (e.g., Codex CLI using direct vendor OAuth) or for **correlation telemetry** when additional client-side visibility is desired.

Production contract summary:

- Raw OTEL collector ports (`4317`, `4318`, `13133`) remain bound to `127.0.0.1`
- `OTEL_PUBLISH_HOST` is not a supported production escape hatch
- Remote OTEL clients must use `https://<gateway-domain>/otel`
- Remote OTEL ingress requires `Authorization: Bearer ${OTEL_INGEST_AUTH_TOKEN}`

## Production vs Demo Configuration

The AI Control Plane provides two OTEL collector configurations:

| Profile | Config File | Purpose | Exporters |
|---------|-------------|---------|-----------|
| Demo | `config.yaml` | Local development, debugging | `debug`, `file` |
| Production | `config.production.yaml` | Production telemetry with remote export | `otlphttp/primary` |

### Configuration Selection

The `OTEL_COLLECTOR_CONFIG_FILE` environment variable selects the configuration:

```bash
# Demo configuration (local-only collector ports bound to 127.0.0.1)
OTEL_COLLECTOR_CONFIG_FILE=config.yaml make up-production

# Production configuration (collector export hardened; remote ingest must use HTTPS)
OTEL_COLLECTOR_CONFIG_FILE=config.production.yaml make up-production
```

### Production Configuration Features

The production configuration (`config.production.yaml`) includes:

- **Remote OTLP export**: Sends telemetry to a configured backend (e.g., Datadog, Honeycomb, custom)
- **Memory limiter**: Prevents OOM conditions under high load
- **Probabilistic sampling**: Controls costs via configurable sampling rate
- **Attribute sanitization**: Removes sensitive headers (Authorization) and hashes user IDs
- **Retry with backoff**: Ensures reliable delivery
- **Sending queue**: Buffers telemetry during network issues

### Required Production Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Yes | - | Remote OTLP endpoint URL (https://...) |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | No | `http/protobuf` | Production collector export protocol (must be `http/protobuf`) |
| `OTEL_EXPORTER_OTLP_AUTH_HEADER` | No | - | Auth header for remote exporter endpoint |
| `OTEL_INGEST_AUTH_TOKEN` | Yes for remote ingest | - | Shared secret enforced at the HTTPS `/otel/*` ingress |
| `OTEL_RESOURCE_ENVIRONMENT` | Yes | `demo` | Environment tag (must be non-`demo` in production) |
| `OTEL_RESOURCE_DEPLOYMENT` | Yes | `local` | Deployment identifier (e.g., `us-east-1`) |
| `OTEL_MEMORY_LIMIT_MIB` | No | `256` | Memory limit in MiB |
| `OTEL_BATCH_SIZE` | No | `512` | Batch size for telemetry |
| `OTEL_TRACES_SAMPLING_PERCENT` | No | `10` | Sampling percentage (0-100) |

**Production ingress rules**:

- Do not expose raw OTEL ports beyond localhost
- Do not rely on `OTEL_PUBLISH_HOST` for remote access
- Use the TLS Caddy ingress in `demo/config/caddy/Caddyfile.prod`
- Run `make validate-config-production` before applying production changes

### Cost and Cardinality Controls

Production deployments should configure sampling to control costs:

```bash
# High-volume production: 1% sampling
OTEL_TRACES_SAMPLING_PERCENT=1

# Medium-volume: 10% sampling (default)
OTEL_TRACES_SAMPLING_PERCENT=10

# Low-volume or critical path: 100% sampling
OTEL_TRACES_SAMPLING_PERCENT=100
```

## Why OTEL is Needed

### Route-Based Governance Model

1. **API Key Mode** (Gateway Enforcement)
   - All traffic flows through LiteLLM gateway
   - Full policy enforcement (models, budgets, rate limits)
   - Central logging in PostgreSQL audit log
   - **Status:** Fully implemented

2. **Subscription-Backed Upstream** (Gateway-Routed)
   - Tool routes through gateway with subscription upstream (ChatGPT provider, Claude OAuth forwarding)
   - Gateway enforces policies (models, budgets, rate limits)
   - Upstream billing via vendor subscription
   - Telemetry in PostgreSQL audit logs
   - **Status:** Fully implemented (requires `forward_client_headers_to_llm_api: true` for Claude)

3. **Direct Subscription / Bypass** (OTEL Telemetry)
   - Tool authenticates directly with vendor (OAuth) without gateway routing
   - Gateway cannot enforce inline (bypass scenario)
   - Telemetry via OTEL export from tool
   - **Status:** Optional for demo, required for production bypass scenarios

### When to Use OTEL

| Scenario | Gateway Routing | OTEL Needed |
|----------|-----------------|-------------|
| API-key mode | Yes | No (PostgreSQL logs sufficient) |
| Claude MAX through gateway | Yes | Optional (dual telemetry possible) |
| Codex with ChatGPT provider | Yes | No (PostgreSQL logs sufficient) |
| Codex direct OAuth (bypass) | No | Yes (for visibility) |
| OpenCode with Codex plugin | No | Yes (for visibility) |

### OTEL Data Flow

```
+--------+         +--------+         +--------+
| Codex  | OTLP    | OTEL   | Export  | SIEM / |
| (Direct+-------->Collector+--------> Logs   |
| OAuth) |         |        |         |        |
+--------+         +--------+         +--------+
                           |
                           v
                     /var/log/otel/
                    telemetry.jsonl
```

## Quick Start

### Demo Mode (Local Development)

OTEL collector is **opt-in** (Compose profile `otel`) for demo mode and is not started by `make up` or CI.

```bash
# Start OTEL collector service (enables the 'otel' Compose profile)
make up-production

# Or start directly with docker compose:
cd demo && docker compose --profile otel up -d otel-collector

# Verify it's running
make otel-health
```

### Production Mode

Production deployments can enable OTEL collector via the `production` profile when bypass/correlation telemetry export is required:

```bash
# Set required environment variables
cat >> demo/.env << 'EOF'
OTEL_EXPORTER_OTLP_ENDPOINT=https://your-otel-backend.example.com
OTEL_EXPORTER_OTLP_AUTH_HEADER="Api-Key your-api-key"
OTEL_RESOURCE_ENVIRONMENT=production
OTEL_RESOURCE_DEPLOYMENT=us-east-1
OTEL_TRACES_SAMPLING_PERCENT=10
EOF

# Validate and start with production profile
make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
make up-production

# Verify OTEL collector health
make otel-health
```

### 2. Configure Codex for OTEL Export (Direct OAuth Mode Only)

```bash
# Local-only demo path
export OTEL_EXPORTER_OTLP_ENDPOINT="http://127.0.0.1:4318"
export OTEL_EXPORTER_OTLP_PROTOCOL="http/protobuf"
export OTEL_SERVICE_NAME="codex-cli"

# Run Codex (in direct OAuth mode, not gateway-routed)
codex
```

For remote deployment (client machine + gateway host), use the TLS-protected OTLP/HTTP ingress. This is the only supported remote production path:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT="https://gateway.example.com/otel"
export OTEL_EXPORTER_OTLP_PROTOCOL="http/protobuf"
export OTEL_EXPORTER_OTLP_HEADERS="Authorization=Bearer ${OTEL_INGEST_AUTH_TOKEN}"
```

### 3. View Telemetry

```bash
# View recent telemetry
make logs

# View OTEL collector logs
make logs
```

## Configuration

### Environment Variables

| Variable | Demo / local default | Remote / production path |
|----------|----------------------|--------------------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://127.0.0.1:4318` | `https://gateway.example.com/otel` |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` | `http/protobuf` |
| `OTEL_EXPORTER_OTLP_HEADERS` | unset | `Authorization=Bearer ${OTEL_INGEST_AUTH_TOKEN}` |
| `OTEL_SERVICE_NAME` | `codex-cli` | `codex-cli` |

Raw production collector ports stay on `127.0.0.1` even when the gateway is exposed.
If you need remote OTEL ingest, route it through the authenticated TLS ingress
instead of publishing `4317`, `4318`, or `13133`.

### OTEL Collector Configuration

Located at `demo/config/otel-collector/config.yaml`:

- **Receivers:** OTLP gRPC (4317), OTLP HTTP (4318)
- **Processors:** Memory limiter, batch, resource enrichment
- **Exporters:** Console (debug), file (persistent)

## Telemetry Data

### What Codex Exports

**Metrics:**
- Request count per model
- Token usage (prompt + completion)
- Latency/timing data
- Error rates

**Traces:**
- Request spans with timing
- Provider interaction details
- Tool usage metadata

**Logs:**
- Client-side events
- Configuration changes
- Error messages

**What is NOT included:**
- Prompt content (redacted by default)
- Response content (redacted by default)
- OAuth tokens (never exported)

## Architecture

### Local Deployment

```
Client machine (127.0.0.1)
+--------+
| Codex  |
| (Direct|
| OAuth) |
+--------+
    |
    | OTEL Export
    v
+--------+
| OTEL   |
|Collector|
+--------+
    |
    v
/var/log/otel/
telemetry.jsonl
```

### Remote Deployment

For remote deployment (client machine + gateway host), do not publish raw collector ports beyond localhost. Remote clients must use the HTTPS ingress path on the gateway:

```
Client machine (CLIENT_HOST)      Gateway host (GATEWAY_HOST)
+--------+                       +----------------------+
| Codex  |                       | TLS/Auth Reverse     |
| (Direct| HTTPS /otel/* ----->  | Proxy -> OTEL        |
| OAuth) |                       | Collector (localhost)|
+--------+                       +----------------------+
                                         |
                                         v
                                   /var/log/otel/
                                  telemetry.jsonl
```

1. Update `OTEL_EXPORTER_OTLP_ENDPOINT` to the gateway HTTPS OTEL path:
   ```bash
   export OTEL_EXPORTER_OTLP_ENDPOINT="https://GATEWAY_HOST/otel"
   export OTEL_EXPORTER_OTLP_PROTOCOL="http/protobuf"
   export OTEL_EXPORTER_OTLP_HEADERS="Authorization=Bearer ${OTEL_INGEST_AUTH_TOKEN}"
   ```

2. Ensure firewall allows HTTPS to the gateway and keep raw collector ports bound to localhost only.

3. Test connectivity from the client machine:
   ```bash
   curl -I https://GATEWAY_HOST/otel/v1/traces
   ```

## Verification

### Health Check

The OTEL collector uses a **two-layer health check** approach:

1. **Docker Compose healthcheck:** Validates configuration file correctness using `otelcol-contrib validate` (distroless image constraint—no shell for HTTP probes).

2. **Runtime health probe:** `make otel-health` probes the actual HTTP health endpoint.

```bash
# Authoritative runtime probe (requires OTEL container to be running)
make otel-health
```

Expected output:
```
Checking OTEL collector...
✓ OTEL collector health: OK

Available endpoints:
  - OTLP gRPC: 127.0.0.1:4317
  - OTLP HTTP:  127.0.0.1:4318
```

**Important distinction:**
- Docker Compose shows "healthy" when the configuration is valid (does NOT verify endpoint readiness)
- `make otel-health` verifies the HTTP endpoint at `http://127.0.0.1:13133/` returns HTTP 200
- `make health` warns if OTEL isn't running, but fails if OTEL is running and unhealthy

### Checking Container Status

```bash
# View Docker-level health (config validation)
docker compose ps otel-collector

# View detailed health status
docker compose inspect --format='{{.State.Health.Status}}' otel-collector
```

### Test Telemetry Flow

1. Start OTEL collector: `make up-production`
2. Configure Codex with OTEL variables
3. Make a request in Codex (in direct OAuth mode)
4. Check telemetry: `make logs`

### Troubleshooting

**No telemetry received:**
- Verify Codex OTEL environment variables are set
- Check network connectivity to OTEL collector
- Review OTEL collector logs: `make logs`
- Ensure Codex is in direct OAuth mode (not gateway-routed)

**OTEL collector not starting:**
- Check Docker: `docker compose ps`
- Check ports: `netstat -tulpn | grep 4317` (or `ss -tulpn | grep 4317`)
- Check logs: `make logs`

## Integration with SIEM

### Normalized Telemetry Format

OTEL telemetry follows the **normalized evidence schema** defined in `demo/config/normalized_schema.yaml`. This ensures consistency across gateway-routed evidence (API-key and routed subscription) and direct/bypass telemetry (OTEL).

**Key normalized fields in OTEL telemetry:**

```json
{
  "timestamp": "2026-02-02T10:30:00Z",
  "severity": "INFO",
  "trace_id": "trace-abc123",
  "span_id": "span-def456",
  "resource": {
    "service.name": "codex-cli",
    "deployment.environment": "demo"
  },
  "attributes": {
    "ai.model": "gpt-5.2",
    "ai.provider": "openai",
    "ai.tokens.prompt": 150,
    "ai.tokens.completion": 200,
    "ai.cost.amount": 0.0025,
    "ai.policy.action": "allowed",
    "ai.policy.rule": "approved-model-list",
    "principal.id": "user-alice",
    "principal.type": "user",
    "source.type": "otel"
  }
}
```

**Field mapping to normalized schema:**

| OTEL Field | Normalized Schema | Description |
|------------|-------------------|-------------|
| `trace_id` | `correlation.trace.id` | Distributed trace identifier |
| `span_id` | `correlation.span.id` | Span within the trace |
| `timestamp` | `ai.request.timestamp` | Request timestamp |
| `attributes.ai.model` | `ai.model.id` | Model identifier |
| `attributes.ai.provider` | `ai.provider` | AI provider |
| `attributes.ai.tokens.*` | `ai.tokens.*` | Token consumption |
| `attributes.ai.cost.amount` | `ai.cost.amount` | Request cost |
| `attributes.ai.policy.action` | `policy.action` | Policy outcome |
| `attributes.principal.id` | `principal.id` | User/service identity |
| `resource.service.name` | `source.service.name` | Source service |

### File Export

Telemetry is written to `demo/logs/otel/telemetry.jsonl` in JSON Lines format.

**View recent telemetry:**
```bash
make logs
```

## Trace Workflow for Supported Deployments

OTEL traces are only relevant for the **direct/bypass** path or for optional client-side correlation telemetry. Gateway-routed traffic already has its operational source of truth in PostgreSQL audit data plus the typed status/reporting surface.

### Step 1: verify the collector is up

```bash
make otel-health
```

### Step 2: inspect recent trace-bearing records

```bash
jq -r '
  select(.trace_id != null) |
  [.timestamp, .trace_id, (.attributes["principal.id"] // "unknown"), (.attributes["ai.policy.action"] // "unknown"), (.attributes["ai.model"] // "unknown")] |
  @tsv
' demo/logs/otel/telemetry.jsonl | tail -n 20
```

### Step 3: focus one investigation on a single trace

```bash
TRACE_ID="trace-abc123"
jq --arg trace_id "$TRACE_ID" 'select(.trace_id == $trace_id)' demo/logs/otel/telemetry.jsonl
```

### Step 4: correlate to gateway-routed evidence when dual telemetry exists

Use the OTEL `trace_id` together with the gateway `request_id`/normalized trace field in your SIEM, or search the PostgreSQL audit log directly when the request also traversed LiteLLM.

For day-2 operator triage, pair OTEL trace inspection with:

- `make operator-report WIDE=1`
- `make operator-dashboard`
- `docs/observability/OPERATOR_SIGNAL_REFERENCE.md`
- `docs/security/SIEM_INTEGRATION.md`

### SIEM Integration Examples

**Splunk:**
```bash
# Forward to Splunk
splunk add oneshot /path/to/telemetry.jsonl -sourcetype json

# Search normalized fields
index=ai_gateway sourcetype=otel_logs
| eval model=ai.model
| eval user=principal.id
| stats count by model, user
```

**Elasticsearch:**
```bash
# Bulk import to Elasticsearch
curl -X POST "localhost:9200/otel-telemetry/_bulk" \
  -H "Content-Type: application/x-ndjson" \
  --data-binary @/path/to/telemetry.jsonl
```

**Grafana Loki:**
```bash
# Forward to Loki
loki-client --file=/path/to/telemetry.jsonl
```

### Correlation with Gateway Logs

The `trace_id` field enables correlation between OTEL telemetry and PostgreSQL audit logs:

**Example correlation query (Splunk):**
```spl
index=ai_gateway (sourcetype=litellm_audit OR sourcetype=otel_logs)
| eval normalized_user=coalesce(user_id, principal.id)
| eval trace=coalesce(request_id, trace_id)
| stats values(sourcetype) as sources by trace, normalized_user
| where mvcount(sources) > 1
```

**Cross-source detection example:**
- Gateway logs show a user exceeded their token budget
- OTEL telemetry shows the same user switched to direct subscription mode
- Correlation via `principal.id` reveals the full activity pattern

For complete SIEM integration guidance, see [SIEM_INTEGRATION.md](../security/SIEM_INTEGRATION.md).

## Limitations

### OTEL vs Compliance API

| Feature | OTEL | Compliance API |
|---------|------|----------------|
| Availability | All tiers | Enterprise only |
| Prompt content | Redacted | Full export |
| Response content | Redacted | Full export |
| User attribution | Service name only | Full user ID |
| Regulatory compliance | Limited | Audit-ready |

**Recommendation:** Use OTEL for operational monitoring. Use Compliance API for regulatory audit.

### Direct Subscription Governance

OTEL provides visibility but NOT enforcement. For governance:

1. **Preventive controls:** Route through gateway with subscription-backed upstream
2. **Detective controls:** OTEL telemetry + SIEM alerts
3. **Audit controls:** Compliance API (enterprise only)

## Routing vs Enforcement

**Key distinction:**
- **Routing** is about whether the tool sends requests to the gateway
- **Enforcement** is about whether the gateway can reliably block/shape traffic

**Gateway-routed subscription modes** (both can enforce):
- Claude Code MAX → through gateway → OAuth forwarded → Anthropic subscription billing
- Codex → through gateway → ChatGPT provider → OpenAI subscription billing

**Direct subscription** (cannot enforce, OTEL only):
- Codex → direct to OpenAI OAuth
- OpenCode → with Codex subscription plugin

## Makefile Reference

| Target | Purpose |
|--------|---------|
| `make up-production` | Start OTEL collector (enables Compose profile `otel`) |
| `make down` | Stop OTEL collector |
| `make logs` | View collector logs |
| `make otel-health` | Health check |
| `make logs` | View recent telemetry |
| `make up-production` | Start with production profile (OTEL enabled) |
| `make validate-config-production` | Validate production configuration |

**Note:** OTEL collector is opt-in for demo mode (profile `otel`) and not started by `make up` or CI. Use `make up-production` for demo mode. Production deployments use `make up-production` which includes OTEL with the `production` profile.

## Related Documentation

- [OPERATOR_SIGNAL_REFERENCE.md](OPERATOR_SIGNAL_REFERENCE.md) - Canonical host-first signal and dashboard surface map
- [SIEM_INTEGRATION.md](../security/SIEM_INTEGRATION.md) - Complete SIEM integration guide
- [Normalized Schema](../../demo/config/normalized_schema.yaml) - Evidence schema definition
- [Local Demo Implementation Plan](../LOCAL_DEMO_PLAN.md) - End-to-end demo flow
- [Enterprise Strategy](../ENTERPRISE_STRATEGY.md) - Strategic context
- [DETECTION.md](../security/DETECTION.md) - Detection rules for gateway-mode logs

## References

- [LiteLLM Claude Code MAX Subscription Guide](https://docs.litellm.ai/docs/tutorials/claude_code_max_subscription)
- [LiteLLM ChatGPT Provider Documentation](https://docs.litellm.ai/docs/providers/chatgpt)
- [LiteLLM OpenAI Codex Tutorial](https://docs.litellm.ai/docs/tutorials/openai_codex)

## Security Considerations

1. **Prompt redaction:** OTEL should NOT export prompt/response content
2. **Token safety:** Never export OAuth tokens
3. **Network exposure:** Raw OTEL ports must stay localhost-only; remote OTEL access must use authenticated HTTPS `/otel/*`
4. **Log retention:** Telemetry logs contain usage metadata (treat accordingly)
