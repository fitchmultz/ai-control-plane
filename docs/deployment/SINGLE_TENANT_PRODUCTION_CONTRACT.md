# Single-Tenant Production Deployment Contract

This document defines the **canonical configuration contract** for single-tenant production deployments of the AI Control Plane. It specifies required vs optional settings, secrets management, and deployment invariants that must be satisfied for production use.

## Overview

The AI Control Plane supports a **single-tenant deployment model** where one instance serves one customer/organization. While multi-tenant isolation is not required, production deployments must enforce:

- Strong authentication (no placeholder secrets)
- TLS encryption for external exposure
- Secure database credentials
- Observable and auditable configuration

## Deployment Topologies

The AI Control Plane supports two deployment topologies:

### 1. Embedded Database (Single Host)

All services run on a single Docker host with PostgreSQL as a container.

```
┌─────────────────────────────────────────────────────────────┐
│                    Customer Environment                      │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              Docker Host (Single Tenant)               │  │
│  │  ┌─────────────────┐      ┌──────────────────────┐   │  │
│  │  │  Caddy Proxy    │─────▶│  LiteLLM Gateway     │   │  │
│  │  │  (TLS/TLS)      │      │  Port 4000           │   │  │
│  │  │  Port 443/80    │      │                      │   │  │
│  │  └─────────────────┘      └──────────┬───────────┘   │  │
│  │                                      │                │  │
│  │                                      ▼                │  │
│  │                         ┌──────────────────────┐     │  │
│  │                         │  PostgreSQL 18       │     │  │
│  │                         │  Port 5432           │     │  │
│  │                         │  (Docker container)  │     │  │
│  │                         └──────────────────────┘     │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

**Use when**: Quick setup, development, testing, or single-host production.

### 2. Split-Host (Gateway + External Database)

The LiteLLM gateway runs on a gateway host, connecting to PostgreSQL on a separate database host.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Customer Environment                               │
│                                                                              │
│  ┌───────────────────────────────┐    ┌─────────────────────────────────┐  │
│  │      Gateway Host             │    │         DB Host                 │  │
│  │  ┌───────────────────────┐    │    │  ┌─────────────────────────┐   │  │
│  │  │  Caddy Proxy          │    │    │  │  PostgreSQL 18          │   │  │
│  │  │  Port 443/80          │    │    │  │  Port 5432              │   │  │
│  │  └───────────┬───────────┘    │    │  │  (Managed/External)     │   │  │
│  │              │                │    │  └────────────┬────────────┘   │  │
│  │              ▼                │    │               ▲                │  │
│  │  ┌───────────────────────┐    │    │     TCP/5432  │                │  │
│  │  │  LiteLLM Gateway      │────┼────┼───────────────┘                │  │
│  │  │  Port 4000            │    │    │                                  │  │
│  │  └───────────────────────┘    │    └─────────────────────────────────┘  │
│  └───────────────────────────────┘                                          │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Network Requirements for Split-Host:**

| Source | Destination | Port | Protocol | Purpose |
|--------|-------------|------|----------|---------|
| Gateway Host | DB Host | 5432 | TCP | PostgreSQL connection |
| Clients | Gateway Host | 443 | TCP | HTTPS API access |
| Clients | Gateway Host | 80 | TCP | HTTP redirect/ACME |

**Use when**: Enterprise requirements mandate database host separation, existing PostgreSQL infrastructure, or DBA-managed database instances.

## Configuration Inputs

Production secrets are sourced from the canonical host file
`/etc/ai-control-plane/secrets.env`. `demo/.env` is a runtime sync target used
by Compose and is refreshed from the canonical file via
`./scripts/acpctl.sh bridge prepare_secrets_env`.

The following sections define required and optional variables.

### Required Variables

These variables **must** be present and valid for any deployment:

| Variable | Type | Description | Validation Rules |
|----------|------|-------------|------------------|
| `LITELLM_MASTER_KEY` | Secret | Admin authentication key for gateway | >=32 chars, no whitespace, not placeholder |
| `LITELLM_SALT_KEY` | Secret | Persistent salt for key encryption | >=32 chars, no whitespace, not placeholder |
| `DATABASE_URL` | Secret-ish | PostgreSQL connection string | Non-empty, valid format, no whitespace |

### Database Mode Configuration

| Variable | Type | Description | Default | Validation Rules |
|----------|------|-------------|---------|------------------|
| `ACP_DATABASE_MODE` | Non-secret | Database deployment mode | `embedded` | Must be `embedded` or `external` |

### Embedded PostgreSQL Variables (ACP_DATABASE_MODE=embedded)

When using the embedded PostgreSQL service (default):

| Variable | Type | Description | Validation Rules |
|----------|------|-------------|------------------|
| `POSTGRES_USER` | Non-secret | Database user | Non-empty |
| `POSTGRES_PASSWORD` | Secret | Database password | Production: >=16 chars, not demo default |
| `POSTGRES_DB` | Non-secret | Database name | Non-empty |

**Consistency Requirement**: If `DATABASE_URL` points to host `postgres` (embedded mode), the credentials must match `POSTGRES_*` variables.

### External Database Variables (ACP_DATABASE_MODE=external)

When using an external PostgreSQL host (split-host mode):

| Variable | Type | Description | Validation Rules |
|----------|------|-------------|------------------|
| `DATABASE_URL` | Secret | External PostgreSQL connection string | Must specify external host (not `postgres`), production requires `sslmode` |

**External DATABASE_URL Requirements**:
- Must use `postgresql://` scheme
- Must NOT use host `postgres` (which refers to the embedded service)
- Production profile requires `sslmode=require` for non-local hosts
- Format: `postgresql://user:password@db-host.example.com:5432/dbname?sslmode=require`

