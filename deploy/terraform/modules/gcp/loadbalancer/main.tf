# GCP Global HTTP(S) Load Balancer Module for AI Control Plane
# Creates a global forwarding rule, target proxy, backend service, and health check

terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 4.0"
    }
  }
}

locals {
  default_tags = {
    ManagedBy = "terraform"
    Module    = "loadbalancer"
  }

  all_tags = merge(local.default_tags, var.tags)

  # Determine which SSL certificate to use
  use_managed_ssl = var.enable_https && length(var.managed_ssl_certificate_domains) > 0 && var.ssl_certificate == null
  use_self_managed_ssl = var.enable_https && var.ssl_certificate != null
}

#------------------------------------------------------------------------------
# Global IP Address
#------------------------------------------------------------------------------

resource "google_compute_global_address" "this" {
  name        = var.name
  description = "Global IP address for ${var.name} load balancer"

  labels = local.all_tags
}

#------------------------------------------------------------------------------
# Managed SSL Certificate (Google-managed)
#------------------------------------------------------------------------------

resource "google_compute_managed_ssl_certificate" "this" {
  count = local.use_managed_ssl ? 1 : 0

  name = "${var.name}-cert"

  managed {
    domains = var.managed_ssl_certificate_domains
  }

  lifecycle {
    create_before_destroy = true
  }
}

#------------------------------------------------------------------------------
# Health Check
#------------------------------------------------------------------------------

resource "google_compute_health_check" "this" {
  name                = "${var.name}-health-check"
  description         = "Health check for LiteLLM on port ${var.health_check_port}"
  check_interval_sec  = var.health_check_interval
  timeout_sec         = var.health_check_timeout
  healthy_threshold   = var.health_check_healthy_threshold
  unhealthy_threshold = var.health_check_unhealthy_threshold

  http_health_check {
    port         = var.health_check_port
    request_path = var.health_check_path
  }

  log_config {
    enable = var.health_check_logging_enabled
  }

  labels = local.all_tags
}

#------------------------------------------------------------------------------
# Container-Native Network Endpoint Group (NEG) - Optional
#------------------------------------------------------------------------------

resource "google_compute_network_endpoint_group" "this" {
  count = var.create_neg ? 1 : 0

  name                  = var.neg_name != null ? var.neg_name : "${var.name}-neg"
  network_endpoint_type = "GCE_VM_IP_PORT"
  network               = var.network
  subnetwork            = var.subnetwork
  zone                  = var.neg_zone != null ? var.neg_zone : "${var.region}-a"
}

#------------------------------------------------------------------------------
# Serverless NEG for GKE (Container-Native LB) - Optional
#------------------------------------------------------------------------------

resource "google_compute_region_network_endpoint_group" "this" {
  count = var.create_serverless_neg ? 1 : 0

  name                  = var.neg_name != null ? var.neg_name : "${var.name}-serverless-neg"
  network_endpoint_type = "SERVERLESS"
  region                = var.region

  cloud_run {
    service = var.cloud_run_service
  }
}

#------------------------------------------------------------------------------
# Backend Service
#------------------------------------------------------------------------------

resource "google_compute_backend_service" "this" {
  name        = var.backend_service_name
  description = "Backend service for AI Control Plane"
  port_name   = "http"
  protocol    = "HTTP"
  timeout_sec = var.backend_timeout_sec

  health_checks = [google_compute_health_check.this.id]

  # Use NEG if created, otherwise use instance group or serverless NEG
  dynamic "backend" {
    for_each = var.create_neg ? [1] : []
    content {
      group = google_compute_network_endpoint_group.this[0].id
    }
  }

  dynamic "backend" {
    for_each = var.create_serverless_neg ? [1] : []
    content {
      group = google_compute_region_network_endpoint_group.this[0].id
    }
  }

  # Backend with instance group if specified
  dynamic "backend" {
    for_each = var.instance_group != null && !var.create_neg && !var.create_serverless_neg ? [1] : []
    content {
      group = var.instance_group
    }
  }

  # Connection draining settings
  connection_draining_timeout_sec = var.connection_draining_timeout_sec

  # Logging configuration
  log_config {
    enable      = var.backend_logging_enabled
    sample_rate = var.backend_logging_sample_rate
  }

  # CDN configuration (disabled by default for API workloads)
  enable_cdn = var.enable_cdn

  # Security settings
  dynamic "security_policy" {
    for_each = var.security_policy != null ? [1] : []
    content {
      name = var.security_policy
    }
  }

  # IAP configuration
  dynamic "iap" {
    for_each = var.iap_enabled ? [1] : []
    content {
      oauth2_client_id     = var.iap_oauth2_client_id
      oauth2_client_secret = var.iap_oauth2_client_secret
    }
  }

  labels = local.all_tags
}

