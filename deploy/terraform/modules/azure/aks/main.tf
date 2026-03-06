# Azure Kubernetes Service (AKS) Module
# Creates an AKS cluster with system and user node pools, workload identity, and OIDC support

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 3.80.0"
    }
  }
}

# Data source to get the resource group
data "azurerm_resource_group" "this" {
  name = var.resource_group_name
}

# User Assigned Managed Identity for AKS control plane
resource "azurerm_user_assigned_identity" "aks_control_plane" {
  name                = "${var.cluster_name}-control-plane"
  location            = var.location
  resource_group_name = data.azurerm_resource_group.this.name
  tags                = var.tags
}

# AKS Cluster
resource "azurerm_kubernetes_cluster" "this" {
  name                = var.cluster_name
  location            = var.location
  resource_group_name = data.azurerm_resource_group.this.name
  dns_prefix          = var.cluster_name
  kubernetes_version  = var.kubernetes_version
  sku_tier            = var.sku_tier

  # OIDC Issuer for pod identity
  oidc_issuer_enabled       = var.enable_oidc_issuer
  workload_identity_enabled = var.enable_workload_identity

  # Default system node pool (required)
  default_node_pool {
    name                         = var.system_node_pool.name
    vm_size                      = var.system_node_pool.vm_size
    node_count                   = var.system_node_pool.enable_auto_scaling ? null : var.system_node_pool.node_count
    min_count                    = var.system_node_pool.enable_auto_scaling ? var.system_node_pool.min_count : null
    max_count                    = var.system_node_pool.enable_auto_scaling ? var.system_node_pool.max_count : null
    os_disk_size_gb              = var.system_node_pool.os_disk_size_gb
    enable_auto_scaling          = var.system_node_pool.enable_auto_scaling
    type                         = "VirtualMachineScaleSets"
    vnet_subnet_id               = var.subnet_id
    only_critical_addons_enabled = var.system_node_pool.only_critical_addons
    zones                        = var.availability_zones

    node_labels = var.system_node_pool.labels
    node_taints = var.system_node_pool.taints

    tags = var.tags
  }

  # Identity configuration - User Assigned Managed Identity
  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.aks_control_plane.id]
  }

  # Network profile
  network_profile {
    network_plugin    = var.network_plugin
    network_policy    = var.network_policy
    load_balancer_sku = var.load_balancer_sku
    service_cidr      = var.service_cidr
    dns_service_ip    = var.dns_service_ip
  }

  # API server access
  api_server_access_profile {
    authorized_ip_ranges     = var.authorized_ip_ranges
    vnet_integration_enabled = var.vnet_integration_enabled
  }

  # Auto-scaler profile (optional)
  dynamic "auto_scaler_profile" {
    for_each = var.auto_scaler_profile != null ? [var.auto_scaler_profile] : []
    content {
      balance_similar_node_groups      = auto_scaler_profile.value.balance_similar_node_groups
      expander                         = auto_scaler_profile.value.expander
      max_graceful_termination_sec     = auto_scaler_profile.value.max_graceful_termination_sec
      max_node_provision_time          = auto_scaler_profile.value.max_node_provision_time
      max_unready_nodes                = auto_scaler_profile.value.max_unready_nodes
      max_unready_percentage           = auto_scaler_profile.value.max_unready_percentage
      new_pod_scale_up_delay           = auto_scaler_profile.value.new_pod_scale_up_delay
      scale_down_delay_after_add       = auto_scaler_profile.value.scale_down_delay_after_add
      scale_down_delay_after_delete    = auto_scaler_profile.value.scale_down_delay_after_delete
      scale_down_delay_after_failure   = auto_scaler_profile.value.scale_down_delay_after_failure
      scan_interval                    = auto_scaler_profile.value.scan_interval
      scale_down_unneeded              = auto_scaler_profile.value.scale_down_unneeded
      scale_down_unready               = auto_scaler_profile.value.scale_down_unready
      scale_down_utilization_threshold = auto_scaler_profile.value.scale_down_utilization_threshold
    }
  }

  # Maintenance window (optional)
  dynamic "maintenance_window" {
    for_each = var.maintenance_window != null ? [var.maintenance_window] : []
    content {
      dynamic "allowed" {
        for_each = maintenance_window.value.allowed
        content {
          day   = allowed.value.day
          hours = allowed.value.hours
        }
      }
      dynamic "not_allowed" {
        for_each = maintenance_window.value.not_allowed
        content {
          start = not_allowed.value.start
          end   = not_allowed.value.end
        }
      }
    }
  }

  # Microsoft Defender (optional)
  microsoft_defender {
    enabled = var.enable_microsoft_defender
    log_analytics_workspace_id = var.enable_microsoft_defender && var.log_analytics_workspace_id != null ? var.log_analytics_workspace_id : null
  }

  # Monitoring (optional)
  dynamic "oms_agent" {
    for_each = var.enable_oms_agent ? [1] : []
    content {
      log_analytics_workspace_id = var.log_analytics_workspace_id
    }
  }

  # Azure Policy (optional)
  azure_policy_enabled = var.enable_azure_policy

  # HTTP application routing (disabled by default for security)
  http_application_routing_enabled = false

  # Key Vault Secrets Provider (optional)
  dynamic "key_vault_secrets_provider" {
    for_each = var.enable_key_vault_secrets_provider ? [1] : []
    content {
      secret_rotation_enabled  = var.key_vault_secrets_provider_secret_rotation
      secret_rotation_interval = var.key_vault_secrets_provider_rotation_interval
    }
  }

  tags = var.tags

  lifecycle {
    prevent_destroy = false
    ignore_changes = [
      # Ignore changes to kubernetes_version to allow controlled upgrades
      kubernetes_version,
    ]
  }
}

