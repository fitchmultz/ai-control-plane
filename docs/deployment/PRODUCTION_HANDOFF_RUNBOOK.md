# AI Control Plane - Production Handoff Runbook

## Purpose

This runbook provides operational procedures for customer handoff of production AI Control Plane deployments. It covers single-tenant deployments using the canonical Linux-host Docker deployment with an optional Kubernetes/Helm track for organizations that already operate Kubernetes.

**Audience**: Site Reliability Engineers, DevOps teams, and security operators responsible for production deployments.

**Scope**: Single-tenant production deployments only. All deployments serve exactly one customer/organization.

Operator interface order in this runbook:
1. Use `make` targets as the default operator entrypoint.
2. Use `./scripts/acpctl.sh` for migrated typed workflows (currently `./scripts/acpctl.sh ci should-run-runtime`).
3. Use direct scripts/systemctl as secondary compatibility or break-glass controls.

---

## Table of Contents

1. [Deployment Checklist](#deployment-checklist)
2. [Secrets and Key Lifecycle](#secrets-and-key-lifecycle)
3. [Backup and Restore](#backup-and-restore)
4. [Upgrade Procedures](#upgrade-procedures)
5. [Incident Response](#incident-response)
6. [Validation Steps](#validation-steps)

---

## Deployment Checklist

### Pre-Deployment

- [ ] **Deployment Topology**: Choose embedded DB (single host) or split-host (gateway + external DB)
- [ ] **DNS Configuration**: Domain name configured and pointing to deployment host
- [ ] **TLS Strategy**: 
  - Docker Compose: Caddy with Let's Encrypt or internal CA
  - Kubernetes: Ingress TLS with cert-manager or pre-created secrets
- [ ] **Network Rules**: 
  - **Source of Truth**: Canonical network firewall contract defines all required flows
  - **Handoff Packet**: `docs/deployment/network_firewall_contract.*` (markdown, csv, json)
  - **Key Ports** (defined in contract):
    - Database port (5432) NOT exposed externally (embedded mode) OR
    - Database port (5432) accessible from gateway host only (split-host mode)
    - Gateway port (4000) bound to localhost unless TLS-enabled
    - Caddy ports (80/443) exposed for TLS mode
  - **Artifact Set**: Treat `demo/config/network_firewall_contract.yaml` plus `docs/deployment/network_firewall_contract.*` as the tracked source bundle for firewall reviews

### Environment Setup

- [ ] **Master Key**: Generate secure master key (>=32 characters)
  ```bash
  openssl rand -base64 48 | tr -d '\n='
  ```
- [ ] **Salt Key**: Generate persistent salt key (>=32 characters, NEVER change after initial set)
  ```bash
  openssl rand -base64 48 | tr -d '\n='
  ```
- [ ] **Database Mode**: Choose and configure database deployment mode
  - Embedded mode (default): PostgreSQL runs as Docker container
  - Split-host mode: PostgreSQL runs on separate host (see Split-Host Deployment below)
- [ ] **Database Credentials**: 
  - Embedded: Strong password for PostgreSQL (>=16 characters for production)
  - Split-host: External DATABASE_URL with credentials (managed separately)
- [ ] **Provider Keys**: API keys for upstream LLM providers (if not using mock/offline mode)

### Configuration Files

- [ ] **Canonical Secrets File**: `/etc/ai-control-plane/secrets.env` (0600/0640, non-symlink) for Docker Compose host deployments
- [ ] **Compose Runtime Env File**: `demo/.env` refreshed from canonical secrets via `make host-secrets-refresh`
- [ ] **TLS Certificates**: Caddyfile configured or Ingress TLS secrets created
- [ ] **Model Configuration**: `litellm.yaml` with approved model aliases

### Validation

- [ ] **Production Gate (Required Before Customer Handoff)**: Run the enterprise production readiness gate
  ```bash
  # For host-first deployments (default)
  make ci-nightly SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
  
  # With optional Kubernetes profile checks (when K8s deployment planned)
  LITELLM_MASTER_KEY='<redacted>' CI_PRODUCTION_K8S=1 make ci-nightly \
    SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env \
    NAMESPACE=acp RELEASE=acp
  ```
  
  **Evidence Requirement**: Capture and archive `ci-nightly` output log with timestamp:
  ```bash
  make ci-nightly SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env 2>&1 | \
    tee "ci-nightly-$(hostname)-$(date +%Y%m%d-%H%M%S).log"
  ```

- [ ] **Config Validation**: Run production profile validation (also covered by ci-nightly gate)
  ```bash
  make validate-config-production
  # Secondary secrets-file variant (TLS mode):
  make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
  ```

- [ ] **Observability Validation**: Verify OTEL configuration for production
  ```bash
  # Ensure required OTEL environment variables are set without grepping demo/.env
  ./scripts/acpctl.sh env get OTEL_EXPORTER_OTLP_ENDPOINT
  ./scripts/acpctl.sh env get OTEL_RESOURCE_ENVIRONMENT
  
  # Start with production profile (includes OTEL)
  make up-production
  
  # Verify OTEL collector health
  make otel-health
  ```

- [ ] **Host Preflight Gate**: Validate host operational readiness
  ```bash
  # Run host preflight checks
  make host-preflight
  ```
  
  **Evidence Requirement**: Capture preflight output with timestamp and hostname:
  ```bash
  make host-preflight 2>&1 | tee "host-preflight-$(hostname)-$(date +%Y%m%d-%H%M%S).log"
  ```

- [ ] **Declarative Preflight** (if using host orchestrator): Check mode validation
  ```bash
  make host-check INVENTORY=deploy/ansible/inventory/hosts.yml
  ```
  
  Note: Declarative check/apply validates the tracked Ansible inventory and playbook surface. Run `make host-preflight` separately on the gateway host before service installation.
- [ ] **Helm Validation**: For Kubernetes deployments
  ```bash
  make helm-validate
  ```
- [ ] **License Boundary Check**: Verify no restricted components
  ```bash
  make license-check
  ```

- [ ] **Runtime + Security Baseline**: Run nightly and manual-heavy gates, then archive readiness artifacts
  ```bash
  # Nightly gate (runtime + release verification)
  make ci-nightly SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env

  # On-demand heavy gate (includes hardened image build/scan)
  make ci-manual-heavy

  # Build and verify release bundle artifacts for handoff
  make release-bundle
  make release-bundle-verify
  ```

  > **Acceptance Requirement:** Handoff evidence MUST include a successful nightly gate log and release-bundle verification output before production cutover.
  >
  > Performance benchmark harness commands are not part of the current public-snapshot command surface; run external load testing in customer environments when additional sizing evidence is required.

---

## Split-Host Deployment (Gateway + External DB)

### Overview

In split-host deployments, the LiteLLM gateway runs on a dedicated gateway host while PostgreSQL runs on a separate database host. This topology is required by some enterprise security policies.

### Network Architecture

```
┌──────────────────────────┐      TCP/5432      ┌──────────────────────────┐
│    Gateway Host          │◄──────────────────►│    DB Host               │
│    - Caddy Proxy         │      (PostgreSQL)  │    - PostgreSQL          │
│    - LiteLLM Gateway     │                    │    - Managed/External    │
│    - Presidio (PII)      │                    │                          │
└──────────────────────────┘                    └──────────────────────────┘
           ▲
           │ HTTPS/443
    ┌──────┴──────┐
    │   Clients   │
    └─────────────┘
```

### Prerequisites

1. **Database Host**: PostgreSQL 18+ running and accessible from gateway host
2. **Network Connectivity**: Gateway host can reach DB host on TCP/5432
3. **Database User**: Dedicated user created with appropriate permissions
4. **Firewall Rules**: DB host allows inbound 5432 from gateway host only

### Configuration

1. **Set database mode to external** in `.env`:
   ```bash
   ACP_DATABASE_MODE=external
   ```

2. **Configure external DATABASE_URL** in `.env`:
   ```bash
   DATABASE_URL=postgresql://litellm:<password>@db-host.example.com:5432/litellm?sslmode=require
   ```

3. **Remove or comment out** POSTGRES_* variables (not needed in external mode):
   ```bash
   # POSTGRES_USER=litellm      # Not used in external mode
   # POSTGRES_PASSWORD=...      # Not used in external mode
   # POSTGRES_DB=litellm        # Not used in external mode
   ```

### Validation Steps

1. **Verify network connectivity** from gateway host:
   ```bash
   # Test TCP connectivity to database
   nc -zv db-host.example.com 5432
   
   # Canonical runtime checks
   make health
   make doctor
   ```

2. **Validate configuration**:
   ```bash
   make validate-config-production
   ```

3. **Start services** (postgres container will NOT be started):
   ```bash
   ACP_DATABASE_MODE=external make up
   ```

4. **Verify health**:
   ```bash
   make health
   ```

### Troubleshooting Split-Host Issues

| Symptom | Cause | Solution |
|---------|-------|----------|
| "Connection refused" on gateway startup | Firewall blocking 5432 | Verify DB host firewall allows gateway IP |
| "sslmode required" error | Missing sslmode in URL | Add `?sslmode=require` to DATABASE_URL |
| "unknown host" error | DNS resolution failure | Use IP address or verify DNS configuration |
| Gateway health check fails | External DB not accessible | Run connectivity test from gateway host |

### Security Considerations

1. **Use SSL/TLS**: Always use `sslmode=require` for production external connections
2. **Firewall Rules**: Restrict DB host port 5432 to gateway host IP only
3. **Credential Management**: Store DATABASE_URL credentials securely (e.g., Vault, AWS Secrets Manager)
4. **Network Segmentation**: Place DB host in isolated network segment
5. **Audit Logging**: Enable PostgreSQL audit logging on external DB host

### Handoff Artifacts

The following artifacts must be included in customer handoff bundles:

- [ ] **Versioned Deployment Bundle**: `demo/logs/release-bundles/ai-control-plane-deploy-<version>.tar.gz`
  - Generated via: `make release-bundle`
  - Contains: Canonical deployment files, install manifest, checksums for integrity verification
  - **Build and Verify**:
    ```bash
    # Build bundle with default version (git short sha)
    make release-bundle
    
    # Build with explicit version
    make release-bundle RELEASE_BUNDLE_VERSION=v2026.02.11
    
    # Build to custom output directory
    make release-bundle RELEASE_BUNDLE_VERSION=v1.0.0 RELEASE_BUNDLE_OUT_DIR=./bundles
    ```
  - **Verify Bundle** (for integrity validation during handoff):
    ```bash
    # Verify specific bundle
    make release-bundle-verify RELEASE_BUNDLE_PATH=demo/logs/release-bundles/ai-control-plane-deploy-v2026.02.11.tar.gz
    
    # Secondary typed path
    ./scripts/acpctl.sh deploy release-bundle verify --bundle demo/logs/release-bundles/ai-control-plane-deploy-v2026.02.11.tar.gz
    ```
  - **Bundle Contents**:
    - `payload/` - Canonical deployment files (Makefile, docker-compose.yml, configs, docs)
    - `install-manifest.txt` - Sorted list of included files (bill-of-materials style)
    - `sha256sums.txt` - SHA-256 checksums for all payload files
  - **Sidecar Files**:
    - `ai-control-plane-deploy-<version>.tar.gz.sha256` - Tarball checksum for download verification
  - **Tamper Detection**: The verify command validates both tarball and payload checksums
  - **CI Enforcement**: The release bundle contract is validated by `make script-tests`

- [ ] **License Summary**: `docs/deployment/THIRD_PARTY_LICENSE_SUMMARY.md`
  - Generated via: `make license-report-update`
  - Contains: Third-party component inventory, compliance status, override records

- [ ] **Supply-Chain Evidence**: `demo/logs/supply-chain/`
  - Generated via: `make supply-chain-gate`
  - Expiry drift guard: `make supply-chain-allowlist-expiry-check`
  - Contains: policy summary + allowlist lifecycle evidence
  - **Default Runtime Image Source**: digest-pinned images declared in `demo/docker-compose.yml`
  - **Baseline**: gate outcome is evaluated from the current policy file and digest-pinned image set at execution time
  - **Archive Command**:
    ```bash
    make supply-chain-gate 2>&1 | tee "supply-chain-$(hostname)-$(date +%Y%m%d-%H%M%S).log"
    tar -czf "supply-chain-evidence-$(date +%Y%m%d).tar.gz" demo/logs/supply-chain/
    ```
  - **Evidence Files**:
    - `summary.json` - Aggregated security posture summary
    - *(Optional in downstream production pipelines)* SBOM/CVE/provenance outputs from dedicated scanners/signers
  - **Remediation Workflow**:
    1. Review findings: `jq '.policy_id, .allowlist_count, .status' demo/logs/supply-chain/summary.json`
    2. For accepted risks: Add to `demo/config/supply_chain_vulnerability_policy.json` allowlist with expiry
    3. For patches: Update image digests in compose/helm files, re-run gate
    4. Re-generate evidence after fixes

---

## Secrets and Key Lifecycle

### Master Key

The `LITELLM_MASTER_KEY` is the administrative key for the gateway.

**Rotation Procedure**:
1. Generate new master key
2. Update environment/secret with new key
3. Restart gateway services
4. **Important**: Existing virtual keys remain valid after master key rotation
5. Update any administrative tools/scripts with new key

**Storage**:
- Docker Compose: canonical `/etc/ai-control-plane/secrets.env` (source of truth) with `demo/.env` as synced runtime file
- Kubernetes: `kubectl create secret generic` or External Secrets Operator
- Never commit to version control

**Rotation Workflow (Docker Compose Host)**:
1. Update secrets in your provider/system (Vault, cloud secret manager, or host-managed file)
2. Refresh runtime env from canonical source:
   ```bash
   make host-secrets-refresh \
     SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env \
     HOST_COMPOSE_ENV_FILE=demo/.env \
     SECRETS_FETCH_HOOK=/usr/local/bin/fetch-acp-secrets.sh
   ```
3. Validate contract:
   ```bash
   make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
   ```
4. Restart service:
   ```bash
   sudo systemctl restart ai-control-plane
   ```

### Salt Key

The `LITELLM_SALT_KEY` is used for encrypting virtual keys in the database.

**CRITICAL INVARIANT**: The salt key **MUST NEVER CHANGE** after initial deployment. Changing the salt key will invalidate all existing virtual keys.

**Backup Requirement**: Store salt key in secure backup (password manager, vault) alongside database backups.

### Virtual Keys

Virtual keys are generated per-user/application and scoped to specific models.

**Generation**:
```bash
# Docker Compose
make key-gen ALIAS=app-name BUDGET=100.00

# Direct API call
curl -X POST http://localhost:4000/key/generate \
  -H "Authorization: Bearer $LITELLM_MASTER_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "key_alias": "app-name",
    "max_budget": 100.00,
    "models": ["claude-haiku-4-5"],
    "budget_duration": "30d"
  }'
```

**Rotation**:
1. Generate new virtual key
2. Update application configuration
3. Revoke old key after confirmation:
   ```bash
   make key-revoke ALIAS=<alias>
   ```

---

## Backup and Restore

### Docker Compose (Embedded PostgreSQL)

**Backup**:
```bash
make db-backup
# Creates: demo/backups/litellm-backup-YYYYMMDD-HHMMSS.sql.gz
```

**Restore**:
```bash
# List available backups
ls -la demo/backups/

# Restore specific backup
./scripts/acpctl.sh db restore demo/backups/litellm-backup-YYYYMMDD-HHMMSS.sql.gz

# Validate backup archive integrity before restore
gzip -t demo/backups/litellm-backup-YYYYMMDD-HHMMSS.sql.gz
```

**Automated Restore Drill (RTO/RPO Evidence)**:
```bash
# Run weekly restore drill with objective metrics
make dr-drill

# Evidence reports generated:
# - demo/logs/drills/restore-drill-<timestamp>.json
# - demo/logs/drills/restore-drill-<timestamp>.md
```

**Verification**:
```bash
make db-status
```

### Kubernetes/Helm (Embedded PostgreSQL)

**Backup**:
```bash
# Find the PostgreSQL pod
kubectl get pods -n <namespace> -l app.kubernetes.io/component=postgresql

# Create backup
kubectl exec -n <namespace> <postgres-pod> -- \
  pg_dump -U litellm litellm | gzip > backup_$(date +%Y%m%d_%H%M%S).sql.gz
```

**Restore**:
```bash
# Copy backup to pod
kubectl cp backup_YYYYMMDD_HHMMSS.sql.gz <namespace>/<postgres-pod>:/tmp/backup.sql.gz

# Restore
kubectl exec -n <namespace> <postgres-pod> -- \
  bash -c "gunzip < /tmp/backup.sql.gz | psql -U litellm litellm"
```

**Automated Restore Drill (RTO/RPO Evidence)**:
Run the staged restore drill using the Kubernetes procedure in [DISASTER_RECOVERY.md](DISASTER_RECOVERY.md). The public snapshot no longer ships a dedicated Helm DR Make wrapper.

**Safety Notes:**
- Drill uses temporary database only (never touches production tables)
- Blocks execution against 'prod' namespaces unless explicitly allowed
- Requires `--yes` confirmation (enforced by Makefile)

### External Database

For external PostgreSQL:

1. Use provider-specific backup tools (RDS snapshots, Cloud SQL exports, etc.)
2. Document restore procedures in customer-specific runbook
3. Test restore procedures quarterly

**Restore Drill for External Databases**:

For external databases, perform restore drills using provider-specific snapshot restore:

```bash
# 1. Create provider snapshot/backup
# 2. Restore to temporary instance (provider-specific)
# 3. Run connectivity and integrity checks
# 4. Document RTO/RPO achieved
# 5. Destroy temporary instance
```

**Restore Verification**:
1. Verify virtual keys are accessible:
   ```bash
   make db-status
   ```
2. Check budget records intact
3. Validate spend logs present

---

## Upgrade Procedures

### Docker Compose (In-Place Upgrade)

The host-first public command surface now performs in-place upgrades. The supported workflow is: validate, backup, converge the new checkout or config, then restart the systemd-managed service and re-run health checks.

**Upgrade Steps**:

1. **Backup**: Create a database backup before changing the running host.
   ```bash
   make db-backup
   ```

2. **Refresh checkout/config**: Pull the tracked repository state or apply the approved configuration change.

3. **Validate the production contract**:
   ```bash
   make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
   make host-preflight
   ```

4. **Refresh secrets and restart the service**:
   ```bash
   make host-secrets-refresh \
     SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env \
     HOST_COMPOSE_ENV_FILE=demo/.env
   make host-service-restart
   ```

5. **Verify runtime health**:
   ```bash
   make health
   make db-status
   make prod-smoke
   ```

**Rollback (if needed)**:

If issues are detected after restart, restore the last known-good repo/config and restart the service again. If data integrity is involved, restore the database backup first:

```bash
./scripts/acpctl.sh db restore demo/backups/litellm-backup-YYYYMMDD-HHMMSS.sql.gz
make host-service-restart
```

#### Go/No-Go Decision Matrix

| Check | Command | Go Criteria | No-Go Action |
|-------|---------|-------------|--------------|
| Pre-checks | `make validate-config-production` | Exit 0 | Fix config issues |
| Host readiness | `make host-preflight` | Exit 0 | Fix host/runtime contract issues |
| Runtime health | `make health` | Exit 0 | Inspect service logs before restart retry |
| Smoke tests | `make prod-smoke` | Exit 0 | Roll back repo/config and restart |

#### Evidence Capture Checklist

After any upgrade or rollback, capture the following:

1. **Service status**:
   ```bash
   make host-service-status
   ```

2. **Gateway health**:
   ```bash
   make health
   ```

3. **Database status**:
   ```bash
   make db-status
   ```

4. **Recent service logs**:
   ```bash
   journalctl -u ai-control-plane --since "1 hour ago" --no-pager
   ```

### Declarative Host Orchestrator

**Day-0 Bootstrap**:
1. **Prepare inventory**: Create `deploy/ansible/inventory/hosts.yml`
2. **Validate**: Run check mode to verify configuration
   ```bash
   make host-check INVENTORY=deploy/ansible/inventory/hosts.yml
   ```
3. **Apply**: Converge gateway host
   ```bash
   make host-apply INVENTORY=deploy/ansible/inventory/hosts.yml
   ```

**Day-2 Drift Correction**:
1. **Check**: Identify drift between desired and actual state
   ```bash
   make host-check INVENTORY=deploy/ansible/inventory/hosts.yml
   ```
2. **Apply**: Re-converge to desired state
   ```bash
   make host-apply INVENTORY=deploy/ansible/inventory/hosts.yml
   ```

**Upgrade Procedure**:
1. **Backup**: Create database backup
   ```bash
   # On gateway host
   make db-backup
   ```
2. **Update inventory**: Update repository reference if needed
3. **Validate**: Run check mode
   ```bash
   make host-check INVENTORY=deploy/ansible/inventory/hosts.yml
   ```
4. **Apply**: Converge with updated configuration
   ```bash
   make host-apply INVENTORY=deploy/ansible/inventory/hosts.yml
   ```

**Rollback**:
1. **Restore database**: On gateway host
   ```bash
   ./scripts/acpctl.sh db restore demo/backups/litellm-backup-YYYYMMDD-HHMMSS.sql.gz
   ```
2. **Re-apply pinned configuration**: If using version-pinned inventory
   ```bash
   make host-apply INVENTORY=deploy/ansible/inventory/hosts.yml
   ```

### Systemd Host Service Management

**Day-0 Bootstrap**:
1. **Install and enable service**: Install systemd service during handoff
   ```bash
   make host-install
   ```
2. **Verify boot persistence**: Confirm service is enabled for automatic start
   ```bash
   systemctl is-enabled ai-control-plane
   ```
   Expected output: `enabled`

**Operational Controls**:
1. **Check service status**:
   ```bash
   make host-service-status
   # Secondary direct control:
   systemctl status ai-control-plane
   ```
2. **Start service**:
   ```bash
   make host-service-start
   # Secondary direct control:
   sudo systemctl start ai-control-plane
   ```
3. **Stop service**:
   ```bash
   make host-service-stop
   # Secondary direct control:
   sudo systemctl stop ai-control-plane
   ```
4. **Restart service** (after config changes):
   ```bash
   make host-service-restart
   # Secondary direct control:
   sudo systemctl restart ai-control-plane
   ```
5. **View logs** (follow mode):
   ```bash
   journalctl -u ai-control-plane -f
   ```
6. **View recent logs** (last 100 lines):
   ```bash
   journalctl -u ai-control-plane -n 100
   ```

**Rollback/Uninstall Sequence**:
1. **Stop and disable service**:
   ```bash
   make host-uninstall
   ```
2. **Manual cleanup** (if needed):
   ```bash
   # Remove service file
   sudo rm -f /etc/systemd/system/ai-control-plane.service
   
   # Remove repository (preserves backups in demo/backups/)
   sudo rm -rf /opt/ai-control-plane
   
   # Reload systemd
   sudo systemctl daemon-reload
   ```

**Evidence Capture**:
1. **Service status output**:
   ```bash
   systemctl status ai-control-plane --no-pager > systemd-status-$(date +%Y%m%d-%H%M%S).log
   ```
2. **Unit file verification** (if systemd-analyze available):
   ```bash
   systemd-analyze verify /etc/systemd/system/ai-control-plane.service
   ```
3. **Journal logs for recent activity**:
   ```bash
   journalctl -u ai-control-plane --since "1 hour ago" --no-pager > systemd-journal-$(date +%Y%m%d-%H%M%S).log
   ```

**Boot Persistence Verification**:
1. **Verify service starts on boot**:
   ```bash
   systemctl is-enabled ai-control-plane
   ```
2. **Host service checks**: Secondary direct checks for systemd availability and service state:
   ```bash
   command -v systemctl
   systemctl is-enabled ai-control-plane
   systemctl is-active ai-control-plane
   ```
3. **Unit syntax validation** (if `systemd-analyze` is available):
   ```bash
   systemd-analyze verify /etc/systemd/system/ai-control-plane.service
   ```

### Kubernetes/Helm

**Upgrade Steps**:
1. **Backup**: Create database backup
   ```bash
   kubectl exec -n <namespace> <postgres-pod> -- \
     pg_dump -U litellm litellm | gzip > pre_upgrade_$(date +%Y%m%d_%H%M%S).sql.gz
   ```

2. **Validate**: Check Helm chart
   ```bash
   make helm-validate
   ```

3. **Upgrade**: Apply new chart version
   ```bash
   helm upgrade --install <release> ./deploy/helm/ai-control-plane \
     -n <namespace> \
     -f values.production.yaml
   ```

4. **Validate**: Run smoke tests
   ```bash
   make helm-smoke NAMESPACE=<namespace> RELEASE=<release>
   ```

5. **Rollback** (if needed):
   ```bash
   helm rollback <release> <revision> -n <namespace>
   
   # Or restore database and redeploy
   kubectl cp backup.sql.gz <namespace>/<postgres-pod>:/tmp/
   kubectl exec -n <namespace> <postgres-pod> -- \
     bash -c "gunzip < /tmp/backup.sql.gz | psql -U litellm litellm"
   ```

---

## Incident Response

### Compromised Virtual Key

**Detection**: 
- Anomalous usage patterns in logs
- Budget exhaustion alerts
- User reports

**Response**:
1. **Immediate**: Revoke the key
   ```bash
   make key-revoke ALIAS=<alias>
   ```
2. **Verify**: Check spend logs for unauthorized usage
   ```bash
   make db-status
   ```
3. **Notify**: Inform key owner/applications team
4. **Reissue**: Generate new key for legitimate use

### Compromised Master Key

**Severity**: HIGH - Administrative access compromised

**Response**:
1. **Immediate**: Rotate master key
   - Generate new master key
   - Update environment/secret
   - Restart gateway services
2. **Audit**: Review all recent key generations and spend logs
3. **Verify**: Check for unauthorized virtual keys
   ```bash
   make db-status
   ```
4. **Reissue**: Rotate all virtual keys as precaution

### OAuth Token Leakage

**Detection**:
- Review logs for Authorization headers
- Run verification:
  ```bash
  make tls-verify
  ```

**Response**:
1. **Identify**: Locate leaked tokens in logs
2. **Revoke**: Revoke OAuth tokens at provider
3. **Clear**: Truncate or rotate log files
4. **Verify**: Re-run verification check
5. **Review**: Check Caddy/LiteLLM config to prevent future logging

### Database Corruption/Failure

**Response**:
1. **Assess**: Determine extent of corruption
2. **Restore**: From most recent valid backup
   ```bash
   ./scripts/acpctl.sh db restore demo/backups/litellm-backup-YYYYMMDD-HHMMSS.sql.gz
   ```
3. **Verify**: 
   ```bash
   make health
   make db-status
   ```
4. **Root Cause**: Review logs for corruption cause

### Performance Degradation Triggers

Rollback indicators based on current runtime validation evidence:
- **Latency Spike**: p95 latency >2x the latest successful smoke-test baseline
- **Error Rate Increase**: Error rate materially higher than the latest successful smoke-test baseline
- **Health Regression**: `make health` or `make validate-detections` no longer returns success

Baseline sources:
- Latest smoke-test output from `make prod-smoke` / `make prod-smoke-local-tls`
- Latest readiness logs captured from `make ci-nightly`
- Latest verified release evidence from `make release-bundle`

---

## Observability Setup

### OpenTelemetry (OTEL) Collector

Production deployments require telemetry export to a centralized observability backend.

**Required Environment Variables**:
```bash
# Add to demo/.env or your secrets management system
OTEL_EXPORTER_OTLP_ENDPOINT=https://your-otel-backend.example.com
OTEL_EXPORTER_OTLP_AUTH_HEADER="Api-Key your-api-key"
OTEL_RESOURCE_ENVIRONMENT=production
OTEL_RESOURCE_DEPLOYMENT=us-east-1
OTEL_TRACES_SAMPLING_PERCENT=10  # Adjust based on volume/cost
```

**Starting with Production Profile**:
```bash
# Validates config and starts with OTEL collector
make up-production
```

**Verification**:
```bash
# Check OTEL collector health
make otel-health

# Verify telemetry export (check your observability backend)
# Most backends show incoming telemetry within 1-2 minutes
```

**Cost Controls**:
- Configure `OTEL_TRACES_SAMPLING_PERCENT` based on expected volume
- Default 10% sampling for medium-volume deployments
- High-volume: 1% sampling
- Critical paths: 100% sampling with targeted filtering

### Kubernetes Alerting (Helm)

The Helm chart includes baseline alerts for production monitoring:

| Alert | Severity | Description |
|-------|----------|-------------|
| ACPGatewayUnavailable | critical | No gateway replicas available |
| ACPGatewayHighErrorRate | warning | Error rate > 10% for 5 minutes |
| ACPAuthFailuresHigh | warning | Auth failures > 1/sec |
| ACPDetectionErrorsHigh | warning | Detection/guardrail errors |
| ACPBackupStale | critical | No backup in 25+ hours |

**Configure Alert Thresholds** (values.yaml):
```yaml
monitoring:
  alerts:
    authFailureRateThreshold: 1
    authFailureWindow: 5m
    detectionErrorWindow: 10m
    gatewayLatencyThreshold: 5
```

## Validation Steps

### Post-Deployment Validation

After any deployment or upgrade, run these validation steps:

**1. Production Gate (Required)**:
```bash
# Enterprise production readiness gate - runs all production invariants
make ci-nightly SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env

# With optional Kubernetes profile checks
CI_PRODUCTION_K8S=1 make ci-nightly \
  SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env \
  NAMESPACE=acp RELEASE=acp
```

**Evidence Capture**: Archive the gate output for compliance:
```bash
make ci-nightly SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env 2>&1 | \
  tee "ci-nightly-$(hostname)-$(date +%Y%m%d-%H%M%S).log"
```

**2. Supply-Chain Validation** (also covered by ci-nightly):
```bash
# Validate policy contract + digest pinning + allowlist expiry windows
make supply-chain-gate

# Check allowlist expiry windows (warn/fail policy)
make supply-chain-allowlist-expiry-check

# Review security summary
jq '.policy_id, .allowlist_count, .status' demo/logs/supply-chain/summary.json
```

Current accepted exception scope is defined by `demo/config/supply_chain_vulnerability_policy.json` and should be reviewed before each production handoff.

**Evidence Capture**: Archive supply-chain evidence:
```bash
# Generate and archive evidence bundle
tar -czf "supply-chain-evidence-$(date +%Y%m%d).tar.gz" demo/logs/supply-chain/
```

**3. Configuration Validation** (also covered by ci-nightly):
```bash
make validate-config-production
```

**4. Observability Validation**:
```bash
# Verify OTEL collector health
make otel-health

# Check telemetry is being received (backend-specific)
# Example: Datadog - check APM > Services > ai-control-plane
```

**5. Smoke Tests**:

For Docker Compose with TLS:
```bash
make prod-smoke-local-tls
```

For existing deployment:
```bash
export LITELLM_MASTER_KEY=your-master-key
make prod-smoke PUBLIC_URL=https://gateway.example.com
```

For Helm deployment:
```bash
export LITELLM_MASTER_KEY=your-master-key
make helm-smoke NAMESPACE=acp RELEASE=acp
```

**6. Health Check**:
```bash
make health
```

**7. Database Verification**:
```bash
make db-status
```

### Smoke Test Reference

The production smoke tests validate:

| Check | Description | Failure Indication |
|-------|-------------|-------------------|
| Health endpoint | Gateway reachable | Network/DNS issues |
| Auth enforcement | No anonymous access | Security misconfiguration |
| Models configured | At least one model available | Config issue |
| Virtual key generation | Admin API working | Auth/database issue |
| Virtual key validation | Generated key works | Key scope/permission issue |
| Request path (mock) | Full request cycle | Upstream connectivity |

---

## References

- [Single-Tenant Production Contract](./SINGLE_TENANT_PRODUCTION_CONTRACT.md) - Configuration invariants
- [Kubernetes/Helm Guide](./KUBERNETES_HELM.md) - Helm deployment details
- [TLS Setup Guide](./TLS_SETUP.md) - HTTPS/TLS configuration
- [Database Guide](../DATABASE.md) - PostgreSQL operations
- [Demo Runbook](../RUNBOOK.md) - Demo environment procedures

---

## Emergency Contacts

Document customer-specific emergency contacts here:

- **Primary On-Call**: 
- **Security Team**: 
- **Vendor Support**:
  - LiteLLM: 
  - Cloud Provider:

---

*This runbook should be customized with customer-specific details before handoff.*
