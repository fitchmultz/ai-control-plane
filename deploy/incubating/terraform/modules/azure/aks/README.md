# Azure AKS Terraform Module

Terraform module for deploying Azure Kubernetes Service (AKS) clusters with support for Workload Identity, OIDC, and flexible node pool configuration.

## Features

- **AKS Cluster** with configurable Kubernetes version (default: 1.29)
- **System Node Pool** for critical cluster components
- **User Node Pools** with configurable scaling, labels, and taints
- **Azure Workload Identity** support (enabled by default)
- **OIDC Issuer** for pod identity (enabled by default)
- **User Assigned Managed Identity (UAMI)** for secure authentication
- **Auto-scaling** support for all node pools
- **Network policies** (Calico/Azure CNI)
- **Availability Zones** support
- **ACR integration** with automatic role assignment

## Requirements

| Name | Version |
|------|---------|
| terraform | >= 1.5.0 |
| azurerm | >= 3.80.0 |

## Usage

### Basic Example

```hcl
module "aks" {
  source = "./modules/azure/aks"

  cluster_name        = "my-aks-cluster"
  resource_group_name = "my-resource-group"
  location            = "westus2"
  subnet_id           = "/subscriptions/xxx/resourceGroups/xxx/providers/Microsoft.Network/virtualNetworks/xxx/subnets/aks"

  kubernetes_version = "1.29"

  system_node_pool = {
    name                 = "system"
    vm_size              = "Standard_D4s_v3"
    node_count           = 2
    min_count            = 1
    max_count            = 4
    os_disk_size_gb      = 128
    enable_auto_scaling  = true
    only_critical_addons = true
    labels               = {}
    taints               = ["CriticalAddonsOnly=true:NoSchedule"]
  }

  node_pools = {
    "general" = {
      vm_size             = "Standard_D4s_v3"
      node_count          = 3
      min_count           = 1
      max_count           = 10
      os_disk_size_gb     = 128
      enable_auto_scaling = true
      labels = {
        "workload-type" = "general"
      }
      taints = []
    }
    "gpu" = {
      vm_size             = "Standard_NC6s_v3"
      node_count          = 1
      min_count           = 0
      max_count           = 4
      os_disk_size_gb     = 256
      enable_auto_scaling = true
      labels = {
        "workload-type" = "gpu"
        "nvidia.com/gpu" = "true"
      }
      taints = ["nvidia.com/gpu=true:NoSchedule"]
    }
  }

  enable_workload_identity = true
  enable_oidc_issuer       = true

  acr_id = "/subscriptions/xxx/resourceGroups/xxx/providers/Microsoft.ContainerRegistry/registries/myacr"

  tags = {
    Environment = "production"
    Team        = "platform"
  }
}
```

### Workload Identity Example

```hcl
module "aks" {
  source = "./modules/azure/aks"

  cluster_name        = "aks-workload-id"
  resource_group_name = "rg-aks"
  location            = "eastus"
  subnet_id           = azurerm_subnet.aks.id

  enable_workload_identity = true
  enable_oidc_issuer       = true

  node_pools = {
    "workload" = {
      vm_size             = "Standard_D8s_v3"
      node_count          = 2
      min_count           = 2
      max_count           = 10
      os_disk_size_gb     = 128
      enable_auto_scaling = true
      labels = {}
      taints = []
    }
  }
}

# Create a managed identity for your workload
resource "azurerm_user_assigned_identity" "app" {
  name                = "my-app-identity"
  location            = module.aks.location
  resource_group_name = "rg-aks"
}

# Federated credential for Workload Identity
resource "azurerm_federated_identity_credential" "app" {
  name                = "my-app-federated-credential"
  resource_group_name = "rg-aks"
  parent_id           = azurerm_user_assigned_identity.app.id
  issuer              = module.aks.oidc_issuer_url
  subject             = "system:serviceaccount:default:my-app-sa"
  audiences           = ["api://AzureADTokenExchange"]
}
```

## Inputs

### Required Variables

| Name | Description | Type |
|------|-------------|------|
| `cluster_name` | Name of the AKS cluster | `string` |
| `resource_group_name` | Name of the resource group | `string` |
| `location` | Azure region | `string` |
| `subnet_id` | Subnet ID for AKS nodes | `string` |

### Optional Variables

