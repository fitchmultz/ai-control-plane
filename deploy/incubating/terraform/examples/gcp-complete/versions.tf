# -----------------------------------------------------------------------------
# Terraform and Provider Versions
# -----------------------------------------------------------------------------
# This file defines the required Terraform version and provider versions
# for the GCP Complete Example.
# -----------------------------------------------------------------------------

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 5.0.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = ">= 5.0.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 2.23.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = ">= 2.12.0"
    }
    random = {
      source  = "hashicorp/random"
      version = ">= 3.5.0"
    }
  }

  # -----------------------------------------------------------------------------
  # Backend Configuration (Optional)
  # -----------------------------------------------------------------------------
  # Uncomment and configure to use GCS backend for state storage.
  # Using a GCS backend is recommended for production deployments.
  #
  # backend "gcs" {
  #   bucket = "your-terraform-state-bucket"
  #   prefix = "ai-control-plane/gcp-complete"
  # }
}

# -----------------------------------------------------------------------------
# Provider Configuration
# -----------------------------------------------------------------------------

# Google Cloud Provider
provider "google" {
  project = var.project_id
  region  = var.region
}

provider "google-beta" {
  project = var.project_id
  region  = var.region
}

# Kubernetes Provider - configured using GKE cluster data
provider "kubernetes" {
  host  = "https://${module.gke.endpoint}"
  token = data.google_client_config.default.access_token
  cluster_ca_certificate = base64decode(
    module.gke.ca_certificate
  )
}

# Helm Provider - configured using GKE cluster data
provider "helm" {
  kubernetes = {
    host  = "https://${module.gke.endpoint}"
    token = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(
      module.gke.ca_certificate
    )
  }
}

# -----------------------------------------------------------------------------
# Data Sources
# -----------------------------------------------------------------------------

# Get the default client config for authentication
data "google_client_config" "default" {}
