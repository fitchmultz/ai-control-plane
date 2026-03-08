# AI Control Plane - Agent Guidelines

Guidelines for contributors and agents working on the AI Control Plane infrastructure project.

---

## Project Overview

**Infrastructure-as-Code / Demo Reference Implementation**

This is an infrastructure-first demo reference implementation with a typed operator core (`acpctl`). We build Docker configurations, documentation, and tooling for enterprise AI governance demos. Docker containers ARE the product, not a deployment artifact.

**Key Characteristics:**
- Infrastructure configuration plus typed operational tooling (`acpctl`)
- Health checks for runtime validation; unit tests for typed modules
- Documentation is a primary output alongside working configurations

### Product Intent

- Working reference implementation for enterprise AI governance controls
- Canonical strategy source for go-to-market messaging and deployment readiness
- Primary baseline: host-first production deployment with strict controls (public-sector grade by default)
- Kubernetes is an optional track and must not dilute host-first readiness

### Collaboration Charter

- User and agent operate as co-owners of delivery quality
- Pushback is expected when it improves correctness, security, or execution speed
- Prioritize evidence-backed decisions over politeness-driven ambiguity

---

## Quick Reference

| What you need | Where to find it |
|---------------|------------------|
| All make targets | `make help` |
| Shell script tests | `make script-tests` |
| CI gate | `make ci` |
| PR CI gate (fast deterministic) | `make ci-pr` |
| Performance baseline | `make performance-baseline` |
| Pilot closeout bundle | `make pilot-closeout-bundle` |
| Service health | `make health` |
| Project structure | `tree -L 2 -d` |
| Deployment docs | `docs/DEPLOYMENT.md` |
| Database reference | `docs/DATABASE.md` |
| External tooling references (OpenCode/Claude/Codex/Cursor/LiteLLM) | `docs/tooling/TOOLING_REFERENCE_LINKS.md` |

### Essential Commands

```bash
make install     # Initial setup (creates `demo/.env`, pulls images)
make install-ci  # CI setup (creates `demo/.env` only)
make ci-pr       # Fast deterministic PR gate
make ci          # Full CI gate (REQUIRED before claiming completion; runtime uses pinned offline image fallback)
make up          # Start services
make performance-baseline # Run local reference-host performance baseline
make pilot-closeout-bundle # Build the local pilot closeout artifact set
make health      # Verify services
```

---

## Repository Structure

```
├── AGENTS.md          # This file
├── Makefile           # Main orchestrator (includes mk/*.mk modules)
├── mk/                # Makefile modules (organized by domain)
├── cmd/acpctl/        # Go CLI implementation (split into focused files)
├── internal/          # Typed implementation modules
├── scripts/           # acpctl wrapper, bridge implementations, script tests
├── demo/              # Docker Compose configs, runtime fixtures, logs
├── docs/              # Strategic and technical documentation
├── local/             # Reserved local workspace (do not rely on scripts here)
└── deploy/            # Helm charts, Terraform, Ansible
```

---

## Core Principles

- **Docker-first:** ALL services run in Docker; never install dependencies on host
- **Delete before adding:** Net-negative diffs are wins when behavior stays correct
- **Evidence over opinion:** Tests, data constraints, and benchmarks settle debates
- **Fix root cause:** If the same issue appears elsewhere, fix all occurrences
- **Canonical subprocess execution:** Operator-facing subprocesses should use `internal/proc` with caller context propagation and bounded deadlines; avoid bare `exec.Command` in CLI/internal execution paths
- **Runtime health contract:** route gateway and database health through the typed `internal/gateway` and `internal/db` services, then adapt operator output in `internal/status` / `internal/doctor`; do not reintroduce collector-local HTTP probes or `docker exec psql` helpers
- **Abstract patterns:** Three occurrences = must be abstracted unless explicitly justified
- **Thin shell scripts:** Keep orchestration in shell; move complex logic to typed modules
- **Operator interface:** Use `acpctl` for typed workflows, Make for day-to-day, shell as fallback
- **acpctl command metadata:** `cmd/acpctl/command_registry.go` is the canonical source for root commands, grouped subcommands, completion ordering, and bridge compatibility entries
- **Config ownership:** `internal/config` is the only Go package that may touch process env or repo-local `.env`; other packages must consume typed config from it
- **Validation/security ownership:** canonical deployment/config scan scope lives in `internal/policy`; structural validators live in `internal/validation`; security policy enforcement lives in `internal/security`
- **Readiness gate plan:** `demo/config/readiness_evidence.yaml` is the tracked source of truth for readiness evidence gate membership; `internal/release/readiness_plan.go` materializes it
- **Artifact-run ownership:** generated readiness and closeout runs must use the shared `internal/release` artifact-run helpers for inventories, latest pointers, and run-directory verification; avoid bespoke run-dir lifecycle code
- **Onboarding ownership:** `acpctl onboard` / `internal/onboard` own onboarding product logic; `scripts/libexec/onboard_impl.sh` is a compatibility shim only
- **User config safety:** home-directory tool config writes must be ACP-managed, atomic, private (`0600` files / `0700` dirs), and conflict-aware; never overwrite unmanaged user config or emit world-readable backups
- **Helm deployment contract:** `deploy/helm/ai-control-plane/values.yaml` is production-only; demo paths must opt in via `examples/values.demo.yaml` or `examples/values.offline.yaml` with `profile=demo` and `demo.enabled=true`
- **Remote OTEL ingress:** keep raw collector ports localhost-bound; remote OTLP/HTTP clients must use the TLS Caddy `/otel/*` route with `OTEL_INGEST_AUTH_TOKEN`

