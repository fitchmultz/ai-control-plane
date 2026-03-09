# AI Control Plane - Deployment Guide

This guide provides comprehensive instructions for deploying the AI Control Plane with a canonical Linux host, Docker-first deployment track and optional secondary tracks for teams that need them.

Operator interface order used throughout this guide:
1. Use `make` targets as the default operator entrypoint.
2. Use `./scripts/acpctl.sh` for migrated typed workflows (currently `./scripts/acpctl.sh ci should-run-runtime`).
3. Use direct shell scripts/systemctl/UI as secondary compatibility or break-glass paths.

## Table of Contents

1. [Overview](#1-overview)
2. [Deployment Tracks](#2-deployment-tracks-choose-this-order)
   - [2.1 Linux Host, Docker-First (Default)](#21-linux-host-docker-first-default)
   - [2.2 Optional: Portainer Operator Layer](#22-optional-portainer-operator-layer)
   - [2.3 Managed LibreChat Frontend (Browser UI)](#23-managed-librechat-frontend-browser-ui)
   - [2.4 Optional: Kubernetes (Helm)](#24-optional-kubernetes-helm)
   - [2.5 Optional: Terraform (Cloud Provisioning)](#25-optional-terraform-cloud-provisioning)
3. [Prerequisites](#3-prerequisites)
4. [Deployment Steps](#4-deployment-steps)
   - [4.1 Single Machine Mode](#41-single-machine-mode-local)
   - [4.2 Remote Gateway Host Mode](#42-remote-gateway-host-mode)
   - [4.3 Declarative Host-First Deployment](#43-declarative-host-first-deployment-recommended-for-production)
   - [4.4 Systemd Host Service Management](#44-systemd-host-service-management-production-recommended)
5. [Environment Configuration](#5-environment-configuration)
6. [Network Configuration](#6-network-configuration)
7. [Database](#7-database)
8. [Verification](#8-verification)
9. [Troubleshooting](#9-troubleshooting)
10. [DLP and Guardrails](#10-dlp-and-guardrails)
11. [Security Considerations](#11-security-considerations)
12. [Next Steps](#12-next-steps)

---

## 1. Overview

The AI Control Plane demo environment consists of two core services:

- **LiteLLM Proxy**: Central API gateway for unified AI model access (port 4000)
- **PostgreSQL**: Persistent storage for virtual keys, budgets, and usage logs (port 5432)

Configuration files:
- `demo/docker-compose.yml` - Service orchestration
- `demo/config/litellm.yaml` - LiteLLM routing and authentication rules
- `demo/.env` - Environment variables (created from `.env.example`)

> **Production Note**: For production deployments, see the
> [Single-Tenant Production Contract](deployment/SINGLE_TENANT_PRODUCTION_CONTRACT.md)
> which defines the canonical configuration contract, validation rules, and
> deployment invariants. Production secrets source of truth is
> `/etc/ai-control-plane/secrets.env`; refresh `demo/.env` from it with
> `make host-secrets-refresh` and run
> `make ci-nightly SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env`
> before customer handoff (`make validate-config-production` remains available
> for focused config-only checks).

---

## 2. Deployment Tracks (Choose This Order)

### 2.1 Linux Host, Docker-First (Default)

All services run on a Linux host using Docker Compose. This is the **default and recommended** deployment mode for most production and pilot deployments.

**Single Machine Mode:**

| Service | Endpoint |
|---------|----------|
| LiteLLM Gateway | `http://127.0.0.1:4000` (localhost only) |
| LiteLLM WebUI (optional) | `http://127.0.0.1:4000/ui` (localhost only) |
| PostgreSQL | Internal (Docker network; not published by default) |

**LiteLLM WebUI Access (Optional):**
- Username: `admin`
- Password: Value of `LITELLM_MASTER_KEY` from `demo/.env`

Find your master key:
```bash
./scripts/acpctl.sh env get LITELLM_MASTER_KEY
```

**Remote Gateway Host Mode:**

Services run on a gateway host (server/lab machine), and client machines connect over the network. Use this mode when you want the gateway to be hosted centrally while running tools elsewhere.

| Component | Host | Endpoint |
|-----------|------|----------|
| Gateway Host (services) | `GATEWAY_HOST` | LiteLLM: `https://gateway.example.com`<br>LiteLLM WebUI (optional): `https://gateway.example.com/ui` |
| Client Machine (AI tools) | `CLIENT_HOST` | Connects to `GATEWAY_HOST` |

### 2.2 Optional: Portainer Operator Layer

Portainer provides optional GUI-based management for Docker environments. Use this if your team already uses Portainer to operate Docker hosts.

See [Portainer Operations Guide](deployment/PORTAINER.md) for instructions on managing an already-deployed Linux host Docker stack with Portainer.

> Portainer is **optional convenience tooling**, not a required runtime dependency. The canonical deployment path remains Docker-first host deployment.

### 2.3 Managed LibreChat Frontend (Browser UI)

LibreChat provides a governed web-based chat interface for non-technical users. All traffic routes through the LiteLLM gateway for policy enforcement.

| Service | Endpoint | Notes |
|---------|----------|-------|
| LibreChat UI | `http://127.0.0.1:3080` | Browser-based chat interface |
| LiteLLM Gateway | `http://127.0.0.1:4000` (localhost only) | Backend API (same as base deployment) |

**Quick Start:**
```bash
# 1. Generate encryption keys and virtual key
echo "LIBRECHAT_CREDS_KEY=$(openssl rand -hex 32)" >> demo/.env
echo "LIBRECHAT_CREDS_IV=$(openssl rand -hex 16)" >> demo/.env
echo "LIBRECHAT_MEILI_MASTER_KEY=$(openssl rand -base64 32)" >> demo/.env
echo "JWT_SECRET=$(openssl rand -hex 32)" >> demo/.env
echo "JWT_REFRESH_SECRET=$(openssl rand -hex 32)" >> demo/.env
make key-gen ALIAS=librechat-managed BUDGET=10.00
# Add the generated key to demo/.env as LIBRECHAT_LITELLM_API_KEY

# 2. Start core stack (includes LibreChat services)
make up

# 3. Verify
make librechat-health
```

**Access:** Open http://127.0.0.1:3080 in your browser, create an account, and start chatting.

See [LibreChat Tooling Guide](tooling/LIBRECHAT.md) for complete documentation including troubleshooting, rollback procedures, and security considerations.

### 2.4 Optional: Kubernetes (Helm)

Kubernetes is a **supported secondary track** for teams that already operate Kubernetes platforms and need cluster-native scaling and platform integrations. It is not required for standard Linux-host deployments.

| Component | Endpoint | Notes |
|-----------|----------|-------|
| LiteLLM Gateway | `https://ai-control-plane.example.com` | Configurable via Ingress |
| LiteLLM WebUI | `https://ai-control-plane.example.com/ui` | Same endpoint as gateway |
| PostgreSQL | Internal (ClusterIP) | External DB recommended for production |

**Quick Start:**
```bash
# Install with Helm
helm upgrade --install acp ./deploy/helm/ai-control-plane -n acp --create-namespace

# See KUBERNETES_HELM.md for complete documentation
```

See [Kubernetes (Helm) Deployment Guide](deployment/KUBERNETES_HELM.md) for complete instructions including TLS setup, external database configuration, and production hardening.

### 2.5 Optional: Terraform (Cloud Provisioning)

Terraform provides optional cloud infrastructure provisioning and Kubernetes platform bootstrap for AWS, Azure, and GCP. Use this track only if you need cloud infrastructure automation.

See [Terraform Deployment Guide](deployment/TERRAFORM.md) for cloud-specific deployment instructions.

---

## 3. Prerequisites

### Software Requirements

| Component | Minimum Version | Notes |
|-----------|-----------------|-------|
| Docker | Latest stable | <https://docs.docker.com/get-docker/> |
| Docker Compose | V2 (preferred), V1 supported | Use `docker compose` (V2) or `docker-compose` (V1) |
| curl | Any | For health checks |

### Hardware Requirements

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| CPU | 2 cores | 4+ cores |
| RAM | 4 GB | 8+ GB |
| Disk | 10 GB free | 20+ GB free |

### Network Requirements

**Single Machine Mode:**
- Port 4000 must be available on localhost
- PostgreSQL (5432) is not published by default; use `make db-shell` / `make db-status` on the host

**Remote Gateway Host Mode (Optional):**
- Network connectivity between client and gateway host
- Port 4000 accessible from client machine
- PostgreSQL (5432) is optional and requires intentional exposure (or SSH port forwarding)
- Firewall rules configured to allow traffic (see [Network Configuration](#6-network-configuration))

---

## 4. Deployment Steps

### 4.1 Single Machine Mode (Local)

#### Step 1: Clone the Repository

```bash
git clone <repository-url>
cd ai-control-plane
```

#### Step 2: Set Up Environment

Use the provided Makefile target:

```bash
make install
```

This creates `demo/.env` from the example template and pulls required Docker images.

Secondary/manual fallback (direct Compose usage):

```bash
cp demo/.env.example demo/.env
# Edit demo/.env with your configuration
```

#### Step 3: Generate Secure Keys (Recommended)

For production or secure environments, generate cryptographically random keys:

```bash
# Generate master key
openssl rand -base64 48 | tr -d '\n='

# Generate salt key
openssl rand -base64 48 | tr -d '\n='
```

Edit `demo/.env` and replace the placeholder values:
- `LITELLM_MASTER_KEY`
- `LITELLM_SALT_KEY`

#### Step 4: Start Services

```bash
# Personal-use path: LiteLLM core only (gateway + DB + guardrails)
make up-core

# Full standard package (includes managed LibreChat stack)
make up
```

Secondary/manual fallback (direct Compose usage):

```bash
cd demo
docker compose up -d
```

#### Step 5: Verify Deployment

```bash
make health
```

Expected output:
```
=== AI Control Plane Health Check ===
1. Docker Container Status
✓ PostgreSQL container is running
✓ LiteLLM container is running
2. LiteLLM Gateway Health Endpoint
✓ LiteLLM health endpoint is accessible (authorized HTTP 200)
3. LiteLLM Models Endpoint
✓ LiteLLM models endpoint is accessible
4. PostgreSQL Connectivity
✓ PostgreSQL is accepting connections
=== Health Check Summary ===
Health check: PASSED
```

---

### 4.2 Remote Gateway Host Mode

#### Step 1: Prepare the Docker Host

SSH into the gateway host:

```bash
ssh user@GATEWAY_HOST
```

#### Step 2: Clone and Set Up Repository

```bash
# Clone repository
git clone <repository-url>
cd ai-control-plane

# Set up environment
make install
```

#### Step 3: Configure Production Secrets

Use the canonical production secrets file and keep `demo/.env` as the synced
runtime file:

```bash
# Generate keys
openssl rand -base64 48 | tr -d '\n='  # For LITELLM_MASTER_KEY
openssl rand -base64 48 | tr -d '\n='  # For LITELLM_SALT_KEY

# Create canonical secrets file with secure permissions
sudo install -d -m 750 /etc/ai-control-plane
sudo cp demo/.env.example /etc/ai-control-plane/secrets.env
sudo chmod 600 /etc/ai-control-plane/secrets.env

# Refresh compose runtime file (optional fetch hook)
make host-secrets-refresh \
  SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env \
  HOST_COMPOSE_ENV_FILE=demo/.env

# Validate production contract
make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
```

#### Step 4: Configure Firewall (if needed)

Ensure the gateway port (4000) is accessible from client machines.

> **Note:** PostgreSQL (5432) is not published by default. Only open 5432 if you
> intentionally publish the database port or need direct SIEM DB-connect access.

```bash
# Example: UFW on Ubuntu/Debian
sudo ufw allow 4000/tcp

# Example: firewalld on RHEL/CentOS
sudo firewall-cmd --permanent --add-port=4000/tcp
sudo firewall-cmd --reload
```

#### Step 5: Start Services

```bash
make up
```

#### Step 6: Verify from Docker Host

```bash
make health
```

#### Step 7: Verify Network Connectivity from Client Machine

From the client machine, run these connectivity checks:

```bash
# Optional ICMP reachability check
ping -c 2 GATEWAY_HOST

# Validate gateway health endpoint (200 or 401 expected)
curl -sS -o /dev/null -w '%{http_code}\n' "https://GATEWAY_HOST/health"
```

> Use a short-lived **virtual key** for client-side authenticated checks.
> Do not copy `LITELLM_MASTER_KEY` to client machines.

If the repository is available on the client machine, you can also run:

```bash
GATEWAY_HOST=GATEWAY_HOST make health
```

---

### 4.3 Declarative Host-First Deployment (Recommended for Production)

The declarative deployment orchestrator provides idempotent, repeatable host convergence using Ansible. This approach reduces operational variance and enables infrastructure-as-code practices.

**When to use:**
- Managing multiple gateway hosts
- Requiring idempotent, repeatable deployments
- Integrating with CI/CD pipelines
- Reducing manual command-runbook execution

**Prerequisites:**
- Ansible 2.16+ installed on control machine (or Docker for containerized fallback)
- SSH access to gateway hosts
- Repository cloned on control machine

#### Step 1: Create Inventory

Copy the example inventory and customize:

```bash
cp deploy/ansible/inventory/hosts.example.yml deploy/ansible/inventory/hosts.yml
```

Edit `deploy/ansible/inventory/hosts.yml` with your gateway host details:

```yaml
all:
  children:
    gateway:
      hosts:
        acp-gateway-1:
          ansible_host: 192.168.1.122
          ansible_user: ubuntu
          acp_repo_path: /opt/ai-control-plane
          acp_env_file: /opt/ai-control-plane/demo/.env
          acp_tls_mode: tls
          acp_public_url: https://gateway.example.com
```

**Inventory contract:**
- `ansible_host`: IP or hostname of gateway
- `ansible_user`: SSH username
- `acp_repo_path`: Path where repository is cloned on gateway
- `acp_env_file`: Path to environment file on gateway
- `acp_tls_mode`: `tls`
- `acp_public_url`: Public URL for smoke checks

#### Step 2: Dry-Run (Check Mode)

Validate what would change without making modifications:

```bash
# Using Makefile
make host-check INVENTORY=deploy/ansible/inventory/hosts.yml

# Typed entrypoint equivalent
./scripts/acpctl.sh host check --inventory deploy/ansible/inventory/hosts.yml
```

#### Step 3: Apply (Converge)

Deploy or update the gateway host:

```bash
# Using Makefile
make host-apply INVENTORY=deploy/ansible/inventory/hosts.yml

# Typed entrypoint equivalent
./scripts/acpctl.sh host apply --inventory deploy/ansible/inventory/hosts.yml
```

The apply operation is idempotent—running it multiple times will converge to the same state without unnecessary changes.

#### Step 4: Verify Deployment

The apply command automatically runs health and smoke checks. You can also verify manually:

```bash
# SSH to gateway and run health check
ssh ubuntu@192.168.1.122 'cd /opt/ai-control-plane && make health'
```

#### Advanced: Override Variables via CLI

Override inventory variables without editing the file:

```bash
./scripts/acpctl.sh host apply \
  --inventory deploy/ansible/inventory/hosts.yml \
  --limit acp-gateway-1 \
  --repo-path /var/lib/acp \
  --env-file /etc/acp/gateway.env \
  --tls-mode tls \
  --public-url https://gateway.example.com
```

The direct script path is a compatibility bridge to the tracked Ansible workflow. It still requires a real local `ansible-playbook` installation.

### 4.4 Systemd Host Service Management (Production Recommended)

Systemd integration provides boot-persistent service management for the AI Control Plane gateway. This is the recommended approach for production deployments where services must survive host reboots and integrate with standard Linux operational controls.

**When to use:**
- Production deployments requiring automatic service startup on boot
- Environments using standard Linux operations tooling (systemctl, journalctl)
- Hosts managed via configuration management or IaC pipelines
- Scenarios requiring service dependencies, resource limits, or restart policies

**Prerequisites:**
- systemd installed and running (standard on most modern Linux distributions)
- Root or sudo privileges for installing system-wide units
- Repository cloned and environment configured (see sections 4.1 or 4.2)

#### Step 1: Install the Systemd Service

Install and enable the systemd service using the Makefile:

```bash
make host-install
```

This command:
- Installs the `ai-control-plane.service` unit to the system directory
- Reloads systemd to recognize the new unit
- Enables the service for automatic startup on boot
- Starts the service immediately

**Privilege assumptions:**
The installation typically requires root or sudo access because it writes to the system unit directory (`/etc/systemd/system/`). Run with sudo or as root:

```bash
sudo make host-install
```

#### Step 2: Verify Service Status

Check the service status using the Makefile target:

```bash
make host-service-status
```

Or using systemctl directly:

```bash
systemctl status ai-control-plane.service
```

#### Available Makefile Targets

| Target | Description |
|--------|-------------|
| `host-install` | Install and enable the systemd service |
| `host-uninstall` | Stop, disable, and remove the systemd service |
| `host-service-status` | Check the systemd service status |
| `host-service-start` | Start the systemd service |
| `host-service-stop` | Stop the systemd service |
| `host-service-restart` | Restart the systemd service |

#### Boot Persistence

Once installed, the service is configured to:
- **Start on boot**: Enabled via `systemctl enable`
- **Track compose stack state**: Uses `Type=oneshot` with `RemainAfterExit=yes`
- **Respect service dependencies**: Waits for Docker service availability

The service definition ensures the AI Control Plane starts automatically after system reboots without manual intervention.

#### Preflight Validation

Before making system-level changes, validate host readiness and deployment configuration:

```bash
make host-preflight
make host-check INVENTORY=deploy/ansible/inventory/hosts.yml
```

These checks verify prerequisites and inventory-backed deployment settings before `make host-install`.

#### Secondary Direct Script Usage

For advanced use cases or integration with automation, use the deployment script directly as a secondary path:

```bash
# Install with default options
./scripts/acpctl.sh host install

# Install with explicit secrets/runtime paths
./scripts/acpctl.sh host install \
  --env-file /etc/ai-control-plane/secrets.env \
  --compose-env-file demo/.env

# Start the service after refreshing secrets
./scripts/acpctl.sh host service-start \
  --env-file /etc/ai-control-plane/secrets.env \
  --compose-env-file demo/.env

# Uninstall the service
./scripts/acpctl.sh host uninstall

# Show help
./scripts/acpctl.sh host --help
```

**Direct systemctl operations:**
```bash
# Start the service
sudo systemctl start ai-control-plane

# Stop the service
sudo systemctl stop ai-control-plane

# Restart after config changes
sudo systemctl restart ai-control-plane

# View logs
sudo journalctl -u ai-control-plane -f

# Disable autostart on boot
sudo systemctl disable ai-control-plane
```

---

## 5. Environment Configuration

> See the [Single-Tenant Production Contract](deployment/SINGLE_TENANT_PRODUCTION_CONTRACT.md)
> for the complete configuration specification including required vs optional
> variables, secret management, and validation rules.

### Required Variables

| Variable | Purpose | Default Value | Security Notes |
|----------|---------|---------------|----------------|
| `ACP_DATABASE_MODE` | Database deployment mode | `embedded` | Set `external` for externally managed PostgreSQL |
| `LITELLM_MASTER_KEY` | Admin key for generating virtual keys | `sk-litellm-master-change-me` | **CHANGE THIS** - Controls admin access |
| `LITELLM_SALT_KEY` | Persistent salt for key encryption | `sk-litellm-salt-change-me` | **CHANGE THIS** - Regenerating invalidates encrypted data |
| `DATABASE_URL` | PostgreSQL connection string | `postgresql://litellm:litellm@postgres:5432/litellm` | Contains database credentials |

`ACP_DATABASE_MODE` is the authoritative switch for database behavior. `DATABASE_URL` alone does not imply external mode because the embedded demo/reference stack also defines `DATABASE_URL`. For external PostgreSQL deployments, set `ACP_DATABASE_MODE=external` explicitly.

### Network Configuration

| Variable | Purpose | Default Value | Security Notes |
|----------|---------|---------------|----------------|
| `LITELLM_PUBLISH_HOST` | Interface to bind LiteLLM | `127.0.0.1` | Use `0.0.0.0` only with TLS in production |
| `CADDY_PUBLISH_HOST` | Interface to bind Caddy (TLS) | `127.0.0.1` | Use `0.0.0.0` for external TLS access |

### Optional Provider Keys

| Variable | Provider | Where to Obtain |
|----------|----------|-----------------|
| `OPENAI_API_KEY` | OpenAI | <https://platform.openai.com/api-keys> |
| `ANTHROPIC_API_KEY` | Anthropic | <https://console.anthropic.com/settings/keys> |
| `GEMINI_API_KEY` | Google Gemini | <https://makersuite.google.com/app/apikey> |

**Note**: The gateway can operate without provider keys for demo/testing purposes using mock configurations.

### Key Generation Commands

Generate secure keys using OpenSSL:

```bash
# Master key (48 bytes, base64-encoded)
openssl rand -base64 48 | tr -d '\n='

# Salt key (48 bytes, base64-encoded)
openssl rand -base64 48 | tr -d '\n='
```

### Master Key Validation

Scripts that generate virtual keys enforce validation rules for `LITELLM_MASTER_KEY`:

| Rule | Requirement | Error if violated |
|------|-------------|-------------------|
| Required | Must be non-empty | `LITELLM_MASTER_KEY is required` |
| No whitespace | Must not contain spaces, tabs, `\n`, `\r` | `must not contain whitespace` |
| Minimum length | Must be >= 32 characters | `looks too short` |
| Not placeholder | Must not be `sk-litellm-master-change-me` | `is still the placeholder value` |

**Why this matters:**
- Prevents accidental use of placeholder values in production
- Ensures sufficient entropy for secure authentication
- Avoids header injection risks from whitespace/newlines
- Provides clear error messages early (before API calls)

---

## 6. Network Configuration

### Single Machine Mode

The gateway is published on localhost. PostgreSQL is internal by default (Docker network only):

<!--
Project Color Scheme:
- Gateway (LiteLLM): Orange (#f26522) highlight emphasis
- Database: Blue (#0089cf) secondary accent for data
- Local Machine border: Gray (#444444)
-->

```
┌─────────────────────────────────────┐
│         Local Machine               │
│  ┌───────────────────────────────┐  │
│  │  LiteLLM: 127.0.0.1:4000      │  │
│  │  ═══════════════════════════  │  │
│  │  PostgreSQL: internal only    │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
```

### Remote Gateway Host Mode (Optional)

<!--
Project Color Scheme:
- Gateway Host: Orange (#f26522) emphasis — this is the control plane
- Client Machine: Gray (#444444) — standard component
- Network connection: Dashed line represents connectivity
-->

```
┌─────────────────────────────────────┐
│      Client Machine                 │
│      CLIENT_HOST                    │
│  ┌───────────────────────────────┐  │
│  │  Claude Code, Codex CLI,      │  │
│  │  OpenCode, Cursor             │  │
│  │  → Connect to GATEWAY_HOST    │  │
│  └───────────────────────────────┘  │
└──────────────┬────────────────────────┘
               │ Network
               ▼
╔═════════════════════════════════════╗
║  ╔═══════════════════════════════╗  ║
║  ║  Gateway Host                 ║  ║
║  ║  GATEWAY_HOST                 ║  ║
║  ║  ───────────────────────────  ║  ║
║  ║  Caddy TLS: 0.0.0.0:443       ║  ║
║  ║  LiteLLM: internal-only       ║  ║
║  ║  PostgreSQL: internal only    ║  ║
║  ╚═══════════════════════════════╝  ║
╚═════════════════════════════════════╝
```

### Firewall Configuration

**Ports to Open:**

| Port | Service | When Required |
|------|---------|---------------|
| 443 | TLS gateway ingress | Remote mode only |
| 5432 | PostgreSQL | Only if you intentionally publish DB access |

**Example UFW rules:**
```bash
# Allow from a specific client subnet
sudo ufw allow from CLIENT_SUBNET to any port 443
# Optional: only if you publish Postgres for direct access
# sudo ufw allow from CLIENT_SUBNET to any port 5432
```

### Outbound Egress Controls (Enterprise)

For production environments, implement **default-deny egress** to AI provider endpoints:

| Control | Purpose | Implementation |
|---------|---------|----------------|
| **Default-deny egress** | Prevent direct API access | Only gateway may reach `api.openai.com`, `api.anthropic.com` |
| **SWG/CASB** | Govern browser AI usage | Block personal tenants, allow enterprise workspaces |
| **MDM/Endpoint** | Enforce tool configuration | Force gateway routing via managed configs |

**Why this matters:** Without egress controls, developers can bypass the gateway using personal API keys or subscriptions, creating a governance blind spot.

**Local demo limitation:** This demo environment cannot prove egress blocking. Use the AWS lab or enterprise network infrastructure for validation.

### Customer Network Hardening Requirements (Mandatory)

The gateway alone cannot enforce enterprise-wide egress policy. The following controls must be implemented in customer-owned network/security infrastructure:

| Requirement | Control Owner | Description |
|---|---|---|
| Provider endpoint egress restrictions | Customer (network/security) | Default-deny outbound access so only approved gateway paths can reach provider APIs. |
| SWG/CASB tenant governance | Customer (security/web gateway) | Block personal tenants and unapproved AI SaaS categories; allow approved enterprise tenants only. |
| Endpoint configuration enforcement | Customer (endpoint/IT) | Use MDM/endpoint policy to enforce gateway base URLs and prevent local override of managed settings. |
| Firewall and DNS policy lifecycle | Customer (network/security) | Maintain endpoint/domain policies as providers evolve. |
| Evidence retention and SIEM onboarding | Customer (security operations) | Ingest and retain gateway/OTEL/compliance evidence per policy. |

### Project vs Customer Responsibility Split

| Responsibility Area | Project Provides | Customer Must Provide |
|---|---|---|
| Architecture and control design | Reference patterns, deployment guidance, control mapping | Approval and adaptation to enterprise topology/policy |
| Gateway enforcement | LiteLLM policy configuration and routed-path controls | Enterprise traffic routing and endpoint enforcement to keep traffic on governed paths |
| Network/SWG/CASB execution | Rule design support and validation criteria | Production firewall/SWG/CASB rule implementation and operations |
| Endpoint posture | Tooling guidance and onboarding automation | MDM/device management policy rollout and enforcement |
| Audit operations | Evidence schema, detections, runbooks | SIEM operations, retention policy, compliance ownership |

See:
- [Network and Endpoint Enforcement Demo](demos/NETWORK_ENDPOINT_ENFORCEMENT_DEMO.md)
- [GO_TO_MARKET_SCOPE.md](GO_TO_MARKET_SCOPE.md) - validated baseline and customer-environment validation boundary

## Observability and Monitoring

### OpenTelemetry (OTEL) Collector

The AI Control Plane includes an OpenTelemetry collector for telemetry that does not originate from gateway-managed logs (for example direct/bypass client paths) and for optional client-side correlation data.

#### Configuration Profiles

| Profile | Config File | Use Case | Exporters |
|---------|-------------|----------|-----------|
| Demo | `config.yaml` | Local development and bypass-path rehearsal | `debug`, `file` |
| Production | `config.production.yaml` | Production bypass/correlation telemetry export | `otlphttp/primary` |

#### Production Setup

**1. Configure Environment Variables** (in `demo/.env`, when OTEL export is in scope):

```bash
# Required: Remote OTLP endpoint
OTEL_EXPORTER_OTLP_ENDPOINT=https://your-otel-backend.example.com

# Optional: Authentication header
OTEL_EXPORTER_OTLP_AUTH_HEADER="Api-Key your-api-key"

# Required for remote client -> gateway OTEL ingress
OTEL_INGEST_AUTH_TOKEN=replace-with-long-random-token

# Required: Resource attributes for tagging
OTEL_RESOURCE_ENVIRONMENT=production
OTEL_RESOURCE_DEPLOYMENT=us-east-1

# Cost control: Sampling percentage
OTEL_TRACES_SAMPLING_PERCENT=10
```

**2. Validate and Start**:

```bash
# Validate production configuration
make validate-config-production

# Start with production profile (includes OTEL collector)
make up-production

# Verify OTEL collector health
make otel-health
```

#### Cost and Cardinality Controls

Production deployments should configure sampling to manage costs:

| Volume Level | Sampling | Use Case |
|--------------|----------|----------|
| High (>1M traces/day) | 1% | Large production deployments |
| Medium (100K-1M/day) | 10% (default) | Standard production |
| Low (<100K/day) | 50-100% | Small deployments, critical paths |

#### Supported Backends

The OTEL collector exports to any OTLP-compatible backend:

- **Datadog**: `OTEL_EXPORTER_OTLP_ENDPOINT=https://trace.agent.datadoghq.com`
- **Honeycomb**: `OTEL_EXPORTER_OTLP_ENDPOINT=https://api.honeycomb.io`
- **Grafana Cloud**: `OTEL_EXPORTER_OTLP_ENDPOINT=https://tempo-prod-XX-prod-us-east-XX.grafana.net`
- **Custom**: Any OTLP HTTP endpoint

### Kubernetes Monitoring (Helm)

The Helm chart includes Prometheus alerting rules for production monitoring.

**Enable Monitoring** (values.yaml):

```yaml
monitoring:
  serviceMonitor:
    enabled: true
    labels:
      release: prometheus  # Match your Prometheus Operator release
  
  prometheusRule:
    enabled: true
    labels:
      release: prometheus
  
  alerts:
    authFailureRateThreshold: 1
    authFailureWindow: 5m
    detectionErrorWindow: 10m
    gatewayLatencyThreshold: 5
```

**Baseline Alerts**:

| Alert | Severity | Description |
|-------|----------|-------------|
| ACPGatewayUnavailable | critical | No gateway replicas available |
| ACPGatewayHighErrorRate | warning | Error rate > 10% for 5 minutes |
| ACPAuthFailuresHigh | warning | Auth failures > 1/sec |
| ACPDetectionErrorsHigh | warning | Detection/guardrail errors |
| ACPBackupStale | critical | No backup in 25+ hours |
| ACPPodCPUHigh | warning | Pod CPU > 80% |
| ACPPodMemoryHigh | warning | Pod memory > 85% |

### Network Firewall Contract

The canonical network firewall contract defines all network flows for the AI Control Plane and serves as the source of truth for firewall configuration. The contract is version-controlled and published in multiple artifact formats for different consumption patterns.

**Generated Artifacts:**

| Artifact | Format | Purpose |
|----------|--------|---------|
| `docs/deployment/network_firewall_contract.md` | Markdown | Human-readable table for documentation and review |
| `docs/deployment/network_firewall_contract.csv` | CSV | Machine-readable for firewall import and SIEM ingestion |
| `docs/deployment/network_firewall_contract.json` | JSON | Structured data for automation and validation pipelines |

**Canonical Source:**

The contract is defined in `demo/config/network_firewall_contract.yaml`. This YAML file specifies:
- Required network flows (service-to-service and external)
- Port requirements and protocols
- Exposure semantics (`localhost`, `internal_only`, `public`)
- Manifest references and operational justifications per flow
- Description and justification for each rule

**Artifact ownership:**

The public snapshot publishes the canonical YAML source together with the checked-in Markdown, CSV, and JSON artifacts under `docs/deployment/`. There is no public `make` or `acpctl` regeneration wrapper for these files; update the YAML source and regenerated artifacts together in the same change when the contract changes.

---

## 7. Database

The AI Control Plane uses PostgreSQL 18 for persistent storage. This section covers database initialization, backup, and restore procedures.

For comprehensive database documentation, see [DATABASE.md](DATABASE.md).

### Database Initialization

LiteLLM automatically initializes the database schema on first startup when `DATABASE_URL` is configured. No manual migration is required.

**What gets created automatically:**
- `LiteLLM_VerificationToken` - Virtual key metadata (aliases, budgets, expiry)
- `LiteLLM_UserTable` - User metadata (if used)
- `LiteLLM_BudgetTable` - Budget records referenced by keys
- `LiteLLM_SpendLogs` - Usage/cost logs (“audit log” source)
- `LiteLLM_ProxyModelTable` - Proxy/global budgets (if configured)

**Note:** Table names and columns are an implementation detail of the LiteLLM
image tag used by this repo. Prefer `make db-status` for the canonical view.

**Verify initialization:**
```bash
make db-status
```

### Database Status

Check database health, table row counts, and recent activity:

```bash
make db-status
```

This displays:
- Database size and connection count
- Table statistics and row counts
- Virtual keys and budget usage
- Recent audit log entries

### Database Backup

Create timestamped backups of the PostgreSQL database:

```bash
make db-backup
```

**Backups are stored in:** `demo/backups/`

**Backup with custom name (typed CLI path):**
```bash
./scripts/acpctl.sh db backup my-backup-name
```

**Example backup cleanup** (keep last 7 days):
```bash
find demo/backups/ -name "litellm-backup-*.sql.gz" -mtime +7 -delete
```

### Database Restore

Restore the database from a backup file:

```bash
# List available backups
ls -1 demo/backups/*.sql.gz

# Restore latest backup
make db-restore

# Restore specific backup (typed CLI path)
./scripts/acpctl.sh db restore demo/backups/litellm-backup-20240128.sql.gz
```

**Important:**
- Restore overwrites all existing data
- LiteLLM service is stopped during restore
- Validate backup integrity before restore (for example: `gzip -t demo/backups/<backup>.sql.gz`)

### Volume Persistence

Database data is persisted in a Docker named volume (`pgdata`):

| Operation | Command | Data Preserved? |
|-----------|---------|-----------------|
| Stop services | `make down` | Yes |
| Restart services | `make up` | Yes |
| Clean artifacts | `make clean` | **No** (deleted) |

**Always backup before cleaning:**
```bash
make db-backup                          # Backup first
make clean                              # Then clean (prompts for confirmation)
# Or for scripts/automation:
# make clean-force                    # Skip confirmation
```

### Direct Database Access (Secondary/Debug)

Connect to the PostgreSQL database directly:

```bash
# Interactive psql session
docker exec -it $(docker compose ps -q postgres) \
  psql -U litellm -d litellm

# Run a single query
docker exec $(docker compose ps -q postgres) \
  psql -U litellm -d litellm -c "SELECT * FROM \"LiteLLM_VerificationToken\" LIMIT 5;"
```

---

## 8. Verification

### Health Check Commands

#### Using Makefile (Recommended)

```bash
# Full health check
make health

# Check running containers
make ps

# View service logs
make logs
```

#### Using Docker Compose Directly

```bash
cd demo

# Check container status
docker compose ps

# View logs (follow mode)
docker compose logs -f

# Check specific service logs
docker compose logs litellm
docker compose logs postgres
```

#### Using the Health Check Script

```bash
make health

# With verbose output
make health
```

#### Manual Verification with curl

```bash
# Test LiteLLM health endpoint
curl -H "Authorization: Bearer $LITELLM_MASTER_KEY" http://127.0.0.1:4000/health
# Remote mode: curl -H "Authorization: Bearer $LITELLM_MASTER_KEY" https://gateway.example.com/health

# Test models endpoint (requires authentication)
curl -H "Authorization: Bearer $LITELLM_MASTER_KEY" \
  http://127.0.0.1:4000/v1/models

# Test database connectivity
docker compose exec postgres \
  psql -U litellm -d litellm -c "SELECT version();"
```

### Expected Health Check Output

```
=== AI Control Plane Health Check ===

1. Docker Container Status
✓ PostgreSQL container is running
✓ LiteLLM container is running

2. LiteLLM Gateway Health Endpoint
✓ LiteLLM health endpoint is accessible (authorized HTTP 200)

3. LiteLLM Models Endpoint
✓ LiteLLM models endpoint is accessible

4. PostgreSQL Connectivity
✓ PostgreSQL is accepting connections

=== Health Check Summary ===
Health check: PASSED

All services are healthy and ready for use.
```

### License Boundary Validation

Before deploying to production, verify that no restricted commercial components are included:

```bash
# Check third-party license compliance
make license-check

# This validates:
# - No LiteLLM Enterprise components in packaging
# - License summary report is up to date
# - All third-party components comply with policy
```

The license boundary guard scans packaging-sensitive files (Makefile, Docker Compose files, gateway configs, shell scripts, and deployment manifests) for restricted patterns and ensures compliance with the third-party license policy defined in `docs/policy/THIRD_PARTY_LICENSE_MATRIX.json`.

To update the license summary report (required for customer handoff):

```bash
make license-report-update
```

The generated report (`docs/deployment/THIRD_PARTY_LICENSE_SUMMARY.md`) includes:
- Complete third-party component inventory
- License compliance status
- Any approved override records

### Runtime and Capacity Validation (Public Snapshot)

This public snapshot uses canonical CI/runtime gates as the supported capacity-validation surface.

#### Validation Workflow

Follow this workflow before production cutover:

1. **Run nightly gate** for runtime + release verification
2. **Run manual-heavy gate** for hardened-image/supply-chain validation
3. **Run smoke checks** against the target runtime
4. **Archive evidence** from readiness logs + release bundles

```bash
# Step 1: Runtime + release evidence gate
make ci-nightly

# Step 2: On-demand heavy gate
make ci-manual-heavy

# Step 3: Runtime smoke against the active runtime configuration
make prod-smoke

# Step 4: Capture release evidence artifacts
make release-bundle
make release-bundle-verify
```

#### Capacity Tuning Guidance

| Validation Outcome | Recommended Action |
|-------------------|--------------------|
| `ci-nightly` + smoke PASS | Keep baseline sizing |
| Smoke latency regressions | Increase CPU/memory and/or reduce expected concurrency |
| Runtime/detection failures | Fix config/policy issues before scaling |
| Heavy gate failures | Resolve hardened-image/supply-chain issues before cutover |

#### Evidence Locations

- Readiness logs: `demo/logs/evidence/readiness-<TIMESTAMP>/`
- Release bundles: `demo/logs/release-bundles/`
- Supply-chain summaries: `demo/logs/supply-chain/summary.json`

> Legacy benchmark harness command references from private iterations are retired from the current public-snapshot Make target surface. Use external load-testing tooling in customer environments when deeper performance profiling is required.

`make prod-smoke` is a real runtime gate. It fails when the gateway is unreachable, when authorized `/v1/models` checks cannot run successfully, or when the database/readiness contract is not healthy. `make helm-smoke` is a real repository Helm gate that validates tracked Helm surfaces and runs `helm lint`; it does not probe a live cluster.

---

## 9. Troubleshooting

### Services Fail to Start

**Symptoms:** Containers exit immediately or restart loops

**Diagnosis:**
```bash
# Check container status
make ps

# View logs for errors
make logs

# Check specific service logs
cd demo
docker compose logs litellm
docker compose logs postgres
```

**Common Causes:**
- Port conflicts (ports 4000 or 5432 already in use)
- Invalid environment variables in `.env`
- Missing or corrupted configuration files

**Solutions:**
```bash
# Stop services and remove volumes (prompts for confirmation)
make clean
# Or for scripts: make clean-force

# Restart services
make up

# Check for port conflicts
lsof -i :4000
lsof -i :5432
```

### Mode-Switching Issues

**Symptoms:** Authentication failed errors against PostgreSQL after switching between standard (`make up`) and offline (`make up-offline`) modes.

**Diagnosis:**
Standard and Offline modes may use different volumes if not configured correctly, or may attempt to reuse a volume initialized with a different password.

**Solutions:**
1. Ensure `demo/.env` has consistent `POSTGRES_PASSWORD`.
2. Clear volumes if switching from a stale environment:
   ```bash
   make clean-force
   ```
   *Note: This deletes all database data.*

---

### Port Conflicts

**Symptoms:** Error "bind: address already in use"

**Diagnosis:**
```bash
# Check what's using the ports
lsof -i :4000
lsof -i :5432

# Or with netstat
netstat -tulpn | grep -E '4000|5432'
```

**Solutions:**
1. Stop the conflicting service
2. Or modify `demo/docker-compose.yml` to use different ports

---

### Authentication Failures

**Symptoms:** 401 Unauthorized errors when accessing endpoints, or validation errors from scripts

**Diagnosis:**
```bash
# Read the current master key without sourcing demo/.env
./scripts/acpctl.sh env get LITELLM_MASTER_KEY

# Check key length (should be >= 32 chars)
key="$(./scripts/acpctl.sh env get LITELLM_MASTER_KEY)"
echo "Key length: ${#key} chars"
```

**Common Causes:**
- `LITELLM_MASTER_KEY` mismatch between `.env` and client requests
- `.env` file not loaded (restart containers after editing)
- Using wrong key (master key vs. virtual key)
- **Still using placeholder value** (`sk-litellm-master-change-me`)
- **Key contains whitespace/newlines** (copy-paste error)
- **Key too short** (< 32 characters)

**Solutions:**
```bash
# Generate a new secure key (produces ~64 characters)
openssl rand -base64 48 | tr -d '\n='

# Edit .env and replace the placeholder
nano demo/.env

# Restart containers after editing .env
make down
make up
```

**Validation Errors from Key Generation:**
If you see errors like these when running `make key-gen`:
- `Error: LITELLM_MASTER_KEY is still the placeholder value` → Generate a real key with the openssl command above
- `Error: LITELLM_MASTER_KEY must not contain whitespace` → Remove spaces/tabs/newlines from the key
- `Error: LITELLM_MASTER_KEY looks too short` → Use a key with at least 32 characters

---

### Network Connectivity Issues (Remote Mode)

**Symptoms:** Client cannot reach gateway host services

**Diagnosis:**

**Step 1: Run the automated connectivity test script**

```bash
# From client machine
ping -c 2 GATEWAY_HOST
  curl -sS -o /dev/null -w '%{http_code}\n' "https://GATEWAY_HOST/health"
```

**Step 2: Manual diagnosis (if script unavailable)**

```bash
# From client: Test network connectivity
ping GATEWAY_HOST

# Test port accessibility
telnet GATEWAY_HOST 4000
nc -zv GATEWAY_HOST 4000

# From Docker host: Check services are listening
ss -tulpn | grep -E '4000|5432'
```

**Common Causes:**
- Firewall blocking ports
- Services binding to localhost instead of 0.0.0.0
- Network routing issues

**Solutions:**
```bash
# Check firewall rules
sudo ufw status
sudo firewall-cmd --list-all

# Ensure ports are allowed
sudo ufw allow 4000/tcp
# Optional: only if you publish Postgres for direct access
# sudo ufw allow 5432/tcp

# Verify Docker port bindings
docker compose ps
```

---

### Database Issues

**Symptoms:** LiteLLM cannot connect to PostgreSQL, or queries fail

**Diagnosis:**
```bash
# Check database status
make db-status

# Check PostgreSQL container health
docker compose ps postgres

# Check PostgreSQL logs
docker compose logs postgres

# Test database connection
docker compose exec postgres \
  psql -U litellm -d litellm -c "SELECT 1;"
```

**Common Causes:**
- PostgreSQL not ready (wait longer after `make up`)
- Incorrect `DATABASE_URL` in `.env`
- Volume corruption
- Tables not created (LiteLLM creates them on first startup)

**Solutions:**
```bash
# Wait for PostgreSQL to be ready
docker compose exec postgres pg_isready -U litellm

# Check if tables exist
make db-status

# If tables missing, restart LiteLLM (it creates tables on startup)
make down && make up

# Recreate volumes (WARNING: deletes all data)
make db-backup                          # Backup first!
make clean                              # Prompts for confirmation
# Or: make clean-force                # For scripts
make up
```

**For more database troubleshooting, see [DATABASE.md](DATABASE.md).**

---

### Container Restart Loops

**Symptoms:** Containers continuously restart

**Diagnosis:**
```bash
# Check restart count
docker compose ps

# View recent logs
docker compose logs --tail=50 litellm
docker compose logs --tail=50 postgres
```

**Common Causes:**
- Configuration errors in `litellm.yaml`
- Resource constraints (insufficient RAM/CPU)
- Health check failures

**Solutions:**
```bash
# Validate YAML configuration
yamllint demo/config/litellm.yaml

# Check system resources
docker stats

# Increase Docker resource limits (Docker Desktop settings)
```

---

## 10. DLP and Guardrails

Data Loss Prevention (DLP) and guardrails provide content-level policy enforcement to prevent sensitive data from being sent to AI models.

### Guardrail Architecture Decision

The control plane uses two guardrail layers:

| Layer | What It Covers | Why It Exists |
|---|---|---|
| LiteLLM native guardrails | In-memory checks such as content filtering and prompt-injection detection | Lightweight baseline controls without external DLP services |
| Presidio guardrails | Deterministic PII/entity detection and custom recognizers | High-fidelity DLP for enterprise-specific data patterns |

This architecture keeps metadata-only evidence defaults while still allowing inline blocking/masking on routed traffic.

Guardrail lifecycle control points in this deployment:

| Stage | Purpose | Typical Implementation |
|---|---|---|
| Pre-call | Block or sanitize risky input before provider calls | Presidio DLP policies, LiteLLM content/prompt filters |
| In-call | Constrain behavior during generation/tool execution | Tool allowlists, parameter/schema constraints (when configured) |
| Post-call | Improve policy quality and response readiness | Detection tuning, SIEM correlation, monthly efficacy review |

Reference documentation:
- LiteLLM guardrails quickstart: <https://docs.litellm.ai/docs/proxy/guardrails/quick_start>
- LiteLLM content filter guardrail: <https://docs.litellm.ai/docs/proxy/guardrails/ai_guardrails/litellm_content_filter>
- LiteLLM prompt injection detection: <https://docs.litellm.ai/docs/proxy/guardrails/ai_guardrails/prompt_injection_detection>
- LiteLLM Presidio integration: <https://docs.litellm.ai/docs/proxy/guardrails/ai_guardrails/presidio>

**Running the DLP Demo:**

```bash
make demo-scenario SCENARIO=6
```

This demonstrates:
- Inline guardrail policy actions (blocked, redacted, allowed)
- Metadata-only audit evidence capture
- SIEM integration readiness

**Important:** This repo intentionally avoids storing request bodies in PostgreSQL.
Guardrail checks run inline before upstream calls while persisted evidence stays metadata-focused unless transcript handling is explicitly enabled by policy.

See `docs/observability/OTEL_SETUP.md` for telemetry guidance.

### Production DLP: Presidio Integration

For real-time blocking and redaction, integrate Microsoft Presidio with LiteLLM.

**Architecture:**

<!--
Project Color Scheme:
- LiteLLM Gateway: Orange (#f26522) — central control plane
- Presidio (DLP): Blue (#0089cf) — secondary/data processing
- Client Tool: Gray (#444444) — standard client component
-->

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   Client Tool   │────▶│  LiteLLM Gateway │────▶│  Presidio       │
│                 │     │  ═══════════════ │     │  Analyzer       │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                               │
                               ▼
                        ┌──────────────────┐
                        │  Presidio        │
                        │  Anonymizer      │
                        └──────────────────┘
```

**Configuration in litellm.yaml:**

```yaml
guardrails:
  - guardrail_name: "presidio-dlp-block"
    litellm_params:
      guardrail: presidio
      mode: "pre_call"
      presidio_analyzer_api_base: "http://presidio-analyzer:3000"
      presidio_anonymizer_api_base: "http://presidio-anonymizer:3000"
      pii_entities_config:
        US_SSN: "BLOCK"
        US_PASSPORT: "BLOCK"
        AWS_ACCESS_KEY: "BLOCK"
        EMAIL_ADDRESS: "MASK"
        PHONE_NUMBER: "REDACT"
      default_on: true
```

**Policy Actions:**

| Action | Behavior | Use Case |
|--------|----------|----------|
| `BLOCK` | Request rejected with 400 error | High-risk data (SSN, credentials) |
| `MASK` | Content replaced with `****` | Medium-risk PII (email, phone) |
| `REDACT` | Content removed entirely | Compliance requirements |
| `ALLOW` | No action taken | Approved content |

**Normalized Schema Mapping:**

When DLP guardrails trigger, the following fields are populated:

```yaml
policy:
  action: "blocked"        # or "redacted", "allowed"
  rule: "dlp-ssn-detected" # or "dlp-aws-key-detected"
  reason: "DLP: SSN pattern detected"
```

See `demo/config/normalized_schema.yaml` for complete schema details.

### Alternative: Custom Callback

For simpler regex-based DLP without Presidio:

```yaml
# In litellm.yaml
callbacks: ["demo/dlp_callback.py"]
```

Example callback (`demo/dlp_callback.py`):

```python
import re

def pre_call_hook(user_api_key_dict, cache, data, call_type):
    """Block requests containing SSN patterns."""
    content = data.get("messages", [{}])[0].get("content", "")

    ssn_pattern = r'\d{3}-\d{2}-\d{4}'
    if re.search(ssn_pattern, content):
        raise ValueError("DLP: SSN pattern detected - request blocked")

    return data
```

### DLP Best Practices

1. **Start with Guardrails**: Implement Presidio for real-time blocking/masking
2. **Use External Telemetry**: Route content-bearing logs to SIEM for analysis
3. **Gradual Enforcement**: Begin with alerting, then move to blocking
4. **Entity Tuning**: Fine-tune Presidio entities for your use case
5. **False Positive Review**: Monitor blocked requests for false positives
6. **Audit Everything**: Log all DLP decisions for compliance

### Compliance Mapping

| Regulation | DLP Requirement | Implementation |
|------------|-----------------|----------------|
| GDPR | PII protection | Email, phone, address masking |
| HIPAA | PHI protection | Medical record number blocking |
| PCI-DSS | Card data protection | Credit card number blocking |
| SOX | Financial data | Account number redaction |

---

## 11. Security Considerations

### Never Commit Secrets

The `.gitignore` at the repository root explicitly excludes `demo/.env` to prevent accidental credential exposure.

**If you have accidentally committed secrets:**
1. Rotate the compromised keys immediately
2. Remove sensitive data from git history using `git filter-repo` or BFG Repo-Cleaner
3. Consider enabling GitHub secret scanning

### OAuth Token Safety

**CRITICAL**: When using subscription mode (e.g., Claude Code with OAuth login), the gateway forwards OAuth tokens to upstream providers. Ensure:

- LiteLLM is configured not to log Authorization headers
- Any reverse proxy does not log headers
- Stored traffic has headers stripped before persistence
- Logs are reviewed to ensure no tokens are present

### Key Management

**For Production:**
- Use a proper secrets management system (HashiCorp Vault, AWS Secrets Manager, etc.)
- Rotate keys regularly
- Use different keys for different environments
- Implement key expiry policies

**For Demo/Development:**
- Generate unique keys per deployment
- Never use default placeholder keys in production
- Store keys securely (password manager, encrypted file)

### CI/Production Resource Separation

To prevent CI cleanup operations from affecting production deployments:

**Option 1: Use a different slot name**
```bash
ACP_SLOT=production docker compose up -d
```

**Option 2: Use a different project name**
```bash
docker compose -p acp-production up -d
```

Either approach ensures production resources are not affected by CI cleanup operations, which target the CI slot (`ci-runtime`) and CI project names (`acp-active`, `acp-standby`).

**CI Cleanup Targets:**
| Cleanup Type | Affected Resources |
|--------------|-------------------|
| `make down` | Volumes/containers with slot: `ci-runtime` |
| `make down` | Projects: `acp-active`, `acp-standby` |
| `make ps` | Verifies no CI resources remain |

**Production-Safe Names:**
- Slot names: `production`, `prod`, `live`, or customer-specific identifiers
- Project names: `acp-production`, `acp-prod`, or organization-specific prefixes

### TLS/HTTPS

**For Production:**
- **Enable TLS** using the provided Caddy reverse proxy configuration
- Use Let's Encrypt for automatic certificate management
- Configure valid domain name with DNS A record
- Enforce HTTPS-only connections
- See [deployment/TLS_SETUP.md](deployment/TLS_SETUP.md) for complete setup guide

**For Demo/Development:**
- Plain HTTP remains localhost-only for single-machine troubleshooting
- Any remote access, including demos, must terminate with TLS
- Quick start: `make up-tls` (see [deployment/TLS_SETUP.md](deployment/TLS_SETUP.md))

**Quick Reference:**
| Mode | Command | Endpoint | Certificate |
|------|---------|----------|-------------|
| HTTP (localhost only) | `make up` | http://localhost:4000 | None |
| HTTPS (local) | `make up-tls` | https://localhost | Self-signed |
| HTTPS (prod) | `make up-tls` | https://your-domain.com | Let's Encrypt |

**OAuth Token Safety:**
- When using subscription mode (Claude Code OAuth), tokens are forwarded upstream
- TLS encrypts tokens in transit
- Caddy's default logging does NOT log Authorization headers (safe by default)
- Verify: `docker compose logs caddy \| grep -i authorization` (should return nothing)

**TLS Environment Variables:**

| Variable | Purpose | When Required |
|----------|---------|---------------|
| `CADDY_PUBLISH_HOST` | Interface to bind Caddy ports | Always for TLS mode |
| `CADDYFILE_PATH` | Path to Caddyfile | Always for TLS mode |
| `CADDY_ACME_CA` | Certificate authority (`internal` or `letsencrypt`) | Always for TLS mode |
| `CADDY_DOMAIN` | Domain name for certificates | `CADDY_ACME_CA=letsencrypt` |
| `CADDY_EMAIL` | Email for cert notifications | `CADDY_ACME_CA=letsencrypt` |
| `LITELLM_PUBLIC_URL` | Public HTTPS URL | TLS mode |

**Key Files:**
- `demo/docker-compose.tls.yml` - TLS-enabled Docker Compose override
- `demo/config/caddy/Caddyfile.dev` - Self-signed certificate configuration
- `demo/config/caddy/Caddyfile.prod` - Let's Encrypt configuration (uses env vars)
- `docs/deployment/TLS_SETUP.md` - Comprehensive TLS documentation
- `docs/deployment/SINGLE_TENANT_PRODUCTION_CONTRACT.md` - TLS requirements for production

### Supply-Chain Security

The AI Control Plane includes a mandatory supply-chain gate for this public snapshot that validates policy contracts, digest pinning, and allowlist expiry windows.

**Current Baseline:**
- Gate status is policy-driven and evaluated at runtime from `demo/config/supply_chain_vulnerability_policy.json`
- Temporary exceptions are tracked in the allowlist with explicit expiry and ticket ownership
- Default runtime images are digest-pinned in `demo/docker-compose.yml` (and Helm values where applicable)

**Running the Supply-Chain Gate:**

```bash
# Gate: policy contract + digest pinning + allowlist expiry
make supply-chain-gate

# Check allowlist expiry windows (default warn<45d fail<14d)
make supply-chain-allowlist-expiry-check

# Generate deterministic policy summary artifact
make supply-chain-report
```

**Evidence Artifacts** (for compliance and customer handoff):
- `demo/logs/supply-chain/summary.json` - Supply-chain policy summary

> SBOM/CVE/provenance artifacts are not generated by default public-snapshot make targets. Integrate dedicated tools (for example Syft/Trivy/Cosign) in downstream production pipelines when required.

**Policy Configuration:**

Policy file: `demo/config/supply_chain_vulnerability_policy.json`

Default enforcement:
- Policy contract must be structurally valid
- Require digest-pinned image references
- Allowlist expiry windows must satisfy warn/fail thresholds
- Severity gate thresholds are enforced from policy configuration

Example allowlist entry for accepted risks:
```json
{
  "id": "CVE-2024-XXXX",
  "package": "vulnerable-package",
  "image": "ghcr.io/fitchmultz/acp/litellm-hardened@sha256:...",
  "expires_on": "2024-12-31",
  "justification": "Pending upstream patch",
  "owner": "security-team",
  "ticket": "SEC-1234"
}
```

**Remediation Workflow:**

1. **Review Findings**: Check `summary.json` for policy status and allowlist coverage
   ```bash
   jq '.policy_id, .allowlist_count, .status' demo/logs/supply-chain/summary.json
   ```

2. **Patch**: Update image digests in compose/Helm files to patched versions

3. **Allowlist**: Add temporary exceptions with expiry date and ticket reference

4. **Expiry Guard**: Run `make supply-chain-allowlist-expiry-check` to catch near-term expiry drift

5. **Re-verify**: Run `make supply-chain-gate` to confirm resolution

**CI Integration:**

The supply-chain gate and allowlist expiry check run automatically during:
- `make ci` - Standard CI gate
- `make ci-nightly` - Production readiness gate

Failure blocks deployment until resolved or explicitly allowed.

**Key Files:**
- `demo/config/supply_chain_vulnerability_policy.json` - Policy configuration
- `make supply-chain-gate` (`mk/security.mk`) - Policy + digest + expiry gate implementation
- `make supply-chain-allowlist-expiry-check` (`scripts/libexec/check_supply_chain_allowlist_expiry_impl.py`) - Expiry reminder/fail-window enforcement
- `docs/deployment/PRODUCTION_HANDOFF_RUNBOOK.md` - Handoff evidence requirements

---

## 12. Next Steps

### Generate Virtual Keys

Create per-user or per-service virtual keys for authentication:

```bash
# Using Makefile
make key-gen ALIAS=your-user-or-service-name BUDGET=10.00

# Secondary direct-script path
make key-gen ALIAS=your-alias BUDGET=10.00
```

### Configure AI Tools

Refer to the following documents for tool-specific configuration:

**Core Tools:**
- **Claude Code**: See `tooling/CLAUDE_CODE_TESTING.md` and `LOCAL_DEMO_PLAN.md`
- **Codex CLI**: See `tooling/CODEX.md` and `LOCAL_DEMO_PLAN.md`

**Optional Tools:**
- **OpenCode**: See `tooling/OPENCODE.md` - LiteLLM integration and bypass scenarios
- **Cursor**: See `tooling/CURSOR.md` - Custom base URL setup and per-user key tracking

### Additional Documentation

| Document | Description |
|----------|-------------|
| [README.md](../README.md) | Project overview and quick start |
| [Demo Environment README](../demo/README.md) | Detailed demo environment documentation |
| [Database Documentation](DATABASE.md) | Database schema, backup, restore, and maintenance |
| [Enterprise AI Control Plane Strategy](ENTERPRISE_STRATEGY.md) | Complete strategy and architecture overview |
| [Local Demo Implementation Plan](LOCAL_DEMO_PLAN.md) | Single-server local demo implementation plan |
| [Go-To-Market Scope And Readiness](GO_TO_MARKET_SCOPE.md) | Validated baseline and customer-environment proof boundary |

### Useful Makefile Targets

| Target | Description |
|--------|-------------|
| `make help` | Show all available targets |
| `make install` | Set up dependencies and environment |
| `make validate-config` | Validate deployment configuration against contract |
| `make up` | Start Docker services (validates config first) |
| `make up-tls` | Start Docker services with TLS (validates config first) |
| `make down` | Stop Docker services (preserves volumes) |
| `make health` | Health check gateway |
| `make logs` | View Docker logs (follow mode) |
| `make ps` | Show running containers |
| `make key-gen` | Generate a LiteLLM virtual key |
| `make db-backup` | Backup PostgreSQL database |
| `make db-restore` | Restore PostgreSQL database from backup |
| `make db-status` | Show database status and statistics |
| `make clean` | Remove artifacts + logs. DESTRUCTIVE: deletes Docker volumes. Prompts; use `FORCE=1` for non-interactive. |
| `make demo-scenario SCENARIO=6` | Run DLP/guardrails demo |
| `make demo-scenario SCENARIO=7` | Run Network/Endpoint Enforcement demo |
| `make ci` | Run full CI gate (format drift check, generate, lint, type-check, build, supply-chain-gate, supply-chain-allowlist-expiry-check, script-tests, license-report-update, runtime validation via pinned offline images) |
| `make build` | Build/recreate Docker containers (requires Docker+Compose; exits 2 if missing) |
