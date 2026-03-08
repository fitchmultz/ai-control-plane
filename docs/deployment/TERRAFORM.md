# Terraform Deployment Guide

> Terraform is an optional deployment track for cloud infrastructure provisioning and Kubernetes platform bootstrap.
> For direct Linux-host deployments, use the default Docker-first path in `../DEPLOYMENT.md`.

This guide provides comprehensive instructions for deploying the AI Control Plane on AWS, Azure, and GCP using Terraform.

## Table of Contents

1. [Overview](#1-overview)
2. [Prerequisites](#2-prerequisites)
3. [Quick Start](#3-quick-start)
4. [Provider-Specific Deployment](#4-provider-specific-deployment)
5. [Configuration](#5-configuration)
6. [State Management](#6-state-management)
7. [Security](#7-security)
8. [Operations](#8-operations)
9. [Troubleshooting](#9-troubleshooting)
10. [Reference](#10-reference)

---

## 1. Overview

The Terraform modules provide a production-safe infrastructure baseline for the AI Control Plane. They create:

- **Network Infrastructure**: VPC/VNet with public/private subnets, NAT for egress
- **Kubernetes Cluster**: Managed EKS/AKS/GKE with autoscaling node pools
- **Database**: Managed PostgreSQL with backup and high availability
- **Load Balancer**: TLS-terminated ingress only
- **Identity**: Cloud-native pod identity for secure API access
- **Application**: Automated Helm deployment of the AI Control Plane

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Cloud Account                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                           VPC/VNet                                   │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │   │
│  │  │  Public      │  │  Private     │  │  Private (Database)      │  │   │
│  │  │  Subnet      │  │  Subnet      │  │  Subnet                  │  │   │
│  │  │              │  │              │  │                          │  │   │
│  │  │  Load        │  │  EKS/AKS/    │  │  RDS/Azure DB/           │  │   │
│  │  │  Balancer    │  │  GKE Nodes   │  │  Cloud SQL               │  │   │
│  │  └──────────────┘  └──────┬───────┘  └────────────┬─────────────┘  │   │
│  │                           │                       │                 │   │
│  │                           ▼                       │                 │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │              Kubernetes Cluster                              │   │   │
│  │  │  ┌───────────────────────────────────────────────────────┐  │   │   │
│  │  │  │              AI Control Plane Namespace                │  │   │   │
│  │  │  │  ┌─────────────────────────────────────────────────┐  │  │   │   │
│  │  │  │  │         LiteLLM Gateway (Pods)                   │  │  │   │   │
│  │  │  │  │                                                  │  │  │   │   │
│  │  │  │  │  ┌──────────────┐  ┌─────────────────────────┐  │  │  │   │   │
│  │  │  │  │  │   Ingress    │  │   Secrets (keys, DB)    │  │  │  │   │   │
│  │  │  │  │  │   Controller │  │                         │  │  │  │   │   │
│  │  │  │  │  └──────────────┘  └─────────────────────────┘  │  │  │   │   │
│  │  │  │  └─────────────────────────────────────────────────┘  │  │   │   │
│  │  │  └───────────────────────────────────────────────────────┘  │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Prerequisites

### Required Tools

| Tool | Version | Installation |
|------|---------|--------------|
| Terraform | >= 1.5.0 | [Download](https://developer.hashicorp.com/terraform/downloads) |
| kubectl | >= 1.25 | [Install](https://kubernetes.io/docs/tasks/tools/) |
| Helm | >= 3.12 | [Install](https://helm.sh/docs/intro/install/) |

### Cloud Provider CLIs

Choose the CLI for your cloud provider:

| Provider | CLI | Installation |
|----------|-----|--------------|
| AWS | AWS CLI v2 | [Install](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html) |
| Azure | Azure CLI | [Install](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli) |
| GCP | gcloud CLI | [Install](https://cloud.google.com/sdk/docs/install) |

### Authentication

Before deploying, authenticate with your cloud provider:

**AWS:**
```bash
aws configure
# Or use environment variables:
export AWS_ACCESS_KEY_ID="your-key"
export AWS_SECRET_ACCESS_KEY="your-secret"
export AWS_REGION="us-east-1"
```

**Azure:**
```bash
az login
az account set --subscription "your-subscription-id"
```

**GCP:**
```bash
gcloud auth application-default login
gcloud config set project your-project-id
```

---

## 3. Quick Start

### 3.1 AWS Quick Start

```bash
# Navigate to the AWS example
cd deploy/terraform/examples/aws-complete

# Copy the example variables file
cp terraform.tfvars.example terraform.tfvars

# Edit with your settings
# Required: aws_region, environment
# Optional: name_prefix, cluster_name, etc.

# Initialize Terraform
terraform init

# Review the deployment plan (guardrails fail public-open or TLS-off settings)
terraform plan

# Deploy
terraform apply

# Get outputs
terraform output

# Configure kubectl
aws eks update-kubeconfig --region $(terraform output -raw aws_region) --name $(terraform output -raw cluster_name)

# Access the application
terraform output application_url
```

### 3.2 Azure Quick Start

```bash
cd deploy/terraform/examples/azure-complete

cp terraform.tfvars.example terraform.tfvars
# Edit: resource_group_name, location, environment

terraform init
terraform plan
terraform apply

# Get kubeconfig
az aks get-credentials --resource-group $(terraform output -raw resource_group_name) --name $(terraform output -raw cluster_name)

# Access the application
terraform output application_url
```

### 3.3 GCP Quick Start

```bash
cd deploy/terraform/examples/gcp-complete

cp terraform.tfvars.example terraform.tfvars
# Edit: project_id, region, environment

terraform init
terraform plan
terraform apply

# Get credentials
gcloud container clusters get-credentials $(terraform output -raw cluster_name) --region $(terraform output -raw region)

# Access the application
terraform output application_url
```

---

## 4. Provider-Specific Deployment

### 4.1 AWS Deployment

The AWS example creates:
- VPC with 3 AZs, public and private subnets
- EKS cluster with managed node groups
- RDS PostgreSQL in private subnets
- Application Load Balancer (optional)
- IRSA for pod identity

**Key Files:**
- `main.tf` - Infrastructure orchestration
- `variables.tf` - Input variables
- `terraform.tfvars` - Your configuration (not in Git)

**Environment-Specific Defaults:**

| Environment | Nodes | Instance Type | RDS Class | Multi-AZ |
|-------------|-------|---------------|-----------|----------|
| dev | 1-3 | t3.medium | db.t3.micro | No |
| staging | 2-5 | t3.medium | db.t3.small | Yes |
| production | 2-10 | t3.large | db.t3.medium | Yes |

### 4.2 Azure Deployment

The Azure example creates:
- Resource Group
- VNet with subnets for AKS and database
- AKS cluster with system and user node pools
- Azure Database for PostgreSQL - Flexible Server
- Workload Identity support

**Key Features:**
- Private endpoints for database (optional)
- Azure CNI networking
- Cluster autoscaling
- Workload Identity federation

**Environment-Specific Defaults:**

| Environment | Nodes | VM Size | PostgreSQL SKU | HA |
|-------------|-------|---------|----------------|-----|
| dev | 1-3 | Standard_B2s | B_Standard_B2s | No |
| staging | 2-5 | Standard_B2s | B_Standard_B2s | No |
| production | 2-10 | Standard_D2s_v3 | GP_Standard_D2s_v3 | Yes |

### 4.3 GCP Deployment

The GCP example creates:
- VPC network with subnets and secondary ranges
- GKE cluster with Workload Identity
- Cloud SQL PostgreSQL with private IP
- Cloud NAT for egress

**Key Features:**
- VPC-native GKE with alias IP ranges
- Workload Identity for GKE
- Cloud SQL with private IP
- Container-native load balancing support

**Environment-Specific Defaults:**

| Environment | Nodes | Machine Type | SQL Tier | Availability |
|-------------|-------|--------------|----------|--------------|
| dev | 1-3 | e2-medium (spot) | db-f1-micro | ZONAL |
| staging | 2-5 | e2-medium | db-g1-small | ZONAL |
| production | 2-10 | e2-standard-2 | db-n1-standard-2 | REGIONAL |

---

## 5. Configuration

### 5.1 Common Variables

All examples support these common variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `environment` | Environment type (dev/staging/production) | `dev` |
| `name_prefix` | Prefix for resource names | `ai-control-plane` |
| `kubernetes_version` | Kubernetes version | `1.29` |

### 5.2 Environment-Specific Behavior

Setting `environment` automatically configures:

- Resource sizing (nodes, database tier)
- High availability (Multi-AZ for production)
- Backup retention (7 days for production)
- Deletion protection (enabled for production)
- Helm profile (`production` profile for production)

### 5.3 Environment Validation Calibration

Use canonical CI/runtime gates to validate Terraform environment assumptions before apply:

```bash
# Runtime + release-evidence gate
make ci-nightly

# Heavy security/image validation gate
make ci-manual-heavy

# Optional smoke validation against a deployed endpoint
make prod-smoke PUBLIC_URL=https://gateway.example.com
```

**How to calibrate Terraform variables from validation results:**

| Validation Outcome | Terraform Adjustment |
|-------------------|----------------------|
| `ci-nightly` + smoke PASS | Keep baseline environment defaults |
| Smoke latency regressions | Increase instance size / node resources |
| Runtime/detection failures | Fix config and policy issues before scaling |
| Heavy gate failures | Resolve hardened-image/supply-chain issues before apply |

**Example: Applying Validation Findings**

```hcl
# Example: increase node resources after smoke regressions
node_groups = {
  general = {
    instance_types = ["t3.large"]
    desired_size   = 3
    # ... rest of config
  }
}
```

Customer-specific load testing can be layered on top of this baseline, but benchmark harness commands from older private iterations are not part of the public-snapshot Make target surface.

### 5.4 Customizing Node Pools / Node Groups

**AWS (EKS Node Groups):**
```hcl
node_groups = {
  general = {
    desired_size   = 3
    min_size       = 2
    max_size       = 10
    instance_types = ["t3.medium"]
    capacity_type  = "ON_DEMAND"  # or "SPOT"
    disk_size      = 50
    labels = {
      workload = "general"
    }
    taints = []
  }
}
```

**Azure (AKS Node Pools):**
```hcl
node_pools = {
  default = {
    vm_size             = "Standard_D2s_v3"
    node_count          = 3
    min_count           = 2
    max_count           = 10
    enable_auto_scaling = true
    labels = {
      workload = "general"
    }
    taints = []
  }
}
```

**GCP (GKE Node Pools):**
```hcl
node_pools = {
  default = {
    machine_type       = "e2-standard-2"
    initial_node_count = 3
    min_count          = 2
    max_count          = 10
    preemptible        = false
    labels = {
      workload = "general"
    }
  }
}
```

### 5.5 Database Configuration

**AWS RDS:**
```hcl
db_instance_class        = "db.t3.medium"
db_allocated_storage     = 100
multi_az                 = true
backup_retention_period  = 7
deletion_protection      = true
```

**Azure PostgreSQL:**
```hcl
postgresql_sku_name = "GP_Standard_D2s_v3"
postgresql_storage_mb = 65536
high_availability_enabled = true
geo_redundant_backup_enabled = true
```

**GCP Cloud SQL:**
```hcl
cloudsql_tier = "db-n1-standard-2"
cloudsql_disk_size = 100
cloudsql_availability_type = "REGIONAL"
```

### 5.6 Helm Configuration

All secrets are automatically generated and passed to Helm:

```hcl
# Auto-generated (can override)
litellm_master_key = null  # Auto-generated if null
litellm_salt_key   = null  # Auto-generated if null

# Helm values overrides
helm_values = {
  litellm = {
    replicaCount = 3
    resources = {
      limits = {
        cpu    = "2000m"
        memory = "2Gi"
      }
    }
  }
}
```

---

## 6. State Management

### 6.1 Why Remote State?

Terraform state contains sensitive information. Remote backends provide:
- **Collaboration**: Team members can share state
- **Security**: State is encrypted at rest
- **Locking**: Prevents concurrent modifications
- **Versioning**: Recover from accidental changes

### 6.2 Setting Up Remote State

See `deploy/terraform/backend-examples/` for configuration templates.

**AWS S3 + DynamoDB:**

```bash
# Create bucket
aws s3 mb s3://ai-control-plane-tfstate --region us-east-1
aws s3api put-bucket-versioning \
  --bucket ai-control-plane-tfstate \
  --versioning-configuration Status=Enabled

# Create DynamoDB table for locking
aws dynamodb create-table \
  --table-name terraform-locks \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST

# Configure backend
cat > backend.tf << 'EOF'
terraform {
  backend "s3" {
    bucket         = "ai-control-plane-tfstate"
    key            = "aws/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "terraform-locks"
  }
}
EOF

terraform init -migrate-state
```

### 6.3 State Commands

```bash
# View current state
terraform state list

# Show specific resource
terraform state show aws_eks_cluster.this

# Import existing resources
terraform import aws_vpc.this vpc-12345678

# Remove resource from state (doesn't delete it)
terraform state rm aws_instance.example
```

---

## 7. Security

### 7.1 Secrets Management

**Auto-Generated Secrets (Default):**
- RDS/PostgreSQL/Cloud SQL passwords
- LiteLLM master key
- LiteLLM salt key

All secrets are:
- Marked as `sensitive` in Terraform outputs
- Stored in Kubernetes Secrets
- Never logged to console

**Using Existing Secrets:**

Override the auto-generation:

```hcl
database_password    = "your-secure-password"
litellm_master_key   = "sk-your-master-key"
litellm_salt_key     = "sk-your-salt-key"
```

### 7.2 Network Security

**Private Subnets:**
- Databases run in private subnets (no public access)
- Kubernetes nodes run in private subnets
- NAT gateways provide egress only

**Security Groups / NSGs / Firewall:**
- Minimal required ports open
- Database access restricted to Kubernetes node security groups
- Load balancers in public subnets only

### 7.3 Pod Identity

**AWS IRSA:**
```yaml
# Automatically configured
serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123:role/ai-control-plane-irsa
```

**Azure Workload Identity:**
```yaml
# Automatically configured
serviceAccount:
  annotations:
    azure.workload.identity/client-id: "00000000-0000-0000-0000-000000000000"
```

**GCP Workload Identity:**
```yaml
# Automatically configured
serviceAccount:
  annotations:
    iam.gke.io/gcp-service-account: ai-control-plane@project.iam.gserviceaccount.com
```

---

## 8. Operations

### 8.1 Scaling

**Scale Node Group:**
```bash
# Edit terraform.tfvars
node_groups = {
  general = {
    desired_size = 5
    min_size     = 3
    max_size     = 20
    # ...
  }
}

terraform apply
```

**Scale LiteLLM:**
```bash
# Via Helm values
helm upgrade acp ../../../helm/ai-control-plane \
  --namespace acp \
  --set litellm.replicaCount=5
```

### 8.2 Upgrades

**Kubernetes Upgrade:**
```bash
# Update kubernetes_version variable
kubernetes_version = "1.30"

terraform apply
```

**Helm Chart Upgrade:**
```bash
# Terraform will handle this automatically on next apply
# Or manually:
helm upgrade acp ../../../helm/ai-control-plane --namespace acp
```

### 8.3 Backup and Restore

**Database Backups:**

All databases are configured with automated backups:
- AWS RDS: Daily snapshots with configurable retention
- Azure PostgreSQL: Automated backups with geo-redundancy option
- GCP Cloud SQL: Automated backups and point-in-time recovery

**Manual Database Backup:**

```bash
# AWS RDS
aws rds create-db-snapshot \
  --db-instance-identifier ai-control-plane-db \
  --db-snapshot-identifier backup-$(date +%Y%m%d)

# Azure
az postgres flexible-server backup create \
  --resource-group my-rg \
  --name ai-control-plane-db \
  --backup-name backup-$(date +%Y%m%d)

# GCP
gcloud sql backups create --instance=ai-control-plane-db
```

### 8.4 Monitoring

Access CloudWatch/Azure Monitor/Stackdriver:

```bash
# AWS CloudWatch
aws logs tail /aws/eks/ai-control-plane/cluster --follow

# Azure Monitor
az monitor metrics list-namespaces --resource "your-aks-resource-id"

# GCP Stackdriver
gcloud logging read "resource.type=k8s_container" --limit=50
```

### 8.5 Destroying Infrastructure

**⚠️ WARNING: This deletes all resources and data!**

```bash
# Review what will be destroyed
terraform plan -destroy

# Destroy everything
terraform destroy

# Note: Some resources may have deletion protection enabled
# You'll need to disable it first for production databases
```

---

## 9. Troubleshooting

### 9.1 Terraform Issues

**"Error: Failed to query available provider packages"**
```bash
# Update providers
terraform init -upgrade
```

**"Error: Error acquiring the state lock"**
```bash
# Find lock ID from error message, then:
terraform force-unlock <LOCK_ID>
```

**"Provider configuration not present"**
```bash
# Reinitialize
terraform init
```

### 9.2 Kubernetes Issues

**"Unable to connect to the server"**
```bash
# Update kubeconfig
aws eks update-kubeconfig --region us-east-1 --name <cluster-name>
# or
az aks get-credentials --resource-group <rg> --name <cluster>
# or
gcloud container clusters get-credentials <cluster> --region us-central1
```

**"Failed to create deployment" (Helm)**
```bash
# Check events
kubectl get events --namespace acp --sort-by='.lastTimestamp'

# Check pods
kubectl get pods --namespace acp
kubectl describe pod -n acp <pod-name>

# Check logs
kubectl logs -n acp deployment/acp-ai-control-plane-litellm
```

### 9.3 Database Issues

**"Connection refused" or "Connection timeout"**

1. Check security group rules allow traffic from EKS nodes
2. Verify database is in `available` state
3. Check subnet routing

```bash
# Test connectivity from a pod
kubectl run -it --rm debug --image=postgres --restart=Never -- psql $DATABASE_URL
```

### 9.4 Load Balancer Issues

**ALB/Azure App Gateway/GCLB not routing traffic:**

1. Verify target health in console
2. Check security groups allow traffic from LB
3. Verify backend is listening on correct port (4000)

---

## 10. Reference

### 10.1 Module Reference

See individual module README files:
- `deploy/terraform/modules/aws/*/README.md`
- `deploy/terraform/modules/azure/*/README.md`
- `deploy/terraform/modules/gcp/*/README.md`
- `deploy/terraform/modules/common/*/README.md`

### 10.2 Terraform Commands

| Command | Description |
|---------|-------------|
| `terraform init` | Initialize working directory |
| `terraform plan` | Show execution plan |
| `terraform apply` | Execute plan |
| `terraform destroy` | Destroy all resources |
| `terraform validate` | Validate configuration |
| `terraform fmt` | Format configuration files |
| `terraform state list` | List resources in state |
| `terraform output` | Show outputs |
| `terraform refresh` | Refresh state |
| `terraform import` | Import existing resource |

### 10.3 File Structure

```
deploy/terraform/
├── modules/
│   ├── aws/           # AWS-specific modules
│   ├── azure/         # Azure-specific modules
│   ├── gcp/           # GCP-specific modules
│   └── common/        # Cloud-agnostic modules
├── examples/
│   ├── aws-complete/  # AWS example
│   ├── azure-complete/ # Azure example
│   └── gcp-complete/  # GCP example
└── backend-examples/  # Backend configuration templates
```

### 10.4 Additional Resources

- [Terraform Documentation](https://developer.hashicorp.com/terraform/docs)
- [AWS EKS Documentation](https://docs.aws.amazon.com/eks/)
- [Azure AKS Documentation](https://docs.microsoft.com/en-us/azure/aks/)
- [GCP GKE Documentation](https://cloud.google.com/kubernetes-engine/docs)
- [Helm Chart Documentation](./KUBERNETES_HELM.md)
