# Azure Complete Example - Outputs
# All outputs from the Azure complete infrastructure deployment

#------------------------------------------------------------------------------
# Resource Group Outputs
#------------------------------------------------------------------------------

output "resource_group_name" {
  description = "Name of the Azure Resource Group"
  value       = azurerm_resource_group.main.name
}

output "resource_group_id" {
  description = "ID of the Azure Resource Group"
  value       = azurerm_resource_group.main.id
}

output "resource_group_location" {
  description = "Azure region where resources are deployed"
  value       = azurerm_resource_group.main.location
}

#------------------------------------------------------------------------------
# Network Outputs
#------------------------------------------------------------------------------

output "vnet_id" {
  description = "ID of the Virtual Network"
  value       = module.network.vnet_id
}

output "vnet_name" {
  description = "Name of the Virtual Network"
  value       = module.network.vnet_name
}

output "vnet_cidr" {
  description = "CIDR block of the Virtual Network"
  value       = module.network.vnet_cidr
}

output "subnet_ids" {
  description = "Map of subnet names to their IDs"
  value       = module.network.subnet_ids
}

output "subnet_names" {
  description = "Map of subnet names"
  value       = module.network.subnet_names
}

#------------------------------------------------------------------------------
# AKS Cluster Outputs
#------------------------------------------------------------------------------

output "cluster_name" {
  description = "Name of the AKS cluster"
  value       = module.aks.cluster_name
}

output "cluster_id" {
  description = "ID of the AKS cluster"
  value       = module.aks.cluster_id
}

output "cluster_fqdn" {
  description = "FQDN of the AKS cluster"
  value       = module.aks.fqdn
}

output "kube_config_raw" {
  description = "Raw kubeconfig for kubectl access (sensitive)"
  value       = module.aks.kube_config_raw
  sensitive   = true
}

output "kube_config_command" {
  description = "Command to configure kubectl for the cluster"
  value       = "az aks get-credentials --resource-group ${azurerm_resource_group.main.name} --name ${module.aks.cluster_name} --overwrite-existing"
}

output "cluster_host" {
  description = "Kubernetes API server host"
  value       = module.aks.host
}

output "oidc_issuer_url" {
  description = "OIDC issuer URL for Workload Identity"
  value       = module.aks.oidc_issuer_url
}

output "workload_identity_enabled" {
  description = "Whether Workload Identity is enabled"
  value       = module.aks.workload_identity_enabled
}

output "control_plane_identity" {
  description = "Control plane managed identity information"
  value       = module.aks.control_plane_identity
}

output "node_resource_group" {
  description = "Resource group containing AKS node resources"
  value       = module.aks.node_resource_group
}

output "system_node_pool" {
  description = "System node pool configuration"
  value       = module.aks.system_node_pool
}

output "user_node_pools" {
  description = "User node pools configuration"
  value       = module.aks.node_pools
}

#------------------------------------------------------------------------------
# PostgreSQL Outputs
#------------------------------------------------------------------------------

output "postgresql_server_name" {
  description = "Name of the PostgreSQL server"
  value       = module.postgresql.server_name
}

output "postgresql_fqdn" {
  description = "Fully qualified domain name of the PostgreSQL server"
  value       = module.postgresql.fqdn
}

output "postgresql_database_name" {
  description = "Name of the PostgreSQL database"
  value       = module.postgresql.database_name
}

output "postgresql_admin_username" {
  description = "PostgreSQL administrator username"
  value       = module.postgresql.administrator_login
}

output "postgresql_connection_string_masked" {
  description = "PostgreSQL connection string with password masked"
  value       = "postgresql://${var.postgresql_admin_username}:****@${module.postgresql.fqdn}:5432/${var.postgresql_database_name}?sslmode=require"
  sensitive   = false
}

output "postgresql_private_endpoint_ip" {
  description = "Private IP address of the PostgreSQL private endpoint"
  value       = module.postgresql.private_endpoint_private_ip
}

#------------------------------------------------------------------------------
# Kubernetes Outputs
#------------------------------------------------------------------------------

output "kubernetes_namespace" {
  description = "Kubernetes namespace where AI Control Plane is deployed"
  value       = module.namespace.namespace_name
}

