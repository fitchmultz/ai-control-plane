#------------------------------------------------------------------------------
# AWS Complete Example - Outputs
#------------------------------------------------------------------------------
# This file defines all outputs for the AI Control Plane deployment.
#------------------------------------------------------------------------------

#------------------------------------------------------------------------------
# Cluster Outputs
#------------------------------------------------------------------------------

output "cluster_endpoint" {
  description = "Endpoint for the EKS cluster API server"
  value       = module.eks.cluster_endpoint
}

output "cluster_name" {
  description = "Name of the EKS cluster"
  value       = module.eks.cluster_name
}

output "cluster_arn" {
  description = "ARN of the EKS cluster"
  value       = module.eks.cluster_arn
}

output "cluster_version" {
  description = "Kubernetes version of the cluster"
  value       = module.eks.cluster_version
}

output "cluster_certificate_authority_data" {
  description = "Base64 encoded certificate data for cluster CA"
  value       = module.eks.cluster_certificate_authority_data
  sensitive   = true
}

output "kubeconfig_command" {
  description = "Command to update kubeconfig for kubectl access"
  value       = "aws eks update-kubeconfig --region ${var.aws_region} --name ${module.eks.cluster_name}"
}

#------------------------------------------------------------------------------
# VPC Outputs
#------------------------------------------------------------------------------

output "vpc_id" {
  description = "ID of the VPC"
  value       = module.vpc.vpc_id
}

output "vpc_cidr_block" {
  description = "CIDR block of the VPC"
  value       = module.vpc.vpc_cidr_block
}

output "private_subnet_ids" {
  description = "List of private subnet IDs"
  value       = module.vpc.private_subnet_ids
}

output "public_subnet_ids" {
  description = "List of public subnet IDs"
  value       = module.vpc.public_subnet_ids
}

#------------------------------------------------------------------------------
# Database Outputs
#------------------------------------------------------------------------------

output "database_endpoint" {
  description = "Connection endpoint of the RDS instance"
  value       = module.rds.db_instance_endpoint
}

output "database_address" {
  description = "Hostname of the RDS instance"
  value       = module.rds.db_instance_address
}

output "database_port" {
  description = "Port on which the RDS instance accepts connections"
  value       = module.rds.db_instance_port
}

output "database_name" {
  description = "Name of the default database"
  value       = module.rds.db_instance_name
}

output "database_username" {
  description = "Master username for the database"
  value       = module.rds.db_instance_username
}

output "database_url" {
  description = "PostgreSQL connection URL (sensitive)"
  value       = module.rds.database_url
  sensitive   = true
}

#------------------------------------------------------------------------------
# IRSA Outputs
#------------------------------------------------------------------------------

output "irsa_role_arn" {
  description = "ARN of the IAM role created for IRSA"
  value       = module.irsa.iam_role_arn
}

output "irsa_role_name" {
  description = "Name of the IAM role created for IRSA"
  value       = module.irsa.iam_role_name
}

#------------------------------------------------------------------------------
# Helm Release Outputs
#------------------------------------------------------------------------------

output "helm_release_name" {
  description = "Name of the Helm release"
  value       = module.helm_release.release_name
}

output "helm_release_namespace" {
  description = "Namespace where the Helm release is deployed"
  value       = module.helm_release.release_namespace
}

output "helm_release_status" {
  description = "Status of the Helm release"
  value       = module.helm_release.release_status
}

output "helm_chart_version" {
  description = "Version of the chart that was deployed"
  value       = module.helm_release.chart_version
}

#------------------------------------------------------------------------------
# Application URLs
#------------------------------------------------------------------------------

output "application_url" {
  description = "URL to access the AI Control Plane (LiteLLM gateway)"
  value       = var.enable_ingress ? "https://${var.ingress_host}" : "Use kubectl port-forward: kubectl port-forward svc/${var.helm_release_name}-litellm 4000:4000 -n ${var.namespace}"
}

output "application_https_url" {
  description = "HTTPS URL to access the AI Control Plane when ingress is enabled"
  value       = var.enable_ingress ? "https://${var.ingress_host}" : null
}

output "litellm_health_endpoint" {
  description = "Health check endpoint for LiteLLM"
  value       = var.enable_ingress ? "https://${var.ingress_host}/health" : "kubectl exec -n ${var.namespace} deployment/${var.helm_release_name}-litellm -- curl -s -H 'Authorization: Bearer <master-key>' http://localhost:4000/health"
}

#------------------------------------------------------------------------------
# Secrets Outputs (Sensitive)
#------------------------------------------------------------------------------

output "litellm_master_key" {
  description = "Master key for LiteLLM admin authentication (sensitive)"
  value       = var.litellm_master_key
  sensitive   = true
}

output "litellm_salt_key" {
  description = "Salt key for LiteLLM encryption (sensitive - never change after set)"
  value       = var.litellm_salt_key
  sensitive   = true
}

#------------------------------------------------------------------------------
# Command Helpers
#------------------------------------------------------------------------------

output "kubectl_commands" {
  description = "Useful kubectl commands for managing the deployment"
  value       = <<-EOT

    # Get pods in the namespace
    kubectl get pods -n ${var.namespace}

    # View LiteLLM logs
    kubectl logs -n ${var.namespace} -l app.kubernetes.io/component=litellm --tail=100 -f

    # Port-forward to access LiteLLM locally
    kubectl port-forward svc/${var.helm_release_name}-litellm 4000:4000 -n ${var.namespace}

    # Check service account and optional IRSA binding
    kubectl describe sa ${var.helm_release_name}-sa -n ${var.namespace}

    # Get secret information
    kubectl get secret ${module.secrets.secret_name} -n ${var.namespace}

  EOT
}

output "next_steps" {
  description = "Next steps after deployment"
  value       = <<-EOT

    ==========================================
    AI Control Plane Deployment Complete!
    ==========================================

    1. Configure kubectl:
       aws eks update-kubeconfig --region ${var.aws_region} --name ${module.eks.cluster_name}

    2. Verify deployment:
       kubectl get pods -n ${var.namespace}

    3. Access LiteLLM:
       ${var.enable_ingress ? "URL: https://${var.ingress_host}" : "Port-forward: kubectl port-forward svc/${var.helm_release_name}-litellm 4000:4000 -n ${var.namespace}"}

    4. Check health:
       ${var.enable_ingress ? "curl https://${var.ingress_host}/health" : "kubectl exec -n ${var.namespace} deployment/${var.helm_release_name}-litellm -- curl -s -H 'Authorization: Bearer <master-key>' http://localhost:4000/health"}

    5. View logs:
       kubectl logs -n ${var.namespace} -l app.kubernetes.io/component=litellm -f

    ==========================================
    Important Notes
    ==========================================

    - Master Key (sensitive): Used for LiteLLM admin API authentication
    - Salt Key (sensitive): Used for encryption - NEVER CHANGE after initial setup
    - Database: External RDS PostgreSQL instance
    - IRSA: Optional and scoped only to the AWS APIs you explicitly grant

  EOT
}
