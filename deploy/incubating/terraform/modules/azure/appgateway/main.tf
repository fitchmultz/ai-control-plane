# Azure Application Gateway Module for AI Control Plane
# Creates an Application Gateway v2 with public IP, backend pools, and listeners

terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 3.0"
    }
  }
}

locals {
  default_tags = {
    ManagedBy = "terraform"
    Module    = "appgateway"
  }

  all_tags = merge(local.default_tags, var.tags)

  # Determine if WAF is enabled based on SKU tier and variable
  is_waf_sku = var.sku_tier == "WAF_v2"
  waf_config = local.is_waf_sku && var.waf_enabled ? [1] : []
}

#------------------------------------------------------------------------------
# Public IP Address
#------------------------------------------------------------------------------

resource "azurerm_public_ip" "this" {
  name                = "${var.name}-pip"
  resource_group_name = var.resource_group_name
  location            = var.location
  allocation_method   = "Static"
  sku                 = "Standard"
  zones               = ["1", "2", "3"]

  tags = merge(
    local.all_tags,
    {
      Name = "${var.name}-pip"
    }
  )
}

#------------------------------------------------------------------------------
# SSL Certificate (if HTTPS is enabled)
#------------------------------------------------------------------------------

# SSL certificates can be managed in several ways:
# 1. Via azurerm_application_gateway_ssl_certificate resource with data from Key Vault
# 2. Via azurerm_application_gateway_ssl_certificate resource with file content
# 3. Via the ssl_certificate block within azurerm_application_gateway (used here)
#
# This module uses option 3 - inline ssl_certificate block within the gateway resource
# to allow proper lifecycle management and avoid conflicts with App Gateway's
# built-in certificate handling.
#
# For production use, certificates should be stored in Azure Key Vault and
# referenced using key_vault_secret_id instead of inline data.

#------------------------------------------------------------------------------
# Application Gateway
#------------------------------------------------------------------------------