output "kubernetes_secret_name" {
  description = "Name of the Kubernetes secret containing credentials"
  value       = module.secrets.secret_name
}

#------------------------------------------------------------------------------
# Helm Release Outputs
#------------------------------------------------------------------------------

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

#------------------------------------------------------------------------------
# Application Access Outputs
#------------------------------------------------------------------------------

output "application_url" {
  description = "URL to access the AI Control Plane application"
  value       = var.ingress_enabled ? "https://${var.ingress_host}" : "Internal ClusterIP - use kubectl port-forward"
}

output "application_internal_endpoint" {
  description = "Internal Kubernetes endpoint for the application"
  value       = "${var.helm_release_name}-litellm.${var.helm_namespace}.svc.cluster.local:4000"
}

#------------------------------------------------------------------------------
# Connection Commands
#------------------------------------------------------------------------------

output "connect_kubectl" {
  description = "Command to connect kubectl to the cluster"
  value       = "az aks get-credentials --resource-group ${azurerm_resource_group.main.name} --name ${module.aks.cluster_name} --overwrite-existing"
}

output "connect_port_forward" {
  description = "Command to port-forward to the LiteLLM service for local access"
  value       = "kubectl port-forward svc/${var.helm_release_name}-litellm 4000:4000 -n ${var.helm_namespace}"
}

output "check_pods" {
  description = "Command to check pod status"
  value       = "kubectl get pods -n ${var.helm_namespace}"
}

output "check_logs" {
  description = "Command to view LiteLLM logs"
  value       = "kubectl logs -l app.kubernetes.io/component=litellm -n ${var.helm_namespace} --tail=100 -f"
}

output "get_master_key" {
  description = "Command to retrieve the LiteLLM master key from the secret"
  value       = "kubectl get secret ${module.secrets.secret_name} -n ${var.helm_namespace} -o jsonpath='{.data.LITELLM_MASTER_KEY}' | base64 -d"
  sensitive   = true
}

output "get_database_url" {
  description = "Command to retrieve the database URL from the secret"
  value       = "kubectl get secret ${module.secrets.secret_name} -n ${var.helm_namespace} -o jsonpath='{.data.DATABASE_URL}' | base64 -d"
  sensitive   = true
}

#------------------------------------------------------------------------------
# Secrets (Sensitive)
#------------------------------------------------------------------------------

output "litellm_master_key" {
  description = "LiteLLM master key"
  value       = var.litellm_master_key
  sensitive   = true
}

output "litellm_salt_key" {
  description = "LiteLLM salt key"
  value       = var.litellm_salt_key
  sensitive   = true
}

output "postgresql_admin_password" {
  description = "PostgreSQL administrator password (auto-generated)"
  value       = random_password.postgresql.result
  sensitive   = true
}

#------------------------------------------------------------------------------
# Summary Output
#------------------------------------------------------------------------------

output "deployment_summary" {
  description = "Summary of the deployment"
  value       = <<EOF

╔══════════════════════════════════════════════════════════════════════════════╗
║                    AI Control Plane Deployment Complete                      ║
╠══════════════════════════════════════════════════════════════════════════════╣
  Environment:        ${var.environment}
  Location:           ${azurerm_resource_group.main.location}
  Resource Group:     ${azurerm_resource_group.main.name}
╠══════════════════════════════════════════════════════════════════════════════╣
  AKS Cluster:        ${module.aks.cluster_name}
  PostgreSQL Server:  ${module.postgresql.server_name}
  Namespace:          ${var.helm_namespace}
╠══════════════════════════════════════════════════════════════════════════════╣
  Connection Steps:
  1. Authenticate:    az login
  2. Configure kubectl:
                      az aks get-credentials --resource-group ${azurerm_resource_group.main.name} --name ${module.aks.cluster_name}
  3. Check pods:      kubectl get pods -n ${var.helm_namespace}
  4. Port forward:    kubectl port-forward svc/${var.helm_release_name}-litellm 4000:4000 -n ${var.helm_namespace}
  5. Access:          ${var.ingress_enabled ? "https://${var.ingress_host}" : "Local-only port-forward access"}
╚══════════════════════════════════════════════════════════════════════════════╝
EOF
}
