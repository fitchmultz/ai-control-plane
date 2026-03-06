# -----------------------------------------------------------------------------
# Cluster Outputs
# -----------------------------------------------------------------------------

output "cluster_id" {
  description = "The unique identifier of the GKE cluster"
  value       = google_container_cluster.primary.id
}

output "cluster_name" {
  description = "The name of the GKE cluster"
  value       = google_container_cluster.primary.name
}

output "cluster_location" {
  description = "The location (region or zone) of the GKE cluster"
  value       = google_container_cluster.primary.location
}

output "endpoint" {
  description = "The endpoint IP address of the Kubernetes master"
  value       = google_container_cluster.primary.endpoint
  sensitive   = true
}

output "ca_certificate" {
  description = "The base64-encoded public certificate for the cluster's Certificate Authority"
  value       = google_container_cluster.primary.master_auth[0].cluster_ca_certificate
  sensitive   = true
}

output "master_version" {
  description = "The current Kubernetes master version"
  value       = google_container_cluster.primary.master_version
}

output "workload_identity_pool" {
  description = "The Workload Identity Pool for the cluster (format: PROJECT_ID.svc.id.goog)"
  value       = var.enable_workload_identity ? "${var.project_id}.svc.id.goog" : null
}

# -----------------------------------------------------------------------------
# Network Outputs
# -----------------------------------------------------------------------------

output "network" {
  description = "The VPC network self_link where the cluster is deployed"
  value       = google_container_cluster.primary.network
}

output "subnetwork" {
  description = "The subnetwork self_link where the cluster is deployed"
  value       = google_container_cluster.primary.subnetwork
}

output "pods_range_name" {
  description = "The name of the secondary IP range for pods"
  value       = var.pods_secondary_range_name
}

output "services_range_name" {
  description = "The name of the secondary IP range for services"
  value       = var.services_secondary_range_name
}

# -----------------------------------------------------------------------------
# Private Cluster Outputs
# -----------------------------------------------------------------------------

output "private_endpoint" {
  description = "The internal IP address of the Kubernetes master when private endpoint is enabled"
  value       = var.enable_private_nodes ? google_container_cluster.primary.private_cluster_config[0].private_endpoint : null
  sensitive   = true
}

output "public_endpoint" {
  description = "The external IP address of the Kubernetes master when private endpoint is disabled"
  value       = google_container_cluster.primary.private_cluster_config[0].public_endpoint
  sensitive   = true
}

output "master_ipv4_cidr_block" {
  description = "The CIDR block for the master endpoint"
  value       = var.enable_private_nodes ? var.master_ipv4_cidr_block : null
}

# -----------------------------------------------------------------------------
# Node Pool Outputs
# -----------------------------------------------------------------------------

output "node_pools" {
  description = "Map of node pool names to their details"
  value = {
    for name, pool in google_container_node_pool.pools : name => {
      name           = pool.name
      id             = pool.id
      instance_group_urls = pool.managed_instance_group_urls
      node_count     = pool.node_count
      version        = pool.version
    }
  }
}

output "node_pool_names" {
  description = "List of node pool names"
  value       = [for pool in google_container_node_pool.pools : pool.name]
}

# -----------------------------------------------------------------------------
# Service Account Outputs
# -----------------------------------------------------------------------------

output "service_account_email" {
  description = "The email address of the service account used by GKE nodes"
  value       = google_service_account.gke_nodes.email
}

output "service_account_name" {
  description = "The fully-qualified name of the service account used by GKE nodes"
  value       = google_service_account.gke_nodes.name
}

output "service_account_id" {
  description = "The account ID of the service account used by GKE nodes"
  value       = google_service_account.gke_nodes.account_id
}

# -----------------------------------------------------------------------------
# Connection Outputs
# -----------------------------------------------------------------------------

output "kubectl_connection_command" {
  description = "gcloud command to get credentials for kubectl"
  value       = "gcloud container clusters get-credentials ${var.cluster_name} --region=${var.region} --project=${var.project_id}"
}

# -----------------------------------------------------------------------------
# Attribute Outputs
# -----------------------------------------------------------------------------

output "release_channel" {
  description = "The release channel of the cluster"
  value       = google_container_cluster.primary.release_channel[0].channel
}

output "datapath_provider" {
  description = "The datapath provider for the cluster"
  value       = google_container_cluster.primary.datapath_provider
}

output "enable_workload_identity" {
  description = "Whether Workload Identity is enabled on the cluster"
  value       = var.enable_workload_identity
}

output "enable_private_nodes" {
  description = "Whether private nodes are enabled on the cluster"
  value       = var.enable_private_nodes
}

output "cluster_resource_labels" {
  description = "The resource labels applied to the cluster"
  value       = google_container_cluster.primary.resource_labels
}

output "maintenance_window" {
  description = "The maintenance window configuration"
  value = {
    start_time = var.maintenance_start_time
    end_time   = var.maintenance_end_time
    recurrence = var.maintenance_recurrence
  }
}
