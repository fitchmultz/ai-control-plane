#------------------------------------------------------------------------------
# AWS Complete Example - Terraform and Provider Versions
#------------------------------------------------------------------------------
# This file defines the required Terraform version and provider versions
# for deploying the AI Control Plane on AWS with EKS, RDS, and IRSA.
#------------------------------------------------------------------------------

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 2.23"
    }
    helm = {
      source  = "hashicorp/helm"
      version = ">= 2.11"
    }
    random = {
      source  = "hashicorp/random"
      version = ">= 3.5"
    }
    tls = {
      source  = "hashicorp/tls"
      version = ">= 4.0"
    }
  }
}

#------------------------------------------------------------------------------
# AWS Provider Configuration
#------------------------------------------------------------------------------

provider "aws" {
  region                      = var.aws_region
  access_key                  = var.validation_only ? "validation-access-key" : null
  secret_key                  = var.validation_only ? "validation-secret-key" : null
  skip_credentials_validation = var.validation_only
  skip_metadata_api_check     = var.validation_only
  skip_region_validation      = var.validation_only
  skip_requesting_account_id  = var.validation_only

  default_tags {
    tags = {
      Environment = var.environment
      Project     = "ai-control-plane"
      ManagedBy   = "terraform"
    }
  }
}

#------------------------------------------------------------------------------
# Kubernetes Provider Configuration (uses EKS cluster data)
#------------------------------------------------------------------------------

provider "kubernetes" {
  host                   = module.eks.cluster_endpoint
  cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)

  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    command     = "aws"
    args        = ["eks", "get-token", "--cluster-name", module.eks.cluster_name]
  }
}

#------------------------------------------------------------------------------
# Helm Provider Configuration (uses EKS cluster data)
#------------------------------------------------------------------------------

provider "helm" {
  kubernetes = {
    host                   = module.eks.cluster_endpoint
    cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)

    exec = {
      api_version = "client.authentication.k8s.io/v1beta1"
      command     = "aws"
      args        = ["eks", "get-token", "--cluster-name", module.eks.cluster_name]
    }
  }
}
