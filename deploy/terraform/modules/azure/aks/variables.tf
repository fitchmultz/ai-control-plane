# Variables for Azure AKS Module

# ---------------------------------------------------------------------------------------------------------------------
# Required Variables
# ---------------------------------------------------------------------------------------------------------------------

variable "cluster_name" {
  description = "Name of the AKS cluster"
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9][a-z0-9-]{0,58}[a-z0-9]$", var.cluster_name))
    error_message = "Cluster name must be 1-60 characters, lowercase alphanumeric and hyphens, start and end with alphanumeric."
  }
}

variable "resource_group_name" {
  description = "Name of the resource group where AKS will be deployed"
  type        = string
}

variable "location" {
  description = "Azure region where AKS will be deployed"
  type        = string
}

variable "subnet_id" {
  description = "Subnet ID for AKS nodes"
  type        = string
}

# ---------------------------------------------------------------------------------------------------------------------
# Cluster Configuration
# ---------------------------------------------------------------------------------------------------------------------

variable "kubernetes_version" {
  description = "Kubernetes version for the AKS cluster"
  type        = string
  default     = "1.29"
}

variable "sku_tier" {
  description = "SKU tier for the AKS cluster (Free or Standard)"
  type        = string
  default     = "Standard"

  validation {
    condition     = contains(["Free", "Standard", "Premium"], var.sku_tier)
    error_message = "SKU tier must be Free, Standard, or Premium."
  }
}

variable "availability_zones" {
  description = "Availability zones for AKS nodes"
  type        = list(string)
  default     = ["1", "2", "3"]
}

# ---------------------------------------------------------------------------------------------------------------------
# System Node Pool Configuration
# ---------------------------------------------------------------------------------------------------------------------

