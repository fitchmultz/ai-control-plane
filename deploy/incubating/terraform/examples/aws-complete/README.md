# AWS Complete Example - AI Control Plane

This Terraform example deploys the complete AI Control Plane infrastructure on AWS, including:

> Validation boundary: this example is backed by explicit internal `make tf-*` checks plus the AWS hardening and cost documents. It remains an incubating Terraform surface and does not replace the host-first primary production contract.

- **VPC** with public and private subnets across multiple availability zones
- **EKS** (Elastic Kubernetes Service) cluster with managed node groups
- **RDS** (Relational Database Service) PostgreSQL instance for LiteLLM
- **IRSA** (IAM Roles for Service Accounts) for pod-level AWS permissions
- **ALB** (Application Load Balancer) for external access (optional)
- **Helm** deployment of the AI Control Plane with external database

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              AWS Cloud                                       │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                           VPC (10.0.0.0/16)                         │   │
│  │                                                                     │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                    Public Subnets                           │   │   │
│  │  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │   │   │
│  │  │  │   ALB       │    │   NAT GW    │    │   Bastion   │     │   │   │
│  │  │  │  (optional) │    │             │    │   (opt)     │     │   │   │
│  │  │  └─────────────┘    └─────────────┘    └─────────────┘     │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                                                                     │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                   Private Subnets                           │   │   │
│  │  │                                                             │   │   │
│  │  │  ┌─────────────────────────────────────────────────────┐   │   │   │
│  │  │  │              EKS Cluster (Kubernetes)               │   │   │   │
│  │  │  │  ┌─────────────────────────────────────────────┐   │   │   │   │
│  │  │  │  │         Managed Node Group(s)               │   │   │   │   │
│  │  │  │  │  ┌─────────────┐      ┌─────────────┐       │   │   │   │   │
│  │  │  │  │  │ LiteLLM Pod │      │ LiteLLM Pod │  ...  │   │   │   │   │
│  │  │  │  │  │  (IRSA)     │      │  (IRSA)     │       │   │   │   │   │
│  │  │  │  │  └─────────────┘      └─────────────┘       │   │   │   │   │
│  │  │  │  └─────────────────────────────────────────────┘   │   │   │   │
│  │  │  └─────────────────────────────────────────────────────┘   │   │   │
│  │  │                                                             │   │   │
│  │  │  ┌─────────────────────────────────────────────────────┐   │   │   │
│  │  │  │         RDS PostgreSQL (Multi-AZ opt)               │   │   │   │
│  │  │  │  ┌─────────────────────────────────────────────┐   │   │   │   │
│  │  │  │  │  Database: litellm                          │   │   │   │   │
│  │  │  │  │  User: litellm                              │   │   │   │   │
│  │  │  │  │  Encryption: Enabled                        │   │   │   │   │
│  │  │  │  └─────────────────────────────────────────────┘   │   │   │   │
│  │  │  └─────────────────────────────────────────────────────┘   │   │   │
│  │  │                                                             │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         IAM / IRSA                                  │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │  Service Account Role (Pod Identity)                        │   │   │
│  │  │  - Secrets Manager Access                                   │   │   │
│  │  │  - CloudWatch Logs Access                                   │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- **Terraform** >= 1.5.0
- **AWS CLI** configured with appropriate credentials
- **kubectl** for Kubernetes management
- **Helm** v3 for chart deployment
- An **AWS account** with sufficient permissions

## Quick Start

### 1. Configure AWS Credentials

```bash
aws configure
# OR
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_DEFAULT_REGION="us-east-1"
```

### 2. Initialize Terraform

```bash
cd deploy/incubating/terraform/examples/aws-complete
terraform init
```

### 3. Configure Variables

Copy the example file and customize:

```bash
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your settings
```

### 4. Plan and Apply

```bash
terraform plan
terraform apply
```

### 5. Configure kubectl

```bash
aws eks update-kubeconfig --region us-east-1 --name ai-control-plane-dev
```

## Configuration

### Environment-Specific Defaults

The module provides environment-specific defaults based on the `environment` variable:

| Setting | dev | staging | production |
|---------|-----|---------|------------|
| RDS Instance | db.t3.micro | db.t3.small | db.t3.medium |
| Multi-AZ | false | true | true |
| Single NAT Gateway | true | false | false |
| Node Instance Type | t3.medium | t3.medium | t3.large |
| Node Min/Max | 1-3 | 2-5 | 2-10 |
| Helm Profile | production | production | production |

> The AWS example always deploys the Helm chart using the production profile. Environment selection changes infrastructure sizing defaults, not the Helm security contract.