resource "azurerm_application_gateway" "this" {
  name                = var.name
  resource_group_name = var.resource_group_name
  location            = var.location
  enable_http2        = true

  sku {
    name = var.sku_tier
    tier = var.sku_tier
    # capacity is only used when autoscale is not configured
    capacity = var.enable_autoscale ? null : var.sku_capacity
  }

  dynamic "autoscale_configuration" {
    for_each = var.enable_autoscale ? [1] : []
    content {
      min_capacity = var.autoscale_min_capacity
      max_capacity = var.autoscale_max_capacity
    }
  }

  #------------------------------------------------------------------------------
  # Gateway IP Configuration
  #------------------------------------------------------------------------------

  gateway_ip_configuration {
    name      = "gateway-ip-configuration"
    subnet_id = var.subnet_id
  }

  #------------------------------------------------------------------------------
  # Frontend IP Configurations
  #------------------------------------------------------------------------------

  frontend_ip_configuration {
    name                 = "public-frontend-ip"
    public_ip_address_id = azurerm_public_ip.this.id
  }

  #------------------------------------------------------------------------------
  # Frontend Ports
  #------------------------------------------------------------------------------

  frontend_port {
    name = "http-port"
    port = var.frontend_port
  }

  dynamic "frontend_port" {
    for_each = var.enable_https ? [1] : []
    content {
      name = "https-port"
      port = var.frontend_https_port
    }
  }

  #------------------------------------------------------------------------------
  # Backend Address Pool
  #------------------------------------------------------------------------------

  backend_address_pool {
    name         = "aks-backend-pool"
    ip_addresses = var.backend_ip_addresses
    fqdns        = var.backend_fqdns
  }

  #------------------------------------------------------------------------------
  # Backend HTTP Settings
  #------------------------------------------------------------------------------

  backend_http_settings {
    name                  = "http-settings"
    cookie_based_affinity = "Disabled"
    port                  = var.backend_port
    protocol              = var.backend_protocol
    request_timeout       = 60
    probe_name            = var.health_probe_enabled ? "health-probe" : null
  }

  #------------------------------------------------------------------------------
  # Health Probe
  #------------------------------------------------------------------------------

  dynamic "probe" {
    for_each = var.health_probe_enabled ? [1] : []
    content {
      name                                      = "health-probe"
      protocol                                  = var.health_probe_protocol
      path                                      = var.health_probe_path
      interval                                  = var.health_probe_interval
      timeout                                   = var.health_probe_timeout
      unhealthy_threshold                       = var.health_probe_unhealthy_threshold
      pick_host_name_from_backend_http_settings = true

      match {
        status_code = var.health_probe_match_status_codes
      }
    }
  }

  #------------------------------------------------------------------------------
  # HTTP Listener
  #------------------------------------------------------------------------------

  http_listener {
    name                           = "http-listener"
    frontend_ip_configuration_name = "public-frontend-ip"
    frontend_port_name             = "http-port"
    protocol                       = "Http"
  }

  dynamic "http_listener" {
    for_each = var.enable_https ? [1] : []
    content {
      name                           = "https-listener"
      frontend_ip_configuration_name = "public-frontend-ip"
      frontend_port_name             = "https-port"
      protocol                       = "Https"
      ssl_certificate_name           = var.ssl_certificate_name
    }
  }

  #------------------------------------------------------------------------------
  # Request Routing Rules
  #------------------------------------------------------------------------------

  # HTTP rule - redirect to HTTPS if HTTPS is enabled, otherwise forward
  request_routing_rule {
    name                       = "http-rule"
    rule_type                  = "Basic"
    http_listener_name         = "http-listener"
    backend_address_pool_name  = var.enable_https ? null : "aks-backend-pool"
    backend_http_settings_name = var.enable_https ? null : "http-settings"
    redirect_configuration_name = var.enable_https ? "http-to-https-redirect" : null
    priority                   = 100
  }

  dynamic "request_routing_rule" {
    for_each = var.enable_https ? [1] : []
    content {
      name                       = "https-rule"
      rule_type                  = "Basic"
      http_listener_name         = "https-listener"
      backend_address_pool_name  = "aks-backend-pool"
      backend_http_settings_name = "http-settings"
      priority                   = 200
    }
  }

  #------------------------------------------------------------------------------
  # Redirect Configuration (HTTP to HTTPS)
  #------------------------------------------------------------------------------

  dynamic "redirect_configuration" {
    for_each = var.enable_https ? [1] : []
    content {
      name                 = "http-to-https-redirect"
      redirect_type        = "Permanent"
      target_listener_name = "https-listener"
      include_path         = true
      include_query_string = true
    }
  }

  #------------------------------------------------------------------------------
  # WAF Configuration (WAF_v2 only)
  #------------------------------------------------------------------------------

  dynamic "waf_configuration" {
    for_each = local.waf_config
    content {
      enabled          = true
      firewall_mode    = var.waf_mode
      rule_set_type    = var.waf_rule_set_type
      rule_set_version = var.waf_rule_set_version

      # Default exclusion for health probes
      exclusion {
        match_variable          = "RequestUri"
        selector                = "/health"
        selector_match_operator = "Contains"
      }
    }
  }

  #------------------------------------------------------------------------------
  # SSL Policy (recommended security settings)
  #------------------------------------------------------------------------------

  ssl_policy {
    policy_type = "Predefined"
    policy_name = "AppGwSslPolicy20220101S1"
  }

  tags = merge(
    local.all_tags,
    {
      Name = var.name
    }
  )

  # Ensure the public IP is created before the gateway
  depends_on = [azurerm_public_ip.this]

  lifecycle {
    ignore_changes = [
      # Ignore changes to SSL certificate data as it may be managed externally
      ssl_certificate,
      # Ignore backend address pool changes as they may be managed by Kubernetes
      backend_address_pool,
    ]
  }
}

#------------------------------------------------------------------------------
# Diagnostic Settings (optional)
#------------------------------------------------------------------------------

resource "azurerm_monitor_diagnostic_setting" "this" {
  count = var.enable_diagnostics && var.log_analytics_workspace_id != null ? 1 : 0

  name                       = "${var.name}-diagnostics"
  target_resource_id         = azurerm_application_gateway.this.id
  log_analytics_workspace_id = var.log_analytics_workspace_id

  enabled_log {
    category = "ApplicationGatewayAccessLog"
  }

  enabled_log {
    category = "ApplicationGatewayPerformanceLog"
  }

  enabled_log {
    category = "ApplicationGatewayFirewallLog"
  }

  metric {
    category = "AllMetrics"
    enabled  = true
  }
}
