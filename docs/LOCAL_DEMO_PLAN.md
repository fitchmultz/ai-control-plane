# Local Demo Implementation Plan (Single-Server)

This document is the **current** implementation plan for the local demo environment as it exists today: **one server** runs the AI Control Plane (Docker) and can also run the client tools locally.

## What this doc is responsible for

- The single-server workflow we are actively building and demoing.
- The minimal, repeatable commands to bring the environment up, verify health, and run demo scenarios.
- The "optional remote client" variant **without** device-specific assumptions.

## What this doc does NOT cover

- Production hardening (multi-host, HA, multi-region, formal change management).
- Customer-environment or cloud-specific validation work.
- Vendor-specific enterprise governance exports setup beyond the local demo (those are referenced, not implemented here).

## Invariants / assumptions (callers must respect)

- Docker is the runtime; do not install services directly on the host.
- Secrets live in `demo/.env` (gitignored). Never commit API keys or tokens.
- **API-key mode** and **subscription-backed upstream (routed)** can both be enforced at the gateway; **direct subscription (bypass)** is detection-first (logging/investigation), with enforcement limited by what the vendor/client allows.
- Gateway operator contract:
  - Local default: `http://127.0.0.1:4000`
  - Full override: `GATEWAY_URL` (or `ACP_GATEWAY_URL`)
  - Derived URL: `GATEWAY_HOST` + `LITELLM_PORT` + `ACP_GATEWAY_SCHEME`/`ACP_GATEWAY_TLS`
  - For remote TLS on standard 443, prefer `GATEWAY_URL=https://gateway.example.com`
  - Read secrets with `./scripts/acpctl.sh env get ...`; never source `demo/.env`

---

## 0) Demo outcomes (what you will show)

The goal is not slides; it's **evidence**:

1. **Approved path (API-key tools)**: traffic goes through the gateway; allowlists/budgets apply; logs + detections are centralized.
2. **Subscription-backed upstream (routed)**: tools route through the gateway with subscription upstream for centralized telemetry and enforcement (gateway-side controls apply).
3. **Direct subscription (bypass)**: tools authenticate directly to vendor; OTEL/compliance exports provide evidence; enforcement requires egress controls.
4. **Governance reporting**: detections run, SIEM queries are consistent, and an executive summary can be generated.

---

## 1) Bring up the local demo (single server)

From repo root on the server:

```bash
make install
make up-core
make health
```

Use `make up` only after the managed LibreChat environment variables in `demo/.env` are populated. The core gateway proof path for pilots is `make up-core`.

Useful status commands:

```bash
make ps
make logs
make db-status
```

---

## 2) Create a demo key (API-key enforcement path)

Generate a LiteLLM virtual key:

```bash
make key-gen ALIAS=demo-user BUDGET=10.00
```

Keep the generated key handy; you'll use it as `OPENAI_API_KEY` (or tool-specific equivalent) when pointing clients at the gateway.

Related references:
- Policy contracts: `policy/APPROVED_MODELS.md`, `policy/BUDGETS_AND_RATE_LIMITS.md`
- Detection rules: `security/DETECTION.md`

---

## 3) Onboard tools (API-key mode vs subscription modes)

The repo provides a unified onboarding entrypoint:

```bash
make onboard-help
```

### 3.1 Codex CLI

API-key mode (enforceable via gateway):

```bash
./scripts/acpctl.sh onboard codex
```

Subscription-backed upstream (routed through gateway, enforceable):

```bash
# First, authenticate with ChatGPT on the gateway host
make chatgpt-login

# Then onboard Codex
make onboard-codex
```

### 3.2 Claude Code

API-key mode (enforceable via gateway):

```bash
make onboard-claude
```

Subscription-backed upstream (routed through gateway, enforceable):

```bash
./scripts/acpctl.sh onboard claude
```

Direct subscription (OTEL telemetry for visibility) is currently demonstrated with Codex:

```bash
make up-production
./scripts/acpctl.sh onboard codex
```

### 3.3 OpenCode

Gateway mode (for tool-specific integration details):

```bash
make onboard-opencode
```

### 3.4 Cursor

```bash
make onboard-cursor
```

Tooling references:
- `tooling/CODEX.md`
- `tooling/CLAUDE_CODE_TESTING.md`
- `tooling/OPENCODE.md`
- `tooling/CURSOR.md`

---

## 4) Optional: remote client machine (no device assumptions)

If you run tools on a separate client machine, the server (Docker host) must be reachable from that client.

1) On the client machine, validate connectivity to the gateway host:

```bash
export GATEWAY_URL="${GATEWAY_URL:-https://${GATEWAY_HOST}}"
ping -c 2 "${GATEWAY_HOST}"
curl -sS -o /dev/null -w '%{http_code}\n' "${GATEWAY_URL}/health"
```

2) Then onboard tools from the client, pointing them at the remote host:

```bash
./scripts/acpctl.sh onboard codex
./scripts/acpctl.sh onboard claude
```

For each wizard run, either export `GATEWAY_URL` ahead of time or enter `GATEWAY_HOST` as the host and enable HTTPS/TLS when prompted.

