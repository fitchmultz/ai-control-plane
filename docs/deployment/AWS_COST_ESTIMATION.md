<!--
Purpose: Provide a basic cost-estimation model for the AWS-first incubating Terraform path.
Responsibilities:
  - Map IaC environment defaults to cost components.
  - Provide formulas and refresh guidance for monthly estimation.
  - Prevent overstated cost certainty.
Scope:
  - deploy/incubating/terraform/examples/aws-complete.
Usage:
  - Use before buyer discussions or internal AWS sizing reviews.
Invariants/Assumptions:
  - Estimates are based on list-price modeling, not billing truth.
  - Actual costs vary by region, purchase model, traffic, storage growth, and usage.
-->

# AWS Cost Estimation Model

This document provides a **basic** monthly cost-estimation model for the AWS Terraform example at `deploy/incubating/terraform/examples/aws-complete`.

## Disclaimer

These are **estimates**, not billing truth.

- Use **current AWS list prices** for the target region before external commitments.
- Actual cost varies by:
  - region
  - Savings Plans / Reserved Instances
  - NAT data processing
  - ALB LCUs
  - RDS storage growth and backup retention
  - traffic volume and log volume

## Model assumptions

- Region baseline: `us-east-1` unless replaced
- Month length: `730` hours
- Purchase model: on-demand list pricing unless replaced
- Environment defaults come from `examples/aws-complete/main.tf`

## Environment sizing from IaC

| Environment | EKS nodes | Node type | RDS class | Multi-AZ | NAT gateways |
| --- | --- | --- | --- | --- | --- |
| dev | 2 desired (1-3) | `t3.medium` | `db.t3.micro` | no | 1 |
| staging | 2 desired (2-5) | `t3.medium` | `db.t3.small` | yes | 3 |
| production | 3 desired (2-10) | `t3.large` | `db.t3.medium` | yes | 3 |

## Pricing inputs to refresh

Fill these from the AWS Pricing Calculator or current AWS list pricing for the chosen region.

| Symbol | Description |
| --- | --- |
| `EKS_CTRL` | EKS cluster control plane monthly price |
| `T3_MEDIUM_HR` | EC2 `t3.medium` hourly price |
| `T3_LARGE_HR` | EC2 `t3.large` hourly price |
| `RDS_T3_MICRO_HR` | RDS PostgreSQL `db.t3.micro` hourly price |
| `RDS_T3_SMALL_HR` | RDS PostgreSQL `db.t3.small` hourly price |
| `RDS_T3_MEDIUM_HR` | RDS PostgreSQL `db.t3.medium` hourly price |
| `NAT_HR` | NAT Gateway hourly price |
| `NAT_GB` | NAT Gateway data processing price per GB |
| `ALB_HR` | ALB hourly price |
| `ALB_LCU_HR` | ALB LCU hourly price |
| `RDS_GP3_GB` | RDS gp3 storage per GB-month |
| `S3_STD_GB` | S3 Standard per GB-month |
| `S3_IA_GB` | S3 Standard-IA per GB-month |
| `S3_GLACIER_GB` | S3 Glacier / archive tier per GB-month |

## Per-component monthly formulas

### EKS

| Component | Formula |
| --- | --- |
| Cluster control plane | `1 * EKS_CTRL` |
| Dev nodes | `2 * T3_MEDIUM_HR * 730` |
| Staging nodes | `2 * T3_MEDIUM_HR * 730` |
| Production nodes | `3 * T3_LARGE_HR * 730` |

### RDS

Use the environment default compute class plus storage and backup growth.

| Environment | Formula |
| --- | --- |
| Dev | `(RDS_T3_MICRO_HR * 730) + (20 * RDS_GP3_GB)` |
| Staging | `(2 * RDS_T3_SMALL_HR * 730) + (20 * RDS_GP3_GB)` |
| Production | `(2 * RDS_T3_MEDIUM_HR * 730) + (20 * RDS_GP3_GB)` |

> For this basic model, Multi-AZ is modeled as approximately double the single-instance compute footprint. Refresh with exact regional pricing if you need a tighter estimate.

### NAT gateway

| Environment | Formula |
| --- | --- |
| Dev | `(1 * NAT_HR * 730) + (monthly_nat_gb * NAT_GB)` |
| Staging | `(3 * NAT_HR * 730) + (monthly_nat_gb * NAT_GB)` |
| Production | `(3 * NAT_HR * 730) + (monthly_nat_gb * NAT_GB)` |

### S3 backup storage

`backup_replication_enabled` defaults to `true`, so include backup bucket storage in the model.

A simple blended monthly estimate:

```text
S3 backups = (standard_gb * S3_STD_GB) + (ia_gb * S3_IA_GB) + (archive_gb * S3_GLACIER_GB)
```

Start with a conservative estimate using the retention/lifecycle rules in `main.tf`:

- days 0-30: Standard
- days 31-90: Standard-IA
- days 91+: Glacier/archive tier
- expiration at `backup_retention_days`

### ALB (optional)

ALB cost applies only when `enable_ingress=true`.

```text
ALB monthly = (ALB_HR * 730) + (LCU_hours * ALB_LCU_HR)
```

For a low-traffic baseline, start with `LCU_hours = 730` and adjust upward with real request/throughput expectations.

## Monthly estimate summary table

Populate the rightmost column with actual numbers after refreshing the pricing inputs.

| Environment | Summary formula | Notes |
| --- | --- | --- |
| Dev | `EKS_CTRL + (2*T3_MEDIUM_HR*730) + (RDS_T3_MICRO_HR*730) + (20*RDS_GP3_GB) + (1*NAT_HR*730) + NAT data + S3 backups + optional ALB` | Cheapest path; single NAT and single-AZ RDS |
| Staging | `EKS_CTRL + (2*T3_MEDIUM_HR*730) + (2*RDS_T3_SMALL_HR*730) + (20*RDS_GP3_GB) + (3*NAT_HR*730) + NAT data + S3 backups + optional ALB` | Multi-AZ RDS and multi-NAT increase cost sharply |
| Production | `EKS_CTRL + (3*T3_LARGE_HR*730) + (2*RDS_T3_MEDIUM_HR*730) + (20*RDS_GP3_GB) + (3*NAT_HR*730) + NAT data + S3 backups + optional ALB` | Higher fixed cost floor even before traffic |

## Cost optimization guidance

- **Right-size node groups first.** Compute dominates quickly.
- Use **Savings Plans / Reserved Instances** for steady-state production.
- Keep **dev** on a single NAT and single-AZ RDS unless a stronger requirement exists.
- Revisit `desired_size`, `min_size`, and HPA settings after observing real usage.
- Disable public ingress unless required; ALB and traffic add cost.
- Review whether S3 backup replication is needed in every non-production environment.
- Track NAT data volume; NAT processing is often the surprise line item.
- Use exact region pricing before customer-facing cost conversations.

## Boundary statement

This model is sufficient for **basic AWS-first planning**. It is not a replacement for:

- AWS Pricing Calculator outputs
- customer-specific traffic forecasting
- organization-specific discount programs
- production finance approval
