#------------------------------------------------------------------------------
# Azure Application Gateway Module - Variables
#------------------------------------------------------------------------------

#------------------------------------------------------------------------------
# General
#------------------------------------------------------------------------------

variable "name" {
  description = "Name of the Application Gateway and related resources"
  type        = string
}

variable "resource_group_name" {
  description = "Name of the resource group where the Application Gateway will be created"
  type        = string
}

variable "location" {
  description = "Azure region where the Application Gateway will be created"
  type        = string
}

variable "subnet_id" {
  description = "ID of the subnet where the Application Gateway will be deployed"
  type        = string
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}

#------------------------------------------------------------------------------
# SKU Configuration
#------------------------------------------------------------------------------

variable "sku_tier" {
  description = "SKU tier for the Application Gateway (Standard_v2 or WAF_v2)"
  type        = string
  default     = "Standard_v2"

  validation {
    condition     = contains(["Standard_v2", "WAF_v2"], var.sku_tier)
    error_message = "SKU tier must be either 'Standard_v2' or 'WAF_v2'."
  }
}

variable "sku_capacity" {
  description = "Number of instances for the Application Gateway (autoscale not configured)"
  type        = number
  default     = 2
}

variable "enable_autoscale" {
  description = "Enable autoscaling for the Application Gateway"
  type        = bool
  default     = false
}

variable "autoscale_min_capacity" {
  description = "Minimum number of instances for autoscaling"
  type        = number
  default     = 2
}

variable "autoscale_max_capacity" {
  description = "Maximum number of instances for autoscaling"
  type        = number
  default     = 10
}

#------------------------------------------------------------------------------
# Frontend Configuration
#------------------------------------------------------------------------------

variable "frontend_port" {
  description = "Frontend port for HTTP traffic"
  type        = number
  default     = 80
}

variable "frontend_https_port" {
  description = "Frontend port for HTTPS traffic"
  type        = number
  default     = 443
}

variable "enable_https" {
  description = "Enable HTTPS listener"
  type        = bool
  default     = false
}

variable "ssl_certificate_path" {
  description = "Path to the SSL certificate file (PFX format). Required if enable_https is true"
  type        = string
  default     = null
}

variable "ssl_certificate_password" {
  description = "Password for the SSL certificate file"
  type        = string
  sensitive   = true
  default     = null
}

variable "ssl_certificate_name" {
  description = "Name for the SSL certificate in Application Gateway"
  type        = string
  default     = "ssl-cert"
}

#------------------------------------------------------------------------------
# Backend Configuration
#------------------------------------------------------------------------------

variable "backend_port" {
  description = "Backend port for the backend HTTP settings (LiteLLM port)"
  type        = number
  default     = 4000
}

variable "backend_protocol" {
  description = "Protocol for backend communication"
  type        = string
  default     = "Http"

  validation {
    condition     = contains(["Http", "Https"], var.backend_protocol)
    error_message = "Backend protocol must be either 'Http' or 'Https'."
  }
}

variable "backend_ip_addresses" {
  description = "List of backend IP addresses for the backend pool"
  type        = list(string)
  default     = []
}

variable "backend_fqdns" {
  description = "List of backend FQDNs for the backend pool"
  type        = list(string)
  default     = []
}

#------------------------------------------------------------------------------
# Health Probe
#------------------------------------------------------------------------------

variable "health_probe_enabled" {
  description = "Enable health probe"
  type        = bool
  default     = true
}

variable "health_probe_path" {
  description = "Path for health probe requests"
  type        = string
  default     = "/health"
}

variable "health_probe_protocol" {
  description = "Protocol for health probe requests"
  type        = string
  default     = "Http"

  validation {
    condition     = contains(["Http", "Https"], var.health_probe_protocol)
    error_message = "Health probe protocol must be either 'Http' or 'Https'."
  }
}

variable "health_probe_interval" {
  description = "Interval between health probes in seconds"
  type        = number
  default     = 30
}

variable "health_probe_timeout" {
  description = "Timeout for health probe requests in seconds"
  type        = number
  default     = 30
}

variable "health_probe_unhealthy_threshold" {
  description = "Number of consecutive failed health probes before marking unhealthy"
  type        = number
  default     = 3
}

variable "health_probe_match_status_codes" {
  description = "HTTP status codes to accept as healthy"
  type        = list(string)
  default     = ["200-399"]
}

#------------------------------------------------------------------------------
# WAF Configuration (WAF_v2 only)
#------------------------------------------------------------------------------

variable "waf_enabled" {
  description = "Enable WAF (only applicable when sku_tier is WAF_v2)"
  type        = bool
  default     = false
}

variable "waf_mode" {
  description = "WAF mode (Detection or Prevention)"
  type        = string
  default     = "Detection"

  validation {
    condition     = contains(["Detection", "Prevention"], var.waf_mode)
    error_message = "WAF mode must be either 'Detection' or 'Prevention'."
  }
}

variable "waf_rule_set_type" {
  description = "Type of WAF rule set"
  type        = string
  default     = "OWASP"
}

variable "waf_rule_set_version" {
  description = "Version of WAF rule set"
  type        = string
  default     = "3.2"
}

#------------------------------------------------------------------------------
# Logging and Diagnostics
#------------------------------------------------------------------------------

variable "enable_diagnostics" {
  description = "Enable diagnostic settings"
  type        = bool
  default     = false
}

variable "log_analytics_workspace_id" {
  description = "Log Analytics Workspace ID for diagnostics"
  type        = string
  default     = null
}