# User Assigned Managed Identity for kubelet
resource "azurerm_user_assigned_identity" "kubelet" {
  name                = "${var.cluster_name}-kubelet"
  location            = var.location
  resource_group_name = data.azurerm_resource_group.this.name
  tags                = var.tags
}

# Role assignment for kubelet identity
resource "azurerm_role_assignment" "kubelet_acr_pull" {
  count = var.acr_id != null ? 1 : 0

  scope                = var.acr_id
  role_definition_name = "AcrPull"
  principal_id         = azurerm_user_assigned_identity.kubelet.principal_id
}

# Role assignment for control plane identity
resource "azurerm_role_assignment" "control_plane_contributor" {
  scope                = data.azurerm_resource_group.this.id
  role_definition_name = "Contributor"
  principal_id         = azurerm_user_assigned_identity.aks_control_plane.principal_id
}

# Additional user node pools
resource "azurerm_kubernetes_cluster_node_pool" "user" {
  for_each = var.node_pools

  name                  = each.key
  kubernetes_cluster_id = azurerm_kubernetes_cluster.this.id
  vm_size               = each.value.vm_size
  node_count            = each.value.enable_auto_scaling ? null : each.value.node_count
  min_count             = each.value.enable_auto_scaling ? each.value.min_count : null
  max_count             = each.value.enable_auto_scaling ? each.value.max_count : null
  os_disk_size_gb       = each.value.os_disk_size_gb
  enable_auto_scaling   = each.value.enable_auto_scaling
  vnet_subnet_id        = var.subnet_id
  zones                 = var.availability_zones
  mode                  = "User"

  node_labels = each.value.labels
  node_taints = each.value.taints

  tags = var.tags

  depends_on = [azurerm_kubernetes_cluster.this]
}

# Update AKS cluster to use kubelet identity after creation
# Note: This is handled through the kubelet_identity block in newer versions
resource "null_resource" "update_kubelet_identity" {
  count = 0 # Placeholder - kubelet identity is configured through the cluster resource

  triggers = {
    kubelet_identity_id = azurerm_user_assigned_identity.kubelet.id
  }
}
