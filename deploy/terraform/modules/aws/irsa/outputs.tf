#------------------------------------------------------------------------------
# AWS IRSA (IAM Roles for Service Accounts) Module - Outputs
#------------------------------------------------------------------------------

output "iam_role_arn" {
  description = "ARN of the IAM role created for IRSA"
  value       = aws_iam_role.this.arn
}

output "iam_role_name" {
  description = "Name of the IAM role created for IRSA"
  value       = aws_iam_role.this.name
}
