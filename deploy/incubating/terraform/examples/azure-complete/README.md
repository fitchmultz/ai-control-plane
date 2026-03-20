# Azure Complete Example

This Terraform example deploys a complete AI Control Plane infrastructure on Microsoft Azure, including:

- **Azure Resource Group** - Logical container for all resources
- **Virtual Network (VNet)** - Network isolation with subnets for AKS and PostgreSQL
- **Azure Kubernetes Service (AKS)** - Managed Kubernetes cluster with Workload Identity
- **Azure Database for PostgreSQL** - Flexible Server with private endpoint
- **Kubernetes Namespace & Secrets** - Properly configured for security
- **Helm Release** - AI Control Plane deployment with LiteLLM gateway

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Resource Group                                  │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                     Virtual Network (VNet)                       │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │    │
│  │  │  AKS Subnet │  │  DB Subnet  │  │    Private Endpoint     │  │    │
│  │  │  10.0.1.0/24│  │  10.0.2.0/24│  │    (PostgreSQL)         │  │    │
│  │  └──────┬──────┘  └──────┬──────┘  └─────────────────────────┘  │    │
│  │         │                │                                      │    │
│  │  ┌──────▼────────────────▼──────────────────────────────────┐  │    │
│  │  │              Network Security Groups                      │  │    │
│  │  │  - Allow HTTPS to AKS                                     │  │    │
│  │  │  - Allow PostgreSQL only from AKS subnet                  │  │    │
│  │  └───────────────────────────────────────────────────────────┘  │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                           │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │              Azure Kubernetes Service (AKS)                      │    │
│  │  ┌──────────────┐  ┌─────────────────────────────────────────┐  │    │
│  │  │ System Pool  │  │           User Node Pools               │  │    │
│  │  │ (Addons)     │  │  - Workload pods                        │  │    │
│  │  └──────────────┘  │  - Horizontal Pod Autoscaler (optional) │  │    │
│  │                    └─────────────────────────────────────────┘  │    │
│  │                                                                 │    │
│  │  Features:                                                      │    │
│  │  - Azure Workload Identity enabled                              │    │
│  │  - OIDC Issuer for pod authentication                           │    │
│  │  - Calico network policy                                        │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                           │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │         Azure Database for PostgreSQL - Flexible Server          │    │
│  │  - Private endpoint connection from AKS                          │    │
│  │  - SSL enforcement enabled                                       │    │
│  │  - Automated backups                                             │    │
│  │  - Geo-redundant backup (production)                             │    │
│  │  - High availability (production)                                │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

## Prerequisites

### Required Tools

- **Terraform** >= 1.5.0
- **Azure CLI** (az) - For authentication
- **kubectl** - For Kubernetes cluster access
- **kubelogin** - For Azure AD authentication to AKS

### Azure Requirements

- Azure subscription with sufficient quota
- Contributor or Owner access to create resources
- Registered providers: `Microsoft.ContainerService`, `Microsoft.DBforPostgreSQL`, `Microsoft.Network`

### Authentication

```bash
# Login to Azure
az login

# Set your subscription
az account set --subscription "Your Subscription Name"
```

## Quick Start

### 1. Copy Example Variables

```bash
cd deploy/incubating/terraform/examples/azure-complete
cp terraform.tfvars.example terraform.tfvars
```

### 2. Customize Variables

Edit `terraform.tfvars` for your environment:

```hcl
resource_group_name = "rg-ai-control-plane-dev"
location            = "East US"
environment         = "dev"
name_prefix         = "aicp"
```

### 3. Initialize Terraform

```bash
terraform init
```

### 4. Plan Deployment

```bash
terraform plan -out=tfplan
```

### 5. Apply Deployment

```bash
terraform apply tfplan
```

### 6. Configure kubectl

```bash
# Get cluster credentials
az aks get-credentials --resource-group rg-ai-control-plane-dev --name ai-control-plane --overwrite-existing

# Verify connection
kubectl get nodes
```

### 7. Access the Application

```bash
# Port-forward to access locally
kubectl port-forward svc/acp-litellm 4000:4000 -n acp

# Access remains localhost-only when port-forwarding; use TLS ingress for shared access
```

## Configuration

### Environment Selection

The example supports three environments with sensible defaults:

| Environment | PostgreSQL SKU | Node Pool VM | Replicas | Features |
|-------------|---------------|--------------|----------|----------|
| dev | B_Standard_B2s | Standard_B2s | 1 | Basic setup |
| staging | GP_Standard_D2s_v3 | Standard_B2s | 1 | Better performance |
| production | GP_Standard_D4s_v3 | Standard_D4s_v3 | 2+ | HA, HPA, PDB |

### Production Configuration

For production deployments, use this minimal configuration:

```hcl
environment = "production"
resource_group_name = "rg-ai-control-plane-prod"
location = "East US"
name_prefix = "aicp-prod"

litellm_replica_count = 2
enable_autoscaling = true
ingress_enabled = true
ingress_host = "ai-control-plane.yourcompany.com"
ingress_class_name = "nginx"
ingress_tls_secret_name = "ai-control-plane-tls"
ingress_cluster_issuer = "letsencrypt-prod"
```

Production will automatically:
- Use larger VM sizes for nodes (Standard_D4s_v3)
- Enable geo-redundant backups
- Enable high availability for PostgreSQL
- Configure Pod Disruption Budget
- Enable Horizontal Pod Autoscaler
- Set production profile for Helm chart

### Custom Node Pools

