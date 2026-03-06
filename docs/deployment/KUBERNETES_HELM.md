# Kubernetes (Helm) Deployment Guide

This guide provides comprehensive instructions for deploying the AI Control Plane on Kubernetes using Helm.

> This is an optional secondary deployment track for teams with existing Kubernetes operations maturity.
> If you are deploying to Linux hosts without a Kubernetes platform requirement, start with `../DEPLOYMENT.md`.

## Table of Contents

1. [Overview](#1-overview)
2. [Prerequisites](#2-prerequisites)
3. [Quick Start](#3-quick-start)
4. [Installation](#4-installation)
5. [Configuration](#5-configuration)
6. [TLS Setup](#6-tls-setup)
7. [External Database](#7-external-database)
8. [Operations](#8-operations)
9. [Troubleshooting](#9-troubleshooting)

---

## 1. Overview

The AI Control Plane Helm chart deploys:

- **LiteLLM Gateway**: Central API gateway for AI model access
- **PostgreSQL** (optional): Embedded database for quick starts
- **Mock Upstream** (optional): For offline demos/testing

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Kubernetes Cluster                    │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                    Ingress (TLS)                       │  │
│  │              (nginx, traefik, etc.)                    │  │
│  └─────────────────────────┬─────────────────────────────┘  │
│                            │                                 │
│  ┌─────────────────────────▼─────────────────────────────┐  │
│  │              LiteLLM Gateway Service                   │  │
│  │                   (ClusterIP)                          │  │
│  └─────────────────────────┬─────────────────────────────┘  │
│                            │                                 │
│  ┌─────────────────────────▼─────────────────────────────┐  │
│  │              LiteLLM Deployment                        │  │
│  │              (Replicas: configurable)                  │  │
│  └─────────────────────────┬─────────────────────────────┘  │
│                            │                                 │
│  ┌─────────────────────────▼─────────────────────────────┐  │
│  │              PostgreSQL StatefulSet                    │  │
│  │         (Optional: use external for production)        │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Prerequisites

### Cluster Requirements

| Component | Minimum Version | Notes |
|-----------|-----------------|-------|
| Kubernetes | 1.25+ | Tested on 1.25-1.29 |
| Helm | 3.12+ | Required for chart installation |

### Optional Components

| Component | Purpose | When Needed |
|-----------|---------|-------------|
| Ingress Controller | External HTTPS access | Production deployments |
| cert-manager | Automatic TLS certificates | Production with Let's Encrypt |
| External Secrets Operator | Secrets management | Production (recommended) |
| Prometheus Operator | Monitoring | Optional observability |

### Resource Requirements

| Profile | CPU | Memory | Storage |
|---------|-----|--------|---------|
| Demo (embedded Postgres) | 500m | 512Mi | 5Gi |
| Production (external DB) | 1000m | 1Gi | - |
| Production (HA) | 2000m | 2Gi | - |

### Validating Profile Readiness Before Kubernetes Cutover

> **Recommendation:** Run the canonical CI tiers on your target host before production Helm cutover.

Use this validation sequence before final Helm sizing decisions:

```bash
# Runtime + release-evidence gate
make ci-nightly

# On-demand heavy security/image checks
make ci-manual-heavy

# Optional runtime smoke against your planned public URL
make prod-smoke PUBLIC_URL=https://gateway.example.com
```

**Apply results to Helm values:** use runtime and smoke evidence to tune resource `requests`, `limits`, and replica counts.

| Validation Outcome | Helm Resource Adjustment |
|-------------------|-------------------------|
| `ci-nightly` + smoke PASS | Use default profile values |
| Smoke latency regressions | Increase CPU/memory requests; verify node sizing |
| Runtime instability/errors | Investigate networking/config before scaling |
| Heavy gate failures | Resolve hardened-image/supply-chain issues before cutover |

```yaml
# Example: Adjusted values based on smoke/validation regressions
litellm:
  resources:
    requests:
      cpu: 1500m      # Increased from 1000m due to latency failures
      memory: 1Gi
    limits:
      cpu: 3000m      # Increased from 2000m
      memory: 2Gi

  # If throughput failed, consider horizontal scaling
  replicaCount: 3     # Increased from 2
```

---

## 3. Quick Start

### Demo Installation (Quick Start)

```bash
# Add the Helm repository (or use local chart)
# helm repo add acp https://<your-github-org>.github.io/ai-control-plane

# Install with defaults (embedded PostgreSQL)
helm install acp ./deploy/helm/ai-control-plane -n acp --create-namespace

# Port-forward to access locally
kubectl port-forward -n acp svc/acp-ai-control-plane-litellm 4000:4000

# Access the UI
curl http://localhost:4000/health
```

### Production Installation

```bash
# Create namespace
kubectl create namespace acp

# Create secrets (see Secrets section below)
kubectl create secret generic ai-control-plane-secrets \
  --from-literal=LITELLM_MASTER_KEY='your-secure-master-key-32-chars-min' \
  --from-literal=LITELLM_SALT_KEY='your-secure-salt-key-32-chars-min' \
  --from-literal=DATABASE_URL='postgresql://user:pass@postgres-host:5432/litellm' \
  -n acp

# Install with production values
helm upgrade --install acp ./deploy/helm/ai-control-plane -n acp \
  -f ./deploy/helm/ai-control-plane/examples/values.production.yaml
```

---

## 4. Installation

### Install from Local Chart

```bash
# Clone the repository
git clone <repository-url>
cd ai-control-plane

# Install
helm upgrade --install acp ./deploy/helm/ai-control-plane -n acp --create-namespace
```

### Upgrade

```bash
# Upgrade to new version
helm upgrade acp ./deploy/helm/ai-control-plane -n acp

# Upgrade with new values
helm upgrade acp ./deploy/helm/ai-control-plane -n acp -f my-values.yaml
```

### Rollback

```bash
# List releases
helm history acp -n acp

# Rollback to previous version
helm rollback acp 1 -n acp
```

### Uninstall

```bash
# Uninstall (preserves PVCs by default)
helm uninstall acp -n acp

# Delete namespace and all resources
kubectl delete namespace acp
```

---

## 5. Configuration

### Configuration Methods

1. **values.yaml**: Default configuration (don't modify directly)
2. **Custom values file**: `-f my-values.yaml`
3. **Command line**: `--set key=value`
4. **Existing secrets**: Reference external secrets

### Secrets Configuration

**Option 1: Chart creates secrets (dev/demo only)**

```yaml
secrets:
  create: true
  litellm:
    masterKey: "your-secure-master-key"
    saltKey: "your-secure-salt-key"
```

**Option 2: Reference existing secrets (production)**

```bash
# Create secret manually
kubectl create secret generic ai-control-plane-secrets \
  --from-literal=LITELLM_MASTER_KEY='...' \
  --from-literal=LITELLM_SALT_KEY='...' \
  --from-literal=DATABASE_URL='...' \
  -n acp
```

```yaml
secrets:
  create: false
  existingSecret:
    name: ai-control-plane-secrets
    masterKeyKey: LITELLM_MASTER_KEY
    saltKeyKey: LITELLM_SALT_KEY
    databaseUrlKey: DATABASE_URL
```

**Option 3: External Secrets Operator (production recommended)**

```yaml
# Install ESO: https://external-secrets.io/latest/
# Create ExternalSecret resource referencing your secret store
```

### Profile Selection

```yaml
# Demo profile - allows placeholders, embedded postgres ok
profile: demo

# Production profile - enforces strong secrets
profile: production
```

---

## 6. TLS Setup

### Option 1: cert-manager with Let's Encrypt (Recommended)

```yaml
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
  hosts:
    - host: ai-control-plane.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: ai-control-plane-tls
      hosts:
        - ai-control-plane.example.com
```

Prerequisites:
```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Create ClusterIssuer for Let's Encrypt
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

### Option 2: Pre-existing TLS Secret

```bash
# Create TLS secret from certificate files
kubectl create secret tls ai-control-plane-tls \
  --cert=path/to/cert.crt \
  --key=path/to/key.key \
  -n acp
```

```yaml
ingress:
  enabled: true
  className: nginx
  tls:
    - secretName: ai-control-plane-tls
      hosts:
        - ai-control-plane.example.com
```

### Option 3: LoadBalancer Service (Cloud Provider TLS)

```yaml
ingress:
  enabled: false

litellm:
  service:
    type: LoadBalancer
    # Add cloud provider annotations as needed
```

### Security Note: OAuth Token Safety

**IMPORTANT**: When using subscription mode (e.g., Claude Code with OAuth), OAuth tokens are forwarded to upstream providers. Ensure your Ingress controller is configured to NOT log Authorization headers:

```yaml
# nginx ingress example
nginx.ingress.kubernetes.io/configuration-snippet: |
  # Do not log Authorization header
  access_log /var/log/nginx/access.log combined if=$loggable;
```

---

## 7. External Database

For production, use an external PostgreSQL database rather than the embedded StatefulSet.

### Configuration

```yaml
# Disable embedded PostgreSQL
postgres:
  enabled: false

# Reference external database
externalDatabase:
  existingSecret: "ai-control-plane-secrets"
  existingSecretKey: "DATABASE_URL"
```

### Database URL Format

```
postgresql://username:password@hostname:5432/database
```

### Cloud Managed Databases

**AWS RDS:**
```bash
# Create secret with RDS endpoint
kubectl create secret generic ai-control-plane-secrets \
  --from-literal=DATABASE_URL='postgresql://litellm:password@my-rds.abc123.us-east-1.rds.amazonaws.com:5432/litellm' \
  -n acp
```

**Google Cloud SQL:**
```bash
# Use Cloud SQL Auth Proxy sidecar or private IP
kubectl create secret generic ai-control-plane-secrets \
  --from-literal=DATABASE_URL='postgresql://litellm:password@10.0.0.3:5432/litellm' \
  -n acp
```

**Azure Database for PostgreSQL:**
```bash
kubectl create secret generic ai-control-plane-secrets \
  --from-literal=DATABASE_URL='postgresql://litellm@my-server:password@my-server.postgres.database.azure.com:5432/litellm' \
  -n acp
```

---

## 8. Operations

### Health Checks

```bash
# Check pod status
kubectl get pods -n acp

# Check logs
kubectl logs -n acp -l app.kubernetes.io/component=litellm

# Check health endpoint (port-forward first)
kubectl port-forward -n acp svc/acp-ai-control-plane-litellm 4000:4000
curl http://localhost:4000/health
```

### Scaling

```bash
# Manual scaling
kubectl scale deployment acp-ai-control-plane-litellm --replicas=3 -n acp

# Or via Helm upgrade
helm upgrade acp ./deploy/helm/ai-control-plane -n acp --set litellm.replicaCount=3
```

### Backup and Restore

**Embedded PostgreSQL:**

```bash
# Backup
kubectl exec -n acp acp-ai-control-plane-postgres-0 -- \
  pg_dump -U litellm litellm | gzip > backup-$(date +%Y%m%d).sql.gz

# Restore
kubectl exec -n acp -i acp-ai-control-plane-postgres-0 -- \
  psql -U litellm litellm < backup-20240101.sql
```

**External Database:** Use provider-specific backup tools.

### Monitoring

Enable Prometheus ServiceMonitor:

```yaml
monitoring:
  serviceMonitor:
    enabled: true
    labels:
      release: prometheus
```

Optional: enable runbook URLs in alert annotations for your repo/docs host:

```yaml
monitoring:
  prometheusRule:
    enabled: true
  alerts:
    runbookBaseUrl: "https://github.com/<your-org>/ai-control-plane/blob/main"
```

Key metrics:
- `litellm_proxy_total_requests_metric_total` - Total gateway request count
- `litellm_proxy_failed_requests_metric_total` - Total failed gateway requests
- `litellm_request_total_latency_metric_bucket` - Request latency histogram buckets
- `litellm_guardrail_errors_total` - Guardrail/detection error count
- Container resource usage via cAdvisor

#### Advanced ServiceMonitor Configuration

For enterprise deployments requiring TLS, label manipulation, or metric filtering:

```yaml
monitoring:
  serviceMonitor:
    enabled: true
    labels:
      release: prometheus
    
    # Preserve conflicting labels from targets
    honorLabels: true
    
    # Add labels from pod to all scraped metrics
    targetLabels:
      - app.kubernetes.io/name
      - app.kubernetes.io/component
    
    # Filter metrics before ingestion (reduce storage)
    metricRelabelings:
      - sourceLabels: [__name__]
        regex: 'go_goroutines|go_gc_duration_seconds.*'
        action: drop
      - sourceLabels: [__name__]
        regex: 'litellm_.*'
        action: keep
    
    # Relabel targets before scraping
    relabelings:
      - sourceLabels: [__meta_kubernetes_pod_label_team]
        targetLabel: team
      - sourceLabels: [__meta_kubernetes_namespace]
        targetLabel: kubernetes_namespace
    
    # TLS configuration for mTLS environments
    tlsConfig:
      caFile: /etc/prometheus/secrets/ca.crt
      certFile: /etc/prometheus/secrets/client.crt
      keyFile: /etc/prometheus/secrets/client.key
      serverName: litellm.metrics.svc.cluster.local
      insecureSkipVerify: false
```

**Common Patterns:**

| Use Case | Configuration |
|----------|---------------|
| Drop high-cardinality metrics | `metricRelabelings` with `action: drop` |
| Preserve application labels | `honorLabels: true` |
| Add team/namespace context | `relabelings` with `targetLabel` |
| Secure scraping | `tlsConfig` with certificates |
| Multi-tenant Prometheus | `targetLabels` for tenant identification |

### Production Smoke Tests

Validate your Helm deployment using the production smoke test harness. This performs runtime validation of the deployment contract without requiring ingress to be configured.

**Prerequisites:**
- `kubectl` configured for the target cluster
- `LITELLM_MASTER_KEY` environment variable set

**Basic Usage:**
```bash
# Validate via port-forward (no ingress required)
export LITELLM_MASTER_KEY=your-master-key
make helm-smoke NAMESPACE=acp RELEASE=acp
```

**With Public URL:**
```bash
# Test both port-forward and public ingress
export LITELLM_MASTER_KEY=your-master-key
make helm-smoke NAMESPACE=acp RELEASE=acp \
  PUBLIC_URL=https://gateway.example.com
```

**What it validates:**
1. **Gateway reachability**: Health endpoint responds
2. **Auth enforcement**: No anonymous access to protected endpoints
3. **Models configured**: At least one model is available
4. **Virtual key generation**: Admin API works with master key
5. **Key validation**: Generated keys work on public endpoints
6. **Request path**: Full request cycle (when mock models configured)

**CI Integration:**
```bash
# Extended CI includes production smoke tests
make ci-nightly

# Optional Kubernetes production profile checks via enterprise gate
CI_PRODUCTION_K8S=1 make ci-nightly \
  SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env \
  NAMESPACE=acp RELEASE=acp
```

See [Production Handoff Runbook](./PRODUCTION_HANDOFF_RUNBOOK.md) for full operational procedures.

---

## 9. Troubleshooting

### Pod not starting

```bash
# Check events
kubectl describe pod -n acp -l app.kubernetes.io/component=litellm

# Check logs
kubectl logs -n acp -l app.kubernetes.io/component=litellm --previous
```

### Database connection issues

```bash
# Test connectivity from LiteLLM pod
kubectl exec -n acp deployment/acp-ai-control-plane-litellm -- \
  python -c "import psycopg2; conn = psycopg2.connect('$DATABASE_URL'); print('OK')"
```

### Ingress not working

```bash
# Check ingress status
kubectl get ingress -n acp

# Check ingress controller logs
kubectl logs -n ingress-nginx deployment/ingress-nginx-controller
```

### Secret issues

```bash
# Verify secrets exist
kubectl get secrets -n acp

# Verify secret content (base64 decoded)
kubectl get secret ai-control-plane-secrets -n acp -o json | \
  jq -r '.data.LITELLM_MASTER_KEY' | base64 -d
```

---

## References

- [Helm Chart Source](../../deploy/helm/ai-control-plane)
- [Production Contract](./SINGLE_TENANT_PRODUCTION_CONTRACT.md)
- [Main Deployment Guide](../DEPLOYMENT.md)
- [LiteLLM Documentation](https://docs.litellm.ai/)