| Name | Description | Type | Default |
|------|-------------|------|---------|
| `kubernetes_version` | Kubernetes version | `string` | `"1.29"` |
| `sku_tier` | SKU tier (Free/Standard/Premium) | `string` | `"Standard"` |
| `availability_zones` | Availability zones | `list(string)` | `["1", "2", "3"]` |
| `network_plugin` | Network plugin | `string` | `"azure"` |
| `network_policy` | Network policy | `string` | `"calico"` |
| `load_balancer_sku` | Load balancer SKU | `string` | `"standard"` |
| `service_cidr` | Service CIDR | `string` | `"10.0.0.0/16"` |
| `dns_service_ip` | DNS service IP | `string` | `"10.0.0.10"` |
| `enable_workload_identity` | Enable Workload Identity | `bool` | `true` |
| `enable_oidc_issuer` | Enable OIDC issuer | `bool` | `true` |
| `enable_microsoft_defender` | Enable Microsoft Defender | `bool` | `false` |
| `enable_oms_agent` | Enable OMS agent | `bool` | `false` |
| `enable_azure_policy` | Enable Azure Policy | `bool` | `false` |
| `enable_key_vault_secrets_provider` | Enable Key Vault Secrets Provider | `bool` | `false` |
| `acr_id` | ACR resource ID for AcrPull role | `string` | `null` |
| `tags` | Tags for resources | `map(string)` | `{}` |

### Node Pool Variables

#### `system_node_pool`

| Name | Description | Type | Default |
|------|-------------|------|---------|
| `name` | Node pool name | `string` | `"system"` |
| `vm_size` | VM size | `string` | `"Standard_D4s_v3"` |
| `node_count` | Node count | `number` | `2` |
| `min_count` | Min nodes (for auto-scaling) | `number` | `1` |
| `max_count` | Max nodes (for auto-scaling) | `number` | `4` |
| `os_disk_size_gb` | OS disk size | `number` | `128` |
| `enable_auto_scaling` | Enable auto-scaling | `bool` | `true` |
| `only_critical_addons` | Only critical addons | `bool` | `true` |
| `labels` | Node labels | `map(string)` | `{}` |
| `taints` | Node taints | `list(string)` | `["CriticalAddonsOnly=true:NoSchedule"]` |

#### `node_pools`

Map of user node pools with the following attributes:

| Name | Description | Type | Default |
|------|-------------|------|---------|
| `vm_size` | VM size | `string` | (required) |
| `node_count` | Node count | `number` | `2` |
| `min_count` | Min nodes | `number` | `1` |
| `max_count` | Max nodes | `number` | `10` |
| `os_disk_size_gb` | OS disk size | `number` | `128` |
| `enable_auto_scaling` | Enable auto-scaling | `bool` | `true` |
| `labels` | Node labels | `map(string)` | `{}` |
| `taints` | Node taints | `list(string)` | `[]` |

## Outputs

| Name | Description | Sensitive |
|------|-------------|-----------|
| `cluster_id` | The AKS cluster ID | no |
| `cluster_name` | The cluster name | no |
| `kube_config_raw` | Raw kubeconfig for kubectl | **yes** |
| `host` | Kubernetes API server host | no |
| `client_certificate` | Client certificate for authentication | **yes** |
| `client_key` | Client key for authentication | **yes** |
| `cluster_ca_certificate` | Cluster CA certificate | **yes** |
| `oidc_issuer_url` | OIDC issuer URL for Workload Identity | no |
| `oidc_issuer_enabled` | Whether OIDC issuer is enabled | no |
| `workload_identity_enabled` | Whether Workload Identity is enabled | no |
| `fqdn` | Cluster FQDN | no |
| `private_fqdn` | Private FQDN | no |
| `node_resource_group` | Node resource group name | no |
| `control_plane_identity` | Control plane UAMI details | no |
| `kubelet_identity` | Kubelet UAMI details | no |
| `node_pools` | Map of user node pools | no |
| `system_node_pool` | System node pool details | no |

## Important Notes

### Workload Identity Setup

After deploying the cluster with Workload Identity enabled:

1. Install the Azure Workload Identity webhook:
   ```bash
   helm repo add azure-workload-identity https://azure.github.io/azure-workload-identity/charts
   helm install workload-identity-webhook azure-workload-identity/workload-identity-webhook \
     --namespace azure-workload-identity-system \
     --create-namespace \
     --set azureTenantID="YOUR_TENANT_ID"
   ```

2. Create a service account with the workload identity annotation:
   ```yaml
   apiVersion: v1
   kind: ServiceAccount
   metadata:
     name: my-app-sa
     annotations:
       azure.workload.identity/client-id: "MANAGED_IDENTITY_CLIENT_ID"
   ```

3. Use the service account in your pod spec.

### Security Considerations

- The module uses **User Assigned Managed Identities** for both control plane and kubelet
- System node pool is configured with `only_critical_addons_enabled` by default
- HTTP application routing is disabled for security
- ACR integration automatically assigns `AcrPull` role to kubelet identity

### Version Management

The module ignores changes to `kubernetes_version` to allow controlled upgrades. To upgrade:

1. Update the `kubernetes_version` variable
2. Run `terraform apply`
3. The change will be detected and applied

## License

Apache-2.0 - See the main project [LICENSE](../../../../../../LICENSE) and [NOTICE](../../../../../../NOTICE) files for details.
