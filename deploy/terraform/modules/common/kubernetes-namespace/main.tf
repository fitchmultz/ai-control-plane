# Kubernetes Namespace Resource
# Creates a Kubernetes namespace with optional labels and annotations

resource "kubernetes_namespace" "this" {
  metadata {
    name        = var.name
    labels      = var.labels
    annotations = var.annotations
  }
}
