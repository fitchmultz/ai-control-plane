#------------------------------------------------------------------------------
# Azure Network Module - Variables
#------------------------------------------------------------------------------

variable "resource_group_name" {
  description = "Name of the resource group where resources will be created"
  type        = string
}

variable "location" {
  description = "Azure region where resources will be created"
  type        = string
  default     = "East US"
}

variable "name_prefix" {
  description = "Prefix to be used for all resource names"
  type        = string
  default     = "ai-control-plane"
}

variable "vnet_cidr" {
  description = "CIDR block for the Virtual Network"
  type        = string
  default     = "10.0.0.0/16"
}

variable "subnet_cidrs" {
  description = "Map of subnet names to CIDR blocks"
  type        = map(string)
  default = {
    aks      = "10.0.1.0/24"
    database = "10.0.2.0/24"
  }
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}
