#-------------------------------------------------------------------------------
# Required Variables
#-------------------------------------------------------------------------------

variable "server_name" {
  description = "Name of the Azure PostgreSQL Flexible Server"
  type        = string
}

variable "resource_group_name" {
  description = "Name of the resource group where the server will be created"
  type        = string
}

variable "location" {
  description = "Azure region where the server will be deployed"
  type        = string
}

variable "administrator_password" {
  description = "Password for the PostgreSQL administrator"
  type        = string
  sensitive   = true
}

#-------------------------------------------------------------------------------
# General Settings
#-------------------------------------------------------------------------------

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}

#-------------------------------------------------------------------------------
# Server Settings
#-------------------------------------------------------------------------------

variable "postgresql_version" {
  description = "PostgreSQL version (11-16)"
  type        = string
  default     = "16"
}

variable "sku_name" {
  description = "SKU name for the PostgreSQL Flexible Server (e.g., B_Standard_B2s, GP_Standard_D4s_v3)"
  type        = string
  default     = "B_Standard_B2s"
}

variable "storage_mb" {
  description = "Storage size in MB (minimum 32768, maximum 16777216)"
  type        = number
  default     = 32768
}

variable "administrator_login" {
  description = "Login name for the PostgreSQL administrator"
  type        = string
  default     = "litellm"
}

#-------------------------------------------------------------------------------
# Database Settings
#-------------------------------------------------------------------------------

variable "database_name" {
  description = "Name of the default database to create"
  type        = string
  default     = "litellm"
}

variable "database_collation" {
  description = "Collation for the database"
  type        = string
  default     = "en_US.utf8"
}

variable "database_charset" {
  description = "Character set for the database"
  type        = string
  default     = "UTF8"
}

#-------------------------------------------------------------------------------
# Network Settings
#-------------------------------------------------------------------------------

variable "subnet_id" {
  description = "Optional subnet ID for private endpoint. If not provided, public access with firewall rules is used"
  type        = string
  default     = null
}

variable "private_dns_zone_id" {
  description = "Optional private DNS zone ID for private endpoint"
  type        = string
  default     = null
}

variable "public_network_access_enabled" {
  description = "Whether public network access is enabled (ignored if subnet_id is provided)"
  type        = bool
  default     = false
}

variable "allowed_ip_ranges" {
  description = "Map of IP ranges allowed to access the server (only used when public_network_access_enabled is true)"
  type        = map(string)
  default     = {}
  # Example: { "office" = "203.0.113.0/24", "vpn" = "198.51.100.0/24" }
}

#-------------------------------------------------------------------------------
# Backup Settings
#-------------------------------------------------------------------------------

variable "backup_retention_days" {
  description = "Number of days to retain backups (7-35 days)"
  type        = number
  default     = 7
}

variable "geo_redundant_backup_enabled" {
  description = "Enable geo-redundant backups (only available for Standard and Premium SKUs)"
  type        = bool
  default     = false
}

#-------------------------------------------------------------------------------
# High Availability Settings
#-------------------------------------------------------------------------------

variable "high_availability_enabled" {
  description = "Enable high availability with zone redundancy (only available for Standard and Premium SKUs)"
  type        = bool
  default     = false
}

variable "high_availability_standby_availability_zone" {
  description = "Availability zone for the standby server (1, 2, or 3)"
  type        = string
  default     = null
}

variable "high_availability_mode" {
  description = "High availability mode (ZoneRedundant or SameZone)"
  type        = string
  default     = "ZoneRedundant"
}

#-------------------------------------------------------------------------------
# Maintenance Settings
#-------------------------------------------------------------------------------

variable "maintenance_window_day_of_week" {
  description = "Day of week for maintenance window (0-6, where 0 is Sunday)"
  type        = number
  default     = 0
}

variable "maintenance_window_start_hour" {
  description = "Start hour for maintenance window (0-23)"
  type        = number
  default     = 3
}

variable "maintenance_window_start_minute" {
  description = "Start minute for maintenance window (0-59)"
  type        = number
  default     = 0
}

#-------------------------------------------------------------------------------
# Security Settings
#-------------------------------------------------------------------------------

variable "ssl_enforcement_enabled" {
  description = "Enforce SSL connections"
  type        = bool
  default     = true
}

variable "ssl_minimal_tls_version" {
  description = "Minimum TLS version for SSL connections"
  type        = string
  default     = "TLS1_2"
}

variable "auto_grow_enabled" {
  description = "Enable storage auto-grow"
  type        = bool
  default     = true
}
