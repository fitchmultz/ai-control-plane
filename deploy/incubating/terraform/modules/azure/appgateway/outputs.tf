#------------------------------------------------------------------------------
# Azure Application Gateway Module - Outputs
#------------------------------------------------------------------------------

output "gateway_id" {
  description = "ID of the Application Gateway"
  value       = azurerm_application_gateway.this.id
}

output "gateway_name" {
  description = "Name of the Application Gateway"
  value       = azurerm_application_gateway.this.name
}

output "gateway_resource_group_name" {
  description = "Resource group name of the Application Gateway"
  value       = azurerm_application_gateway.this.resource_group_name
}

output "gateway_location" {
  description = "Location of the Application Gateway"
  value       = azurerm_application_gateway.this.location
}

output "public_ip_address" {
  description = "Public IP address of the Application Gateway"
  value       = azurerm_public_ip.this.ip_address
}

output "public_ip_id" {
  description = "ID of the public IP address"
  value       = azurerm_public_ip.this.id
}

output "public_ip_fqdn" {
  description = "FQDN of the public IP address"
  value       = azurerm_public_ip.this.fqdn
}

output "frontend_ip_configuration" {
  description = "Frontend IP configuration details"
  value = {
    name                            = "public-frontend-ip"
    public_ip_address_id            = azurerm_public_ip.this.id
    private_ip_address              = null
    private_ip_address_allocation   = null
    subnet_id                       = null
  }
}

output "backend_address_pool_id" {
  description = "ID of the backend address pool"
  value       = tolist(azurerm_application_gateway.this.backend_address_pool)[0].id
}

output "backend_address_pool_name" {
  description = "Name of the backend address pool"
  value       = tolist(azurerm_application_gateway.this.backend_address_pool)[0].name
}

output "http_listener_id" {
  description = "ID of the HTTP listener"
  value       = tolist(azurerm_application_gateway.this.http_listener)[0].id
}

output "http_listener_name" {
  description = "Name of the HTTP listener"
  value       = tolist(azurerm_application_gateway.this.http_listener)[0].name
}

output "https_listener_id" {
  description = "ID of the HTTPS listener (null if HTTPS is disabled)"
  value       = var.enable_https ? tolist(azurerm_application_gateway.this.http_listener)[1].id : null
}

output "https_listener_name" {
  description = "Name of the HTTPS listener (null if HTTPS is disabled)"
  value       = var.enable_https ? tolist(azurerm_application_gateway.this.http_listener)[1].name : null
}

output "backend_http_settings_id" {
  description = "ID of the backend HTTP settings"
  value       = tolist(azurerm_application_gateway.this.backend_http_settings)[0].id
}

output "backend_http_settings_name" {
  description = "Name of the backend HTTP settings"
  value       = tolist(azurerm_application_gateway.this.backend_http_settings)[0].name
}

output "health_probe_id" {
  description = "ID of the health probe (null if health probe is disabled)"
  value       = var.health_probe_enabled ? tolist(azurerm_application_gateway.this.probe)[0].id : null
}

output "health_probe_name" {
  description = "Name of the health probe (null if health probe is disabled)"
  value       = var.health_probe_enabled ? tolist(azurerm_application_gateway.this.probe)[0].name : null
}

output "sku_tier" {
  description = "SKU tier of the Application Gateway"
  value       = azurerm_application_gateway.this.sku[0].tier
}

output "sku_capacity" {
  description = "SKU capacity of the Application Gateway"
  value       = azurerm_application_gateway.this.sku[0].capacity
}

output "waf_enabled" {
  description = "Whether WAF is enabled"
  value       = local.is_waf_sku && var.waf_enabled
}
