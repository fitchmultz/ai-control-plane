<!--
Purpose: Document AWS hardening guidance for the incubating AWS Terraform path.
Responsibilities:
  - Map existing IaC guardrails to operator actions.
  - State what the provided IaC does and does not cover.
  - Preserve truthful cloud-boundary claims.
Scope:
  - deploy/incubating/terraform/examples/aws-complete and referenced AWS modules.
Usage:
  - Review before using the AWS example or discussing AWS positioning externally.
Invariants/Assumptions:
  - Guidance is for the provided IaC, not a complete cloud security program.
  - AWS only; Azure/GCP are out of scope here.
-->

# AWS Cloud Hardening Guidance

This document provides hardening guidance for the **AWS-first incubating Terraform path** in this repository.

## Boundary

This is guidance for the provided AWS Terraform and Helm assets. It is **not** a complete cloud security program, compliance package, or managed-operations guarantee.

## Hardening checklist mapped to repo evidence

| Control area | Repo evidence | Required operator action | Covered by repo? |
| --- | --- | --- | --- |
| Private EKS API by default | `examples/aws-complete/variables.tf` defaults `cluster_endpoint_public_access=false`, `cluster_endpoint_private_access=true` | Keep the API private unless there is a named admin access requirement | Yes |
| No wildcard public API access | `cluster_public_access_cidrs` validation forbids `0.0.0.0/0`; `terraform_data.deployment_guardrails` requires explicit CIDRs if public access is enabled | If public API access is required, allowlist only explicit admin CIDRs | Yes |
| TLS required for ingress | `terraform_data.deployment_guardrails` requires `alb_certificate_arn` when `enable_ingress=true` | Do not expose ingress without ACM-backed TLS | Yes |
| Private worker/data plane placement | VPC module creates private subnets; EKS and RDS consume private subnet IDs | Keep nodes and database in private subnets; use NAT only for outbound egress | Yes |
| RDS not public | `modules/aws/rds/main.tf` sets `publicly_accessible = false` | Do not override this behavior | Yes |
| RDS deletion protection | Guardrails require `rds_deletion_protection=true` | Keep enabled in all serious environments | Yes |
| Final snapshot retention | Guardrails require `rds_skip_final_snapshot=false` | Keep final snapshot creation enabled | Yes |
| Storage encryption | RDS sets `storage_encrypted=true`; EKS module creates KMS key for cluster encryption; S3 backups use SSE | Keep encryption enabled and review customer-managed KMS policy where required | Yes |
| Least-privilege pod IAM | `irsa_policy_statements` default to empty; IRSA annotation appears only when explicitly needed | Add only narrowly scoped actions/resources when workload AWS API access is required | Yes |
| Production Helm contract | `values.schema.json` requires external secrets, no embedded Postgres, min replicas, PDB, and network policy | Use the production chart profile only for this AWS path | Yes |
| Backups | RDS backups enabled; optional S3 backup bucket uses versioning, public-access block, and lifecycle rules | Review retention, lifecycle, and restore ownership with the customer | Partially |

## Network isolation requirements

### Required baseline

- Keep **EKS nodes in private subnets**
- Keep **RDS in private subnets**
- Use **NAT gateways for outbound-only egress**
- Keep **cluster endpoint private-only** unless a named admin exception exists
- Keep **public ingress disabled by default**
- If ingress is enabled:
  - require ACM certificate
  - require DNS ownership
  - prefer internal ALB unless external access is explicitly justified

### Additional customer-owned controls not provided here

The provided IaC does **not** create or validate:

- AWS WAF configuration
- Shield Advanced / DDoS response operations
- VPC endpoints / PrivateLink strategy
- centralized egress proxying
- Transit Gateway / multi-VPC segmentation
- organization-wide SCP guardrails

## IAM least-privilege guidance

### IRSA baseline

The AWS example creates IRSA trust wiring, but **gives no workload AWS API permissions by default**.

That is the correct baseline.

### Policy design rules

When adding `irsa_policy_statements`:

- scope by exact action, not `*`
- scope by exact ARN where possible
- prefer a single service account per permission boundary
- document why the workload needs AWS API access at all
- deny broad S3, Secrets Manager, KMS, or CloudWatch wildcards unless a customer-approved exception exists

### Example: narrow S3 write policy

```hcl
irsa_policy_statements = [
  {
    effect    = "Allow"
    actions   = ["s3:PutObject", "s3:GetObject", "s3:ListBucket"]
    resources = [
      "arn:aws:s3:::ai-control-plane-prod-backups",
      "arn:aws:s3:::ai-control-plane-prod-backups/*",
    ]
  }
]
```

## Secrets management

- Do **not** commit secrets into Terraform vars or Git-tracked files
- Use external secret sources for real deployments:
  - AWS Secrets Manager
  - external secret workflows
  - customer-managed secret distribution
- The Helm production contract already expects existing secrets rather than demo literals
- Treat Terraform state as sensitive because it can contain secret values and connection details

## Encryption at rest and in transit

### At rest

- EKS secrets encryption: KMS-backed support exists in the EKS module
- RDS storage encryption: enabled
- S3 backup bucket encryption: enabled
- RDS snapshot tags copied forward: enabled

### In transit

- Public ingress requires TLS certificate wiring
- Database connection string uses `sslmode=require`
- Keep any future admin/API exposure behind TLS and named access controls

## Logging and monitoring

Repo evidence already enables:

- EKS control plane logs:
  - `api`
  - `audit`
  - `authenticator`
  - `controllerManager`
  - `scheduler`
- RDS log exports:
  - `postgresql`
  - `upgrade`
- optional RDS Performance Insights
- Helm-side ServiceMonitor support for production

Operator follow-through still required:

- define CloudWatch retention
- export logs to the customer SIEM if required
- define alert ownership and escalation
- validate EKS audit visibility in the customer account

## Backup and recovery

### Provided by IaC

- RDS automated backups with retention period
- final snapshot protection on destroy
- optional S3 backup bucket with:
  - versioning
  - SSE
  - public access block
  - lifecycle transitions to IA and Glacier

### Not fully covered here

- restore drill automation
- cross-account backup isolation
- multi-region disaster recovery orchestration
- customer-approved RPO/RTO validation
- application-level runtime smoke tests after restore

## Explicitly not covered by this guide

The following items remain customer-owned or out of scope for the provided IaC:

- WAF policy design and managed-rule tuning
- DDoS protection program ownership
- GuardDuty/Security Hub/Macie rollout
- node OS hardening beyond managed service defaults
- admission controller policy suites
- image signing / runtime workload attestation
- VPC endpoint strategy
- org-wide IAM governance and SCPs
- continuous cloud compliance monitoring

## Recommended usage statement

Use this guide exactly as follows:

> The repository contains an AWS-first validated Terraform path with explicit internal IaC checks and hardening guidance. It does not claim a complete cloud security program or generalized multi-cloud production support.
