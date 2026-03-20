#------------------------------------------------------------------------------
# AWS IRSA (IAM Roles for Service Accounts) Module - Variables
#------------------------------------------------------------------------------

variable "oidc_provider_arn" {
  description = "ARN of the EKS OIDC provider"
  type        = string
}

variable "oidc_provider_url" {
  description = "URL of the EKS OIDC provider (without https:// prefix)"
  type        = string
}

variable "namespace" {
  description = "Kubernetes namespace where the service account resides"
  type        = string
  default     = "acp"
}

variable "service_account_name" {
  description = "Name of the Kubernetes service account"
  type        = string
  default     = "ai-control-plane"
}

variable "role_name" {
  description = "Name of the IAM role to create"
  type        = string
  default     = "ai-control-plane-irsa"
}

variable "policy_statements" {
  description = "List of policy statements to include in the inline policy"
  type = list(object({
    effect    = string
    actions   = list(string)
    resources = list(string)
  }))
  default = []
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}
