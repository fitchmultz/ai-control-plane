# Outputs for Kubernetes Namespace Module

output "namespace_name" {
  description = "The name of the created Kubernetes namespace"
  value       = kubernetes_namespace_v1.this.metadata[0].name
}

output "namespace_id" {
  description = "The ID of the created Kubernetes namespace"
  value       = kubernetes_namespace_v1.this.id
}

output "namespace_uid" {
  description = "The UID of the created Kubernetes namespace"
  value       = kubernetes_namespace_v1.this.metadata[0].uid
}
