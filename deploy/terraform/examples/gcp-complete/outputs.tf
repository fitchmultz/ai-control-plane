# -----------------------------------------------------------------------------
# GCP Complete Example - Outputs
# -----------------------------------------------------------------------------
# These outputs provide important information about the deployed infrastructure
# and useful commands for connecting to the resources.
# -----------------------------------------------------------------------------

# -----------------------------------------------------------------------------
# Project Information
# -----------------------------------------------------------------------------

output "project_id" {
  description = "GCP project ID"
  value       = var.project_id
}

output "region" {
  description = "GCP region"
  value       = var.region
}

output "environment" {
  description = "Deployment environment"
  value       = var.environment
}

# -----------------------------------------------------------------------------
# VPC Outputs
# -----------------------------------------------------------------------------

output "vpc_network_name" {
  description = "Name of the VPC network"
  value       = module.vpc.network_name
}

output "vpc_network_id" {
  description = "ID of the VPC network"
  value       = module.vpc.network_id
}

output "gke_subnet_name" {
  description = "Name of the GKE subnet"
  value       = "${local.name}-gke-subnet"
}

output "nat_gateway_ip" {
  description = "External IP address of the Cloud NAT gateway"
  value       = module.vpc.nat_ip
}

# -----------------------------------------------------------------------------
# GKE Cluster Outputs
# -----------------------------------------------------------------------------

output "cluster_name" {
  description = "Name of the GKE cluster"
  value       = module.gke.cluster_name
}

output "cluster_endpoint" {
  description = "Endpoint IP address of the GKE cluster (sensitive)"
  value       = module.gke.endpoint
  sensitive   = true
}

output "cluster_location" {
  description = "Location (region) of the GKE cluster"
  value       = module.gke.cluster_location
}

output "cluster_master_version" {
  description = "Kubernetes master version"
  value       = module.gke.master_version
}

output "workload_identity_pool" {
  description = "Workload Identity Pool for the cluster"
  value       = module.gke.workload_identity_pool
}

output "node_pools" {
  description = "Map of node pool names to their details"
  value       = module.gke.node_pools
}

# -----------------------------------------------------------------------------
# Cloud SQL Outputs
# -----------------------------------------------------------------------------

output "database_instance_name" {
  description = "Name of the Cloud SQL instance"
  value       = module.cloudsql.instance_name
}

output "database_connection_name" {
  description = "Connection name for Cloud SQL Proxy"
  value       = module.cloudsql.connection_name
}

output "database_private_ip" {
  description = "Private IP address of the Cloud SQL instance"
  value       = module.cloudsql.private_ip_address
}

output "database_name" {
  description = "Name of the database"
  value       = module.cloudsql.database_name
}

output "database_user" {
  description = "Database username"
  value       = module.cloudsql.database_user
}

output "database_url_proxy" {
  description = "PostgreSQL connection URL for Cloud SQL Proxy (sensitive)"
  value       = module.cloudsql.database_url_proxy
  sensitive   = true
}

# -----------------------------------------------------------------------------
# Kubernetes Outputs
# -----------------------------------------------------------------------------

output "namespace" {
  description = "Kubernetes namespace for AI Control Plane"
  value       = module.namespace.namespace_name
}

output "secret_name" {
  description = "Name of the Kubernetes secret"
  value       = module.secrets.secret_name
}

# -----------------------------------------------------------------------------
# Helm Release Outputs
# -----------------------------------------------------------------------------

output "helm_release_name" {
  description = "Name of the Helm release"
  value       = module.helm_release.release_name
}

output "helm_release_status" {
  description = "Status of the Helm release"
  value       = module.helm_release.release_status
}

output "helm_release_version" {
  description = "Version of the Helm release"
  value       = module.helm_release.release_version
}

# -----------------------------------------------------------------------------
# Service Account Outputs
# -----------------------------------------------------------------------------

output "workload_identity_service_account" {
  description = "Email of the Workload Identity service account"
  value       = var.enable_workload_identity ? google_service_account.workload_identity[0].email : null
}

output "gke_node_service_account" {
  description = "Email of the GKE node service account"
  value       = module.gke.service_account_email
}

# -----------------------------------------------------------------------------
# Application URLs
# -----------------------------------------------------------------------------

output "application_url" {
  description = "URL to access the AI Control Plane (if ingress is enabled)"
  value       = var.ingress_enabled ? "https://${var.ingress_host}" : "Use kubectl port-forward (see connection commands)"
}

# -----------------------------------------------------------------------------
# Connection Commands
# -----------------------------------------------------------------------------

output "kubectl_connection_command" {
  description = "Command to configure kubectl for the GKE cluster"
  value       = module.gke.kubectl_connection_command
}

output "port_forward_command" {
  description = "Command to port-forward to the LiteLLM service"
  value       = "kubectl port-forward -n ${var.namespace} svc/${var.helm_release_name}-litellm 4000:4000"
}

output "cloud_sql_proxy_command" {
  description = "Command to start Cloud SQL Proxy for database access"
  value       = "cloud-sql-proxy ${module.cloudsql.connection_name} --port 5432"
}

output "get_secrets_command" {
  description = "Command to view the Kubernetes secrets (keys only)"
  value       = "kubectl get secret -n ${var.namespace} ${module.secrets.secret_name} -o jsonpath='{.data}' | jq 'keys'"
}

output "get_master_key_command" {
  description = "Command to retrieve the LiteLLM master key (sensitive)"
  value       = "kubectl get secret -n ${var.namespace} ${module.secrets.secret_name} -o jsonpath='{.data.LITELLM_MASTER_KEY}' | base64 -d"
  sensitive   = true
}

output "health_check_command" {
  description = "Command to check the health of the AI Control Plane"
  value       = "kubectl exec -n ${var.namespace} deployment/${var.helm_release_name}-litellm -- curl -s http://localhost:4000/health || echo 'Health check endpoint not available'"
}

# -----------------------------------------------------------------------------
# Next Steps
# -----------------------------------------------------------------------------

output "next_steps" {
  description = "Recommended next steps after deployment"
  value       = <<-EOT
    
    ✅ AI Control Plane deployment complete!
    
    Next steps:
    
    1. Configure kubectl:
       ${module.gke.kubectl_connection_command}
    
    2. Verify pods are running:
       kubectl get pods -n ${var.namespace}
    
    3. Access the LiteLLM UI (choose one method):
       
       a) Port-forward (local access):
          kubectl port-forward -n ${var.namespace} svc/${var.helm_release_name}-litellm 4000:4000
          Then open: http://localhost:4000
       
       b) Via Ingress (if enabled): ${var.ingress_enabled ? "https://${var.ingress_host}" : "Ingress not enabled - enable via var.ingress_enabled"}
    
    4. Configure API keys in LiteLLM UI or via environment variables
    
    5. View logs:
       kubectl logs -n ${var.namespace} -l app.kubernetes.io/name=litellm -f
    
    6. Database access via Cloud SQL Proxy:
       cloud-sql-proxy ${module.cloudsql.connection_name} --port 5432
       
    Documentation: https://github.com/your-org/ai-control-plane/blob/main/docs/DEPLOYMENT.md
    EOT
}
