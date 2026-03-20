# Terraform Modules for AI Control Plane

This directory contains **incubating** Terraform modules for AI Control Plane cloud deployment paths.

## Status

- **AWS** is the only cloud path with an explicit validation package in this repository:
  - internal `make tf-*` validation targets
  - AWS hardening guidance
  - AWS cost-estimation model
- **Azure and GCP remain incubating** and are not validated for external claims.
- Terraform remains outside the supported host-first operator UX and default CI.

## Internal validation workflow

```bash
make tf-fmt-check
make tf-validate
make tf-security-check
make tf-plan-aws
```

For the Terraform boundary and claim language, see:
- `docs/deployment/TERRAFORM.md`
- `docs/security/AWS_CLOUD_HARDENING.md`
- `docs/deployment/AWS_COST_ESTIMATION.md`

## Overview

The Terraform modules provision the cloud infrastructure needed to run the AI Control Plane:

- **Networking**: VPC/VNet with public/private subnets and NAT gateways
- **Kubernetes**: EKS (AWS), AKS (Azure), or GKE (GCP) clusters
- **Database**: Managed PostgreSQL (RDS, Azure Database, Cloud SQL)
- **Load Balancing**: application-level ingress load balancers
- **Identity**: cloud-native pod identity (IRSA, Workload Identity)
- **Application**: Helm deployment of the AI Control Plane

## Directory Structure

```text
deploy/incubating/terraform/
├── modules/
│   ├── aws/
│   │   ├── vpc/              # VPC, subnets, NAT
│   │   ├── eks/              # EKS cluster, node groups
│   │   ├── rds/              # RDS PostgreSQL
│   │   ├── alb/              # Application Load Balancer
│   │   └── irsa/             # IAM Roles for Service Accounts
│   ├── azure/
│   │   ├── network/          # VNet, subnets, NSGs
│   │   ├── aks/              # AKS cluster, node pools
│   │   ├── postgresql/       # Azure Database for PostgreSQL
│   │   └── appgateway/       # Azure Application Gateway
│   ├── gcp/
│   │   ├── vpc/              # VPC, subnets, Cloud NAT
│   │   ├── gke/              # GKE cluster, node pools
│   │   ├── cloudsql/         # Cloud SQL PostgreSQL
│   │   └── loadbalancer/     # Cloud Load Balancer
│   └── common/
│       ├── helm-release/     # Helm release for AI Control Plane
│       ├── kubernetes-namespace/
│       └── secrets/
├── examples/
│   ├── aws-complete/
│   ├── azure-complete/
│   └── gcp-complete/
└── backend-examples/
    ├── s3-backend.tf
    ├── azurerm-backend.tf
    └── gcs-backend.tf
```

## Quick Start

### AWS validation package

```bash
cd deploy/incubating/terraform/examples/aws-complete
cp terraform.tfvars.example terraform.tfvars
make tf-validate
make tf-plan-aws
```

### Direct manual exploration

```bash
cd deploy/incubating/terraform/examples/aws-complete
terraform init
terraform plan
```

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Terraform | >= 1.5.0 | Infrastructure as Code |
| Docker | current | Terraform container fallback for the internal validation runner |
| kubectl | >= 1.25 | Kubernetes cluster access during manual exploration |
| Helm | >= 3.12 | Package management during manual exploration |

Azure CLI and gcloud CLI are only required when manually exploring those incubating examples.

## Contributing

When modifying modules:

1. Run `make tf-fmt-check`
2. Run `make tf-validate`
3. Update the relevant README and docs when variables or outputs change
4. Keep AWS claim language aligned to `docs/deployment/TERRAFORM.md`

## License

This incubating deployment track is licensed under Apache-2.0. See the main project [LICENSE](../../LICENSE) and [NOTICE](../../NOTICE) files.
