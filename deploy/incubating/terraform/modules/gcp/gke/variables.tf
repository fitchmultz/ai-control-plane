# -----------------------------------------------------------------------------
# Required Variables
# -----------------------------------------------------------------------------

variable "cluster_name" {
  description = "Name of the GKE cluster"
  type        = string
}

variable "project_id" {
  description = "GCP project ID where the cluster will be created"
  type        = string
}

variable "region" {
  description = "GCP region for the cluster"
  type        = string
}

variable "network" {
  description = "VPC network self_link where the cluster will be deployed"
  type        = string
}

variable "subnetwork" {
  description = "Subnetwork self_link where the cluster will be deployed"
  type        = string
}

variable "pods_secondary_range_name" {
  description = "Name of the secondary IP range for pods"
  type        = string
}

variable "services_secondary_range_name" {
  description = "Name of the secondary IP range for services"
  type        = string
}

# -----------------------------------------------------------------------------
# Cluster Configuration
# -----------------------------------------------------------------------------

variable "kubernetes_version" {
  description = "Kubernetes version for the cluster. Use 'latest' for the latest stable version."
  type        = string
  default     = "1.29"
}

variable "release_channel" {
  description = "Release channel for the cluster (UNSPECIFIED, RAPID, REGULAR, or STABLE)"
  type        = string
  default     = "REGULAR"

  validation {
    condition     = contains(["UNSPECIFIED", "RAPID", "REGULAR", "STABLE"], var.release_channel)
    error_message = "Release channel must be one of: UNSPECIFIED, RAPID, REGULAR, STABLE."
  }
}

variable "description" {
  description = "Description of the GKE cluster"
  type        = string
  default     = "GKE cluster managed by Terraform"
}

# -----------------------------------------------------------------------------
# Node Pool Configuration
# -----------------------------------------------------------------------------

variable "node_pools" {
  description = <<EOF
Map of node pool configurations. Each node pool can have the following attributes:
- machine_type: Machine type for nodes (default: e2-medium)
- initial_node_count: Initial number of nodes (default: 1)
- min_count: Minimum number of nodes for autoscaling (optional)
- max_count: Maximum number of nodes for autoscaling (optional)
- disk_size_gb: Disk size in GB (default: 100)
- disk_type: Disk type (default: pd-balanced)
- preemptible: Use preemptible VMs (default: false)
- spot: Use spot VMs (default: false)
- labels: Map of labels to apply to nodes (default: {})
- taints: List of taint objects with key, value, and effect (default: [])
- max_surge: Maximum surge during upgrades (default: 1)
- max_unavailable: Maximum unavailable during upgrades (default: 0)
- enable_gcfs: Enable Google Container File System (default: false)
- enable_gvnic: Enable gVNIC (default: true)
- enable_confidential_nodes: Enable confidential nodes (default: false)
- reservation_affinity_type: Reservation affinity type (default: NO_RESERVATION)
- network_tags: List of network tags (default: [])
EOF

  type = map(object({
    machine_type       = optional(string, "e2-medium")
    initial_node_count = optional(number, 1)
    min_count          = optional(number)
    max_count          = optional(number)
    disk_size_gb       = optional(number, 100)
    disk_type          = optional(string, "pd-balanced")
    preemptible        = optional(bool, false)
    spot               = optional(bool, false)
    labels             = optional(map(string), {})
    taints = optional(list(object({
      key    = string
      value  = string
      effect = string
    })), [])
    max_surge                 = optional(number, 1)
    max_unavailable           = optional(number, 0)
    enable_gcfs               = optional(bool, false)
    enable_gvnic              = optional(bool, true)
    enable_confidential_nodes = optional(bool, false)
    reservation_affinity_type = optional(string, "NO_RESERVATION")
    network_tags              = optional(list(string), [])
  }))

  default = {
    "default" = {
      machine_type              = "e2-medium"
      initial_node_count        = 1
      min_count                 = 1
      max_count                 = 3
      disk_size_gb              = 100
      disk_type                 = "pd-balanced"
      preemptible               = false
      spot                      = false
      labels                    = {}
      taints                    = []
      max_surge                 = 1
      max_unavailable           = 0
      enable_gcfs               = false
      enable_gvnic              = true
      enable_confidential_nodes = false
      reservation_affinity_type = "NO_RESERVATION"
      network_tags              = []
    }
  }
}

# -----------------------------------------------------------------------------
# Networking Configuration
# -----------------------------------------------------------------------------

variable "enable_private_nodes" {
  description = "Enable private nodes (nodes have only private IPs)"
  type        = bool
  default     = true
}

variable "master_ipv4_cidr_block" {
  description = "CIDR block for the master endpoint (used when enable_private_nodes is true)"
  type        = string
  default     = "172.16.0.0/28"
}