Reference topology and network notes:
- `DEPLOYMENT.md`
- Optional TLS: `deployment/TLS_SETUP.md`

---

## 5) Run demo scenarios (evidence collection)

Run a single scenario:

```bash
make demo-scenario SCENARIO=1
```

Run the Network/Endpoint Enforcement scenario (demonstrates bypass threat model):

```bash
make demo-scenario SCENARIO=7
```

Run them all:

```bash
make demo-all
```

When cost matters for API-key testing, prefer lower-cost models:
- Use Claude **Haiku** when testing Anthropic API-key flows.
- Use `gpt-5.2` at low effort when testing OpenAI API-key flows.

---

## 6) Evidence checklist (what to screenshot / export)

- Gateway healthy: `make health`
- Keys/budgets exist: `make db-status`
- Detections run successfully:
  - `make detection`
  - `make validate-detections`
- Governance readiness evidence (exec summary):
  - `make release-bundle`
  - `make release-bundle-verify`
  - `docs/release/PRESENTATION_READINESS_TRACKER.md` reviewed and refreshed for the current pilot window

SIEM guidance:
- `security/SIEM_INTEGRATION.md`

---

## 7) SaaS/Subscription Control Plane demo (three-track model)

This demo showcases **governance across three tracks** for SaaS/subscription AI tools:

### Three-Track Governance Model

| Aspect | API-Key Mode | Subscription-Backed (Routed) | Direct Subscription (Bypass) |
|--------|--------------|------------------------------|------------------------------|
| **Enforcement** | Inline blocking, budgets, rate limits | Inline blocking, budgets, rate limits | Detective (logging + detection) |
| **Attribution** | Virtual key (key_alias) | Virtual key + user identity | User identity (OAuth/email) |
| **Telemetry** | PostgreSQL audit logs | PostgreSQL audit logs | OTEL + compliance exports |
| **Blocking** | Yes—gateway rejects violations | Yes—gateway rejects violations | No—alerts and reports only |

### Canonical Command Sequence

```bash
# 1) Start services
make up
make health

# 2) Claude Code subscription-through-gateway
#    (Dual telemetry: gateway logs + OAuth identity)
./scripts/acpctl.sh onboard claude
make demo-scenario SCENARIO=2

# 3) Codex subscription-through-gateway
#    (ChatGPT provider via LiteLLM)
make chatgpt-login  # On gateway host
make onboard-codex
make demo-scenario SCENARIO=3

# 4) Alternative: Codex direct subscription (OTEL)
#    (When not routing through gateway)
make up-production
./scripts/acpctl.sh onboard codex
# Manually run: codex "test message"
make logs

# 5) Pull compliance exports (OpenAI Enterprise)
#    Fixture mode by default; live mode with credentials
make validate-detections
make db-status

# 6) Run evidence pipeline (merge all sources)
make release-bundle
make release-bundle-verify

# 7) SIEM demonstration
make validate-detections
make validate-siem-queries

# 8) Executive reporting
make release-bundle
make release-bundle-verify
make validate-siem-schema
```

### Enforcement vs Bypass: Key Distinction

**Gateway-routed (enforcement possible):**
- Claude Code MAX → gateway → OAuth forwarded → Anthropic subscription
- Codex → gateway → ChatGPT provider → OpenAI subscription

**Direct subscription (OTEL only, no enforcement):**
- Codex → direct OAuth to OpenAI (bypass)
- OpenCode → with Codex subscription plugin (bypass)

**See also:** [Network and Endpoint Enforcement Demo](demos/NETWORK_ENDPOINT_ENFORCEMENT_DEMO.md) for the "third pillar" demonstrating why egress controls are required for definitive coverage.

### Key Artifacts

| Artifact | Location | Purpose |
|----------|----------|---------|
| Gateway logs | `demo/logs/gateway/gateway_events.jsonl` | API-key + routed subscription traffic |
| OTEL telemetry | `demo/logs/otel/telemetry.jsonl` | Direct vendor auth traffic |
| Compliance exports | `demo/logs/compliance/compliance_events.jsonl` | Vendor audit logs |
| **Unified evidence** | `demo/logs/normalized/evidence.jsonl` | SIEM-ingestable feed |

### Documentation

- **Complete walkthrough:** `docs/demos/SaaS_SUBSCRIPTION_GOVERNANCE_DEMO.md`
- **SIEM integration:** `docs/security/SIEM_INTEGRATION.md`
- **OTEL setup:** `docs/observability/OTEL_SETUP.md`

---

## References

- [LiteLLM Claude Code MAX Subscription Guide](https://docs.litellm.ai/docs/tutorials/claude_code_max_subscription)
- [LiteLLM ChatGPT Provider Documentation](https://docs.litellm.ai/docs/providers/chatgpt)
- [LiteLLM OpenAI Codex Tutorial](https://docs.litellm.ai/docs/tutorials/openai_codex)

---

## 7) Troubleshooting pointers

- First stop: `RUNBOOK.md`
- Deployment and networking: `DEPLOYMENT.md`
- OTEL collector setup (if enabled): `observability/OTEL_SETUP.md`
