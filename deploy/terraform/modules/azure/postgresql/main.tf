# Azure Database for PostgreSQL - Flexible Server Module
# Creates a managed PostgreSQL database for LiteLLM

terraform {
  required_version = ">= 1.0"

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
    Module    = "azure-postgresql"
  }

  all_tags = merge(local.default_tags, var.tags)

  # Determine if private endpoint should be created
  use_private_endpoint = var.subnet_id != null
}

#-------------------------------------------------------------------------------
# PostgreSQL Flexible Server
#-------------------------------------------------------------------------------
resource "azurerm_postgresql_flexible_server" "this" {
  name                = var.server_name
  resource_group_name = var.resource_group_name
  location            = var.location

  # Version and SKU
  version    = var.postgresql_version
  sku_name   = var.sku_name
  storage_mb = var.storage_mb

  # Administrator credentials
  administrator_login    = var.administrator_login
  administrator_password = var.administrator_password

  # Network configuration
  public_network_access_enabled = local.use_private_endpoint ? false : var.public_network_access_enabled

  # Backup configuration
  backup_retention_days        = var.backup_retention_days
  geo_redundant_backup_enabled = var.geo_redundant_backup_enabled

  # High availability (only for Standard and Premium SKUs)
  dynamic "high_availability" {
    for_each = var.high_availability_enabled ? [1] : []
    content {
      mode                      = var.high_availability_mode
      standby_availability_zone = var.high_availability_standby_availability_zone
    }
  }

  # Maintenance window
  maintenance_window {
    day_of_week  = var.maintenance_window_day_of_week
    start_hour   = var.maintenance_window_start_hour
    start_minute = var.maintenance_window_start_minute
  }

  # Security settings
  ssl_enforcement_enabled = var.ssl_enforcement_enabled

  # Auto-grow
  auto_grow_enabled = var.auto_grow_enabled

  # Tags
  tags = local.all_tags

  # Prevent accidental deletion in production
  lifecycle {
    prevent_destroy = false
  }
}

#-------------------------------------------------------------------------------
# PostgreSQL Database
#-------------------------------------------------------------------------------
resource "azurerm_postgresql_flexible_server_database" "this" {
  name      = var.database_name
  server_id = azurerm_postgresql_flexible_server.this.id
  collation = var.database_collation
  charset   = var.database_charset
}

#-------------------------------------------------------------------------------
# Firewall Rules (only when not using private endpoint)
#-------------------------------------------------------------------------------
resource "azurerm_postgresql_flexible_server_firewall_rule" "allowed_ips" {
  for_each = local.use_private_endpoint ? {} : var.allowed_ip_ranges

  name             = each.key
  server_id        = azurerm_postgresql_flexible_server.this.id
  start_ip_address = cidrhost(each.value, 0)
  end_ip_address   = cidrhost(each.value, -1)
}

#-------------------------------------------------------------------------------
# Private Endpoint (optional)
#-------------------------------------------------------------------------------
resource "azurerm_private_endpoint" "this" {
  count = local.use_private_endpoint ? 1 : 0

  name                = "${var.server_name}-pe"
  location            = var.location
  resource_group_name = var.resource_group_name
  subnet_id           = var.subnet_id

  private_service_connection {
    name                           = "${var.server_name}-psc"
    private_connection_resource_id = azurerm_postgresql_flexible_server.this.id
    subresource_names              = ["postgresqlServer"]
    is_manual_connection           = false
  }

  dynamic "private_dns_zone_group" {
    for_each = var.private_dns_zone_id != null ? [1] : []
    content {
      name                 = "${var.server_name}-dns-group"
      private_dns_zone_ids = [var.private_dns_zone_id]
    }
  }

  tags = local.all_tags
}

#-------------------------------------------------------------------------------
# PostgreSQL Configuration - LiteLLM Optimizations
#-------------------------------------------------------------------------------
resource "azurerm_postgresql_flexible_server_configuration" "log_connections" {
  name      = "log_connections"
  server_id = azurerm_postgresql_flexible_server.this.id
  value     = "on"
}

resource "azurerm_postgresql_flexible_server_configuration" "log_disconnections" {
  name      = "log_disconnections"
  server_id = azurerm_postgresql_flexible_server.this.id
  value     = "on"
}

resource "azurerm_postgresql_flexible_server_configuration" "log_checkpoints" {
  name      = "log_checkpoints"
  server_id = azurerm_postgresql_flexible_server.this.id
  value     = "on"
}

# Connection pooling settings for LiteLLM
resource "azurerm_postgresql_flexible_server_configuration" "max_connections" {
  name      = "max_connections"
  server_id = azurerm_postgresql_flexible_server.this.id
  value     = "200"
}
