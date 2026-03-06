# AI Control Plane

Enterprise AI governance with centralized logging, approved-only model access, and bypass detection.

## Overview

The AI Control Plane is a reference implementation demonstrating centralized governance for AI model usage in enterprise environments. It addresses the critical security challenges organizations face when adopting AI tools:

- **Shadow AI usage**: Employees using unapproved AI tools and providers
- **Lack of audit trails**: No visibility into who is using what models, when, and with what data
- **Uncontrolled spend**: No way to attribute costs or enforce budgets by team

This solution provides a unified approach covering both API-key-based usage and subscription-based SaaS tools (ChatGPT Business, Claude Enterprise, Copilot, etc.).

Validated baseline: local host-first reference implementation on a Linux Docker environment. Cloud-specific production claims, egress-enforcement claims, and generalized managed-service positioning remain gated on additional environment-specific validation.

![AI Control Plane architecture overview](docs/images/2026-03-05-19-30-architecture-hero.png)

## Validated Scope

This repository is strongest as a validated reference implementation for:

- governed AI routing through a central gateway for the scoped host-first path
- approved-model enforcement, budget controls, and attribution workflows
- typed operator onboarding, health checks, and release/readiness evidence generation
- SIEM-ready detection validation and telemetry normalization patterns
- pilot execution discipline, customer ownership boundaries, and closeout decisions
- safe adoption guidance for both API-key tooling and scoped browser/workspace governance

What it does **not** prove by itself:

- enterprise-wide bypass prevention without customer-operated network and endpoint controls
- generalized production readiness across every environment shape
- compliance certification, auditor attestation, or customer-specific cloud hardening

## Architecture

