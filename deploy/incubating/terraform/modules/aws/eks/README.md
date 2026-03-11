# AWS EKS Terraform Module

Terraform module for creating and managing an Amazon EKS (Elastic Kubernetes Service) cluster with managed node groups, IAM roles, and OIDC provider for IRSA (IAM Roles for Service Accounts).

## Features

- **EKS Cluster** with configurable Kubernetes version
- **Managed Node Groups** with flexible scaling and instance configurations
- **IAM Roles** for cluster and node groups with proper policy attachments
- **OIDC Provider** for IRSA (IAM Roles for Service Accounts) support
- **Security Groups** for cluster and node communication
- **EKS Addons** (VPC CNI, CoreDNS, kube-proxy)
- **Cluster Autoscaler** IAM permissions (optional)
- **Private/Public Endpoint Access** configuration
- **Cluster Encryption** with KMS (optional)

## Requirements

| Name | Version |
|------|---------|
| terraform | >= 1.5.0 |
| aws | >= 5.0 |
| tls | >= 3.0 |

## Usage

### Basic Example

```hcl
module "eks" {
  source = "./modules/aws/eks"

  cluster_name    = "my-cluster"
  cluster_version = "1.29"
  vpc_id          = module.vpc.vpc_id
  subnet_ids      = module.vpc.private_subnet_ids

  node_groups = {
    general = {
      desired_size   = 3
      min_size       = 1
      max_size       = 10
      instance_types = ["t3.medium"]
      capacity_type  = "ON_DEMAND"
      disk_size      = 50
      labels = {
        role = "general"
      }
    }
    spot = {
      desired_size   = 2
      min_size       = 0
      max_size       = 20
      instance_types = ["t3.large", "t3a.large"]
      capacity_type  = "SPOT"
      disk_size      = 50
      labels = {
        role = "spot-workloads"
      }
      taints = [{
        key    = "spot"
        value  = "true"
        effect = "NO_SCHEDULE"
      }]
    }
  }

  tags = {
    Environment = "production"
    Team        = "platform"
  }
}
```

### Private Cluster Example

```hcl
module "eks" {
  source = "./modules/aws/eks"

  cluster_name                    = "private-cluster"
  cluster_version                 = "1.29"
  vpc_id                          = module.vpc.vpc_id
  subnet_ids                      = module.vpc.private_subnet_ids
  cluster_endpoint_public_access  = false
  cluster_endpoint_private_access = true
  cluster_public_access_cidrs     = []

  node_groups = {
    private = {
      desired_size   = 2
      min_size       = 1
      max_size       = 5
      instance_types = ["t3.medium"]
    }
  }
}
```

### With IRSA Example

```hcl
module "eks" {
  source = "./modules/aws/eks"

  cluster_name = "irsa-cluster"
  vpc_id       = module.vpc.vpc_id
  subnet_ids   = module.vpc.private_subnet_ids
  enable_irsa  = true

  node_groups = {
    default = {
      desired_size   = 2
      instance_types = ["t3.medium"]
    }
  }
}

# Example IRSA role for AWS Load Balancer Controller
module "aws_load_balancer_controller_irsa" {
  source = "./modules/aws/irsa"

  oidc_provider_arn = module.eks.oidc_provider_arn
  role_name         = "aws-load-balancer-controller"
  namespace         = "kube-system"
  service_account   = "aws-load-balancer-controller"

  policy_statements = [
    {
      effect    = "Allow"
      actions   = ["elasticloadbalancing:*"]
      resources = ["*"]
    }
  ]
}
```

## Inputs

### Required Variables

| Name | Description | Type |
|------|-------------|------|
| `cluster_name` | Name of the EKS cluster | `string` |
| `vpc_id` | VPC ID where the cluster will be deployed | `string` |
| `subnet_ids` | List of subnet IDs for the cluster | `list(string)` |

### Cluster Configuration

| Name | Description | Type | Default |
|------|-------------|------|---------|
| `cluster_version` | Kubernetes version | `string` | `"1.29"` |
| `cluster_enabled_log_types` | Control plane logging types | `list(string)` | `["api", "audit", "authenticator", "controllerManager", "scheduler"]` |
| `cluster_endpoint_public_access` | Enable public endpoint | `bool` | `true` |
| `cluster_endpoint_private_access` | Enable private endpoint | `bool` | `true` |
| `cluster_public_access_cidrs` | Allowed CIDRs for public access | `list(string)` | `["0.0.0.0/0"]` |
| `cluster_service_ipv4_cidr` | CIDR for Kubernetes services | `string` | `null` |
| `cluster_ip_family` | IP family (ipv4/ipv6) | `string` | `"ipv4"` |
| `cluster_encryption_config` | Encryption configuration | `object` | `null` |
| `create_kms_key` | Create KMS key for encryption | `bool` | `false` |
| `enable_security_groups_for_pods` | Enable Security Groups for Pods | `bool` | `false` |

### Node Groups