variable "system_node_pool" {
  description = "Configuration for the system node pool"
  type = object({
    name                 = optional(string, "system")
    vm_size              = optional(string, "Standard_D4s_v3")
    node_count           = optional(number, 2)
    min_count            = optional(number, 1)
    max_count            = optional(number, 4)
    os_disk_size_gb      = optional(number, 128)
    enable_auto_scaling  = optional(bool, true)
    only_critical_addons = optional(bool, true)
    labels               = optional(map(string), {})
    taints               = optional(list(string), ["CriticalAddonsOnly=true:NoSchedule"])
  })
  default = {
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
}

# ---------------------------------------------------------------------------------------------------------------------
# User Node Pools Configuration
# ---------------------------------------------------------------------------------------------------------------------

variable "node_pools" {
  description = "Map of user node pools to create"
  type = map(object({
    vm_size             = string
    node_count          = optional(number, 2)
    min_count           = optional(number, 1)
    max_count           = optional(number, 10)
    os_disk_size_gb     = optional(number, 128)
    enable_auto_scaling = optional(bool, true)
    labels              = optional(map(string), {})
    taints              = optional(list(string), [])
  }))
  default = {
    "default" = {
      vm_size             = "Standard_D4s_v3"
      node_count          = 2
      min_count           = 1
      max_count           = 10
      os_disk_size_gb     = 128
      enable_auto_scaling = true
      labels = {
        "workload-type" = "general"
      }
      taints = []
    }
  }
}

# ---------------------------------------------------------------------------------------------------------------------
# Identity Configuration
# ---------------------------------------------------------------------------------------------------------------------

variable "enable_workload_identity" {
  description = "Enable Azure Workload Identity for the cluster"
  type        = bool
  default     = true
}

variable "enable_oidc_issuer" {
  description = "Enable OIDC issuer for the cluster (required for Workload Identity)"
  type        = bool
  default     = true
}

# ---------------------------------------------------------------------------------------------------------------------
# Network Configuration
# ---------------------------------------------------------------------------------------------------------------------

variable "network_plugin" {
  description = "Network plugin for AKS (azure or kubenet)"
  type        = string
  default     = "azure"

  validation {
    condition     = contains(["azure", "kubenet", "none"], var.network_plugin)
    error_message = "Network plugin must be azure, kubenet, or none."
  }
}

variable "network_policy" {
  description = "Network policy for AKS (calico, azure, or none)"
  type        = string
  default     = "calico"

  validation {
    condition     = contains(["calico", "azure", "cilium", "none"], var.network_policy)
    error_message = "Network policy must be calico, azure, cilium, or none."
  }
}

variable "load_balancer_sku" {
  description = "Load balancer SKU (basic or standard)"
  type        = string
  default     = "standard"

  validation {
    condition     = contains(["basic", "standard"], var.load_balancer_sku)
    error_message = "Load balancer SKU must be basic or standard."
  }
}

variable "service_cidr" {
  description = "CIDR for Kubernetes services"
  type        = string
  default     = "10.0.0.0/16"
}

variable "dns_service_ip" {
  description = "IP address within the service CIDR for DNS"
  type        = string
  default     = "10.0.0.10"
}

variable "authorized_ip_ranges" {
  description = "Authorized IP ranges for API server access"
  type        = list(string)
  default     = []
}

variable "vnet_integration_enabled" {
  description = "Enable VNet integration for API server"
  type        = bool
  default     = false
}

variable "acr_id" {
  description = "Resource ID of Azure Container Registry for AcrPull role assignment"
  type        = string
  default     = null
}

# ---------------------------------------------------------------------------------------------------------------------
# Auto Scaler Profile
# ---------------------------------------------------------------------------------------------------------------------

variable "auto_scaler_profile" {
  description = "Auto-scaler profile configuration"
  type = object({
    balance_similar_node_groups      = optional(bool, false)
    expander                         = optional(string, "random")
    max_graceful_termination_sec     = optional(number, 600)
    max_node_provision_time          = optional(string, "15m")
    max_unready_nodes                = optional(number, 3)
    max_unready_percentage             = optional(number, 45)
    new_pod_scale_up_delay           = optional(string, "0s")
    scale_down_delay_after_add       = optional(string, "10m")
    scale_down_delay_after_delete    = optional(string, "10s")
    scale_down_delay_after_failure   = optional(string, "3m")
    scan_interval                    = optional(string, "10s")
    scale_down_unneeded              = optional(string, "10m")
    scale_down_unready               = optional(string, "20m")
    scale_down_utilization_threshold = optional(number, 0.5)
  })
  default = null
}

# ---------------------------------------------------------------------------------------------------------------------
# Maintenance Window
# ---------------------------------------------------------------------------------------------------------------------

variable "maintenance_window" {
  description = "Maintenance window configuration"
  type = object({
    allowed = optional(list(object({
      day   = string
      hours = list(number)
    })), [])
    not_allowed = optional(list(object({
      start = string
      end   = string
    })), [])
  })
  default = null
}

# ---------------------------------------------------------------------------------------------------------------------
# Monitoring and Security
# ---------------------------------------------------------------------------------------------------------------------

variable "enable_microsoft_defender" {
  description = "Enable Microsoft Defender for Containers"
  type        = bool
  default     = false
}

variable "enable_oms_agent" {
  description = "Enable OMS agent for monitoring"
  type        = bool
  default     = false
}

variable "log_analytics_workspace_id" {
  description = "Log Analytics Workspace ID for monitoring"
  type        = string
  default     = null
}

variable "enable_azure_policy" {
  description = "Enable Azure Policy for the cluster"
  type        = bool
  default     = false
}

variable "enable_key_vault_secrets_provider" {
  description = "Enable Azure Key Vault Secrets Provider"
  type        = bool
  default     = false
}

variable "key_vault_secrets_provider_secret_rotation" {
  description = "Enable secret rotation for Key Vault Secrets Provider"
  type        = bool
  default     = true
}

variable "key_vault_secrets_provider_rotation_interval" {
  description = "Secret rotation interval for Key Vault Secrets Provider"
  type        = string
  default     = "2m"
}

# ---------------------------------------------------------------------------------------------------------------------
# Tags
# ---------------------------------------------------------------------------------------------------------------------

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}
