# Kubernetes Secret Resource
# Creates a Kubernetes secret with configurable type and data

terraform {
  required_providers {
    kubernetes = {
      source = "hashicorp/kubernetes"
    }
  }
}

resource "kubernetes_secret_v1" "this" {
  metadata {
    name        = var.secret_name
    namespace   = var.namespace
    labels      = var.labels
    annotations = var.annotations
  }

  type = var.type

  # Convert secret data map to Kubernetes secret data format
  data = var.secret_data
}
