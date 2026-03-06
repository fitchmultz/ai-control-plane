#-------------------------------------------------------------------------------
# Required Variables
#-------------------------------------------------------------------------------

variable "vpc_id" {
  description = "VPC ID where the RDS instance will be created"
  type        = string
}

variable "subnet_ids" {
  description = "List of private subnet IDs for the DB subnet group"
  type        = list(string)
}

variable "password" {
  description = "Master database password"
  type        = string
  sensitive   = true
}

#-------------------------------------------------------------------------------
# General Settings
#-------------------------------------------------------------------------------

variable "identifier" {
  description = "Unique identifier for the RDS instance"
  type        = string
  default     = "ai-control-plane-db"
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}

#-------------------------------------------------------------------------------
# Engine Settings
#-------------------------------------------------------------------------------

variable "engine_version" {
  description = "PostgreSQL engine version"
  type        = string
  default     = "16.3"
}

#-------------------------------------------------------------------------------
# Instance Settings
#-------------------------------------------------------------------------------

variable "instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.t3.micro"
}

variable "multi_az" {
  description = "Enable Multi-AZ deployment for high availability"
  type        = bool
  default     = true
}

#-------------------------------------------------------------------------------
# Storage Settings
#-------------------------------------------------------------------------------

variable "allocated_storage" {
  description = "Initial storage size in GB"
  type        = number
  default     = 20
}

variable "max_allocated_storage" {
  description = "Maximum storage size in GB for autoscaling"
  type        = number
  default     = 100
}

variable "storage_encrypted" {
  description = "Enable storage encryption using AWS KMS"
  type        = bool
  default     = true
}

#-------------------------------------------------------------------------------
# Database Settings
#-------------------------------------------------------------------------------

variable "db_name" {
  description = "Name of the default database to create"
  type        = string
  default     = "litellm"
}

variable "username" {
  description = "Master database username"
  type        = string
  default     = "litellm"
}

#-------------------------------------------------------------------------------
# Network/Security Settings
#-------------------------------------------------------------------------------

variable "allowed_security_groups" {
  description = "List of security group IDs allowed to connect to the database"
  type        = list(string)
  default     = []
}

variable "allowed_cidr_blocks" {
  description = "List of CIDR blocks allowed to connect to the database (use with caution)"
  type        = list(string)
  default     = []

  validation {
    condition     = alltrue([for cidr in var.allowed_cidr_blocks : can(cidrhost(cidr, 0))])
    error_message = "All entries in allowed_cidr_blocks must be valid CIDR blocks."
  }
}

#-------------------------------------------------------------------------------
# Backup Settings
#-------------------------------------------------------------------------------

variable "backup_retention_period" {
  description = "Number of days to retain backups"
  type        = number
  default     = 7
}

variable "backup_window" {
  description = "Preferred backup window (UTC)"
  type        = string
  default     = "03:00-04:00"
}

variable "maintenance_window" {
  description = "Preferred maintenance window (UTC)"
  type        = string
  default     = "Mon:04:00-Mon:05:00"
}

#-------------------------------------------------------------------------------
# Deletion Protection
#-------------------------------------------------------------------------------

variable "deletion_protection" {
  description = "Enable deletion protection"
  type        = bool
  default     = true
}

variable "skip_final_snapshot" {
  description = "Skip final snapshot when destroying the database"
  type        = bool
  default     = false
}

#-------------------------------------------------------------------------------
# Maintenance Settings
#-------------------------------------------------------------------------------

variable "auto_minor_version_upgrade" {
  description = "Enable automatic minor version upgrades"
  type        = bool
  default     = true
}

#-------------------------------------------------------------------------------
# Monitoring Settings
#-------------------------------------------------------------------------------

variable "performance_insights_enabled" {
  description = "Enable Performance Insights"
  type        = bool
  default     = false
}

variable "performance_insights_retention_period" {
  description = "Performance Insights retention period in days (7-731)"
  type        = number
  default     = 7
}
