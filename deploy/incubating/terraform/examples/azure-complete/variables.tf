# Azure Complete Example - Variables
# All input variables for the Azure complete infrastructure deployment

#------------------------------------------------------------------------------
# General Configuration
#------------------------------------------------------------------------------

variable "resource_group_name" {
  description = "Name of the Azure Resource Group where all resources will be created"
  type        = string
  default     = "rg-ai-control-plane"

  validation {
    condition     = can(regex("^[a-z0-9]([-a-z0-9]*[a-z0-9])?$", var.resource_group_name))
    error_message = "Resource group name must be lowercase alphanumeric characters or '-', and must start and end with an alphanumeric character."
  }
}

variable "location" {
  description = "Azure region where resources will be deployed (e.g., East US, West Europe)"
  type        = string
  default     = "East US"
}

variable "environment" {
  description = "Environment tag for all resources (dev, staging, production)"
  type        = string
  default     = "production"

  validation {
    condition     = contains(["dev", "staging", "production"], var.environment)
    error_message = "Environment must be one of: dev, staging, production."
  }
}

variable "name_prefix" {
  description = "Prefix to be used for all resource names"
  type        = string
  default     = "ai-cp"
}

variable "tags" {
  description = "Additional tags to apply to all resources"
  type        = map(string)
  default     = {}
}

#------------------------------------------------------------------------------
# Network Configuration
#------------------------------------------------------------------------------

variable "vnet_cidr" {
  description = "CIDR block for the Virtual Network"
  type        = string
  default     = "10.0.0.0/16"
}

variable "subnet_cidrs" {
  description = "Map of subnet names to CIDR blocks"
  type        = map(string)
  default = {
    aks      = "10.0.1.0/24"
    database = "10.0.2.0/24"
  }
}

#------------------------------------------------------------------------------
# AKS Configuration
#------------------------------------------------------------------------------

variable "cluster_name" {
  description = "Name of the AKS cluster"
  type        = string
  default     = "ai-control-plane"
}

variable "kubernetes_version" {
  description = "Kubernetes version for the AKS cluster"
  type        = string
  default     = "1.29"
}

variable "sku_tier" {
  description = "SKU tier for the AKS cluster (Free, Standard, or Premium)"
  type        = string
  default     = "Standard"
}

variable "system_node_pool" {
  description = "Configuration for the system node pool (for critical addons only)"
  type = object({
    name                 = optional(string, "system")
    vm_size              = optional(string, "Standard_B2s")
    node_count           = optional(number, 1)
    min_count            = optional(number, 1)
    max_count            = optional(number, 3)
    os_disk_size_gb      = optional(number, 128)
    enable_auto_scaling  = optional(bool, true)
    only_critical_addons = optional(bool, true)
    labels               = optional(map(string), {})
    taints               = optional(list(string), ["CriticalAddonsOnly=true:NoSchedule"])
  })
  default = {
    name                 = "system"
    vm_size              = "Standard_B2s"
    node_count           = 1
    min_count            = 1
    max_count            = 3
    os_disk_size_gb      = 128
    enable_auto_scaling  = true
    only_critical_addons = true
    labels               = {}
    taints               = ["CriticalAddonsOnly=true:NoSchedule"]
  }
}

variable "node_pools" {
  description = "Map of user node pools to create for workloads"
  type = map(object({
    vm_size             = string
    node_count          = optional(number, 2)
    min_count           = optional(number, 1)
    max_count           = optional(number, 5)
    os_disk_size_gb     = optional(number, 128)
    enable_auto_scaling = optional(bool, true)
    labels              = optional(map(string), {})
    taints              = optional(list(string), [])
  }))

  # Environment-specific defaults
  default = {
    "general" = {
      vm_size             = "Standard_B2s"
      node_count          = 2
      min_count           = 1
      max_count           = 5
      os_disk_size_gb     = 128
      enable_auto_scaling = true
      labels = {
        workload-type = "general"
      }
      taints = []
    }
  }
}

variable "availability_zones" {
  description = "Availability zones for AKS nodes (use empty list for regions without zones)"
  type        = list(string)
  default     = ["1", "2", "3"]
}

