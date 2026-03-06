# -----------------------------------------------------------------------------
# Helm Release Module - Outputs
# -----------------------------------------------------------------------------

output "release_name" {
  description = "Name of the Helm release"
  value       = helm_release.this.name
}

output "release_namespace" {
  description = "Namespace where the Helm release is deployed"
  value       = helm_release.this.namespace
}

output "release_status" {
  description = "Status of the Helm release (e.g., deployed, failed, pending-upgrade)"
  value       = helm_release.this.status
}

output "release_version" {
  description = "Version of the Helm release (incremented on each update)"
  value       = helm_release.this.version
}

output "chart_version" {
  description = "Version of the chart that was deployed"
  value       = helm_release.this.metadata[0].version
}

output "chart_name" {
  description = "Name of the chart that was deployed"
  value       = helm_release.this.metadata[0].name
}

output "chart_app_version" {
  description = "Application version of the deployed chart"
  value       = helm_release.this.metadata[0].app_version
}

output "release_id" {
  description = "Unique ID of the Helm release"
  value       = helm_release.this.id
}
