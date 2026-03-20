# -----------------------------------------------------------------------------
# GCP Complete Example - Variables
# -----------------------------------------------------------------------------
# This file defines all input variables for the GCP Complete Example.
# See terraform.tfvars.example for sample values.
# -----------------------------------------------------------------------------

# -----------------------------------------------------------------------------
# Required Variables
# -----------------------------------------------------------------------------

variable "project_id" {
  description = "GCP project ID where all resources will be created (required)"
  type        = string

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "Project ID must be a valid GCP project ID (6-30 lowercase letters, digits, or hyphens)."
  }
}

# -----------------------------------------------------------------------------
# General Configuration
# -----------------------------------------------------------------------------

variable "region" {
  description = "GCP region for all resources"
  type        = string
  default     = "us-central1"
}

variable "environment" {
  description = "Environment tag (dev, staging, or production)"
  type        = string
  default     = "production"

  validation {
    condition     = contains(["dev", "staging", "production"], var.environment)
    error_message = "Environment must be one of: dev, staging, production."
  }
}

variable "name_prefix" {
  description = "Prefix for all resource names"
  type        = string
  default     = "ai-cp"
}

# -----------------------------------------------------------------------------
# VPC Configuration
# -----------------------------------------------------------------------------

variable "vpc_cidr" {
  description = "CIDR block for the VPC network"
  type        = string
  default     = "10.0.0.0/16"
}

variable "gke_subnet_cidr" {
  description = "CIDR block for the GKE subnet"
  type        = string
  default     = "10.0.0.0/20"
}

variable "pods_ip_range" {
  description = "IP range for GKE pods (secondary range)"
  type = object({
    name = string
    cidr = string
  })
  default = {
    name = "pods"
    cidr = "10.4.0.0/14"
  }
}

variable "services_ip_range" {
  description = "IP range for GKE services (secondary range)"
  type = object({
    name = string
    cidr = string
  })
  default = {
    name = "services"
    cidr = "10.0.32.0/20"
  }
}

# -----------------------------------------------------------------------------
# GKE Configuration
# -----------------------------------------------------------------------------

variable "kubernetes_version" {
  description = "Kubernetes version for the GKE cluster"
  type        = string
  default     = "1.29"
}

variable "release_channel" {
  description = "GKE release channel (UNSPECIFIED, RAPID, REGULAR, STABLE)"
  type        = string
  default     = "REGULAR"
}

variable "node_pools" {
  description = <<EOF
Map of node pool configurations. Each node pool can have:
- machine_type: Machine type for nodes (default: e2-medium)
- initial_node_count: Initial number of nodes (default: 1)
- min_count: Minimum number of nodes for autoscaling
- max_count: Maximum number of nodes for autoscaling
- disk_size_gb: Disk size in GB (default: 100)
- spot: Use spot VMs (default: false)
- labels: Map of labels to apply to nodes
EOF

  type = map(object({
    machine_type       = optional(string, "e2-medium")
    initial_node_count = optional(number, 1)
    min_count          = optional(number, 1)
    max_count          = optional(number, 3)
    disk_size_gb       = optional(number, 100)
    spot               = optional(bool, false)
    labels             = optional(map(string), {})
  }))

  default = {}
}

variable "enable_private_nodes" {
  description = "Enable private nodes (nodes have only private IPs)"
  type        = bool
  default     = true
}

variable "master_ipv4_cidr_block" {
  description = "CIDR block for the GKE master endpoint"
  type        = string
  default     = "172.16.0.0/28"
}

variable "master_authorized_networks" {
  description = "List of authorized networks for GKE master access"
  type = list(object({
    cidr_block   = string
    display_name = string
  }))
  default = []
}

# -----------------------------------------------------------------------------
# Cloud SQL Configuration
# -----------------------------------------------------------------------------

variable "cloudsql_tier" {
  description = "Cloud SQL machine type tier (overridden per environment if not set)"
  type        = string
  default     = null
}

variable "cloudsql_disk_size" {
  description = "Cloud SQL initial disk size in GB"
  type        = number
  default     = 20
}

variable "cloudsql_disk_autoresize" {
  description = "Enable automatic disk resizing for Cloud SQL"
  type        = bool
  default     = true
}

variable "cloudsql_availability_type" {
  description = "Cloud SQL availability type (ZONAL or REGIONAL)"
  type        = string
  default     = null
}

variable "cloudsql_backup_enabled" {
  description = "Enable automated backups for Cloud SQL"
  type        = bool
  default     = true
}

variable "cloudsql_backup_retention" {
  description = "Number of backups to retain"
  type        = number
  default     = 7
}

variable "database_name" {
  description = "Name of the database to create in Cloud SQL"
  type        = string
  default     = "litellm"
}

variable "database_user" {
  description = "Name of the database user"
  type        = string
  default     = "litellm"
}

# -----------------------------------------------------------------------------
# Helm Release Configuration
# -----------------------------------------------------------------------------

variable "helm_release_name" {
  description = "Name of the Helm release"
  type        = string
  default     = "acp"
}

variable "namespace" {
  description = "Kubernetes namespace for the AI Control Plane"
  type        = string
  default     = "acp"
}

variable "litellm_master_key" {
  description = "LiteLLM master key for authentication"
  type        = string
  sensitive   = true

  validation {
    condition     = length(trimspace(var.litellm_master_key)) >= 32 && trimspace(var.litellm_master_key) == var.litellm_master_key && can(regex("^[^[:space:]]+$", var.litellm_master_key))
    error_message = "litellm_master_key must be provided, be at least 32 characters, and contain no whitespace."
  }
}

variable "litellm_salt_key" {
  description = "LiteLLM salt key for encryption"
  type        = string
  sensitive   = true

  validation {
    condition     = length(trimspace(var.litellm_salt_key)) >= 32 && trimspace(var.litellm_salt_key) == var.litellm_salt_key && can(regex("^[^[:space:]]+$", var.litellm_salt_key))
    error_message = "litellm_salt_key must be provided, be at least 32 characters, and contain no whitespace."
  }
}

variable "ingress_enabled" {
  description = "Enable ingress for the AI Control Plane"
  type        = bool
  default     = false
}

variable "ingress_host" {
  description = "Hostname for the ingress"
  type        = string
  default     = ""
}

variable "ingress_class_name" {
  description = "Ingress class name (e.g., nginx, traefik, gce)"
  type        = string
  default     = "nginx"
}

variable "ingress_tls_secret_name" {
  description = "TLS secret name for the ingress"
  type        = string
  default     = "ai-control-plane-tls"
}

variable "ingress_cluster_issuer" {
  description = "cert-manager ClusterIssuer for TLS automation"
  type        = string
  default     = ""
}

# -----------------------------------------------------------------------------
# Workload Identity Configuration
# -----------------------------------------------------------------------------

variable "enable_workload_identity" {
  description = "Enable Workload Identity for the GKE cluster"
  type        = bool
  default     = true
}

# -----------------------------------------------------------------------------
# Labels
# -----------------------------------------------------------------------------

variable "common_labels" {
  description = "Common labels to apply to all resources"
  type        = map(string)
  default     = {}
}
