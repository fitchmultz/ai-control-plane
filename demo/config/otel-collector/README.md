# OpenTelemetry Collector Configuration

## Overview

This directory contains the OpenTelemetry (OTEL) collector configuration for ingesting telemetry from AI tools that are **not routed through the gateway** (e.g., Codex CLI using direct vendor OAuth) or for **correlation telemetry** when additional client-side visibility is desired.

## When OTEL is Needed

| Scenario | Gateway Routing | OTEL Needed |
|----------|-----------------|-------------|
| API-key mode | Yes | No (PostgreSQL logs sufficient) |
| Claude MAX through gateway | Yes | Optional (dual telemetry possible) |
| Codex with ChatGPT provider | Yes | No (PostgreSQL logs sufficient) |
| Codex direct OAuth (bypass) | No | Yes (for visibility) |
| OpenCode with Codex plugin | No | Yes (for visibility) |

**Note:** Both Claude Code and Codex CLI **can** be routed through the gateway for full enforcement. OTEL is primarily for direct-to-vendor (bypass) scenarios. See [OTEL_SETUP.md](../../../docs/observability/OTEL_SETUP.md) for complete documentation.

## Files

- `config.yaml` - Main OTEL collector configuration

## Quick Start

OTEL collector is **opt-in** (Compose profile `otel`) and is not started by `make up` or CI.

```bash
# Start OTEL collector (enables the 'otel' Compose profile)
make up-production

# Or start directly with docker compose:
cd demo && docker compose --profile otel up -d otel-collector

# Verify it's running
make otel-health

# View telemetry
make logs

# View logs
make logs
```

## Configuration

### Receivers

- **OTLP gRPC** (port 4317): Standard OpenTelemetry Protocol via gRPC
- **OTLP HTTP** (port 4318): Alternative HTTP protocol

### Processors

- **batch**: Batches telemetry data for efficient export
- **resource/add_environment**: Adds environment metadata (demo, local)

### Exporters

- **debug**: Console output for immediate visibility during demo
- **file**: Persistent storage to `/var/log/otel/telemetry.jsonl`

### Health Check

The OTEL collector uses a **two-layer health check** approach due to the distroless container image:

1. **Docker Compose healthcheck:** Validates configuration file correctness using `otelcol-contrib validate`. This is the most reliable signal available in a distroless image (no shell for HTTP probes).

2. **Runtime health probe:** `make otel-health` and `make health` probe the actual HTTP health endpoint at `http://127.0.0.1:13133/` (OTEL collector health extension).

**Endpoints:**
- OTLP gRPC: `4317`
- OTLP HTTP: `4318`
- Health: `13133`

**Verification:**
```bash
# Check Docker-level health (config validation)
docker compose ps otel-collector

# Check runtime health (HTTP endpoint)
make otel-health

# Check all services including OTEL
make health
```

**Note:** The health check behavior in `make health` is:
- **WARN** if OTEL container is not running (optional service)
- **FAIL** if OTEL container is running but health endpoint returns non-200

## Client Configuration

Configure Codex (or other tools) to export telemetry when using **direct OAuth mode** (not gateway-routed):

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT="http://127.0.0.1:4317"
export OTEL_EXPORTER_OTLP_PROTOCOL="grpc"
export OTEL_SERVICE_NAME="codex-cli"
```

For remote deployment (client machine + gateway host):

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT="http://GATEWAY_HOST:4317"
```

## Gateway-Routed Alternative

For full enforcement, route tools through the gateway instead:

**Claude Code MAX:**
```bash
make onboard TOOL=claude MODE=subscription
# Uses ANTHROPIC_BASE_URL + ANTHROPIC_CUSTOM_HEADERS
```

**Codex with ChatGPT subscription:**
```bash
make chatgpt-login  # On gateway host
make onboard TOOL=codex MODE=subscription
# Uses OPENAI_BASE_URL + OPENAI_API_KEY (LiteLLM virtual key)
```

## Telemetry Data

### What is Collected

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

### What is NOT Collected

- Prompt content (redacted by default)
- Response content (redacted by default)
- OAuth tokens (never exported)

## Documentation

See `docs/observability/OTEL_SETUP.md` for complete documentation including:
- Architecture and data flow
- SIEM integration examples
- Troubleshooting guide
- OTEL vs Compliance API comparison
- Routing vs Enforcement distinction

## References

- [LiteLLM Claude Code MAX Subscription Guide](https://docs.litellm.ai/docs/tutorials/claude_code_max_subscription)
- [LiteLLM ChatGPT Provider Documentation](https://docs.litellm.ai/docs/providers/chatgpt)
- [LiteLLM OpenAI Codex Tutorial](https://docs.litellm.ai/docs/tutorials/openai_codex)

## Logs

Telemetry is written to `demo/logs/otel/telemetry.jsonl` in JSON Lines format for SIEM ingestion.
