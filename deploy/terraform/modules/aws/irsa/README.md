# AWS IRSA (IAM Roles for Service Accounts) Terraform Module

This Terraform module creates an IAM role that can be assumed by Kubernetes pods using a specific service account via IAM Roles for Service Accounts (IRSA).

## Overview

IRSA allows pods running on Amazon EKS to assume IAM roles using Kubernetes service accounts. This enables fine-grained access control where specific pods can have specific AWS permissions without sharing node-level IAM credentials.

## Features

- Creates an IAM role with a trust policy for IRSA
- Configurable trust relationship for specific namespace and service account
- Optional inline policy with custom permissions
- Full OIDC provider integration
- Tagging support for all resources

## Usage

### Basic Example

```hcl
module "irsa" {
  source = "./modules/aws/irsa"

  oidc_provider_arn = module.eks.oidc_provider_arn
  oidc_provider_url = module.eks.oidc_provider_url

  namespace            = "acp"
  service_account_name = "ai-control-plane"
  role_name            = "ai-control-plane-irsa"

  tags = {
    Environment = "production"
    Project     = "ai-control-plane"
  }
}
```

### With Custom Policy

```hcl
module "irsa" {
  source = "./modules/aws/irsa"

  oidc_provider_arn = module.eks.oidc_provider_arn
  oidc_provider_url = module.eks.oidc_provider_url

  namespace            = "acp"
  service_account_name = "ai-control-plane"
  role_name            = "ai-control-plane-irsa"

  policy_statements = [
    {
      effect    = "Allow"
      actions   = ["s3:GetObject", "s3:PutObject"]
      resources = ["arn:aws:s3:::my-bucket/*"]
    },
    {
      effect    = "Allow"
      actions   = ["secretsmanager:GetSecretValue"]
      resources = ["arn:aws:secretsmanager:*:*:secret:my-secret-*"]
    }
  ]

  tags = {
    Environment = "production"
    Project     = "ai-control-plane"
  }
}
```

## Requirements

| Name | Version |
|------|---------|
| terraform | >= 1.0 |
| aws | ~> 5.0 |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| `oidc_provider_arn` | ARN of the EKS OIDC provider | `string` | n/a | yes |
| `oidc_provider_url` | URL of the EKS OIDC provider (without https:// prefix) | `string` | n/a | yes |
| `namespace` | Kubernetes namespace where the service account resides | `string` | `"acp"` | no |
| `service_account_name` | Name of the Kubernetes service account | `string` | `"ai-control-plane"` | no |
| `role_name` | Name of the IAM role to create | `string` | `"ai-control-plane-irsa"` | no |
| `policy_statements` | List of policy statements for the inline policy | `list(object)` | `[]` | no |
| `tags` | Tags to apply to all resources | `map(string)` | `{}` | no |

### Policy Statements Format

The `policy_statements` variable accepts a list of objects with the following structure:

```hcl
{
  effect    = "Allow" or "Deny"
  actions   = ["action1", "action2"]
  resources = ["arn:aws:...", "arn:aws:..."]
}
```

## Outputs

| Name | Description |
|------|-------------|
| `iam_role_arn` | ARN of the IAM role created for IRSA |
| `iam_role_name` | Name of the IAM role created for IRSA |

## Trust Policy

The module creates an IAM role with the following trust policy structure:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::ACCOUNT_ID:oidc-provider/oidc.eks.REGION.amazonaws.com/id/CLUSTER_ID"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "oidc.eks.REGION.amazonaws.com/id/CLUSTER_ID:sub": "system:serviceaccount:NAMESPACE:SA_NAME",
          "oidc.eks.REGION.amazonaws.com/id/CLUSTER_ID:aud": "sts.amazonaws.com"
        }
      }
    }
  ]
}
```

## Kubernetes Service Account Annotation

After creating the IAM role, annotate your Kubernetes service account:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ai-control-plane
  namespace: acp
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::ACCOUNT_ID:role/ai-control-plane-irsa
```

## Notes

- The OIDC provider URL should be provided without the `https://` prefix
- The trust policy only allows pods with the exact service account in the exact namespace to assume the role
- If no `policy_statements` are provided, no inline policy is attached to the role
- You can attach additional managed policies to the role outside of this module if needed
