# Portainer Operations (Optional)

This guide describes how to manage an already-deployed Linux host Docker stack with Portainer.

> **Portainer is optional convenience tooling.** The canonical deployment path remains Docker-first host deployment in [`../DEPLOYMENT.md`](../DEPLOYMENT.md).

> **IMPORTANT: Portainer as Optional Operator UI Layer**
> 
> Portainer is **strictly optional** and serves only as a visual observation layer for operators who prefer a GUI. It **MUST NOT** replace repo-managed configuration or the systemd control flow.
> 
> - **Canonical source of truth:** Compose files (`docker-compose.yml`, etc.) and the Makefile workflow
> - **Lifecycle ownership:** systemd manages service state; Portainer does not
> - **Configuration authority:** All changes flow through version-controlled files, never Portainer's internal state
> - **CI/CD independence:** No runtime dependency on Portainer for automated deployments

## Table of Contents

1. [Overview](#overview)
2. [When to Use Portainer](#when-to-use-portainer)
3. [Prerequisites](#prerequisites)
4. [Connecting Portainer to the AI Control Plane](#connecting-portainer-to-the-ai-control-plane)
5. [Portainer Stack Workflow (Optional)](#portainer-stack-workflow-optional)
6. [Systemd vs Portainer Responsibilities](#systemd-vs-portainer-responsibilities)
7. [Common Operations](#common-operations)
8. [Operational Guardrails](#operational-guardrails)
9. [Portainer Mode Contract Checklist](#portainer-mode-contract-checklist)
10. [Troubleshooting](#troubleshooting)

---

## Overview

Portainer is a lightweight management UI for Docker environments. When your team already uses Portainer to operate Docker hosts, you can use it to:

- Monitor container health and status
- View logs and resource usage
- Restart containers during troubleshooting
- Inspect stack configuration (read-only recommended)

Portainer operates **on top of** the existing Docker Compose deployment—it does not replace the canonical deployment workflow.

---

## When to Use Portainer

| Use Portainer | Do Not Use Portainer |
|--------------|---------------------|
| Your team already manages Docker hosts with Portainer | You need to perform initial deployment (use `make up`) |
| You want a GUI for monitoring container health | You need to modify `.env` secrets (use shell/editor) |
| You need quick access to logs across multiple containers | You are deploying for the first time (follow `DEPLOYMENT.md`) |
| You are troubleshooting with a visual interface | You need production contract validation (use `make validate-config-production`) |

---

## Prerequisites

Before using Portainer with the AI Control Plane:

1. **Docker host already deployed** with AI Control Plane services running:
   ```bash
   # Verify services are running
   ssh user@gateway-host 'cd /opt/ai-control-plane && make health'
   ```

2. **Portainer installed** on the Docker host or with agent access:
   ```bash
   # Example: Portainer CE installation
   docker volume create portainer_data
   docker run -d -p 8000:8000 -p 9443:9443 --name portainer \
     --restart=always -v /var/run/docker.sock:/var/run/docker.sock \
     -v portainer_data:/data portainer/portainer-ce:<pin-approved-version>
   ```

3. **Network access** to Portainer UI from your management workstation

---

## Connecting Portainer to the AI Control Plane

### Option 1: Local Docker Environment (Single Host)

If Portainer runs on the same host as the AI Control Plane:

1. Access Portainer at `https://GATEWAY_HOST:9443`
2. Select **Local** environment (Docker socket already mounted)
3. Navigate to **Stacks** to see `ai-control-plane` (or your project name)

### Option 2: Portainer Agent (Remote Management)

For centralized management of multiple gateway hosts:

1. **Deploy Portainer Agent** on each gateway host:
   ```bash
   docker run -d \
     -p 9001:9001 \
     --name portainer_agent \
     --restart=always \
     -v /var/run/docker.sock:/var/run/docker.sock \
     -v /var/lib/docker/volumes:/var/lib/docker/volumes \
     portainer/agent:<pin-approved-version>
   ```

2. **Add Environment** in your central Portainer:
   - Go to **Environments** → **Add environment**
   - Select **Edge Agent** or **Agent** type
   - Enter the gateway host IP and port 9001

---

## Portainer Stack Workflow (Optional)

This section describes how to import the existing Docker Compose stack into Portainer for observation purposes. This workflow is **optional** and does not change the systemd-first operational model.

### Importing the Existing Stack

1. **Verify stack is managed externally** (via systemd) before importing:
   ```bash
   # On the gateway host
   systemctl is-active ai-control-plane
   # Expected output: active

   # Verify compose stack was created by systemd/docker, not Portainer
   docker compose ls --format json | jq -r '.[] | select(.Name=="ai-control-plane") | .Status'
   # Expected: "running(<n>)" or similar, indicating healthy containers
   ```

2. **In Portainer UI**, navigate to **Stacks** → **Add stack**

3. **Select "From repository" or "Web editor"** (do not let Portainer manage the lifecycle):
   - **From repository:** Point to your git repo with the same path used by systemd
   - **Web editor:** Copy/paste the compose file content for read-only inspection

4. **Name the stack** to match the systemd service context (e.g., `ai-control-plane`)

5. **Do NOT enable Portainer's auto-update or webhook features**—updates flow through the host-side workflow

### Stack Update Procedure (Defers to Host-Side Changes)

When configuration changes are needed, **always** defer to the host-side workflow:

```bash
# On the gateway host (canonical path)
ssh user@gateway-host
cd /opt/ai-control-plane
git pull  # Update from repository
make ci-nightly SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env  # Required pre-handoff gate
# Optional focused check:
# make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
sudo systemctl restart ai-control-plane  # Restart via systemd
```

After host-side restart, Portainer will reflect the new state automatically:
- Container images updated → visible in Portainer within seconds
- Environment changes → reflected in container inspect view
- Health status → updated based on new container health checks

**Portainer's role:** Observation only. Never use Portainer's "Update the stack" or "Pull and redeploy" buttons for this deployment.

---

## Systemd vs Portainer Responsibilities

The following table defines the clear boundary between systemd (owner) and Portainer (observer):

| Responsibility | Systemd (Owner) | Portainer (Observer) |
|---------------|-----------------|----------------------|
| **Service lifecycle** | Start, stop, restart, enable, disable | View status only (read-only) |
| **Configuration source** | Git-tracked compose files + canonical `/etc/ai-control-plane/secrets.env` (synced into `demo/.env`) | Display rendered compose (inspection) |
| **Update/upgrade procedure** | `git pull` → `make ci-nightly SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env` → `systemctl restart` | Passive observation of results |
| **Secret management** | Host filesystem (`/etc/ai-control-plane/secrets.env` source of truth; `demo/.env` is synced runtime file) | Never stores or modifies secrets |
| **Health validation** | `make health`, `make ci-nightly` (handoff gate), `make validate-config-production` (config-only) | Visual health indicators (informational) |
| **CI/CD integration** | Required for automated deployment | No runtime dependency |
| **Rollback procedure** | `git checkout` → `systemctl restart` | Not involved in rollback |
| **Disaster recovery** | Host-level backup/restore procedures | No persistent state to recover |

### Clear Boundary Statement

- **systemd owns lifecycle:** All start/stop/restart operations flow through systemd. The systemd service unit is the single source of truth for service state.
- **Portainer is read-only observation:** Operators may use Portainer to view container status, inspect logs, and check resource usage. Portainer must not modify containers, environment variables, or stack configuration.

---

## Common Operations

### View Container Status

1. Navigate to **Containers** in the Portainer UI
2. Look for containers named `ai-control-plane-litellm-*` and `ai-control-plane-postgres-*`
3. Check status column (should show **running**)

### Inspect Logs

1. Click on a container name (e.g., `ai-control-plane-litellm-1`)
2. Select the **Logs** tab
3. Use the search/filter box for specific entries

> **Security Note:** Logs may contain sensitive information. Do not copy log excerpts with tokens or keys when sharing diagnostics.

### Restart a Container

1. Select the container from the list
2. Click **Restart** button (or use **Stop** then **Start**)

> **Caution:** Restarting PostgreSQL will briefly interrupt service. Prefer scheduled maintenance windows. For production, use the systemd workflow instead of ad-hoc restarts.

### View Resource Usage

1. Navigate to **Dashboard** for the environment
2. View CPU, memory, and network I/O graphs
3. Click individual containers for detailed metrics

---

## Operational Guardrails

### Secrets Handling

- **Never** store `.env` file contents in Portainer environment variables
- Manage canonical secrets in `/etc/ai-control-plane/secrets.env`, then sync with `make host-secrets-refresh`
- Treat Portainer as a privileged surface: Docker socket access allows inspection of container environment variables
- Restrict Portainer access to trusted operators and follow least-privilege RBAC

### Configuration Management

- Make configuration changes via `/etc/ai-control-plane/secrets.env` and `demo/config/litellm.yaml` on the host
- Use Portainer for **observation**, not **configuration editing**
- After host-side changes, restart via `systemctl restart ai-control-plane` on the gateway host

### Stack Updates

For stack modifications (image updates, environment changes):

```bash
# On the gateway host (preferred)
ssh user@gateway-host
cd /opt/ai-control-plane
git pull  # If updating from repository
make validate-config-production  # Validate before applying (production)
make host-secrets-refresh SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env HOST_COMPOSE_ENV_FILE=demo/.env
sudo systemctl restart ai-control-plane  # Apply changes via systemd-owned lifecycle
```

Portainer can also recreate containers, but the systemd-first approach ensures:
- Pre-deployment validation (`make validate-config-production` for production handoff)
- Consistent environment variable handling
- Proper health check sequencing
- Integration with systemd service management

---

## Portainer Mode Contract Checklist

Before using Portainer with the AI Control Plane, confirm the following contract items:

| Item | Status | Description |
|------|--------|-------------|
| **Optionality acknowledged** | [ ] | Portainer is strictly optional; all workflows function without it |
| **Compose source-of-truth confirmed** | [ ] | `docker-compose.yml` and overlays in repo are canonical; Portainer does not hold configuration |
| **Stack import procedure documented** | [ ] | Procedure in [Portainer Stack Workflow](#portainer-stack-workflow-optional) section followed |
| **No runtime dependency on Portainer** | [ ] | CI/CD pipelines, automated deployments, and disaster recovery do not require Portainer |
| **Systemd ownership acknowledged** | [ ] | Service lifecycle owned by systemd; Portainer is read-only observation |
| **Secret management boundary confirmed** | [ ] | Secrets source of truth remains `/etc/ai-control-plane/secrets.env` on host; never stored in Portainer |

**Acknowledgment Statement:**

> By using Portainer with this deployment, I acknowledge that:
> 1. Portainer is an optional UI layer and not required for operation
> 2. All configuration authority remains with version-controlled compose files
> 3. Systemd owns service lifecycle; Portainer is for observation only
> 4. No operational procedure depends on Portainer being available

---

## Troubleshooting

### Container Shows as "Unhealthy"

1. Check logs in Portainer for error messages
2. SSH to host and run detailed health check:
   ```bash
   ssh user@gateway-host 'cd /opt/ai-control-plane && make health'
   ```
3. Verify database connectivity:
   ```bash
   ssh user@gateway-host 'cd /opt/ai-control-plane && make db-status'
   ```

### Cannot Connect to Portainer

| Symptom | Solution |
|---------|----------|
| Connection refused | Verify Portainer container is running: `docker ps` |
| Certificate warning | Accept self-signed cert or configure proper TLS |
| Timeout | Check firewall rules for port 9443 (or 9000 for HTTP) |

### Stack Not Visible in Portainer

1. Verify the AI Control Plane was started with `docker compose`:
   ```bash
   # On gateway host
   docker compose ls
   ```
2. Check Portainer environment is pointed at correct Docker socket
3. Ensure the stack directory (`/opt/ai-control-plane` or equivalent) is accessible

---

## References

- [Main Deployment Guide](../DEPLOYMENT.md) - Canonical Docker-first deployment
- [Production Handoff Runbook](./PRODUCTION_HANDOFF_RUNBOOK.md) - Operational procedures
- [Portainer Documentation](https://docs.portainer.io/) - Official Portainer docs
