# Terraform Modules for AI Control Plane

This directory contains Terraform modules for deploying the AI Control Plane infrastructure on AWS, Azure, and Google Cloud Platform.

## Overview

The Terraform modules provision the complete cloud infrastructure required to run the AI Control Plane:

- **Networking**: VPC/VNet with public/private subnets, NAT gateways
- **Kubernetes**: EKS (AWS), AKS (Azure), or GKE (GCP) clusters
- **Database**: Managed PostgreSQL (RDS, Azure Database, Cloud SQL)
- **Load Balancing**: Application-level load balancers for ingress
- **Identity**: Cloud-native pod identity (IRSA, Workload Identity)
- **Application**: Helm deployment of the AI Control Plane

## Directory Structure

```
deploy/terraform/
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
│       ├── kubernetes-namespace/  # K8s namespace
│       └── secrets/          # Kubernetes secrets
├── examples/
│   ├── aws-complete/         # Complete AWS deployment example
│   ├── azure-complete/       # Complete Azure deployment example
│   └── gcp-complete/         # Complete GCP deployment example
└── backend-examples/
    ├── s3-backend.tf         # AWS S3 backend configuration
    ├── azurerm-backend.tf    # Azure Storage backend configuration
    └── gcs-backend.tf        # GCS backend configuration
```

## Quick Start

### Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Terraform | >= 1.5.0 | Infrastructure as Code |
| AWS CLI | >= 2.0 | AWS authentication (if using AWS) |
| Azure CLI | >= 2.50 | Azure authentication (if using Azure) |
| gcloud CLI | >= 450.0 | GCP authentication (if using GCP) |
| kubectl | >= 1.25 | Kubernetes cluster access |
| Helm | >= 3.12 | Package management |

### AWS Deployment

```bash
cd deploy/terraform/examples/aws-complete

# Copy and edit terraform.tfvars
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your settings

# Initialize and apply
terraform init
terraform plan
terraform apply
```

### Azure Deployment

```bash
cd deploy/terraform/examples/azure-complete

# Copy and edit terraform.tfvars
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your settings

# Login to Azure
az login

# Initialize and apply
terraform init
terraform plan
terraform apply
```

### GCP Deployment

```bash
cd deploy/terraform/examples/gcp-complete

# Copy and edit terraform.tfvars
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your settings

# Login to GCP
gcloud auth application-default login

# Initialize and apply
terraform init
terraform plan
terraform apply
```

## Module Reference

### AWS Modules

| Module | Description | Key Resources |
|--------|-------------|---------------|
| `vpc` | VPC with public/private subnets | VPC, Subnets, NAT, IGW |
| `eks` | EKS cluster with managed node groups | EKS Cluster, Node Groups, OIDC |
| `rds` | RDS PostgreSQL instance | RDS Instance, Subnet Group |
| `alb` | Application Load Balancer | ALB, Target Groups, Listeners |
| `irsa` | IAM Roles for Service Accounts | IAM Role, Trust Policy |

### Azure Modules

| Module | Description | Key Resources |
|--------|-------------|---------------|
| `network` | VNet with subnets and NSGs | VNet, Subnets, NSGs |
| `aks` | AKS cluster with node pools | AKS Cluster, Node Pools |
| `postgresql` | Azure Database for PostgreSQL | Flexible Server, Database |
| `appgateway` | Application Gateway v2 | App Gateway, Public IP |

### GCP Modules

| Module | Description | Key Resources |
|--------|-------------|---------------|
| `vpc` | VPC with subnets and Cloud NAT | VPC, Subnets, Router, NAT |
| `gke` | GKE cluster with node pools | GKE Cluster, Node Pools |
| `cloudsql` | Cloud SQL PostgreSQL | SQL Instance, Database |
| `loadbalancer` | Global HTTP(S) Load Balancer | Load Balancer, Backend Service |

### Common Modules

| Module | Description | Key Resources |
|--------|-------------|---------------|
| `helm-release` | Deploy AI Control Plane Helm chart | Helm Release |
| `kubernetes-namespace` | Create K8s namespace | Namespace |
| `secrets` | Create K8s secrets | Secret |

## Configuration

### Environment-Specific Sizing

All examples support environment-specific resource sizing via the `environment` variable:

| Environment | Nodes | Database | Purpose |
|-------------|-------|----------|---------|
| `dev` | 1-3 | db.t3.micro / B_Standard_B2s / db-f1-micro | Development, testing |
| `staging` | 2-5 | db.t3.small / B_Standard_B2s / db-g1-small | Pre-production |
| `production` | 2-10 | db.t3.medium / GP_Standard_D2s_v3 / db-n1-standard-2 | Production workloads |

### Required Secrets

The following secrets are required for deployment:

| Secret | Description | Generation |
|--------|-------------|------------|
| `LITELLM_MASTER_KEY` | Master key for LiteLLM admin API | Auto-generated or provided |
| `LITELLM_SALT_KEY` | Encryption salt for LiteLLM | Auto-generated or provided |
| `DATABASE_URL` | PostgreSQL connection string | Auto-generated from database module |

### State Management

For production use, configure a remote backend. See `backend-examples/` for:
- AWS S3 with DynamoDB locking
- Azure Blob Storage
- Google Cloud Storage

## Outputs

Each example provides the following outputs:

| Output | Description |
|--------|-------------|
| `cluster_endpoint` | Kubernetes API endpoint |
| `database_endpoint` | Database connection endpoint |
| `application_url` | URL to access AI Control Plane |
| `kubeconfig_command` | Command to configure kubectl |
| `connection_commands` | Helpful commands for connecting |

## Security Considerations

1. **State Files**: Terraform state contains sensitive data. Always use:
   - Remote backends with encryption
   - State locking to prevent concurrent modifications
   - Access controls limiting who can read state

2. **Secrets**: Never commit secrets to Git:
   - Use `terraform.tfvars` (gitignored in examples)
   - Auto-generate passwords where possible
   - Use cloud secret managers for production

3. **Network Security**:
   - Private subnets for databases and nodes
   - NAT gateways for egress
   - Security groups/NSGs with minimal access

4. **Pod Identity**:
   - AWS: IRSA (IAM Roles for Service Accounts)
   - Azure: Workload Identity
   - GCP: Workload Identity

## Troubleshooting

### Terraform Init Fails

```bash
# Clear plugin cache
rm -rf .terraform/
rm .terraform.lock.hcl

# Reinitialize
terraform init
```

### Kubernetes Provider Issues

The Kubernetes and Helm providers are configured automatically after EKS/AKS/GKE creation. If you see authentication errors:

```bash
# AWS: Update kubeconfig
aws eks update-kubeconfig --region us-east-1 --name <cluster-name>

# Azure: Get credentials
az aks get-credentials --resource-group <rg> --name <cluster-name>

# GCP: Get credentials
gcloud container clusters get-credentials <cluster-name> --region us-central1
```

### Database Connection Issues

Ensure the database security group/firewall allows connections from the EKS/AKS/GKE node security group.

## Contributing

When modifying modules:

1. Run `terraform fmt` to format code
2. Run `terraform validate` to validate syntax
3. Update README.md with any new variables or outputs
4. Test changes with one of the example configurations

## License

See the main project LICENSE file.

## Additional Documentation

- [Terraform Deployment Guide](../../docs/deployment/TERRAFORM.md)
- [Kubernetes Helm Guide](../../docs/deployment/KUBERNETES_HELM.md)
- [Production Contract](../../docs/deployment/SINGLE_TENANT_PRODUCTION_CONTRACT.md)
