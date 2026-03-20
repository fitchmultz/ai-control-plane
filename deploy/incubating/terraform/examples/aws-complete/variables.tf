#------------------------------------------------------------------------------
# AWS Complete Example - Variables
#------------------------------------------------------------------------------
# This file defines all input variables for the AI Control Plane deployment
# on AWS with EKS, RDS, and IRSA.
#------------------------------------------------------------------------------

#------------------------------------------------------------------------------
# General Configuration
#------------------------------------------------------------------------------

variable "aws_region" {
  description = "AWS region for resource deployment"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment name (dev, staging, production)"
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
  default     = "ai-control-plane"
}

variable "tags" {
  description = "Additional tags to apply to all resources"
  type        = map(string)
  default     = {}
}

variable "validation_only" {
  description = "Enable provider-bootstrap validation mode for internal dry-run planning without live AWS lookups"
  type        = bool
  default     = false
}

variable "validation_account_id" {
  description = "Placeholder AWS account ID used only when validation_only is true"
  type        = string
  default     = "123456789012"

  validation {
    condition     = can(regex("^[0-9]{12}$", var.validation_account_id))
    error_message = "validation_account_id must be a 12-digit AWS account ID string."
  }
}

#------------------------------------------------------------------------------
# VPC Configuration
#------------------------------------------------------------------------------

variable "vpc_cidr" {
  description = "CIDR block for the VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "availability_zones" {
  description = "List of availability zones to use. If empty, uses the first 3 AZs in the region"
  type        = list(string)
  default     = []
}

variable "private_subnet_cidrs" {
  description = "CIDR blocks for private subnets (one per AZ)"
  type        = list(string)
  default     = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
}

variable "public_subnet_cidrs" {
  description = "CIDR blocks for public subnets (one per AZ)"
  type        = list(string)
  default     = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]
}

variable "single_nat_gateway" {
  description = "Use a single NAT gateway for cost savings (not recommended for production)"
  type        = bool
  default     = false
}

#------------------------------------------------------------------------------
# EKS Configuration
#------------------------------------------------------------------------------

variable "cluster_version" {
  description = "Kubernetes version for the EKS cluster"
  type        = string
  default     = "1.29"
}

variable "cluster_endpoint_public_access" {
  description = "Enable public access to the cluster endpoint"
  type        = bool
  default     = false
}

variable "cluster_endpoint_private_access" {
  description = "Enable private access to the cluster endpoint"
  type        = bool
  default     = true
}

variable "cluster_public_access_cidrs" {
  description = "List of CIDR blocks allowed for public access to the cluster endpoint"
  type        = list(string)
  default     = []

  validation {
    condition     = alltrue([for cidr in var.cluster_public_access_cidrs : cidr != "0.0.0.0/0"])
    error_message = "cluster_public_access_cidrs must not include 0.0.0.0/0. Use explicit admin CIDRs only."
  }
}

variable "node_groups" {
  description = "Map of EKS managed node group definitions"
  type = map(object({
    desired_size               = optional(number, 2)
    min_size                   = optional(number, 1)
    max_size                   = optional(number, 5)
    instance_types             = optional(list(string), ["t3.medium"])
    capacity_type              = optional(string, "ON_DEMAND")
    disk_size                  = optional(number, 50)
    max_unavailable_percentage = optional(number, 25)
    labels                     = optional(map(string), {})
    taints = optional(list(object({
      key    = string
      value  = optional(string, null)
      effect = string
    })), [])
  }))
  default = {}
}

variable "enable_cluster_autoscaler" {
  description = "Enable IAM permissions for Cluster Autoscaler"
  type        = bool
  default     = true
}

#------------------------------------------------------------------------------
# RDS Configuration
#------------------------------------------------------------------------------

variable "rds_instance_class" {
  description = "RDS instance class (overridden per environment in terraform.tfvars)"
  type        = string
  default     = "db.t3.micro"
}

variable "rds_engine_version" {
  description = "PostgreSQL engine version"
  type        = string
  default     = "16.3"
}

variable "rds_allocated_storage" {
  description = "Initial storage size in GB"
  type        = number
  default     = 20
}

variable "rds_max_allocated_storage" {
  description = "Maximum storage size in GB for autoscaling"
  type        = number
  default     = 100
}

variable "rds_multi_az" {
  description = "Enable Multi-AZ deployment for high availability"
  type        = bool
  default     = true
}

variable "rds_backup_retention_period" {
  description = "Number of days to retain backups"
  type        = number
  default     = 7
}

variable "rds_deletion_protection" {
  description = "Enable deletion protection for RDS instance"
  type        = bool
  default     = true
}

variable "rds_skip_final_snapshot" {
  description = "Skip final snapshot when destroying the database"
  type        = bool
  default     = false
}

variable "rds_database_name" {
  description = "Name of the default database to create"
  type        = string
  default     = "litellm"
}

variable "rds_username" {
  description = "Master database username"
  type        = string
  default     = "litellm"
}

variable "rds_performance_insights_enabled" {
  description = "Enable Performance Insights"
  type        = bool
  default     = true
}

#------------------------------------------------------------------------------
# Ingress Exposure Configuration
#------------------------------------------------------------------------------

variable "public_ingress_enabled" {
  description = "Expose ingress publicly. Defaults to internal-only ingress."
  type        = bool
  default     = false
}

variable "alb_certificate_arn" {
  description = "ARN of the ACM certificate for HTTPS ingress (required if enable_ingress is true)"
  type        = string
  default     = ""
}

#------------------------------------------------------------------------------
# Helm Chart Configuration
#------------------------------------------------------------------------------

variable "namespace" {
  description = "Kubernetes namespace for the AI Control Plane"
  type        = string
  default     = "acp"
}

variable "helm_release_name" {
  description = "Name of the Helm release"
  type        = string
  default     = "acp"
}

variable "helm_chart_path" {
  description = "Path to the Helm chart directory"
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
  description = "Number of LiteLLM gateway replicas"
  type        = number
  default     = 2
}

variable "litellm_resources" {
  description = "Resource limits and requests for LiteLLM"
  type = object({
    limits = optional(object({
      cpu    = string
      memory = string
    }), { cpu = "1000m", memory = "1Gi" })
    requests = optional(object({
      cpu    = string
      memory = string
    }), { cpu = "250m", memory = "512Mi" })
  })
  default = {}
}

variable "enable_autoscaling" {
  description = "Enable Horizontal Pod Autoscaler for LiteLLM"
  type        = bool
  default     = true
}

variable "enable_ingress" {
  description = "Enable Kubernetes ingress for the AI Control Plane"
  type        = bool
  default     = false
}

variable "ingress_host" {
  description = "Hostname for the ingress"
  type        = string
  default     = ""
}

variable "ingress_class_name" {
  description = "Ingress class name (e.g., alb, nginx, traefik)"
  type        = string
  default     = "alb"
}

variable "irsa_policy_statements" {
  description = "Additional IAM policy statements for the workload service account. Leave empty unless the workload must call AWS APIs."
  type = list(object({
    effect    = string
    actions   = list(string)
    resources = list(string)
  }))
  default = []
}

#------------------------------------------------------------------------------
# Backup Replication Configuration
#------------------------------------------------------------------------------

variable "backup_replication_enabled" {
  description = "Enable cross-region backup replication to S3"
  type        = bool
  default     = true
}

variable "backup_retention_days" {
  description = "Number of days to retain backups in S3"
  type        = number
  default     = 90
}
