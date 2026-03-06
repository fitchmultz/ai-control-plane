# Kubernetes Secret Resource
# Creates a Kubernetes secret with configurable type and data

resource "kubernetes_secret" "this" {
  metadata {
    name      = var.secret_name
    namespace = var.namespace
    labels      = var.labels
    annotations = var.annotations
  }

  type = var.type

  # Convert secret data map to Kubernetes secret data format
  data = var.secret_data
}