### Network Configuration

| Variable | Type | Description | Default | Notes |
|----------|------|-------------|---------|-------|
| `LITELLM_PUBLISH_HOST` | Non-secret | Interface to bind LiteLLM | `127.0.0.1` | Must remain localhost-only in production; expose traffic through Caddy |
| `CADDY_PUBLISH_HOST` | Non-secret | Interface to bind Caddy (TLS) | `127.0.0.1` | Set to `0.0.0.0` for external TLS access |

**Security Rule**: Production traffic must terminate at Caddy. `LITELLM_PUBLISH_HOST`
must stay `127.0.0.1`; if `CADDY_PUBLISH_HOST` is exposed beyond localhost, the
deployment must use the production TLS contract below.

### TLS Configuration (Required for External Exposure)

When the gateway is exposed beyond localhost, TLS is mandatory:

| Variable | Type | Description | When Required |
|----------|------|-------------|---------------|
| `CADDYFILE_PATH` | Non-secret | Path to Caddyfile | Always for TLS mode |
| `CADDY_ACME_CA` | Non-secret | Certificate authority | TLS mode: `internal` (dev) or `letsencrypt` (prod) |
| `CADDY_DOMAIN` | Non-secret | Domain name for certificates | `CADDY_ACME_CA=letsencrypt` |
| `CADDY_EMAIL` | Non-secret | Email for cert notifications | `CADDY_ACME_CA=letsencrypt` |
| `LITELLM_PUBLIC_URL` | Non-secret | Public HTTPS URL | TLS mode: Must start with `https://` |
| `OTEL_INGEST_AUTH_TOKEN` | Secret | Shared secret for Caddy OTEL ingress | Required when remote OTEL clients use `/otel/*` |

**TLS Exposure Contract**:
- Exposed production ingress must use `CADDYFILE_PATH=./config/caddy/Caddyfile.prod`
- Exposed production ingress must use `CADDY_ACME_CA=letsencrypt`
- `LITELLM_PUBLIC_URL` must be `https://<CADDY_DOMAIN>`
- Remote OTEL clients must use `https://<CADDY_DOMAIN>/otel` with `Authorization: Bearer ${OTEL_INGEST_AUTH_TOKEN}`

### Production Secrets File Contract

`SECRETS_ENV_FILE` is the canonical production input and must satisfy all of the
following:

- Path points to a regular file, not a symlink
- File permissions deny group/other access (`0600` or stricter)
- Parent directory is not group/other writable
- File contains the production-only values consumed by `make validate-config-production`

Example setup:

```bash
sudo install -d -m 750 /etc/ai-control-plane
sudo install -m 600 /dev/null /etc/ai-control-plane/secrets.env
```

### OTEL Ingress Contract

Production OTEL has a strict split between local bind and remote ingest:

- Raw collector ports `4317`, `4318`, and `13133` are localhost-only
- `OTEL_PUBLISH_HOST` is not part of the production contract
- Remote clients must use the TLS Caddy `/otel/*` ingress
- `/otel/*` must require `Authorization: Bearer ${OTEL_INGEST_AUTH_TOKEN}`
- The production Caddyfile must proxy authorized OTEL traffic to `otel-collector:4318`

