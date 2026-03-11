variable "instance_name" {
  description = "Name of the Cloud SQL instance"
  type        = string

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]*[a-z0-9]$", var.instance_name))
    error_message = "Instance name must start with a letter, contain only lowercase letters, numbers, and hyphens, and end with a letter or number."
  }
}

variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region for the Cloud SQL instance"
  type        = string
}

variable "database_version" {
  description = "PostgreSQL version for the Cloud SQL instance"
  type        = string
  default     = "POSTGRES_16"

  validation {
    condition     = can(regex("^POSTGRES_", var.database_version))
    error_message = "Database version must be a valid PostgreSQL version (e.g., POSTGRES_16)."
  }
}

variable "tier" {
  description = "Machine type tier for the Cloud SQL instance (e.g., db-f1-micro, db-n1-standard-1)"
  type        = string
  default     = "db-f1-micro"
}

variable "disk_size" {
  description = "Initial disk size in GB"
  type        = number
  default     = 20

  validation {
    condition     = var.disk_size >= 10 && var.disk_size <= 65536
    error_message = "Disk size must be between 10 and 65536 GB."
  }
}

variable "disk_autoresize" {
  description = "Enable automatic disk resizing"
  type        = bool
  default     = true
}

variable "availability_type" {
  description = "Availability type: ZONAL or REGIONAL"
  type        = string
  default     = "ZONAL"

  validation {
    condition     = contains(["ZONAL", "REGIONAL"], var.availability_type)
    error_message = "Availability type must be either ZONAL or REGIONAL."
  }
}

variable "backup_enabled" {
  description = "Enable automated backups"
  type        = bool
  default     = true
}

variable "backup_start_time" {
  description = "Start time for daily backups in HH:MM format (UTC)"
  type        = string
  default     = "03:00"

  validation {
    condition     = can(regex("^([01][0-9]|2[0-3]):([0-5][0-9])$", var.backup_start_time))
    error_message = "Backup start time must be in HH:MM format (24-hour UTC)."
  }
}

variable "backup_retention_count" {
  description = "Number of backups to retain"
  type        = number
  default     = 7

  validation {
    condition     = var.backup_retention_count >= 1 && var.backup_retention_count <= 365
    error_message = "Backup retention count must be between 1 and 365."
  }
}

variable "maintenance_day" {
  description = "Day of the week for maintenance (1 = Monday, 7 = Sunday)"
  type        = number
  default     = 7

  validation {
    condition     = var.maintenance_day >= 1 && var.maintenance_day <= 7
    error_message = "Maintenance day must be between 1 (Monday) and 7 (Sunday)."
  }
}

variable "maintenance_hour" {
  description = "Hour of the day for maintenance (0-23)"
  type        = number
  default     = 4

  validation {
    condition     = var.maintenance_hour >= 0 && var.maintenance_hour <= 23
    error_message = "Maintenance hour must be between 0 and 23."
  }
}

variable "maintenance_track" {
  description = "Maintenance update track: stable or canary"
  type        = string
  default     = "stable"

  validation {
    condition     = contains(["stable", "canary"], var.maintenance_track)
    error_message = "Maintenance track must be either stable or canary."
  }
}

variable "vpc_network" {
  description = "VPC network name for private IP (null for public IP only)"
  type        = string
  default     = null
}

variable "authorized_networks" {
  description = "List of authorized networks for public IP access"
  type = list(object({
    name = string
    cidr = string
  }))
  default = []
}

variable "database_name" {
  description = "Name of the database to create"
  type        = string
  default     = "litellm"
}

variable "user_name" {
  description = "Name of the database user"
  type        = string
  default     = "litellm"
}

variable "user_password" {
  description = "Password for the database user"
  type        = string
  sensitive   = true
}

variable "enable_insights" {
  description = "Enable Query Insights for performance monitoring"
  type        = bool
  default     = false
}

variable "insights_query_length" {
  description = "Maximum query string length for Query Insights"
  type        = number
  default     = 1024
}

variable "deletion_protection" {
  description = "Enable deletion protection for the instance"
  type        = bool
  default     = true
}

variable "labels" {
  description = "Labels to apply to the Cloud SQL instance"
  type        = map(string)
  default     = {}
}
