# Outputs for Azure AKS Module

output "cluster_id" {
  description = "The Kubernetes Managed Cluster ID"
  value       = azurerm_kubernetes_cluster.this.id
}

output "cluster_name" {
  description = "The name of the AKS cluster"
  value       = azurerm_kubernetes_cluster.this.name
}

output "kube_config_raw" {
  description = "Raw Kubernetes config to be used by kubectl and other compatible tools"
  value       = azurerm_kubernetes_cluster.this.kube_config_raw
  sensitive   = true
}

output "kube_config" {
  description = "Kubernetes configuration object"
  value       = azurerm_kubernetes_cluster.this.kube_config
  sensitive   = true
}

output "host" {
  description = "The Kubernetes cluster server host"
  value       = azurerm_kubernetes_cluster.this.kube_config[0].host
}

output "client_certificate" {
  description = "Base64 encoded public certificate used by clients to authenticate to the Kubernetes cluster"
  value       = azurerm_kubernetes_cluster.this.kube_config[0].client_certificate
  sensitive   = true
}

output "client_key" {
  description = "Base64 encoded private key used by clients to authenticate to the Kubernetes cluster"
  value       = azurerm_kubernetes_cluster.this.kube_config[0].client_key
  sensitive   = true
}

output "cluster_ca_certificate" {
  description = "Base64 encoded public CA certificate used as the root of trust for the Kubernetes cluster"
  value       = azurerm_kubernetes_cluster.this.kube_config[0].cluster_ca_certificate
  sensitive   = true
}

output "oidc_issuer_url" {
  description = "The OIDC issuer URL for the cluster (used for Workload Identity)"
  value       = var.enable_oidc_issuer ? azurerm_kubernetes_cluster.this.oidc_issuer_url : null
}

output "oidc_issuer_enabled" {
  description = "Whether OIDC issuer is enabled"
  value       = azurerm_kubernetes_cluster.this.oidc_issuer_enabled
}

output "workload_identity_enabled" {
  description = "Whether Workload Identity is enabled"
  value       = azurerm_kubernetes_cluster.this.workload_identity_enabled
}

output "fqdn" {
  description = "The FQDN of the Azure Kubernetes Managed Cluster"
  value       = azurerm_kubernetes_cluster.this.fqdn
}

output "private_fqdn" {
  description = "The FQDN for private Kubernetes cluster"
  value       = azurerm_kubernetes_cluster.this.private_fqdn
}

output "node_resource_group" {
  description = "Auto-generated Resource Group containing AKS cluster resources"
  value       = azurerm_kubernetes_cluster.this.node_resource_group
}

output "control_plane_identity" {
  description = "The User Assigned Identity used by the AKS control plane"
  value = {
    id           = azurerm_user_assigned_identity.aks_control_plane.id
    client_id    = azurerm_user_assigned_identity.aks_control_plane.client_id
    principal_id = azurerm_user_assigned_identity.aks_control_plane.principal_id
  }
}

output "kubelet_identity" {
  description = "The User Assigned Identity used by the Kubelet"
  value = {
    id           = azurerm_user_assigned_identity.kubelet.id
    client_id    = azurerm_user_assigned_identity.kubelet.client_id
    principal_id = azurerm_user_assigned_identity.kubelet.principal_id
  }
}

output "node_pools" {
  description = "Map of created node pools"
  value = {
    for name, pool in azurerm_kubernetes_cluster_node_pool.user : name => {
      id              = pool.id
      name            = pool.name
      vm_size         = pool.vm_size
      node_count      = pool.node_count
      enable_auto_scaling = pool.enable_auto_scaling
      min_count       = pool.min_count
      max_count       = pool.max_count
      labels          = pool.node_labels
      taints          = pool.node_taints
    }
  }
}

output "system_node_pool" {
  description = "System node pool configuration"
  value = {
    name            = azurerm_kubernetes_cluster.this.default_node_pool[0].name
    vm_size         = azurerm_kubernetes_cluster.this.default_node_pool[0].vm_size
    node_count      = azurerm_kubernetes_cluster.this.default_node_pool[0].node_count
    enable_auto_scaling = azurerm_kubernetes_cluster.this.default_node_pool[0].enable_auto_scaling
    min_count       = azurerm_kubernetes_cluster.this.default_node_pool[0].min_count
    max_count       = azurerm_kubernetes_cluster.this.default_node_pool[0].max_count
  }
}
