# Common Kubernetes Secrets Module

Terraform module for creating Kubernetes Secrets with support for any secret type and integration with external secret managers.

## Overview

This module creates a Kubernetes Secret resource with configurable type, labels, and annotations. It is designed to work with cloud secret managers (AWS Secrets Manager, Azure Key Vault, GCP Secret Manager) by accepting pre-fetched secret data as input—no direct cloud provider dependencies.

## Requirements

- Terraform >= 1.0
- Kubernetes provider >= 2.0 (configured by the caller)

## Usage

### Basic Usage (Opaque Secret)

```hcl
module "app_secrets" {
  source = "./modules/common/secrets"

  namespace   = "ai-control-plane"
  secret_name = "app-config"
  
  secret_data = {
    "api-key"    = var.api_key
    "db-password" = var.db_password
  }
}
```

### TLS Secret

```hcl
module "tls_secret" {
  source = "./modules/common/secrets"

  namespace   = "ai-control-plane"
  secret_name = "gateway-tls"
  type        = "kubernetes.io/tls"
  
  secret_data = {
    "tls.crt" = var.tls_certificate
    "tls.key" = var.tls_private_key
  }
}
```

### Docker Registry Secret

```hcl
module "registry_secret" {
  source = "./modules/common/secrets"

  namespace   = "ai-control-plane"
  secret_name = "regcred"
  type        = "kubernetes.io/dockerconfigjson"
  
  secret_data = {
    ".dockerconfigjson" = jsonencode({
      auths = {
        "https://index.docker.io/v1/" = {
          username = var.registry_username
          password = var.registry_password
          email    = var.registry_email
          auth     = base64encode("${var.registry_username}:${var.registry_password}")
        }
      }
    })
  }
}
```

### Integration with AWS Secrets Manager

```hcl
# Fetch secrets from AWS Secrets Manager
data "aws_secretsmanager_secret_version" "ai_control_plane" {
  secret_id = "ai-control-plane/production"
}

locals {
  secrets = jsondecode(data.aws_secretsmanager_secret_version.ai_control_plane.secret_string)
}

module "ai_control_plane_secrets" {
  source = "./modules/common/secrets"

  namespace   = "ai-control-plane"
  secret_name = "ai-control-plane-secrets"
  
  secret_data = {
    "LITELLM_MASTER_KEY"      = local.secrets.litellm_master_key
    "LITELLM_SALT_KEY"        = local.secrets.litellm_salt_key
    "ANTHROPIC_API_KEY"       = local.secrets.anthropic_api_key
    "OPENAI_API_KEY"          = local.secrets.openai_api_key
    "DATABASE_URL"            = local.secrets.database_url
  }

  labels = {
    "app.kubernetes.io/part-of"   = "ai-control-plane"
    "app.kubernetes.io/component" = "secrets"
  }
}
```

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| `namespace` | Kubernetes namespace where the secret will be created | `string` | n/a | yes |
| `secret_name` | Name of the Kubernetes secret | `string` | `"ai-control-plane-secrets"` | no |
| `secret_data` | Map of secret key-value pairs | `map(string)` | `{}` | no |
| `type` | Type of Kubernetes secret | `string` | `"Opaque"` | no |
| `labels` | Labels to apply to the secret | `map(string)` | `{}` | no |
| `annotations` | Annotations to apply to the secret | `map(string)` | `{}` | no |

### Secret Types

Common Kubernetes secret types supported:

| Type | Use Case |
|------|----------|
| `Opaque` | General-purpose secrets (default) |
| `kubernetes.io/tls` | TLS certificates and keys |
| `kubernetes.io/dockerconfigjson` | Docker registry credentials |
| `kubernetes.io/basic-auth` | Basic authentication credentials |
| `kubernetes.io/ssh-auth` | SSH authentication credentials |
| `kubernetes.io/service-account-token` | Service account tokens |

## Outputs

| Name | Description |
|------|-------------|
| `secret_name` | Name of the created Kubernetes secret |
| `secret_namespace` | Namespace of the created Kubernetes secret |
| `secret_type` | Type of the created Kubernetes secret |
| `secret_data_keys` | List of keys in the secret (values excluded for security) |

## Security Notes

- **Sensitive Data**: The `secret_data` variable is marked as `sensitive = true` to prevent exposure in Terraform logs and outputs.
- **Provider Configuration**: This module requires the Kubernetes provider to be configured by the caller.
- **No Cloud Dependencies**: This module has no direct cloud provider dependencies—secret values must be fetched externally (e.g., via data sources) and passed as inputs.

## Testing

Validate the module with:

```bash
terraform init
terraform validate
terraform plan
```
