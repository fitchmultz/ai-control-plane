#------------------------------------------------------------------------------
# Azure Network Module - Outputs
#------------------------------------------------------------------------------

output "vnet_id" {
  description = "The ID of the Virtual Network"
  value       = azurerm_virtual_network.main.id
}

output "vnet_name" {
  description = "The name of the Virtual Network"
  value       = azurerm_virtual_network.main.name
}

output "vnet_cidr" {
  description = "The address space of the Virtual Network"
  value       = azurerm_virtual_network.main.address_space[0]
}

output "subnet_ids" {
  description = "Map of subnet names to their IDs"
  value       = { for name, subnet in azurerm_subnet.main : name => subnet.id }
}

output "subnet_names" {
  description = "Map of subnet keys to their full names"
  value       = { for name, subnet in azurerm_subnet.main : name => subnet.name }
}

output "nsg_ids" {
  description = "Map of NSG names to their IDs"
  value = {
    aks      = azurerm_network_security_group.aks.id
    database = azurerm_network_security_group.database.id
  }
}

output "nsg_names" {
  description = "Map of NSG types to their full names"
  value = {
    aks      = azurerm_network_security_group.aks.name
    database = azurerm_network_security_group.database.name
  }
}

output "resource_group_name" {
  description = "The name of the resource group"
  value       = data.azurerm_resource_group.main.name
}

output "resource_group_location" {
  description = "The location of the resource group"
  value       = data.azurerm_resource_group.main.location
}