### Optional Provider Keys

| Variable | Type | Description |
|----------|------|-------------|
| `OPENAI_API_KEY` | Secret | OpenAI API access |
| `ANTHROPIC_API_KEY` | Secret | Anthropic API access |
| `GEMINI_API_KEY` | Secret | Google Gemini API access |

## Deployment Profiles

The validation script supports two profiles:

### `demo` Profile

**Use case**: Local development, testing, CI pipelines.

**Characteristics**:
- Allows localhost-only binding
- Permits shorter passwords (but not placeholders)
- TLS is optional
- More lenient for rapid iteration

### `production` Profile

**Use case**: Customer production environments.

**Characteristics**:
- Requires strong database passwords (>=16 characters)
- Requires host secrets file security checks
- Requires LiteLLM to stay localhost-bound behind Caddy
- Requires Let's Encrypt-backed TLS when Caddy is externally exposed
- Requires localhost-only raw OTEL collector ports with authenticated `/otel/*` ingress
- Enforces consistency between `DATABASE_URL` and `POSTGRES_*` variables
- Fails on any placeholder or weak configuration

## Authentication Profile Contract

Production deployments must declare their authentication profile at deployment time.

| Profile | Description | License | Documentation |
|---------|-------------|---------|---------------|
| `oss-first` | Shared key + trusted user context propagation | OSS-only | [Enterprise Auth Architecture](../security/ENTERPRISE_AUTH_ARCHITECTURE.md) |
| `enterprise-enhanced` | Same baseline + enterprise identity controls | LiteLLM Enterprise (if using enterprise features) | [Enterprise Auth Architecture](../security/ENTERPRISE_AUTH_ARCHITECTURE.md) |

**Production Contract Clause:**
> Auth profile must be declared at deployment time (`oss-first` or `enterprise-enhanced`).
> If `enterprise-enhanced` is selected, deployment must document active license coverage
> for any enterprise-gated LiteLLM feature used.

**Validation:**
```bash
# Verify auth profile is documented in deployment manifest
grep -E "auth_profile:\s*(oss-first|enterprise-enhanced)" deployment-manifest.yaml

# If enterprise-enhanced, verify license documentation
grep -E "litellm_enterprise_license:\s*active" deployment-manifest.yaml
```

## Invariants

The following invariants are enforced by validation:

### Security Invariants

1. **No Placeholder Secrets**: `LITELLM_MASTER_KEY` and `LITELLM_SALT_KEY` cannot be the demo placeholder values
2. **Minimum Entropy**: All secrets must be >=32 characters (except `POSTGRES_PASSWORD` which is >=16 in production)
3. **No Whitespace**: Secrets must not contain spaces, tabs, newlines, or carriage returns
4. **Secrets File Hardening**: `SECRETS_ENV_FILE` must be a non-symlink file with locked-down permissions
5. **No Secret Logging**: Validation scripts must never print secret values in error messages

### Configuration Invariants

1. **Consistency**: When using embedded PostgreSQL, `DATABASE_URL` must reference the same credentials as `POSTGRES_*` variables
2. **Ingress Ownership**: Production LiteLLM traffic must be exposed through Caddy, not by binding LiteLLM directly beyond localhost
3. **TLS for Exposure**: When `CADDY_PUBLISH_HOST` is not `127.0.0.1`, `CADDYFILE_PATH=./config/caddy/Caddyfile.prod`, `CADDY_ACME_CA=letsencrypt`, and a valid `https://` public URL are required
4. **Domain Matching**: `LITELLM_PUBLIC_URL` must match `CADDY_DOMAIN` when using Let's Encrypt
5. **OTEL Exposure**: Raw OTEL ports remain localhost-only and remote OTEL ingress must use authenticated `/otel/*`

### Runtime Invariants

1. **Health Checks**: All services must have valid health check configurations
2. **Log Rotation**: Container logs use json-file driver with 10MB max, 3 files (Docker Compose) or equivalent (Kubernetes)
3. **Image Pins**: All Docker images are pinned to specific digests for reproducibility
4. **Host Preflight**: Production deployment requires host preflight gate to pass before orchestration
5. **Least-Privilege Security**: Containers run with security hardening:
   - `no-new-privileges:true` / `allowPrivilegeEscalation: false` - Prevents privilege escalation
   - `cap_drop: ALL` / `capabilities.drop: ["ALL"]` - Drops all Linux capabilities (where safe)
   - `init: true` (Docker) / proper signal handling (Kubernetes)

