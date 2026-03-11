# Kubernetes Namespace Terraform Module

A reusable Terraform module for creating Kubernetes namespaces with support for labels and annotations.

## Features

- Creates a Kubernetes namespace
- Supports custom labels and annotations
- Idempotent creation (safe to run multiple times)
- Input validation for namespace names

## Requirements

| Name | Version |
|------|---------|
| terraform | >= 1.0.0 |
| kubernetes | >= 2.0.0 |

## Providers

| Name | Version |
|------|---------|
| kubernetes | >= 2.0.0 |

**Note:** This module requires the Kubernetes provider to be configured by the caller.

## Usage

### Basic Usage

```hcl
module "namespace" {
  source = "./modules/common/kubernetes-namespace"

  name = "my-application"
}
```

### With Labels and Annotations

```hcl
module "namespace" {
  source = "./modules/common/kubernetes-namespace"

  name = "my-application"

  labels = {
    environment = "production"
    team        = "platform"
    managed-by  = "terraform"
  }

  annotations = {
    "description" = "Namespace for my application"
    "owner"       = "platform-team@example.com"
  }
}
```

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| `name` | Name of the Kubernetes namespace | `string` | n/a | yes |
| `labels` | Labels to apply to the namespace | `map(string)` | `{}` | no |
| `annotations` | Annotations to apply to the namespace | `map(string)` | `{}` | no |

### Namespace Name Validation

The namespace name must:
- Contain only lowercase alphanumeric characters or '-'
- Start and end with an alphanumeric character
- Be a valid DNS subdomain name

## Outputs

| Name | Description |
|------|-------------|
| `namespace_name` | The name of the created Kubernetes namespace |
| `namespace_id` | The ID of the created Kubernetes namespace |
| `namespace_uid` | The UID of the created Kubernetes namespace |

## Example: Using Outputs

```hcl
module "namespace" {
  source = "./modules/common/kubernetes-namespace"
  name   = "my-app"
}

# Reference the namespace in other resources
resource "kubernetes_deployment" "app" {
  metadata {
    name      = "my-deployment"
    namespace = module.namespace.namespace_name
  }
  # ...
}
```

## Idempotency

This module is idempotent - running `terraform apply` multiple times with the same configuration will:
1. Create the namespace if it doesn't exist
2. Update labels/annotations if they have changed
3. Make no changes if the namespace already matches the desired state