variable "master_authorized_networks" {
  description = "List of authorized networks that can access the master endpoint"
  type = list(object({
    cidr_block   = string
    display_name = string
  }))
  default = []
}

# -----------------------------------------------------------------------------
# Workload Identity Configuration
# -----------------------------------------------------------------------------

variable "enable_workload_identity" {
  description = "Enable Workload Identity for the cluster"
  type        = bool
  default     = true
}

variable "workload_identity_bindings" {
  description = <<EOF
Map of Workload Identity bindings to create. Each binding requires:
- google_service_account: The GCP service account email
- namespace: The Kubernetes namespace
- k8s_service_account: The Kubernetes service account name
EOF
  type = map(object({
    google_service_account = string
    namespace              = string
    k8s_service_account    = string
  }))
  default = {}
}

# -----------------------------------------------------------------------------
# Maintenance Configuration
# -----------------------------------------------------------------------------

variable "maintenance_start_time" {
  description = "Start time for maintenance window (RFC 3339 format)"
  type        = string
  default     = "2024-01-01T06:00:00Z"
}

variable "maintenance_end_time" {
  description = "End time for maintenance window (RFC 3339 format)"
  type        = string
  default     = "2024-01-01T12:00:00Z"
}

variable "maintenance_recurrence" {
  description = "Recurrence rule for maintenance window (RFC 5545 RRULE format)"
  type        = string
  default     = "FREQ=WEEKLY;BYDAY=SA,SU"
}

# -----------------------------------------------------------------------------
# Autoscaling Configuration
# -----------------------------------------------------------------------------

variable "enable_cluster_autoscaling" {
  description = "Enable cluster autoscaling (node auto-provisioning)"
  type        = bool
  default     = false
}

variable "cluster_autoscaling_min_cpu" {
  description = "Minimum CPU cores for cluster autoscaling"
  type        = number
  default     = 2
}

variable "cluster_autoscaling_max_cpu" {
  description = "Maximum CPU cores for cluster autoscaling"
  type        = number
  default     = 100
}

variable "cluster_autoscaling_min_memory" {
  description = "Minimum memory (GB) for cluster autoscaling"
  type        = number
  default     = 4
}

variable "cluster_autoscaling_max_memory" {
  description = "Maximum memory (GB) for cluster autoscaling"
  type        = number
  default     = 400
}

# -----------------------------------------------------------------------------
# Security Configuration
# -----------------------------------------------------------------------------

variable "enable_binary_authorization" {
  description = "Enable Binary Authorization for the cluster"
  type        = bool
  default     = true
}

variable "enable_vertical_pod_autoscaling" {
  description = "Enable Vertical Pod Autoscaling"
  type        = bool
  default     = true
}

# -----------------------------------------------------------------------------
# DNS Configuration
# -----------------------------------------------------------------------------

variable "cluster_dns_provider" {
  description = "DNS provider for the cluster (PLATFORM_DEFAULT, CLOUD_DNS, or NONE)"
  type        = string
  default     = "PLATFORM_DEFAULT"

  validation {
    condition     = contains(["PLATFORM_DEFAULT", "CLOUD_DNS", "NONE"], var.cluster_dns_provider)
    error_message = "DNS provider must be one of: PLATFORM_DEFAULT, CLOUD_DNS, NONE."
  }
}

variable "cluster_dns_scope" {
  description = "DNS scope for the cluster (CLUSTER_SCOPE or VPC_SCOPE)"
  type        = string
  default     = "CLUSTER_SCOPE"

  validation {
    condition     = contains(["CLUSTER_SCOPE", "VPC_SCOPE"], var.cluster_dns_scope)
    error_message = "DNS scope must be one of: CLUSTER_SCOPE, VPC_SCOPE."
  }
}

variable "cluster_dns_domain" {
  description = "DNS domain for the cluster"
  type        = string
  default     = "cluster.local"
}

# -----------------------------------------------------------------------------
# Observability Configuration
# -----------------------------------------------------------------------------

variable "logging_components" {
  description = "List of logging components to enable"
  type        = list(string)
  default     = ["SYSTEM_COMPONENTS", "WORKLOADS"]
}

variable "monitoring_components" {
  description = "List of monitoring components to enable"
  type        = list(string)
  default     = ["SYSTEM_COMPONENTS", "APISERVER", "CONTROLLER_MANAGER", "SCHEDULER"]
}

variable "enable_managed_prometheus" {
  description = "Enable Google Cloud Managed Service for Prometheus"
  type        = bool
  default     = true
}

# -----------------------------------------------------------------------------
# Resource Labels
# -----------------------------------------------------------------------------

variable "labels" {
  description = "Labels to apply to the cluster and other resources"
  type        = map(string)
  default     = {}
}
