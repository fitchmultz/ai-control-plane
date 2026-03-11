# Outputs for Kubernetes Secrets Module

output "secret_name" {
  description = "Name of the created Kubernetes secret"
  value       = kubernetes_secret_v1.this.metadata[0].name
}

output "secret_namespace" {
  description = "Namespace of the created Kubernetes secret"
  value       = kubernetes_secret_v1.this.metadata[0].namespace
}

output "secret_type" {
  description = "Type of the created Kubernetes secret"
  value       = kubernetes_secret_v1.this.type
}

output "secret_data_keys" {
  description = "List of keys in the secret (data keys only, not values for security)"
  value       = keys(kubernetes_secret_v1.this.data)
  sensitive   = false
}
