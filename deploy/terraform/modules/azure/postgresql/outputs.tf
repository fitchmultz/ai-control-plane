#-------------------------------------------------------------------------------
# Server Outputs
#-------------------------------------------------------------------------------

output "server_id" {
  description = "The ID of the PostgreSQL Flexible Server"
  value       = azurerm_postgresql_flexible_server.this.id
}

output "server_name" {
  description = "The name of the PostgreSQL Flexible Server"
  value       = azurerm_postgresql_flexible_server.this.name
}

output "fqdn" {
  description = "The fully qualified domain name (FQDN) of the PostgreSQL server"
  value       = azurerm_postgresql_flexible_server.this.fqdn
}

output "private_fqdn" {
  description = "The private FQDN of the PostgreSQL server (when using private endpoint)"
  value       = local.use_private_endpoint ? azurerm_postgresql_flexible_server.this.fqdn : null
}

#-------------------------------------------------------------------------------
# Database Outputs
#-------------------------------------------------------------------------------

output "database_name" {
  description = "The name of the PostgreSQL database"
  value       = azurerm_postgresql_flexible_server_database.this.name
}

output "database_id" {
  description = "The ID of the PostgreSQL database"
  value       = azurerm_postgresql_flexible_server_database.this.id
}

#-------------------------------------------------------------------------------
# Connection Outputs
#-------------------------------------------------------------------------------

output "administrator_login" {
  description = "The administrator login name"
  value       = azurerm_postgresql_flexible_server.this.administrator_login
}

output "database_url" {
  description = "PostgreSQL connection URL (sensitive)"
  value       = "postgresql://${var.administrator_login}:${var.administrator_password}@${azurerm_postgresql_flexible_server.this.fqdn}:5432/${var.database_name}?sslmode=require"
  sensitive   = true
}

output "jdbc_connection_string" {
  description = "JDBC connection string for PostgreSQL (sensitive)"
  value       = "jdbc:postgresql://${azurerm_postgresql_flexible_server.this.fqdn}:5432/${var.database_name}?sslmode=require"
  sensitive   = false
}

#-------------------------------------------------------------------------------
# Private Endpoint Outputs
#-------------------------------------------------------------------------------

output "private_endpoint_id" {
  description = "The ID of the private endpoint (if created)"
  value       = local.use_private_endpoint ? azurerm_private_endpoint.this[0].id : null
}

output "private_endpoint_private_ip" {
  description = "The private IP address of the private endpoint (if created)"
  value       = local.use_private_endpoint ? azurerm_private_endpoint.this[0].private_service_connection[0].private_ip_address : null
}

#-------------------------------------------------------------------------------
# Resource References
#-------------------------------------------------------------------------------

output "resource_group_name" {
  description = "The name of the resource group containing the server"
  value       = var.resource_group_name
}

output "location" {
  description = "The Azure region where the server is deployed"
  value       = var.location
}