## Kubernetes Mapping

The following table maps Docker Compose invariants to their Kubernetes equivalents:

| Invariant | Docker Compose | Kubernetes (Helm) |
|-----------|---------------|-------------------|
| **TLS Required** | Caddy reverse proxy | Ingress TLS (cert-manager or pre-created secret) |
| **Image Pinning** | Image digest in `docker-compose.yml` | Image digest in `values.yaml` |
| **Secrets** | `.env` file (gitignored) | Kubernetes Secrets or External Secrets Operator |
| **Health Checks** | Docker healthcheck | Liveness/Readiness probes |
| **Least-Privilege** | `security_opt`, `cap_drop` | Pod/Container `securityContext` |
| **Log Rotation** | json-file driver | Container runtime configuration |
| **Network Isolation** | Docker networks | NetworkPolicies (optional) |

### Security Context Examples

**Docker Compose:**
```yaml
cap_drop:
  - ALL
security_opt:
  - no-new-privileges:true
```

**Kubernetes (Helm values):**
```yaml
securityContext:
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
  seccompProfile:
    type: RuntimeDefault
```

## Supported Orchestrators

This contract applies to supported deployment orchestrators:

| Orchestrator | Status | Use Case | Documentation |
|--------------|--------|----------|---------------|
| Docker Compose | **Default** | Linux host deployments, single-machine production | [DEPLOYMENT.md](../DEPLOYMENT.md) |
| Kubernetes (Helm) | Optional | Production multi-node clusters for teams already operating Kubernetes | [KUBERNETES_HELM.md](./KUBERNETES_HELM.md) |

Both orchestrators enforce the same invariants (TLS, secrets, image pins, health checks).

## Non-Goals

This contract explicitly does **not** cover:

- **Multi-tenant isolation**: Each deployment serves exactly one tenant
- **Multi-cluster deployments**: Each Kubernetes cluster runs a single tenant
- **External database specifics**: While supported via `DATABASE_URL`, specific external DB configuration is operator-managed
- **Secrets management systems**: Integration with Vault, AWS Secrets Manager, External Secrets Operator, etc. is recommended but not required
- **Specific Ingress controllers**: The Helm chart supports common patterns (nginx, traefik) but doesn't mandate a specific controller

## Validation

Use the provided validation scripts to check configuration:

### Host Preflight Gate

Before running deployment orchestration, the host must pass operational readiness checks:

```bash
# Run host preflight
make host-preflight

# Or run directly against an explicit secrets file
./scripts/acpctl.sh host preflight --secrets-env-file /etc/ai-control-plane/secrets.env
```

**Required Checks**:
- Docker binary and daemon are available
- Docker Compose is available
- systemd is available
- The tracked systemd unit template exists
- The canonical secrets file passes the production validation contract
- The Compose runtime env parent directory exists

**Exit Codes**:
- `0` - All checks passed, host is ready
- `1` - Domain failure (one or more checks failed)
- `2` - Prerequisites not ready (missing tools)
- `64` - Usage error (invalid arguments)

**Integration**: Run `make host-preflight` on the gateway host before `make host-install`. Declarative `make host-check` / `make host-apply` validate the tracked Ansible inventory and playbook surface separately.

### Docker Compose Validation

```bash
# Validate with production profile (recommended for customer deployments)
make validate-config-production

# Validate with TLS requirements
make validate-config-production

# Validate specific env file
make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
```

### Helm Chart Validation

```bash
# Validate Helm chart (lint + template)
make helm-validate

# Or run directly
./scripts/acpctl.sh deploy helm-validate
```

### License Boundary Validation

Verify that no restricted commercial components (e.g., LiteLLM Enterprise) are included in the deployment:

```bash
# Run license boundary check
make license-check

# Regenerate license summary report
make license-report-update
```

This check ensures compliance with third-party license policies and generates the license summary artifact required for customer handoff.

### Supply-Chain Validation

Validate the committed supply-chain policy contract, digest pinning, and allowlist expiry controls:

