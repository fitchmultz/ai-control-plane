# Azure Network Module
# Creates a Virtual Network with subnets and Network Security Groups

terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
}

#------------------------------------------------------------------------------
# Data Sources
#------------------------------------------------------------------------------

data "azurerm_resource_group" "main" {
  name = var.resource_group_name
}

#------------------------------------------------------------------------------
# Virtual Network
#------------------------------------------------------------------------------

resource "azurerm_virtual_network" "main" {
  name                = "${var.name_prefix}-vnet"
  address_space       = [var.vnet_cidr]
  location            = data.azurerm_resource_group.main.location
  resource_group_name = data.azurerm_resource_group.main.name

  tags = merge(
    var.tags,
    {
      Name = "${var.name_prefix}-vnet"
    }
  )
}

#------------------------------------------------------------------------------
# Subnets
#------------------------------------------------------------------------------

resource "azurerm_subnet" "main" {
  for_each = var.subnet_cidrs

  name                 = "${var.name_prefix}-${each.key}-subnet"
  resource_group_name  = data.azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = [each.value]
}

#------------------------------------------------------------------------------
# Network Security Groups
#------------------------------------------------------------------------------

# NSG for AKS subnet
resource "azurerm_network_security_group" "aks" {
  name                = "${var.name_prefix}-aks-nsg"
  location            = data.azurerm_resource_group.main.location
  resource_group_name = data.azurerm_resource_group.main.name

  tags = merge(
    var.tags,
    {
      Name = "${var.name_prefix}-aks-nsg"
    }
  )
}

# NSG rule: Allow inbound HTTPS to AKS
resource "azurerm_network_security_rule" "aks_https" {
  name                        = "AllowHTTPS"
  priority                    = 100
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "443"
  source_address_prefix       = "*"
  destination_address_prefix  = "*"
  resource_group_name         = data.azurerm_resource_group.main.name
  network_security_group_name = azurerm_network_security_group.aks.name
}

# NSG rule: Allow inbound HTTP to AKS
resource "azurerm_network_security_rule" "aks_http" {
  name                        = "AllowHTTP"
  priority                    = 110
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "80"
  source_address_prefix       = "*"
  destination_address_prefix  = "*"
  resource_group_name         = data.azurerm_resource_group.main.name
  network_security_group_name = azurerm_network_security_group.aks.name
}

# NSG for Database subnet
resource "azurerm_network_security_group" "database" {
  name                = "${var.name_prefix}-database-nsg"
  location            = data.azurerm_resource_group.main.location
  resource_group_name = data.azurerm_resource_group.main.name

  tags = merge(
    var.tags,
    {
      Name = "${var.name_prefix}-database-nsg"
    }
  )
}

# NSG rule: Allow PostgreSQL from AKS subnet only
resource "azurerm_network_security_rule" "database_postgresql" {
  name                        = "AllowPostgreSQLFromAKS"
  priority                    = 100
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "5432"
  source_address_prefix       = var.subnet_cidrs["aks"]
  destination_address_prefix  = "*"
  resource_group_name         = data.azurerm_resource_group.main.name
  network_security_group_name = azurerm_network_security_group.database.name
}

# NSG rule: Deny all other inbound to database
resource "azurerm_network_security_rule" "database_deny_inbound" {
  name                        = "DenyAllInbound"
  priority                    = 4096
  direction                   = "Inbound"
  access                      = "Deny"
  protocol                    = "*"
  source_port_range           = "*"
  destination_port_range      = "*"
  source_address_prefix       = "*"
  destination_address_prefix  = "*"
  resource_group_name         = data.azurerm_resource_group.main.name
  network_security_group_name = azurerm_network_security_group.database.name
}

#------------------------------------------------------------------------------
# NSG Subnet Associations
#------------------------------------------------------------------------------

resource "azurerm_subnet_network_security_group_association" "aks" {
  subnet_id                 = azurerm_subnet.main["aks"].id
  network_security_group_id = azurerm_network_security_group.aks.id
}

resource "azurerm_subnet_network_security_group_association" "database" {
  subnet_id                 = azurerm_subnet.main["database"].id
  network_security_group_id = azurerm_network_security_group.database.id
}