<!--
Project Color Scheme Applied:
- Gateway (LiteLLM): Highlighted with Brand Orange (#f26522) emphasis
- Users/Services: Gray (#444444) for client components
- Providers: Gray (#666666) for external services
- Observability: Secondary Blue (#0089cf) for data/logging
-->

```
┌─────────────────────────────────────────────────────────────────┐
│                        Enterprise Users/Services                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐         │
│  │Claude    │  │Codex     │  │OpenCode  │  │Custom    │         │
│  │Code      │  │CLI       │  │          │  │Services  │         │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘         │
│       │             │             │             │                 │
└───────┼─────────────┼─────────────┼─────────────┼─────────────────┘
        │             │             │             │
        ▼             │             │             │
╔═════════════════════════════════════════════════════════════════╗
║  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ ║
║  ░░░░░░  AI CONTROL PLANE GATEWAY  (LITELLM)  ░░░░░░░░░░░░░░░ ║
║  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ ║
║  ┌──────────────────────────────────────────────────────────┐  ║
║  │  • Authentication (per-user/per-service virtual keys)    │  ║
║  │  • Model allowlisting & routing                          │  ║
║  │  • Budget enforcement & rate limits                      │  ║
║  │  • Centralized logging & telemetry                       │  ║
║  │  • Policy enforcement (DLP, guardrails)                  │  ║
║  └──────────────────────────────────────────────────────────┘  ║
╚═══════════════════════╬═════════════════════════════════════════╝
                        ║
        ┌───────────────╫───────────────┐
        ▼               ║               ▼
┌──────────────┐        ║        ┌──────────────┐
│   OpenAI     │        ║        │  Anthropic   │
│   API        │        ║        │  API         │
└──────────────┘        ║        └──────────────┘
        │               ║               │
        └───────────────╫───────────────┘
                        ║
                        ▼
              ┌─────────────────┐
              │  Observability  │
              │  ─────────────  │
              │  • Logs         │
              │  • Metrics      │
              │  • Alerts       │
              │  • SIEM         │
              └─────────────────┘
```

## Quick Start

### Quick Validation Path (10 minutes)

If you are evaluating this repository quickly (for architecture, deployment, and operator quality), run:

> Prerequisites: Docker running, Go installed, and `demo/.env` configured (see Installation).

```bash
make install-ci
make ci-pr
make up-offline
make health
make doctor
make release-bundle
```

Expected outcomes:
- Deterministic checks pass before runtime validation starts
- Services start and pass health checks
- `./scripts/acpctl.sh doctor` reports actionable environment diagnostics
- Release bundle generation succeeds with checksums

> Generated runtime/evidence artifacts are intentionally local-only and gitignored. See [docs/ARTIFACTS.md](docs/ARTIFACTS.md).

### Full Validation Path

For the full validation checklist, see [`docs/release/VALIDATION_CHECKLIST.md`](docs/release/VALIDATION_CHECKLIST.md).

```bash
# Fast deterministic PR-equivalent checks
make ci-pr

# Full local CI gate (includes runtime-aware test stage)
make ci

# If Docker runtime is unavailable locally, use the deterministic subset:
make ci-fast
go test ./...

# Runtime proof (no provider keys required)
make up-offline
make health
make doctor

# Policy/security checks
make public-hygiene-check
make license-check
make supply-chain-gate

# Release artifact workflow
make release-bundle
make release-bundle-verify
```

### Prerequisites

- Docker and Docker Compose (V2 preferred, V1 supported)
- curl (for health checks)
- Go 1.25+ (for typed CLI core, currently `acpctl`)

**For contributors running `make lint` or `make ci`:**
- shellcheck (for shell script linting)
- yamllint (for YAML validation)
- jq (for license/supply-chain policy validation)
- python3 (for supply-chain allowlist expiry checks)
- rg / ripgrep (for restricted-reference scanning in `make license-check`)

**Optional for manual-heavy security tier (`make ci-manual-heavy`):**
- Trivy (for hardened image vulnerability scanning)

Install on Ubuntu/Debian: `sudo apt-get install shellcheck yamllint jq python3 ripgrep golang-go`
Install on macOS: `brew install shellcheck yamllint jq python ripgrep go`

### Installation

1. **Clone and set up environment:**

   ```bash
   make install
   ```

   This creates `demo/.env` from the example template, pulls required Docker images, and builds the local `acpctl` binary used by `./scripts/acpctl.sh`.

   > **Note**: Ensure your environment is clean before starting. If you have other services using port 4000, they will conflict with the gateway.
   >
   > **Database mode contract**: external PostgreSQL deployments must set `ACP_DATABASE_MODE=external` explicitly. `DATABASE_URL` alone does not switch the system into external mode because the embedded demo stack also defines `DATABASE_URL`.

2. **Configure secure keys (REQUIRED for scripts):**

   Generate secure random keys for local use. While the system starts with defaults, repository scripts **reject placeholder keys** for safety.

   ```bash
   # Generate master key
   openssl rand -base64 48 | tr -d '\n='

   # Generate salt key
   openssl rand -base64 48 | tr -d '\n='
   ```

   Edit `demo/.env` and replace the placeholder values for `LITELLM_MASTER_KEY` and `LITELLM_SALT_KEY`.

3. **Start services:**

   ```bash
   # Personal-use path: LiteLLM core only (gateway + DB + guardrails)
   make up-core

   # Full standard package (includes managed LibreChat stack)
   make up
   ```

   Services started:
   - PostgreSQL: internal (not published by default; use `make db-shell` / `make db-status`)
   - LiteLLM Gateway: `http://127.0.0.1:4000`

4. **Verify health:**

   ```bash
   make health
   ```

   Expected output:
   ```
   ✓ LiteLLM health endpoint: OK
   ✓ LiteLLM models endpoint: OK
   ✓ All services are healthy
   ```

### Tool Onboarding (One-Command Setup)

Configure AI tools to use the gateway with a single command:

```bash
# Onboard Claude Code
make onboard TOOL=claude MODE=api-key

# Onboard Codex CLI
make onboard TOOL=codex MODE=api-key

# Onboard OpenCode
make onboard TOOL=opencode MODE=gateway

# Onboard Cursor IDE
make onboard TOOL=cursor

# With verification (checks connectivity)
make onboard TOOL=claude MODE=api-key VERIFY=1

# For remote Docker host
make onboard TOOL=claude MODE=api-key HOST=192.168.1.122
```

See `make onboard-help` for complete usage information.

### Generating a Virtual Key (Manual)

Generate a per-user or per-service virtual key for authentication:

```bash
make key-gen ALIAS=your-user-or-service-name BUDGET=10.00
```

This creates a unique key that can be used to authenticate requests to the gateway while enforcing budgets and logging usage.

### Making a Test Request

```bash
# List available models
curl -H "Authorization: Bearer $LITELLM_MASTER_KEY" \
  http://127.0.0.1:4000/v1/models

# Make a completion request
curl -X POST http://127.0.0.1:4000/v1/chat/completions \
  -H "Authorization: Bearer $LITELLM_VIRTUAL_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "your-model-name",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### Managed Web UI (LibreChat)

For non-technical users who prefer a browser-based chat interface, **LibreChat** is the managed web UI in the standard package and provides governed access through the same LiteLLM gateway:

```bash
# 1. Generate required keys and add to demo/.env
echo "LIBRECHAT_CREDS_KEY=$(openssl rand -hex 32)" >> demo/.env
echo "LIBRECHAT_CREDS_IV=$(openssl rand -hex 16)" >> demo/.env
echo "LIBRECHAT_MEILI_MASTER_KEY=$(openssl rand -base64 32)" >> demo/.env
echo "JWT_SECRET=$(openssl rand -hex 32)" >> demo/.env
echo "JWT_REFRESH_SECRET=$(openssl rand -hex 32)" >> demo/.env
make key-gen ALIAS=librechat-managed BUDGET=10.00
# Add the generated key from the command output:
echo "LIBRECHAT_LITELLM_API_KEY=sk-..." >> demo/.env

# 2. Start full stack (includes LibreChat by default)
make up

# 3. Access the UI at http://127.0.0.1:3080
```

In the managed LibreChat deployment path, chat traffic routes through the LiteLLM gateway with the scoped governance and attribution controls documented in `docs/tooling/LIBRECHAT.md`.

See [LibreChat Tooling Guide](docs/tooling/LIBRECHAT.md) for complete documentation.

### Declarative Host Deployment (Production-Oriented)

For remote Linux hosts, use the declarative host-first orchestrator:

```bash
# Dry-run
make host-check INVENTORY=deploy/ansible/inventory/hosts.yml

# Converge
make host-apply INVENTORY=deploy/ansible/inventory/hosts.yml
```

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md#43-declarative-host-first-deployment-recommended-for-production) for inventory contract, day-0 bootstrap, and day-2 drift correction.

## Offline Demo Mode

Run the AI Control Plane without provider API keys using mock LLM responses. This mode is useful for:

- Air-gapped or restricted network environments
- Cost-free demonstrations
- CI/CD testing
- Development and debugging

### Quick Start (Offline Mode)

```bash
# Start services in offline mode (no API keys required)
make up-offline

# Verify offline mode is working
make demo-offline-test

# Run full offline demo scenarios
make demo-offline

# Stop offline services
make down-offline
```

**Available Models in Offline Mode:**
- `mock-gpt` - Mock OpenAI GPT-style responses
- `mock-claude` - Mock Anthropic Claude-style responses

**Offline Mode Limitations:**
- Responses are deterministic templates (not actual AI-generated content)
- Token counts are estimated (not computed with real tokenizers)
- Intended for demo/testing only - production must use real providers

For more details, see [demo/README.md](demo/README.md#offline-demo-mode).

## Documentation

| Document | Description |
|----------|-------------|
| [Docs Index](docs/README.md) | Start here: documentation landing page |
| [Deployment Guide](docs/DEPLOYMENT.md) | Comprehensive deployment instructions for Docker hosts |
| [Technical Architecture](docs/technical-architecture.md) | Components, control/data flow, and key engineering trade-offs |
| [Enterprise AI Control Plane Strategy](docs/ENTERPRISE_STRATEGY.md) | Complete strategy and architecture overview |
| [Go-To-Market Scope And Readiness](docs/GO_TO_MARKET_SCOPE.md) | Canonical audience, scope, and "ready-to-sell/deploy" criteria |
| [Service Offerings](docs/SERVICE_OFFERINGS.md) | Productized service catalog with deliverables, prerequisites, and SOW templates |
| [Local Demo Implementation Plan](docs/LOCAL_DEMO_PLAN.md) | Single-server local demo implementation plan |
| [Demo Environment](demo/README.md) | Detailed demo environment documentation |
| [Generated Artifacts Policy](docs/ARTIFACTS.md) | What evidence/log outputs are generated locally and intentionally not committed |
| [Presentation Readiness Tracker](docs/release/PRESENTATION_READINESS_TRACKER.md) | Canonical gate status and evidence mapping for release/presentation confidence |

## Project Standards

- [LICENSE](LICENSE)
- [SECURITY.md](SECURITY.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)

## Deployment Track (Default + Optional)

Use this routing by default:

- **Default (recommended): Linux host, Docker-first**
  - Start with [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) for comprehensive host deployment instructions
- **Optional operator layer: Portainer**
  - Use [`docs/deployment/PORTAINER.md`](docs/deployment/PORTAINER.md) if your team manages Docker hosts with Portainer
- **Optional secondary tracks (scenario-driven):**
  - Kubernetes/Helm: [`docs/deployment/KUBERNETES_HELM.md`](docs/deployment/KUBERNETES_HELM.md) for teams already operating Kubernetes
  - Terraform cloud provisioning: [`docs/deployment/TERRAFORM.md`](docs/deployment/TERRAFORM.md) for cloud infrastructure provisioning

## Deployment Modes

This demo supports two deployment configurations:

### Remote Gateway Host Mode (Optional)

In this mode, the gateway runs on a remote Docker host (e.g., a server or lab machine) and client tools connect to it over the network.

- **Gateway Host** (where services run): `GATEWAY_HOST`
  - LiteLLM Gateway: `http://GATEWAY_HOST:4000`
- **Client Machine** (where AI tools run): `CLIENT_HOST`
  - Claude Code, Codex CLI, and other tools connect to the remote gateway

See [Local Demo Implementation Plan](docs/LOCAL_DEMO_PLAN.md) for detailed setup instructions.

### Single Machine (Local Demo Mode)

In this mode, all services run on the same machine.

- Gateway runs on localhost at `http://127.0.0.1:4000`
- PostgreSQL: internal (not published by default; use `make db-shell` / `make db-status`)
- Use `localhost` or `127.0.0.1` in all configuration examples

## Environment Setup

The `demo/.env` file contains all required configuration. Key variables:

### Required Variables

| Variable | Purpose | Default Value |
|----------|---------|---------------|
| `ACP_DATABASE_MODE` | Database deployment mode | `embedded` |
| `LITELLM_MASTER_KEY` | Admin key for generating virtual keys | `sk-litellm-master-change-me` |
| `LITELLM_SALT_KEY` | Persistent salt for key encryption | `sk-litellm-salt-change-me` |
| `DATABASE_URL` | PostgreSQL connection string | `postgresql://litellm:litellm@postgres:5432/litellm` |

`ACP_DATABASE_MODE` is the canonical switch:
- `embedded` — use the Docker-managed PostgreSQL instance in the local/reference stack
- `external` — use an externally managed PostgreSQL instance referenced by `DATABASE_URL`

If you are integrating an external database, set both `ACP_DATABASE_MODE=external` and `DATABASE_URL=...`.

### Optional Provider Keys

Add provider API keys to `demo/.env` for full functionality:

| Variable | Provider | Where to Obtain |
|----------|----------|-----------------|
| `OPENAI_API_KEY` | OpenAI | <https://platform.openai.com/api-keys> |
| `ANTHROPIC_API_KEY` | Anthropic | <https://console.anthropic.com/settings/keys> |
| `GEMINI_API_KEY` | Google Gemini | <https://makersuite.google.com/app/apikey> |

**Note**: The gateway can operate without provider keys using [offline demo mode](#offline-demo-mode), which routes requests to a local mock upstream service for testing.

## Verification and Health Checks

Run these commands to verify your setup:

```bash
# Check service health
make health

# View running containers
make ps

# View service logs
make logs

# Run full CI checks
make ci
```

## Makefile Targets Reference

| Target | Description |
|--------|-------------|
| `make help` | Show all available targets |
| `make install` | Set up dependencies and environment |
| `make up-core` | Start LiteLLM core services only (gateway + DB + guardrails) |
| `make up` | Start full standard package services |
| `make down` | Stop Docker services (preserves volumes, includes embedded DB profile) |
| `make health` | Health check gateway |
| `make logs` | View Docker logs (follow mode) |
| `make ps` | Show running containers |
| `make key-gen` | Generate a LiteLLM virtual key |
| `make onboard` | Onboard an AI tool (one-command setup) |
| `make onboard-help` | Show onboarding help |
| `make clean` | Remove artifacts + logs. DESTRUCTIVE: deletes Docker volumes (db + TLS certs). Prompts; use `FORCE=1` for non-interactive. |
| `make install-binary` | Build local `acpctl` binary (`.bin/acpctl`) used by typed CLI workflows |
- `make ci` | Run full local CI gate (ci-pr + offline runtime startup, runtime checks, and clean teardown) |
| `make build` | Build/recreate Docker containers (requires Docker+Compose; exits 2 if missing) |
| `make release-bundle` | Build versioned deployment bundle with checksums and install manifest |
| `make release-bundle-verify` | Verify release bundle integrity (use `RELEASE_BUNDLE_PATH=...`) |

### Typed CLI Core

- `acpctl` is the typed Go command core used to migrate nontrivial shell logic.
- Local binary path (managed by the wrapper): `.bin/acpctl` (built by `make install`).
- Canonical typed command entrypoint: `./scripts/acpctl.sh ci should-run-runtime` (equivalent direct binary: `acpctl ci should-run-runtime`).
- Expanded operator flows delegate to stable `make` targets where appropriate (`deploy`, `db`, `host`, `demo`, `terraform`), while `validate` and `key` include typed/native implementations for core checks and key lifecycle.
- Migration notes and contracts: `docs/tooling/ACPCTL.md`.

### Offline Mode Targets

| Target | Description |
|--------|-------------|
| `make up-offline` | Start services in offline mode (no API keys required) |
| `make down-offline` | Stop offline mode services |
| `make health-offline` | Health check offline services |
| `make logs-offline` | View offline mode logs |
| `make demo-offline` | Run comprehensive offline demo scenarios |
| `make demo-offline-test` | Quick test of offline mode |

## Security Notice

### OAuth Tokens in Logs

**IMPORTANT**: When using subscription mode (e.g., Claude Code with OAuth login), the gateway forwards OAuth tokens to upstream providers. Ensure:

- LiteLLM is configured not to log Authorization headers
- Any reverse proxy does not log headers
- Stored traffic has headers stripped before persistence

### Never Commit Secrets

The `.gitignore` explicitly excludes `demo/.env` to prevent accidental credential exposure. If you have previously committed API keys:

1. Rotate the compromised keys immediately through your provider's portal
2. Remove the sensitive data from git history using `git filter-repo` or BFG Repo-Cleaner
3. Consider enabling GitHub secret scanning or pre-commit hooks

## Next Steps

1. **Read the strategy document**: [Enterprise AI Control Plane Strategy](docs/ENTERPRISE_STRATEGY.md) for complete context on the architecture and problem space.

2. **Explore the demo environment**: Review [demo/README.md](demo/README.md) for detailed setup instructions and configuration options.

3. **Review implementation plans**: Examine the deployment guides for the [local reference environment](docs/LOCAL_DEMO_PLAN.md) and the broader deployment tracks in [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).

4. **Configure your tools**: Use `make onboard TOOL=<tool>` for one-command setup, or see the implementation plans for manual configuration instructions.

## Status

**Public reference implementation and deployment demo**

- ✅ Local host-first reference environment validated
- ✅ Typed operator workflows (`acpctl`) and runtime-aware local CI validated
- ⚠️ Customer pilot readiness requires rerunning evidence in the target environment and validating enterprise-specific integrations (identity, SIEM, secrets, network controls)
- ⏳ AWS lab / cloud-specific enforcement validation remains future work

This repository is intended to demonstrate deployable governance patterns, reproducible validation workflows, and operational controls. It is **not** a turnkey managed SaaS product.

External-facing readiness artifacts in `docs/release/` are point-in-time certification snapshots. Refresh them before customer reuse.

See [docs/GO_TO_MARKET_SCOPE.md](docs/GO_TO_MARKET_SCOPE.md) for readiness scope and [docs/KNOWN_LIMITATIONS.md](docs/KNOWN_LIMITATIONS.md) for transparent limitations tracking.

## License

This project is licensed under the [MIT License](LICENSE).
