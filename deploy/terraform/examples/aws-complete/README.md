# AWS Complete Example - AI Control Plane

This Terraform example deploys the complete AI Control Plane infrastructure on AWS, including:

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
cd deploy/terraform/examples/aws-complete
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
| Helm Profile | demo | demo | production |

### Key Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `aws_region` | AWS region | `us-east-1` |
| `environment` | Environment name | `dev` |
| `cluster_version` | EKS Kubernetes version | `1.29` |
| `enable_alb` | Create Application Load Balancer | `true` |
| `alb_enable_https` | Enable HTTPS on ALB | `false` |
| `alb_certificate_arn` | ACM certificate ARN | `""` |
| `litellm_replica_count` | Number of LiteLLM pods | `2` |

## Outputs

After deployment, Terraform outputs important information:

```
Outputs:

cluster_endpoint = "https://XXXXXX.eks.amazonaws.com"
cluster_name = "ai-control-plane-dev"
database_endpoint = "ai-control-plane-dev.XXXXXX.us-east-1.rds.amazonaws.com:5432"
alb_dns_name = "acp-dev-XXXXXX.us-east-1.elb.amazonaws.com"
application_url = "http://acp-dev-XXXXXX.us-east-1.elb.amazonaws.com"
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

### Via ALB (if enabled)

```bash
# Get the ALB DNS name
export APP_URL=$(terraform output -raw alb_dns_name)
curl http://$APP_URL/health
```

### Via Port Forwarding

```bash
kubectl port-forward svc/acp-litellm 4000:4000 -n acp
# Access at http://localhost:4000
```

## Security Considerations

### Secrets Management

- **Master Key**: Used for LiteLLM admin API authentication
- **Salt Key**: Used for encryption - **NEVER CHANGE** after initial deployment
- **Database Password**: Auto-generated and stored in Kubernetes secrets

### IRSA (IAM Roles for Service Accounts)

The deployment creates an IAM role linked to the Kubernetes service account:

- Allows pods to assume AWS permissions without access keys
- Current permissions: Secrets Manager, CloudWatch Logs
- Customize in `main.tf` under `module.irsa.policy_statements`

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

- **Project License**: This project uses third-party open-source components. See `docs/policy/THIRD_PARTY_LICENSE_MATRIX.md` for the complete license policy.
- **License Summary**: `docs/deployment/THIRD_PARTY_LICENSE_SUMMARY.md` — Generated compliance report for customer handoff
- **Compliance Check**: Run `make license-check` to verify no restricted components are included