```hcl
node_pools = {
  general = {
    vm_size             = "Standard_D4s_v3"
    node_count          = 3
    min_count           = 2
    max_count           = 10
    os_disk_size_gb     = 128
    enable_auto_scaling = true
    labels = {
      workload-type = "general"
    }
    taints = []
  }
  
  gpu = {
    vm_size             = "Standard_NC6s_v3"
    node_count          = 1
    min_count           = 0
    max_count           = 3
    enable_auto_scaling = true
    labels = {
      workload-type = "gpu"
    }
    taints = ["nvidia.com/gpu=true:NoSchedule"]
  }
}
```

### Workload Identity

Azure Workload Identity is enabled by default. Pods can use managed identities:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ai-control-plane-sa
  namespace: acp
  annotations:
    azure.workload.identity/client-id: <managed-identity-client-id>
```

## Variables

| Name | Description | Default | Required |
|------|-------------|---------|----------|
| `resource_group_name` | Resource group name | `rg-ai-control-plane` | No |
| `location` | Azure region | `East US` | No |
| `environment` | Environment tag (dev/staging/production) | `dev` | No |
| `name_prefix` | Resource name prefix | `ai-cp` | No |
| `cluster_name` | AKS cluster name | `ai-control-plane` | No |
| `kubernetes_version` | Kubernetes version | `1.29` | No |
| `postgresql_sku_name` | PostgreSQL SKU | `B_Standard_B2s` | No |
| `litellm_replica_count` | Number of LiteLLM replicas | `1` | No |
| `ingress_enabled` | Enable ingress | `false` | No |
| `ingress_host` | Ingress hostname | `""` | No |
| `enable_autoscaling` | Enable HPA | `false` | No |

See `variables.tf` for complete list.

## Outputs

### Connection Commands

After deployment, use these commands to connect:

```bash
# Configure kubectl
terraform output -raw connect_kubectl

# Port-forward for local access
terraform output -raw connect_port_forward

# Check pod status
terraform output -raw check_pods

# View logs
terraform output -raw check_logs
```

### Important URLs

| Output | Description |
|--------|-------------|
| `cluster_fqdn` | AKS cluster FQDN |
| `postgresql_fqdn` | PostgreSQL server FQDN |
| `application_url` | Application access URL |
| `kube_config_command` | Command to get kubectl credentials |

### Sensitive Outputs

These outputs contain sensitive information:

```bash
# Get LiteLLM master key
terraform output -raw litellm_master_key

# Get PostgreSQL admin password
terraform output -raw postgresql_admin_password

# Get database connection string
terraform output -raw get_database_url
```

**Warning:** Never commit sensitive outputs to version control!

## Security

### Network Security

- **Private Endpoint**: PostgreSQL is only accessible via private endpoint from the AKS subnet
- **NSG Rules**: 
  - HTTPS (443) and HTTP (80) allowed to AKS
  - PostgreSQL (5432) only from AKS subnet
  - All other inbound traffic to database subnet denied

### Authentication

- **AKS**: Uses Azure AD authentication with `kubelogin`
- **PostgreSQL**: Auto-generated strong password (32 chars)
- **LiteLLM**: Auto-generated master key and salt key (48 chars each)

### Secrets Management

Secrets are stored in Kubernetes and never logged:

```bash
# View secrets in cluster
kubectl get secrets -n acp

# Get specific secret value
kubectl get secret ai-control-plane-secrets -n acp -o jsonpath='{.data.LITELLM_MASTER_KEY}' | base64 -d
```

## Troubleshooting

### Pod Not Starting

```bash
# Check pod status
kubectl get pods -n acp

# Describe pod for events
kubectl describe pod <pod-name> -n acp

# Check logs
kubectl logs <pod-name> -n acp --previous
```

### Database Connection Issues

```bash
# Test connectivity from a pod
kubectl run -it --rm debug --image=postgres:16 --restart=Never -n acp -- psql <database-url>

# Check PostgreSQL firewall rules
az postgres flexible-server show --name <server-name> --resource-group <rg-name>
```

### Workload Identity Issues

```bash
# Check if Workload Identity is enabled
az aks show --name <cluster-name> --resource-group <rg-name> --query "oidcIssuerProfile"

# Verify service account annotations
kubectl get serviceaccount ai-control-plane-sa -n acp -o yaml
```

## Cleanup

To destroy all resources:

```bash
terraform destroy
```

**Warning:** This will permanently delete:
- AKS cluster and all workloads
- PostgreSQL database (unless `prevent_destroy` is set)
- All data in the database

Make sure to back up any important data first!

## Cost Estimation

Estimated monthly costs (as of 2024, subject to change):

| Component | Dev (B2s) | Production (D4s_v3) |
|-----------|-----------|---------------------|
| AKS (2 nodes) | ~$60 | ~$280 |
| PostgreSQL | ~$30 | ~$150 |
| Load Balancer | ~$20 | ~$20 |
| Storage | ~$5 | ~$20 |
| **Total** | **~$115** | **~$470** |

Use [Azure Pricing Calculator](https://azure.microsoft.com/en-us/pricing/calculator/) for accurate estimates.

## References

- [Azure AKS Documentation](https://docs.microsoft.com/en-us/azure/aks/)
- [Azure PostgreSQL Flexible Server](https://docs.microsoft.com/en-us/azure/postgresql/flexible-server/)
- [Azure Workload Identity](https://azure.github.io/azure-workload-identity/docs/)
- [AI Control Plane Helm Chart](../../../helm/ai-control-plane/README.md)
