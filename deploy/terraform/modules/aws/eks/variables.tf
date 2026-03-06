#-------------------------------------------------------------------------------# Required Variables
#-------------------------------------------------------------------------------

variable "cluster_name" {
  description = "Name of the EKS cluster"
  type        = string

  validation {
    condition     = can(regex("^[a-zA-Z0-9][a-zA-Z0-9-_]*$", var.cluster_name))
    error_message = "Cluster name must start with alphanumeric and contain only alphanumeric characters, hyphens, and underscores."
  }
}

variable "vpc_id" {
  description = "VPC ID where the cluster and nodes will be deployed"
  type        = string
}

variable "subnet_ids" {
  description = "List of subnet IDs. For private clusters, use private subnets."
  type        = list(string)
}

#-------------------------------------------------------------------------------# Cluster Configuration
#-------------------------------------------------------------------------------

variable "cluster_version" {
  description = "Kubernetes version for the EKS cluster"
  type        = string
  default     = "1.29"
}

variable "cluster_enabled_log_types" {
  description = "List of cluster control plane logging types to enable"
  type        = list(string)
  default     = ["api", "audit", "authenticator", "controllerManager", "scheduler"]
}

variable "cluster_endpoint_public_access" {
  description = "Enable public access to the cluster endpoint"
  type        = bool
  default     = true
}

variable "cluster_endpoint_private_access" {
  description = "Enable private access to the cluster endpoint"
  type        = bool
  default     = true
}

variable "cluster_public_access_cidrs" {
  description = "List of CIDR blocks allowed for public access to the cluster endpoint"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "cluster_service_ipv4_cidr" {
  description = "CIDR block for Kubernetes services"
  type        = string
  default     = null
}

variable "cluster_ip_family" {
  description = "IP family for the cluster (ipv4 or ipv6)"
  type        = string
  default     = "ipv4"

  validation {
    condition     = contains(["ipv4", "ipv6"], var.cluster_ip_family)
    error_message = "IP family must be either 'ipv4' or 'ipv6'."
  }
}

variable "cluster_encryption_config" {
  description = "Configuration for cluster encryption. Set to null to disable."
  type = object({
    provider_key_arn = string
    resources        = list(string)
  })
  default = null
}

variable "create_kms_key" {
  description = "Create a KMS key for cluster encryption"
  type        = bool
  default     = false
}

variable "enable_security_groups_for_pods" {
  description = "Enable Security Groups for Pods (VPC resource controller)"
  type        = bool
  default     = false
}

#-------------------------------------------------------------------------------# Node Groups
#-------------------------------------------------------------------------------

variable "node_groups" {
  description = "Map of EKS managed node group definitions"
  type = map(object({
    desired_size             = optional(number, 2)
    min_size                 = optional(number, 1)
    max_size                 = optional(number, 5)
    instance_types           = optional(list(string), ["t3.medium"])
    capacity_type            = optional(string, "ON_DEMAND")
    ami_type                 = optional(string, "AL2_x86_64")
    disk_size                = optional(number, 50)
    max_unavailable_percentage = optional(number, 25)
    labels                   = optional(map(string), {})
    taints = optional(list(object({
      key    = string
      value  = optional(string, null)
      effect = string
    })), [])
    launch_template_id       = optional(string, null)
    launch_template_version  = optional(string, null)
    remote_access = optional(object({
      ec2_ssh_key               = string
      source_security_group_ids = optional(list(string), [])
    }), null)
    tags = optional(map(string), {})
  }))
  default = {
    default = {
      desired_size   = 2
      min_size       = 1
      max_size       = 5
      instance_types = ["t3.medium"]
      capacity_type  = "ON_DEMAND"
      disk_size      = 50
      labels = {
        role = "general"
      }
    }
  }
}

variable "node_group_subnet_ids" {
  description = "Optional separate subnet IDs for node groups. Defaults to subnet_ids if not set."
  type        = list(string)
  default     = null
}

variable "node_group_version" {
  description = "Kubernetes version for node groups. Defaults to cluster version if not set."
  type        = string
  default     = null
}

#-------------------------------------------------------------------------------# EKS Addons
#-------------------------------------------------------------------------------

variable "vpc_cni_addon_version" {
  description = "Version of the VPC CNI addon. Set to null for latest."
  type        = string
  default     = null
}

variable "coredns_addon_version" {
  description = "Version of the CoreDNS addon. Set to null for latest."
  type        = string
  default     = null
}

variable "kube_proxy_addon_version" {
  description = "Version of the kube-proxy addon. Set to null for latest."
  type        = string
  default     = null
}

#-------------------------------------------------------------------------------# Features
#-------------------------------------------------------------------------------

variable "enable_cluster_autoscaler" {
  description = "Enable IAM permissions for Cluster Autoscaler"
  type        = bool
  default     = true
}

variable "enable_irsa" {
  description = "Enable IAM Roles for Service Accounts (OIDC provider)"
  type        = bool
  default     = true
}

#-------------------------------------------------------------------------------# Tags
#-------------------------------------------------------------------------------

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}