---

## Non-Negotiables

- `make ci` MUST pass before claiming completion
- `ci-runtime-checks` must remain stateless in CI slot (`ACP_SLOT=ci-runtime`): always teardown CI runtime volumes to avoid stale PostgreSQL major-version data drift
- Make-driven Docker Compose flows must use slot-scoped Compose project names (`ai-control-plane-<slot>`) so CI/runtime stacks do not collide with other local environments
- Caddy TLS configs must stay compatible with pinned Caddy image behavior: use `lb_retries` (not `lb_retry_count`) and scope JSON `Content-Type` enforcement to body methods (`POST|PUT|PATCH`) so GET endpoints like `/v1/models` are not blocked.
- `make`-driven runtime flows now default `LITELLM_IMAGE`/`LIBRECHAT_IMAGE` to locally built hardened images (`ai-control-plane/*:local`); direct `docker compose` still falls back to the pinned registry images declared in compose files.
- `make ci` and `make ci-nightly` intentionally start the offline runtime with compose-pinned fallback images; local hardened image build/scan remains scoped to `make ci-manual-heavy` and local dev targets such as `make up-offline`.
- `make ci-pr` / `make ci-fast` now enforce `validate headers` and `validate env-access`; keep Go file headers compliant and never add direct `os.Getenv` / `envfile.LookupFile` calls outside `internal/config`
- Never commit secrets (API keys, tokens, OAuth tokens)
- Runtime artifacts and internal workflow state are local-only; do not track `demo/logs/`, `handoff-packet/`, `.ralph/`, `docs/presentation/slides-internal/`, or generated `docs/presentation/slides-external/*.png` exports (see `docs/ARTIFACTS.md`)
- All executable scripts: `set -euo pipefail`, terminal-aware colors, `--help` menu
- API testing: use low-cost models (`claude-haiku`, `gpt-4o-mini`)
- Update relevant documentation when behavior changes
- Documentation command examples must use canonical entrypoints (`make <target>` or `./scripts/acpctl.sh ...`); do not append CLI-style `--flags` to `make` commands unless the Make target explicitly supports that variable/flag pattern.
- Before changing onboarding/tooling scripts or docs for OpenCode, Claude Code, Codex, Cursor, Copilot, LibreChat/OpenWebUI, or LiteLLM integrations, review `docs/tooling/TOOLING_REFERENCE_LINKS.md` and align to current upstream guidance.

---

## Exit Code Contract

All scripts follow this standardized contract:

| Code | Constant | Meaning |
|------|----------|---------|
| `0` | `ACP_EXIT_SUCCESS` | Success |
| `1` | `ACP_EXIT_DOMAIN` | Domain non-success (health failed, detections found) |
| `2` | `ACP_EXIT_PREREQ` | Prerequisites not ready (Docker/curl/jq not installed) |
| `3` | `ACP_EXIT_RUNTIME` | Runtime/internal error |
| `64` | `ACP_EXIT_USAGE` | Usage error (unknown flags, missing arguments) |

Source of truth: `internal/exitcodes/exitcodes.go`. Shell scripts must honor the same numeric contract.

### Shell Library Status

Legacy shared shell-library patterns were retired. Keep shell scripts thin and move reusable logic into typed Go modules under `internal/*` and `cmd/acpctl/*`.
- Chargeback machine-readable outputs now route through `acpctl chargeback`; update canonical `demo/scripts/*` sources, then run `make generate` to refresh Helm copies and completions.

---

## Shell Script Template

```bash
#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Terminal-aware color output
if [ -t 1 ]; then
    COLOR_RED='\033[31m'
    COLOR_GREEN='\033[32m'
    COLOR_RESET='\033[0m'
else
    COLOR_RED=''
    COLOR_GREEN=''
    COLOR_RESET=''
fi

show_help() {
    cat << 'EOF'
Usage: script-name [OPTIONS]

Options:
  --verbose   Enable verbose output
  --help      Show this help message

Exit codes: 0=success, 1=domain fail, 2=prereq, 3=runtime, 64=usage
EOF
}

# Parse args, run logic...
```

## Temp File Cleanup Pattern

All temp files MUST use trap-based cleanup to prevent accumulation in `/tmp` during repeated CI runs:

```bash
# Function-local temp (preferred pattern)
temp_file=$(mktemp) || return 1
trap 'rm -f "$temp_file"' RETURN

# Script-level temp
temp_file=$(mktemp) || exit 1
trap 'rm -f "$temp_file"' EXIT

# Safe pattern when variable might be unset
trap 'rm -f "${temp_file:-}"' EXIT

# For long-running scripts, handle interrupts
cleanup() {
    rm -rf "$TEMP_DIR"
    # ... other cleanup
}
trap cleanup EXIT INT TERM
```

For complex cleanup requirements, prefer moving cleanup orchestration into typed `acpctl` workflows with explicit tests.

---

## Security

- **Never log Authorization headers or OAuth tokens**
- **Redact sensitive output** before including in documentation
- **Never commit** `demo/.env` (contains API keys and master keys)
- Use `os.environ/VAR_NAME` references in config files for credentials
- Review logs before sharing—tokens may appear in subscription mode

---

## Pre-Completion Checklist

- [ ] `make ci` passes
- [ ] Services healthy (`make health` exits 0)
- [ ] Documentation updated (if behavior changed)
- [ ] No secrets in logs or committed files
- [ ] Scripts follow standards (`set -euo pipefail`, `--help`, exit codes)