```bash
# Run supply-chain gate (policy contract + digest pinning + allowlist expiry)
make supply-chain-gate

# Check allowlist expiry windows (default warn<45d fail<14d)
make supply-chain-allowlist-expiry-check

# Generate deterministic local policy summary artifact
make supply-chain-report

# Review findings
jq '.policy_id, .allowlist_count, .status' demo/logs/supply-chain/summary.json
```

**Required Evidence Artifacts** (for customer handoff):
- `demo/logs/supply-chain/summary.json` - Aggregated policy-validation summary
- `demo/logs/evidence/readiness-<timestamp>/make-ci-nightly.log` - Runtime + release readiness gate evidence
- `demo/logs/release-bundles/ai-control-plane-deploy-<version>.tar.gz` (+ `.sha256`) - Release artifact integrity evidence

> SBOM, CVE scan, and provenance artifacts are not generated by the default public-snapshot make targets. Integrate dedicated tooling (for example Syft/Trivy/Cosign) in downstream production pipelines if required.

> **Acceptance Requirement:** Production handoff MUST include successful `make ci-nightly` evidence and release-bundle verification output before cutover. Perform additional customer-specific load testing outside this public-snapshot command surface when capacity profiling is required.

**Policy Configuration**:
- Policy file: `demo/config/supply_chain_vulnerability_policy.json`
- Default: Fail policy on CRITICAL/HIGH severity thresholds in the configured contract
- Allowlist support: Time-bounded CVE exceptions with justification
- Baseline status is evaluated from current policy + digest pins at execution time; temporary allowlist exceptions must include explicit expiry and ticket ownership

**Remediation Workflow**:
1. Triage findings: Review `summary.json` for policy status and allowlist footprint
2. Patch: Update image digests to approved patched versions
3. Allowlist: Add temporary exceptions with expiry date and ticket reference
4. Expiry guard: Execute `make supply-chain-allowlist-expiry-check` and resolve near-term exceptions
5. Re-run: Execute `make supply-chain-gate` to verify policy compliance

**CI Integration**: Automatically runs during `make ci` and `make ci-nightly` gates (supply-chain gate + allowlist expiry check).

### CI Integration

The validation is automatically run as part of `make lint` and `make ci`:

```bash
# Fast static checks (includes config validation and Helm lint)
make lint

# Full CI gate (includes runtime validation)
make ci
```

## Quick Start Templates

### Local Development - Docker Compose (Demo Profile)

```bash
# 1. Create environment
cp demo/.env.example demo/.env

# 2. Generate secure keys (optional but recommended)
# The validation script will warn about placeholders

# 3. Validate
make validate-config

# 4. Start services
make up
```

### Production Deployment - Docker Compose (Embedded DB)

```bash
# 1. Create canonical production secrets file
sudo install -d -m 750 /etc/ai-control-plane
sudo cp demo/.env.example /etc/ai-control-plane/secrets.env
sudo chmod 600 /etc/ai-control-plane/secrets.env

# 2. Edit canonical secrets with production settings:
#    - Strong LITELLM_MASTER_KEY and LITELLM_SALT_KEY
#    - Strong POSTGRES_PASSWORD (>=16 chars)
#    - CADDY_DOMAIN, CADDY_EMAIL for TLS
#    - Keep LITELLM_PUBLISH_HOST=127.0.0.1
#    - Set CADDY_PUBLISH_HOST=0.0.0.0 for external TLS exposure
#    - Set CADDYFILE_PATH=./config/caddy/Caddyfile.prod
#    - Set OTEL_INGEST_AUTH_TOKEN if remote OTEL clients will use /otel/*

# 3. Run production gate (required before customer handoff)
make ci-nightly SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env

# Config-only validation (optional focused check)
make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env

# 4. Sync canonical secrets to compose env path
make host-secrets-refresh \
  SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env \
  HOST_COMPOSE_ENV_FILE=demo/.env

# 5. Start with TLS
make up-tls
```

### Split-Host Production Deployment - Docker Compose (External DB)