### Key Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `aws_region` | AWS region | `us-east-1` |
| `environment` | Environment name | `production` |
| `cluster_version` | EKS Kubernetes version | `1.29` |
| `enable_ingress` | Create TLS-terminated ingress | `false` |
| `public_ingress_enabled` | Expose ingress publicly | `false` |
| `alb_certificate_arn` | ACM certificate ARN | `""` |
| `litellm_replica_count` | Number of LiteLLM pods | `2` |

## Outputs

After deployment, Terraform outputs important information:

```
Outputs:

cluster_endpoint = "https://XXXXXX.eks.amazonaws.com"
cluster_name = "ai-control-plane-dev"
database_endpoint = "ai-control-plane-dev.XXXXXX.us-east-1.rds.amazonaws.com:5432"
application_url = "https://ai-control-plane.example.com"
kubeconfig_command = "aws eks update-kubeconfig --region us-east-1 --name ai-control-plane-dev"
```

### Sensitive Outputs

Sensitive values (database URL, master key, salt key) are marked as sensitive:

```bash
terraform output -raw database_url
terraform output -raw litellm_master_key
terraform output -raw litellm_salt_key
```

## Accessing the Application

### Via TLS Ingress (if enabled)

```bash
export APP_URL=$(terraform output -raw application_url)
curl -H "Authorization: Bearer $(terraform output -raw litellm_master_key)" "$APP_URL/health"
```

### Via Port Forwarding

```bash
kubectl port-forward svc/acp-litellm 4000:4000 -n acp
# Access at http://localhost:4000 (localhost-only troubleshooting path)
```

## Security Considerations

### Secrets Management

- **Master Key**: Used for LiteLLM admin API authentication
- **Salt Key**: Used for encryption - **NEVER CHANGE** after initial deployment
- **Database Password**: Auto-generated and stored in Kubernetes secrets
- **Application Keys**: Must be supplied explicitly in `terraform.tfvars`

### IRSA (IAM Roles for Service Accounts)

The deployment creates an IAM role linked to the Kubernetes service account:

- Allows pods to assume AWS permissions without access keys
- Default permissions: none
- Add scoped permissions through `var.irsa_policy_statements` only when the workload must call AWS APIs

### Network Security

- EKS nodes run in private subnets
- RDS is only accessible from EKS security group
- Public access to cluster endpoint can be restricted via `cluster_public_access_cidrs`

## Cost Optimization

### Development

For development environments, consider:

```hcl
environment = "dev"
single_nat_gateway = true
rds_multi_az = false
rds_instance_class = "db.t3.micro"
```

### Production

For production workloads:

```hcl
environment = "production"
rds_deletion_protection = true
rds_skip_final_snapshot = false
enable_autoscaling = true
```

## Troubleshooting

### Pods Not Starting

```bash
# Check pod status
kubectl get pods -n acp

# Check events
kubectl get events -n acp --sort-by=.lastTimestamp

# View pod logs
kubectl logs -n acp -l app.kubernetes.io/component=litellm
```

### Database Connection Issues

```bash
# Test connectivity from a pod
kubectl run -it --rm debug --image=postgres:16 --restart=Never -n acp -- \
  psql $(terraform output -raw database_url)
```

### IRSA Issues

```bash
# Verify service account annotation
kubectl describe sa acp-sa -n acp

# Check pod identity
kubectl exec -n acp deployment/acp-litellm -- aws sts get-caller-identity
```

## Cleanup

To destroy all resources:

```bash
terraform destroy
```

**Warning**: This will delete the RDS database unless `skip_final_snapshot = false` is set.

## Module Dependencies

This example uses the following modules:

| Module | Path | Purpose |
|--------|------|---------|
| vpc | `../../modules/aws/vpc` | VPC and networking |
| eks | `../../modules/aws/eks` | EKS cluster |
| rds | `../../modules/aws/rds` | PostgreSQL database |
| irsa | `../../modules/aws/irsa` | IAM roles for pods |
| alb | `../../modules/aws/alb` | Application load balancer |
| kubernetes-namespace | `../../modules/common/kubernetes-namespace` | K8s namespace |
| secrets | `../../modules/common/secrets` | K8s secrets |
| helm-release | `../../modules/common/helm-release` | Helm deployment |

## License and Compliance

- **Project License**: Apache-2.0. See the main project [`LICENSE`](../../../../../LICENSE) and [`NOTICE`](../../../../../NOTICE) files.
- **Third-Party License Policy**: `docs/policy/THIRD_PARTY_LICENSE_MATRIX.md` defines the complete third-party license boundary.
- **License Summary**: `docs/deployment/THIRD_PARTY_LICENSE_SUMMARY.md` — Generated compliance report for customer handoff
- **Compliance Check**: Run `make license-check` to verify no restricted components are included
