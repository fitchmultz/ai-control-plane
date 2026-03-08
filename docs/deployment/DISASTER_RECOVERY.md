# AI Control Plane - Disaster Recovery Guide

## Table of Contents

1. [Overview](#overview)
2. [RTO/RPO Targets](#rtorpo-targets)
3. [Backup Architecture](#backup-architecture)
   - [Embedded PostgreSQL](#embedded-postgresql-helm-with-postgresenabled-true)
   - [External PostgreSQL](#external-postgresql-aws-rdsazure-postgresql)
4. [Backup Configuration](#backup-configuration)
   - [Enabling Automated Backups](#enabling-automated-backups-helm)
   - [Cross-Region Replication](#cross-region-replication-aws)
5. [Restore Procedures](#restore-procedures)
   - [Scenario 1: Database Corruption](#scenario-1-database-corruption-embedded-postgresql)
   - [Scenario 2: Complete Cluster Loss](#scenario-2-complete-cluster-loss-dr-region-failover)
   - [Scenario 3: Point-in-Time Recovery](#scenario-3-point-in-time-recovery-external-database)
6. [DR Testing Procedures](#dr-testing-procedures)
7. [Monitoring and Alerting](#monitoring-and-alerting)
8. [Troubleshooting](#troubleshooting)
9. [References](#references)

## Overview

This document provides comprehensive disaster recovery (DR) procedures for the AI Control Plane, including backup/restore operations, RTO/RPO targets, and DR testing procedures.

## RTO/RPO Targets

| Environment | RTO (Recovery Time Objective) | RPO (Recovery Point Objective) |
|-------------|------------------------------|--------------------------------|
| Production  | 1 hour                       | 1 hour (with hourly backups)   |
| Staging     | 4 hours                      | 24 hours (with daily backups)  |
| Demo        | N/A - rebuild from scratch   | N/A                            |

## Backup Architecture

### Embedded PostgreSQL (Helm with postgres.enabled: true)

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                        │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              PostgreSQL StatefulSet                    │
│  │                   (Primary)                            │
│  └─────────────────────────┬─────────────────────────────┘  │
│                            │                                 │
│  ┌─────────────────────────▼─────────────────────────────┐  │
│  │              Backup CronJob (Hourly)                   │
│  │         ┌──────────────────────────┐                  │  │
│  │         │  pg_dump → gzip → PVC    │                  │  │
│  │         │  OR                      │                  │  │
│  │         │  pg_dump → gzip → S3     │                  │  │
│  │         └──────────────────────────┘                  │  │
│  └───────────────────────────────────────────────────────┘  │
│                            │                                 │
│  ┌─────────────────────────▼─────────────────────────────┐  │
│  │              Verification CronJob (Daily)              │
│  │         ┌──────────────────────────┐                  │  │
│  │         │  Checksum verification   │                  │  │
│  │         │  Gzip integrity test     │                  │  │
│  │         └──────────────────────────┘                  │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### External PostgreSQL (AWS RDS/Azure PostgreSQL)

Managed database services provide their own backup mechanisms:
- **AWS RDS**: Automated backups with configurable retention (7-35 days)
- **Azure PostgreSQL**: Automated backups with geo-redundancy option
- **GCP Cloud SQL**: Automated backups with point-in-time recovery

## Backup Configuration

### Enabling Automated Backups (Helm)

```yaml
# values-production.yaml
backup:
  enabled: true
  schedule: "0 * * * *"  # Hourly backups

  retention:
    days: 7
    maxCount: 24
    staleAlertThreshold: 7200  # Alert if no backup in 2 hours

  persistence:
    enabled: true
    size: 50Gi
    storageClass: gp2  # AWS EBS

  verification:
    enabled: true
    schedule: "0 1 * * *"  # Daily at 1 AM

  monitoring:
    enabled: true
    prometheusRule:
      enabled: true
      labels:
        release: prometheus
```

### Cross-Region Replication (AWS)

```yaml
# Additional Terraform configuration
backup_replication_enabled = true
backup_retention_days      = 90
```

## Restore Procedures

### Scenario 1: Database Corruption (Embedded PostgreSQL)

**Impact**: Single instance database corruption
**RTO**: 30 minutes
**RPO**: Up to 1 hour (depending on backup frequency)

```bash
# 1. Identify the last good backup
kubectl get jobs -n acp -l app.kubernetes.io/component=backup

# 2. Stop the gateway to prevent writes
kubectl scale deployment acp-ai-control-plane-litellm --replicas=0 -n acp

# 3. Get the backup from PVC or S3
kubectl cp acp/acp-ai-control-plane-postgres-0:/backups/litellm-backup-20240207-020000.sql.gz /tmp/restore.sql.gz

# 4. Restore the database
kubectl exec -n acp acp-ai-control-plane-postgres-0 -- bash -c "
  # Terminate connections
  psql -U litellm -d postgres -c \"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'litellm';\"

  # Drop and recreate database
  dropdb -U litellm litellm
  createdb -U litellm litellm

  # Restore from backup
  gunzip < /backups/litellm-backup-20240207-020000.sql.gz | psql -U litellm litellm
"

# 5. Restart the gateway
kubectl scale deployment acp-ai-control-plane-litellm --replicas=2 -n acp

# 6. Verify restore
kubectl exec -n acp deployment/acp-ai-control-plane-litellm -- \
  curl -s http://localhost:4000/health
```

### Scenario 2: Complete Cluster Loss (DR Region Failover)

**Impact**: Entire Kubernetes cluster unavailable
**RTO**: 1 hour
**RPO**: 1 hour

```bash
# 1. Deploy to DR region using Terraform
cd deploy/terraform/examples/aws-complete
terraform workspace select dr
terraform apply

# 2. Restore database from S3 backup (if using external RDS with cross-region replica)
# OR restore to new RDS instance from snapshot

# 3. Deploy Helm chart with restored database
helm upgrade --install acp ./deploy/helm/ai-control-plane -n acp \
  -f ./deploy/helm/ai-control-plane/examples/values.production.yaml

# 4. Verify deployment
make helm-smoke NAMESPACE=acp RELEASE=acp
```

### Scenario 3: Point-in-Time Recovery (External Database)

For managed databases (RDS, Azure PostgreSQL):

**AWS RDS:**
```bash
# Restore to point in time (PITR)
aws rds restore-db-instance-to-point-in-time \
  --source-db-instance-identifier ai-control-plane-production \
  --target-db-instance-identifier ai-control-plane-restored \
  --restore-time 2024-02-07T02:00:00Z

# Update Kubernetes secret with new endpoint
kubectl patch secret ai-control-plane-secrets -n acp \
  --type='json' \
  -p='[{"op": "replace", "path": "/data/DATABASE_URL", "value":"'$(echo -n 'postgresql://...' | base64)'"}]'

# Restart gateway
kubectl rollout restart deployment/acp-ai-control-plane-litellm -n acp
```

## DR Testing Procedures

### Automated Restore Drills with RTO/RPO Evidence

Restore drills validate that backups are not just intact but actually restorable, and measure objective recovery metrics against defined thresholds.

#### Docker Compose Mode

**Weekly Drill (Recommended):**
```bash
# Run drill using current default behavior
make dr-drill

# Restore a specific backup file when needed (typed CLI path)
./scripts/acpctl.sh db restore demo/backups/litellm-backup-20240128-020000.sql.gz
```

**Evidence Output:**
- JSON report: `demo/logs/drills/restore-drill-<timestamp>.json`
- Markdown report: `demo/logs/drills/restore-drill-<timestamp>.md`

**Report Contents:**
- Drill ID and timestamps
- Backup metadata (age, size, integrity)
- Measured RTO (restore duration in seconds)
- Measured RPO (backup age in seconds)
- Pass/fail status against thresholds
- Required table verification results

#### Kubernetes/Helm Mode

**Monthly Drill in Staging/DR Namespace (Recommended):**

Run the staged restore drill directly with the Kubernetes workflow documented in this runbook. The public snapshot no longer ships a dedicated Helm DR Make wrapper.

Threshold tuning flags from older private iterations are not part of the current public-snapshot target surface.

**Safety Guardrails:**
- Requires `--yes` confirmation (enforced by Makefile)
- Blocks execution against namespaces containing 'prod' unless `--allow-production`
- Restores to temporary database only, never touches production data
- Automatic cleanup of temporary resources

### Chaos Engineering Tests

#### Test 1: Backup Integrity Verification

**File-level verification (existing CronJob):**
```bash
# Run weekly to ensure backup files are valid
kubectl create job --from=cronjob/acp-ai-control-plane-backup-verify backup-test-$(date +%s) -n acp

# Check results
kubectl logs -n acp job/backup-test-...
```

**Note:** This only verifies file integrity (checksums, gzip). For full restore validation with RTO/RPO measurement, use the restore drill commands above.

#### Test 2: Simulated Database Corruption

**WARNING: Only run in non-production environments**

```bash
# 1. Create a snapshot first
kubectl exec -n acp acp-ai-control-plane-postgres-0 -- \
  pg_dump -U litellm litellm | gzip > /tmp/pre-chaos-backup.sql.gz

# 2. Simulate corruption (drop a table)
kubectl exec -n acp acp-ai-control-plane-postgres-0 -- \
  psql -U litellm -c "DROP TABLE IF EXISTS test_corruption;"

# 3. Execute DR procedure
# (Follow Scenario 1 steps above)

# 4. Verify data integrity
kubectl exec -n acp acp-ai-control-plane-postgres-0 -- \
  psql -U litellm -c "SELECT COUNT(*) FROM LiteLLM_VerificationToken;"
```

#### Test 3: Complete Environment Rebuild

**Quarterly DR Drill:**

1. Run automated restore drill for baseline metrics
2. Document the current state
3. Destroy and recreate the environment
4. Restore from latest backup
5. Verify all functionality
6. Document actual RTO achieved and compare with drill results

```bash
# Step 1: Run automated drill for baseline
make dr-drill

# Review the generated report
cat demo/logs/drills/restore-drill-*.md

# Manual quarterly DR drill steps:
# 2. Document current state and record start time
# 3. Destroy the environment: terraform destroy (staging only!)
# 4. Recreate infrastructure: terraform apply
# 5. Restore from latest backup (see Scenario 1 or 3)
# 6. Verify all functionality with: make helm-smoke
# 7. Calculate actual RTO and document lessons learned
```

## Monitoring and Alerting

### Prometheus Alerts

| Alert | Severity | Description | Action |
|-------|----------|-------------|--------|
| ACPBackupJobFailed | warning | Backup job failed | Check job logs, retry manually |
| ACPBackupStale | critical | No backup in 25+ hours | Investigate CronJob, check PVC |
| ACPBackupVerificationFailed | critical | Backup file corrupted | Create new backup immediately, investigate root cause |
| ACPBackupDiskFull | warning | Backup storage > 90% | Increase retention cleanup or expand storage |

### Metrics

```
# Backup verification metrics
acp_backup_verification_total      # Total backups
acp_backup_verification_verified   # Verified backups
acp_backup_verification_failed     # Failed verifications
acp_backup_verification_timestamp  # Last verification time
```

## Troubleshooting

### Backup Job Failing

```bash
# Check job logs
kubectl logs -n acp job/acp-ai-control-plane-backup-...

# Common issues:
# 1. PostgreSQL not accessible - check network policies
# 2. PVC full - expand or clean up old backups
# 3. Credentials incorrect - verify secrets
```

### Slow Backups

- Increase backup job resources (CPU/memory)
- Use streaming compression with lower compression ratio
- Consider incremental backups (not implemented, requires WAL archiving)

### Large Backup Storage

- Reduce retention days
- Enable object storage (S3) with lifecycle policies
- Compress older backups to Glacier/Archive storage

## References

- [Production Handoff Runbook](./PRODUCTION_HANDOFF_RUNBOOK.md)
- [Kubernetes Helm Guide](./KUBERNETES_HELM.md)
- [Single-Tenant Production Contract](./SINGLE_TENANT_PRODUCTION_CONTRACT.md)
- PostgreSQL Documentation: https://www.postgresql.org/docs/16/backup.html
