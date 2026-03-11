# Azure Network Terraform Module

Terraform module for creating an Azure Virtual Network with subnets and Network Security Groups for the AI Control Plane.

## Features

- Virtual Network (VNet) with configurable address space
- Multiple subnets for AKS and database workloads
- Network Security Groups (NSGs) with security rules
  - AKS NSG: Allows HTTP (80) and HTTPS (443) inbound
  - Database NSG: Allows PostgreSQL (5432) only from AKS subnet
- NSG-subnet associations
- Resource group integration (uses existing resource group)

## Requirements

| Name | Version |
|------|---------|
| terraform | >= 1.0 |
| azurerm | ~> 3.0 |

## Providers

| Name | Version |
|------|---------|
| azurerm | ~> 3.0 |

## Usage

### Basic Example

```hcl
module "network" {
  source = "./modules/azure/network"

  resource_group_name = "my-resource-group"
  location            = "East US"
  name_prefix         = "ai-control-plane"

  vnet_cidr = "10.0.0.0/16"

  subnet_cidrs = {
    aks      = "10.0.1.0/24"
    database = "10.0.2.0/24"
  }

  tags = {
    Environment = "production"
    Project     = "ai-control-plane"
  }
}
```

### Custom Subnets

```hcl
module "network" {
  source = "./modules/azure/network"

  resource_group_name = "my-resource-group"
  name_prefix         = "dev"

  vnet_cidr = "192.168.0.0/16"

  subnet_cidrs = {
    aks       = "192.168.1.0/24"
    database  = "192.168.2.0/24"
    app       = "192.168.3.0/24"
  }

  tags = {
    Environment = "development"
  }
}
```

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| resource_group_name | Name of the resource group where resources will be created | `string` | n/a | yes |
| location | Azure region where resources will be created | `string` | `"East US"` | no |
| name_prefix | Prefix to be used for all resource names | `string` | `"ai-control-plane"` | no |
| vnet_cidr | CIDR block for the Virtual Network | `string` | `"10.0.0.0/16"` | no |
| subnet_cidrs | Map of subnet names to CIDR blocks | `map(string)` | `{ aks = "10.0.1.0/24", database = "10.0.2.0/24" }` | no |
| tags | Tags to apply to all resources | `map(string)` | `{}` | no |

## Outputs

| Name | Description |
|------|-------------|
| vnet_id | The ID of the Virtual Network |
| vnet_name | The name of the Virtual Network |
| vnet_cidr | The address space of the Virtual Network |
| subnet_ids | Map of subnet names to their IDs |
| subnet_names | Map of subnet keys to their full names |
| nsg_ids | Map of NSG names to their IDs |
| nsg_names | Map of NSG types to their full names |
| resource_group_name | The name of the resource group |
| resource_group_location | The location of the resource group |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Virtual Network                          │
│                  CIDR: 10.0.0.0/16                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                  AKS Subnet                         │   │
│  │                  10.0.1.0/24                        │   │
│  │                                                       │   │
│  │  NSG Rules:                                           │   │
│  │  - Allow HTTPS (443) from Internet                    │   │
│  │  - Allow HTTP (80) from Internet                      │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                Database Subnet                      │   │
│  │                  10.0.2.0/24                        │   │
│  │                                                       │   │
│  │  NSG Rules:                                           │   │
│  │  - Allow PostgreSQL (5432) from AKS subnet only       │   │
│  │  - Deny all other inbound traffic                     │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Security Notes

- The AKS NSG allows inbound HTTP/HTTPS traffic from any source (configurable via additional rules)
- The Database NSG restricts PostgreSQL access to the AKS subnet only
- A default deny-all rule is applied to the Database NSG for enhanced security
- All resources are tagged for cost allocation and management

## Prerequisites

- An existing Azure Resource Group
- Azure CLI or Terraform Cloud configured with appropriate credentials
- Proper permissions to create networking resources in the subscription
