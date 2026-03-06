#------------------------------------------------------------------------------
# GCP Load Balancer Module - Variables
#------------------------------------------------------------------------------

#------------------------------------------------------------------------------
# General
#------------------------------------------------------------------------------

variable "name" {
  description = "Name of the load balancer and related resources"
  type        = string
  default     = "ai-control-plane-lb"
}

variable "project_id" {
  description = "GCP project ID where the load balancer will be created"
  type        = string
}

variable "region" {
  description = "GCP region for regional resources (NEG, etc.)"
  type        = string
  default     = "us-central1"
}

variable "tags" {
  description = "Labels to apply to all resources"
  type        = map(string)
  default     = {}
}

#------------------------------------------------------------------------------
# Network Configuration
#------------------------------------------------------------------------------

variable "network" {
  description = "VPC network for the NEG (if create_neg is true)"
  type        = string
  default     = "default"
}

variable "subnetwork" {
  description = "Subnetwork for the NEG (if create_neg is true)"
  type        = string
  default     = null
}

#------------------------------------------------------------------------------
# Backend Service Configuration
#------------------------------------------------------------------------------

variable "backend_service_name" {
  description = "Name of the backend service"
  type        = string
  default     = "ai-control-plane-backend"
}

variable "backend_timeout_sec" {
  description = "Timeout for backend service in seconds"
  type        = number
  default     = 60
}

variable "connection_draining_timeout_sec" {
  description = "Connection draining timeout in seconds"
  type        = number
  default     = 300
}

variable "backend_logging_enabled" {
  description = "Enable logging for the backend service"
  type        = bool
  default     = false
}

variable "backend_logging_sample_rate" {
  description = "Sample rate for backend logging (0.0 to 1.0)"
  type        = number
  default     = 1.0
}

variable "enable_cdn" {
  description = "Enable CDN for the backend service (not recommended for API workloads)"
  type        = bool
  default     = false
}

variable "instance_group" {
  description = "Instance group to use as backend (if not using NEG)"
  type        = string
  default     = null
}

#------------------------------------------------------------------------------
# Network Endpoint Group (NEG) Configuration
#------------------------------------------------------------------------------

variable "create_neg" {
  description = "Create a zonal network endpoint group (container-native LB)"
  type        = bool
  default     = false
}

variable "create_serverless_neg" {
  description = "Create a serverless NEG for Cloud Run or GKE"
  type        = bool
  default     = false
}

variable "neg_name" {
  description = "Name of the NEG (optional, defaults to {name}-neg)"
  type        = string
  default     = null
}

variable "neg_zone" {
  description = "Zone for the NEG (defaults to {region}-a)"
  type        = string
  default     = null
}

variable "cloud_run_service" {
  description = "Cloud Run service name (for serverless NEG)"
  type        = string
  default     = null
}

#------------------------------------------------------------------------------
# Health Check Configuration
#------------------------------------------------------------------------------

variable "health_check_path" {
  description = "Path for health check requests"
  type        = string
  default     = "/health"
}

variable "health_check_port" {
  description = "Port for health check requests"
  type        = number
  default     = 4000
}

variable "health_check_interval" {
  description = "Interval between health checks in seconds"
  type        = number
  default     = 10
}

variable "health_check_timeout" {
  description = "Timeout for health check requests in seconds"
  type        = number
  default     = 5
}

variable "health_check_healthy_threshold" {
  description = "Number of consecutive successful health checks required"
  type        = number
  default     = 2
}

variable "health_check_unhealthy_threshold" {
  description = "Number of consecutive failed health checks required"
  type        = number
  default     = 3
}

variable "health_check_logging_enabled" {
  description = "Enable logging for health checks"
  type        = bool
  default     = false
}

#------------------------------------------------------------------------------
# SSL/TLS Configuration
#------------------------------------------------------------------------------

variable "enable_https" {
  description = "Enable HTTPS listener"
  type        = bool
  default     = true
}

variable "enable_http" {
  description = "Enable HTTP listener (disabled when enable_https_redirect is true)"
  type        = bool
  default     = true
}

variable "enable_https_redirect" {
  description = "Redirect HTTP to HTTPS (creates a separate HTTP forwarding rule for redirect)"
  type        = bool
  default     = false
}

variable "ssl_certificate" {
  description = "Self-managed SSL certificate resource ID (for HTTPS). Use either this or managed_ssl_certificate_domains"
  type        = string
  default     = null
}

variable "managed_ssl_certificate_domains" {
  description = "List of domains for Google-managed SSL certificate"
  type        = list(string)
  default     = []
}

variable "ssl_policy" {
  description = "SSL policy resource ID to apply to the HTTPS proxy"
  type        = string
  default     = null
}

variable "create_ssl_policy" {
  description = "Create a new SSL policy"
  type        = bool
  default     = false
}

variable "ssl_policy_profile" {
  description = "SSL policy profile (COMPATIBLE, MODERN, RESTRICTED, CUSTOM)"
  type        = string
  default     = "MODERN"

  validation {
    condition     = contains(["COMPATIBLE", "MODERN", "RESTRICTED", "CUSTOM"], var.ssl_policy_profile)
    error_message = "SSL policy profile must be one of: COMPATIBLE, MODERN, RESTRICTED, CUSTOM."
  }
}

variable "ssl_policy_min_tls_version" {
  description = "Minimum TLS version for SSL policy"
  type        = string
  default     = "TLS_1_2"

  validation {
    condition     = contains(["TLS_1_0", "TLS_1_1", "TLS_1_2"], var.ssl_policy_min_tls_version)
    error_message = "Minimum TLS version must be one of: TLS_1_0, TLS_1_1, TLS_1_2."
  }
}

variable "ssl_policy_custom_features" {
  description = "Custom SSL features (required when profile is CUSTOM)"
  type        = list(string)
  default     = []
}

variable "quic_override" {
  description = "QUIC protocol override (DISABLE, ENABLE, or NONE)"
  type        = string
  default     = "NONE"

  validation {
    condition     = contains(["DISABLE", "ENABLE", "NONE"], var.quic_override)
    error_message = "QUIC override must be one of: DISABLE, ENABLE, NONE."
  }
}

#------------------------------------------------------------------------------
# URL Map Configuration
#------------------------------------------------------------------------------

variable "host_rules" {
  description = "List of host rules for the URL map"
  type = list(object({
    hosts        = list(string)
    path_matcher = string
  }))
  default = []
}

variable "path_matchers" {
  description = "List of path matchers for the URL map"
  type = list(object({
    name            = string
    default_service = optional(string)
    path_rules = optional(list(object({
      paths   = list(string)
      service = string
    })))
  }))
  default = []
}

#------------------------------------------------------------------------------
# Security Configuration
#------------------------------------------------------------------------------

variable "security_policy" {
  description = "Cloud Armor security policy ID to attach to the backend service"
  type        = string
  default     = null
}

variable "iap_enabled" {
  description = "Enable Identity-Aware Proxy for the backend service"
  type        = bool
  default     = false
}

variable "iap_oauth2_client_id" {
  description = "OAuth2 client ID for IAP"
  type        = string
  default     = null
}

variable "iap_oauth2_client_secret" {
  description = "OAuth2 client secret for IAP"
  type        = string
  sensitive   = true
  default     = null
}