variable "enable_workload_identity" {
  description = "Enable Azure Workload Identity for pod authentication"
  type        = bool
  default     = true
}

variable "enable_oidc_issuer" {
  description = "Enable OIDC issuer for the cluster (required for Workload Identity)"
  type        = bool
  default     = true
}

#------------------------------------------------------------------------------
# PostgreSQL Configuration
#------------------------------------------------------------------------------

variable "postgresql_server_name" {
  description = "Name of the Azure PostgreSQL Flexible Server"
  type        = string
  default     = "litellm-db"
}

variable "postgresql_admin_username" {
  description = "Administrator username for PostgreSQL"
  type        = string
  default     = "litellm"
}

variable "postgresql_version" {
  description = "PostgreSQL version"
  type        = string
  default     = "16"
}

variable "postgresql_sku_name" {
  description = "SKU name for PostgreSQL Flexible Server"
  type        = string
  default     = "GP_Standard_D4s_v3"
}

variable "postgresql_storage_mb" {
  description = "Storage size in MB for PostgreSQL"
  type        = number
  default     = 32768
}

variable "postgresql_database_name" {
  description = "Name of the database to create"
  type        = string
  default     = "litellm"
}

variable "postgresql_backup_retention_days" {
  description = "Number of days to retain backups (7-35 days)"
  type        = number
  default     = 7
}

variable "postgresql_geo_redundant_backup_enabled" {
  description = "Enable geo-redundant backups (only for Standard/Premium SKUs)"
  type        = bool
  default     = true
}

variable "postgresql_high_availability_enabled" {
  description = "Enable high availability for PostgreSQL (only for Standard/Premium SKUs)"
  type        = bool
  default     = true
}

variable "log_analytics_workspace_id" {
  description = "Log Analytics Workspace resource ID for Defender and OMS integration"
  type        = string
  default     = ""
}

#------------------------------------------------------------------------------
# Helm Chart Configuration
#------------------------------------------------------------------------------

variable "helm_release_name" {
  description = "Name of the Helm release"
  type        = string
  default     = "acp"
}

variable "helm_namespace" {
  description = "Kubernetes namespace for the Helm release"
  type        = string
  default     = "acp"
}

variable "helm_chart_path" {
  description = "Path to the AI Control Plane Helm chart"
  type        = string
  default     = "../../../helm/ai-control-plane"
}

variable "litellm_master_key" {
  description = "Master key for LiteLLM admin authentication"
  type        = string
  sensitive   = true

  validation {
    condition     = length(trimspace(var.litellm_master_key)) >= 32 && trimspace(var.litellm_master_key) == var.litellm_master_key && can(regex("^[^[:space:]]+$", var.litellm_master_key))
    error_message = "litellm_master_key must be provided, be at least 32 characters, and contain no whitespace."
  }
}

variable "litellm_salt_key" {
  description = "Salt key for LiteLLM encryption"
  type        = string
  sensitive   = true

  validation {
    condition     = length(trimspace(var.litellm_salt_key)) >= 32 && trimspace(var.litellm_salt_key) == var.litellm_salt_key && can(regex("^[^[:space:]]+$", var.litellm_salt_key))
    error_message = "litellm_salt_key must be provided, be at least 32 characters, and contain no whitespace."
  }
}

variable "litellm_replica_count" {
  description = "Number of LiteLLM replicas"
  type        = number
  default     = 2
}

variable "ingress_enabled" {
  description = "Enable ingress for external access"
  type        = bool
  default     = false
}

variable "ingress_host" {
  description = "Hostname for the ingress"
  type        = string
  default     = ""
}

variable "ingress_class_name" {
  description = "Ingress class name (e.g., nginx, traefik)"
  type        = string
  default     = "nginx"
}

variable "ingress_tls_secret_name" {
  description = "TLS secret name for the ingress"
  type        = string
  default     = "ai-control-plane-tls"
}

variable "ingress_cluster_issuer" {
  description = "cert-manager ClusterIssuer for ingress TLS automation"
  type        = string
  default     = ""
}

variable "enable_autoscaling" {
  description = "Enable Horizontal Pod Autoscaler"
  type        = bool
  default     = true
}
