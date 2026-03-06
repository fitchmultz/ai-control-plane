# Azure Complete Example - Terraform Configuration
# This file defines the required Terraform version and provider versions

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    # Azure Resource Manager provider
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 3.80.0"
    }

    # Random provider for generating passwords
    random = {
      source  = "hashicorp/random"
      version = ">= 3.0.0"
    }

    # Kubernetes provider for cluster resources
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 2.25.0"
    }

    # Helm provider for deploying charts
    helm = {
      source  = "hashicorp/helm"
      version = ">= 2.12.0"
    }
  }
}

#------------------------------------------------------------------------------
# Provider Configuration
#------------------------------------------------------------------------------

# Azure provider configuration
provider "azurerm" {
  features {
    resource_group {
      prevent_deletion_if_contains_resources = false
    }
  }
}

# Random provider - no configuration needed
provider "random" {
}

# Kubernetes provider is configured dynamically after AKS is created
# See main.tf for provider configuration

# Helm provider is configured dynamically after AKS is created
# See main.tf for provider configuration