```bash
# 1. Create canonical production secrets file
sudo install -d -m 750 /etc/ai-control-plane
sudo cp demo/.env.example /etc/ai-control-plane/secrets.env
sudo chmod 600 /etc/ai-control-plane/secrets.env

# 2. Edit canonical secrets with split-host settings:
#    - ACP_DATABASE_MODE=external
#    - DATABASE_URL=postgresql://litellm:<password>@db.example.com:5432/litellm?sslmode=require
#    - Strong LITELLM_MASTER_KEY and LITELLM_SALT_KEY
#    - CADDY_DOMAIN, CADDY_EMAIL for TLS
#    - Keep LITELLM_PUBLISH_HOST=127.0.0.1
#    - Set CADDY_PUBLISH_HOST=0.0.0.0 for external TLS exposure
#    - Set CADDYFILE_PATH=./config/caddy/Caddyfile.prod
#    - Set OTEL_INGEST_AUTH_TOKEN if remote OTEL clients will use /otel/*
#    - POSTGRES_* variables are NOT required (external DB managed separately)

# 3. Validate external database connectivity from gateway host
nc -zv <db-host> 5432
make health

# 4. Run production gate (required before customer handoff)
make ci-nightly SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env

# Config-only validation (optional focused check)
make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env

# 5. Sync canonical secrets to compose env path
make host-secrets-refresh \
  SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env \
  HOST_COMPOSE_ENV_FILE=demo/.env

# 6. Start gateway services (postgres container will NOT be started)
ACP_DATABASE_MODE=external make up

# 7. Verify health
make health
```

### Kubernetes Deployment - Helm (Production)

```bash
# 1. Create namespace
kubectl create namespace acp

# 2. Create secrets (use External Secrets Operator for production)
kubectl create secret generic ai-control-plane-secrets \
  --from-literal=LITELLM_MASTER_KEY='your-secure-master-key' \
  --from-literal=LITELLM_SALT_KEY='your-secure-salt-key' \
  --from-literal=DATABASE_URL='postgresql://...' \
  -n acp

# 3. Install with production values
helm upgrade --install acp ./deploy/helm/ai-control-plane -n acp \
  -f ./deploy/helm/ai-control-plane/examples/values.production.yaml

# 4. Verify deployment
kubectl get pods -n acp

# 5. Optional Kubernetes production profile checks (explicitly enabled)
CI_PRODUCTION_K8S=1 make ci-nightly \
  SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env \
  NAMESPACE=acp RELEASE=acp
```

See [KUBERNETES_HELM.md](./KUBERNETES_HELM.md) for complete Kubernetes deployment guidance.

## Exit Codes

The validation script follows the repository's standardized exit code contract:

| Code | Meaning | Usage |
|------|---------|-------|
| 0 | Success | Configuration is valid |
| 1 | Domain failure | Validation failed (invalid config) |
| 2 | Prerequisites not ready | Missing env file, tools not installed |
| 3 | Runtime/internal error | Unexpected error during validation |
| 64 | Usage error | Invalid CLI arguments |

## Runtime Validation

Configuration validation (static checks) is performed by `make validate-config`. Runtime validation (proving the contract on a running deployment) is performed by the production smoke test script.

### Smoke Test Script

The `./scripts/acpctl.sh bridge prod_smoke_test` validates these production invariants against a running deployment:

1. **Reachability**: Gateway health endpoint is accessible
2. **Auth Enforcement**: No anonymous access to `/v1/models`
3. **Models Configured**: At least one model is available
4. **Virtual Key Generation**: Admin API works with master key
5. **Key Validation**: Generated keys work on public endpoints
6. **Request Path**: Full request cycle works (when mock models configured)

### Running Smoke Tests

```bash
# Against local TLS deployment
make prod-smoke-local-tls

# Against specific endpoint
export LITELLM_MASTER_KEY=sk-...
make prod-smoke PUBLIC_URL=https://gateway.example.com

# Against Helm deployment (via port-forward)
make helm-smoke NAMESPACE=acp RELEASE=acp
```

### CI Integration

Production smoke tests are included in the extended CI gate:

```bash
make ci-nightly  # Includes prod-smoke-local-tls
```

## References

- [Main Deployment Guide](../DEPLOYMENT.md) - Complete deployment instructions
- [Kubernetes/Helm Guide](./KUBERNETES_HELM.md) - Kubernetes deployment instructions
- [TLS Setup Guide](./TLS_SETUP.md) - HTTPS/TLS configuration
- [Database Guide](../DATABASE.md) - PostgreSQL operations
- [Production Handoff Runbook](./PRODUCTION_HANDOFF_RUNBOOK.md) - Production operations and handoff
- [Demo Environment README](../../demo/README.md) - Quick start for local development
- [Helm Chart](../../deploy/helm/ai-control-plane) - Helm chart source
