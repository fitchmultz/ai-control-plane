#-------------------------------------------------------------------------------
# Cluster Outputs
#-------------------------------------------------------------------------------

output "cluster_id" {
  description = "The name/id of the EKS cluster"
  value       = aws_eks_cluster.this.id
}

output "cluster_name" {
  description = "The name of the EKS cluster"
  value       = aws_eks_cluster.this.name
}

output "cluster_arn" {
  description = "The ARN of the EKS cluster"
  value       = aws_eks_cluster.this.arn
}

output "cluster_endpoint" {
  description = "The endpoint for the EKS cluster API server"
  value       = aws_eks_cluster.this.endpoint
}

output "cluster_version" {
  description = "The Kubernetes version of the cluster"
  value       = aws_eks_cluster.this.version
}

output "cluster_platform_version" {
  description = "Platform version of the EKS cluster"
  value       = aws_eks_cluster.this.platform_version
}

output "cluster_certificate_authority_data" {
  description = "Base64 encoded certificate data for cluster CA"
  value       = aws_eks_cluster.this.certificate_authority[0].data
}

output "cluster_oidc_issuer_url" {
  description = "The URL on the EKS cluster OIDC Issuer"
  value       = var.enable_irsa ? aws_eks_cluster.this.identity[0].oidc[0].issuer : null
}

output "cluster_security_group_id" {
  description = "Security group ID attached to the EKS control plane"
  value       = aws_eks_cluster.this.vpc_config[0].cluster_security_group_id
}

output "cluster_primary_security_group_id" {
  description = "Cluster security group created by Amazon EKS"
  value       = aws_eks_cluster.this.vpc_config[0].cluster_security_group_id
}

output "cluster_iam_role_arn" {
  description = "IAM role ARN of the EKS cluster"
  value       = aws_iam_role.cluster.arn
}

output "cluster_iam_role_name" {
  description = "IAM role name of the EKS cluster"
  value       = aws_iam_role.cluster.name
}

#-------------------------------------------------------------------------------
# OIDC Provider Outputs
#-------------------------------------------------------------------------------

output "oidc_provider_arn" {
  description = "The ARN of the OIDC Provider"
  value       = var.enable_irsa ? aws_iam_openid_connect_provider.this[0].arn : null
}

output "oidc_provider_url" {
  description = "The URL of the OIDC Provider"
  value       = var.enable_irsa ? aws_iam_openid_connect_provider.this[0].url : null
}

#-------------------------------------------------------------------------------
# Node Group Outputs
#-------------------------------------------------------------------------------

output "node_groups" {
  description = "Map of node group attributes"
  value       = aws_eks_node_group.this
}

output "node_group_names" {
  description = "List of node group names"
  value       = keys(aws_eks_node_group.this)
}

output "node_iam_role_arn" {
  description = "IAM role ARN for EKS node groups"
  value       = aws_iam_role.node.arn
}

output "node_iam_role_name" {
  description = "IAM role name for EKS node groups"
  value       = aws_iam_role.node.name
}

#-------------------------------------------------------------------------------
# Security Group Outputs
#-------------------------------------------------------------------------------

output "cluster_security_group_additional_id" {
  description = "Security group ID created by this module for the cluster"
  value       = aws_security_group.cluster.id
}

output "node_security_group_id" {
  description = "Security group ID for EKS node groups"
  value       = aws_security_group.node.id
}

#-------------------------------------------------------------------------------
# Network Outputs
#-------------------------------------------------------------------------------

output "cluster_vpc_config" {
  description = "VPC configuration for the cluster"
  value       = aws_eks_cluster.this.vpc_config
}

#-------------------------------------------------------------------------------
# Addon Outputs
#-------------------------------------------------------------------------------

output "addon_vpc_cni_version" {
  description = "Version of the VPC CNI addon"
  value       = aws_eks_addon.vpc_cni.addon_version
}

output "addon_coredns_version" {
  description = "Version of the CoreDNS addon"
  value       = aws_eks_addon.coredns.addon_version
}

output "addon_kube_proxy_version" {
  description = "Version of the kube-proxy addon"
  value       = aws_eks_addon.kube_proxy.addon_version
}

#-------------------------------------------------------------------------------
# KMS Outputs
#-------------------------------------------------------------------------------

output "kms_key_arn" {
  description = "ARN of the KMS key for cluster encryption"
  value       = var.create_kms_key && var.cluster_encryption_config == null ? aws_kms_key.eks[0].arn : null
}

output "kms_key_id" {
  description = "ID of the KMS key for cluster encryption"
  value       = var.create_kms_key && var.cluster_encryption_config == null ? aws_kms_key.eks[0].key_id : null
}