#------------------------------------------------------------------------------
# URL Map (Load Balancer)
#------------------------------------------------------------------------------

resource "google_compute_url_map" "this" {
  name            = var.name
  description     = "URL map for ${var.name} load balancer"
  default_service = google_compute_backend_service.this.id

  # Host rules for multi-domain support
  dynamic "host_rule" {
    for_each = var.host_rules
    content {
      hosts        = host_rule.value.hosts
      path_matcher = host_rule.value.path_matcher
    }
  }

  # Path matchers
  dynamic "path_matcher" {
    for_each = var.path_matchers
    content {
      name            = path_matcher.value.name
      default_service = path_matcher.value.default_service != null ? path_matcher.value.default_service : google_compute_backend_service.this.id

      dynamic "path_rule" {
        for_each = lookup(path_matcher.value, "path_rules", [])
        content {
          paths   = path_rule.value.paths
          service = path_rule.value.service
        }
      }
    }
  }
}

#------------------------------------------------------------------------------
# HTTP Proxy
#------------------------------------------------------------------------------

resource "google_compute_target_http_proxy" "this" {
  count = var.enable_https ? 0 : 1

  name    = "${var.name}-http-proxy"
  url_map = google_compute_url_map.this.id
}

#------------------------------------------------------------------------------
# HTTPS Proxy
#------------------------------------------------------------------------------

resource "google_compute_target_https_proxy" "this" {
  count = var.enable_https ? 1 : 0

  name    = "${var.name}-https-proxy"
  url_map = google_compute_url_map.this.id

  ssl_certificates = compact([
    local.use_self_managed_ssl ? var.ssl_certificate : null,
    local.use_managed_ssl ? google_compute_managed_ssl_certificate.this[0].id : null,
  ])

  ssl_policy = var.ssl_policy

  quic_override = var.quic_override
}

#------------------------------------------------------------------------------
# Global Forwarding Rule (HTTP)
#------------------------------------------------------------------------------

resource "google_compute_global_forwarding_rule" "http" {
  count = var.enable_http ? 1 : 0

  name                  = "${var.name}-http"
  description           = "HTTP forwarding rule for ${var.name}"
  ip_protocol           = "TCP"
  load_balancing_scheme = "EXTERNAL"
  port_range            = "80"
  target                = google_compute_target_http_proxy.this[0].id
  ip_address            = google_compute_global_address.this.id
  labels                = local.all_tags
}

#------------------------------------------------------------------------------
# Global Forwarding Rule (HTTPS)
#------------------------------------------------------------------------------

resource "google_compute_global_forwarding_rule" "https" {
  count = var.enable_https ? 1 : 0

  name                  = "${var.name}-https"
  description           = "HTTPS forwarding rule for ${var.name}"
  ip_protocol           = "TCP"
  load_balancing_scheme = "EXTERNAL"
  port_range            = "443"
  target                = google_compute_target_https_proxy.this[0].id
  ip_address            = google_compute_global_address.this.id
  labels                = local.all_tags
}

#------------------------------------------------------------------------------
# HTTP to HTTPS Redirect (Optional)
#------------------------------------------------------------------------------

resource "google_compute_url_map" "http_redirect" {
  count = var.enable_https_redirect ? 1 : 0

  name = "${var.name}-http-redirect"

  default_url_redirect {
    https_redirect         = true
    redirect_response_code = "MOVED_PERMANENTLY_DEFAULT"
    strip_query            = false
  }
}

resource "google_compute_target_http_proxy" "http_redirect" {
  count = var.enable_https_redirect ? 1 : 0

  name    = "${var.name}-http-redirect-proxy"
  url_map = google_compute_url_map.http_redirect[0].id
}

resource "google_compute_global_forwarding_rule" "http_redirect" {
  count = var.enable_https_redirect ? 1 : 0

  name                  = "${var.name}-http-redirect"
  description           = "HTTP to HTTPS redirect for ${var.name}"
  ip_protocol           = "TCP"
  load_balancing_scheme = "EXTERNAL"
  port_range            = "80"
  target                = google_compute_target_http_proxy.http_redirect[0].id
  ip_address            = google_compute_global_address.this.id
  labels                = local.all_tags
}

#------------------------------------------------------------------------------
# SSL Policy (Optional)
#------------------------------------------------------------------------------

resource "google_compute_ssl_policy" "this" {
  count = var.create_ssl_policy ? 1 : 0

  name            = "${var.name}-ssl-policy"
  description     = "SSL policy for ${var.name}"
  profile         = var.ssl_policy_profile
  min_tls_version = var.ssl_policy_min_tls_version
  custom_features = var.ssl_policy_custom_features
}