| Name | Description | Type | Default |
|------|-------------|------|---------|
| `node_groups` | Map of managed node group definitions | `map(object)` | See below |
| `node_group_subnet_ids` | Separate subnet IDs for nodes | `list(string)` | `null` |
| `node_group_version` | Kubernetes version for nodes | `string` | `null` |

#### Node Group Object

```hcl
{
  desired_size               = number           # Default: 2
  min_size                   = number           # Default: 1
  max_size                   = number           # Default: 5
  instance_types             = list(string)     # Default: ["t3.medium"]
  capacity_type              = string           # Default: "ON_DEMAND"
  ami_type                   = string           # Default: "AL2_x86_64"
  disk_size                  = number           # Default: 50
  max_unavailable_percentage = number           # Default: 25
  labels                     = map(string)      # Default: {}
  taints = list(object({    # Default: []
    key    = string
    value  = string
    effect = string
  }))
  launch_template_id         = string           # Default: null
  launch_template_version    = string           # Default: null
  remote_access = object({  # Default: null
    ec2_ssh_key               = string
    source_security_group_ids = list(string)
  })
  tags = map(string)        # Default: {}
}
```

### Features

| Name | Description | Type | Default |
|------|-------------|------|---------|
| `enable_cluster_autoscaler` | Enable Cluster Autoscaler IAM permissions | `bool` | `true` |
| `enable_irsa` | Enable IAM Roles for Service Accounts | `bool` | `true` |

### Tags

| Name | Description | Type | Default |
|------|-------------|------|---------|
| `tags` | Tags for all resources | `map(string)` | `{}` |

## Outputs

| Name | Description |
|------|-------------|
| `cluster_id` | The ID of the EKS cluster |
| `cluster_name` | The name of the EKS cluster |
| `cluster_arn` | The ARN of the EKS cluster |
| `cluster_endpoint` | The endpoint URL for the EKS API server |
| `cluster_version` | The Kubernetes version of the cluster |
| `cluster_certificate_authority_data` | Base64 encoded cluster CA certificate |
| `cluster_oidc_issuer_url` | The OIDC issuer URL for IRSA |
| `cluster_security_group_id` | Security group ID attached to the control plane |
| `oidc_provider_arn` | ARN of the OIDC provider for IRSA |
| `node_iam_role_arn` | IAM role ARN for node groups |
| `node_security_group_id` | Security group ID for node groups |
| `node_groups` | Map of node group attributes |

## Node Group AMI Types

| AMI Type | Description |
|----------|-------------|
| `AL2_x86_64` | Amazon Linux 2 (x86_64) |
| `AL2_x86_64_GPU` | Amazon Linux 2 with GPU support |
| `AL2_ARM_64` | Amazon Linux 2 (ARM64) |
| `BOTTLEROCKET_x86_64` | Bottlerocket (x86_64) |
| `BOTTLEROCKET_ARM_64` | Bottlerocket (ARM64) |
| `WINDOWS_CORE_2019_x86_64` | Windows Server 2019 Core |
| `WINDOWS_FULL_2019_x86_64` | Windows Server 2019 Full |
| `WINDOWS_CORE_2022_x86_64` | Windows Server 2022 Core |
| `WINDOWS_FULL_2022_x86_64` | Windows Server 2022 Full |

## Capacity Types

- `ON_DEMAND` - Standard on-demand instances
- `SPOT` - Spot instances for cost savings (may be interrupted)

## Taint Effects

- `NO_SCHEDULE` - Pods without matching toleration won't be scheduled
- `PREFER_NO_SCHEDULE` - Prefer not to schedule pods without toleration
- `NO_EXECUTE` - Evict pods without matching toleration

## IAM Roles Created

### Cluster Role

- **Name**: `{cluster_name}-cluster-role`
- **Policies**: `AmazonEKSClusterPolicy`
- **Optional**: `AmazonEKSVPCResourceController` (if Security Groups for Pods enabled)

### Node Role

- **Name**: `{cluster_name}-node-role`
- **Policies**:
  - `AmazonEKSWorkerNodePolicy`
  - `AmazonEKS_CNI_Policy`
  - `AmazonEC2ContainerRegistryReadOnly`
  - Custom Cluster Autoscaler policy (if enabled)

## Security Groups

### Cluster Security Group

- Allows outbound traffic to all destinations
- Allows inbound traffic from nodes on port 443

### Node Security Group

- Allows outbound traffic to all destinations
- Allows inter-node communication on all ports
- Allows cluster control plane to communicate with kubelets (port 10250)
- Allows cluster control plane to communicate with nodes (ports 1025-65535)

## Notes

- The `desired_size` in node groups is ignored after creation to prevent conflicts with Cluster Autoscaler
- Node groups use a `create_before_destroy` lifecycle to ensure zero-downtime updates
- CoreDNS addon depends on node groups being available
- IRSA requires the OIDC provider which is only created if `enable_irsa = true`
- Public endpoint access with `0.0.0.0/0` should be restricted in production
