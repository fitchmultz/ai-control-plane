# Helm Release Terraform Module

Terraform module for deploying the AI Control Plane Helm chart to a Kubernetes cluster.

## Features

- Deploys the AI Control Plane Helm chart (LiteLLM gateway + optional PostgreSQL)
- Supports both local chart paths and remote chart repositories
- Configurable namespace creation
- Flexible values configuration via files and inline maps
- Safe deployment options with timeout, atomic, and wait controls
- Deep merge support for values (files + inline values)

## Requirements

| Name | Version |
|------|---------|
| terraform | >= 1.0 |
| helm | >= 2.12.0 |

## Providers

| Name | Version |
|------|---------|
| helm | >= 2.12.0 |

## Usage

### Basic Example

```hcl
# Configure the Helm provider (required by caller)
provider "helm" {
  kubernetes {
    config_path = "~/.kube/config"
  }
}

# Deploy AI Control Plane
module "ai_control_plane" {
  source = "./modules/common/helm-release"

  release_name = "acp"
  namespace    = "ai-control-plane"
  
  chart_path = "../../deploy/helm/ai-control-plane"
}
```

### With Values Files

```hcl
module "ai_control_plane" {
  source = "./modules/common/helm-release"

  release_name = "acp"
  namespace    = "acp"
  
  chart_path   = "../../deploy/helm/ai-control-plane"
  values_files = [
    "./values/base.yaml",
    "./values/production.yaml"
  ]
}
```

### With Inline Values

```hcl
module "ai_control_plane" {
  source = "./modules/common/helm-release"

  release_name = "acp"
  namespace    = "acp"
  
  chart_path = "../../deploy/helm/ai-control-plane"
  
  values = {
    profile = "production"
    
    litellm = {
      replicaCount = 2
      
      resources = {
        limits = {
          cpu    = "2000m"
          memory = "2Gi"
        }
        requests = {
          cpu    = "500m"
          memory = "1Gi"
        }
      }
    }
    
    postgres = {
      enabled = false  # Use external database
    }
    
    secrets = {
      create = false
      existingSecret = {
        name = "ai-control-plane-secrets"
      }
    }
    
    ingress = {
      enabled   = true
      className = "nginx"
      hosts = [{
        host = "ai-gateway.example.com"
        paths = [{
          path     = "/"
          pathType = "Prefix"
        }]
      }]
    }
  }
}
```

### With Existing Namespace

```hcl
module "ai_control_plane" {
  source = "./modules/common/helm-release"

  release_name     = "acp"
  namespace        = "acp"
  create_namespace = false  # Namespace already exists
  
  chart_path = "../../deploy/helm/ai-control-plane"
}
```

### Safe Deployment Options

```hcl
module "ai_control_plane" {
  source = "./modules/common/helm-release"

  release_name = "acp"
  namespace    = "acp"
  chart_path   = "../../deploy/helm/ai-control-plane"
  
  # Safety options for production deployments
  timeout = 900        # 15 minute timeout
  atomic  = true       # Rollback on failure
  wait    = true       # Wait for all resources ready
  
  # Additional safety options
  cleanup_on_fail = true    # Clean up on failed upgrade
  wait_for_jobs   = true    # Wait for any jobs to complete
}
```

### Combined Example (Production Deployment)

```hcl
module "ai_control_plane" {
  source = "./modules/common/helm-release"

  release_name = "acp"
  namespace    = "ai-control-plane"
  description  = "AI Control Plane - Production"

  chart_path = "../../deploy/helm/ai-control-plane"
  
  # Use external secrets and database
  values = {
    profile = "production"
    
    litellm = {
      replicaCount = 3
      mode         = "online"
    }
    
    postgres = {
      enabled = false
    }
    
    secrets = {
      create = false
      existingSecret = {
        name = "acp-secrets"
      }
    }
    
    externalDatabase = {
      existingSecret    = "acp-db-secret"
      existingSecretKey = "DATABASE_URL"
    }
    
    autoscaling = {
      enabled                          = true
      minReplicas                      = 3
      maxReplicas                      = 10
      targetCPUUtilizationPercentage   = 70
      targetMemoryUtilizationPercentage = 80
    }
    
    podDisruptionBudget = {
      enabled      = true
      minAvailable = 2
    }
  }
  
  # Safe deployment settings
  timeout         = 900
  atomic          = true
  wait            = true
  wait_for_jobs   = true
  cleanup_on_fail = true
}
```

## Inputs

### Chart Configuration

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| chart_path | Path to the Helm chart directory (relative to module caller or absolute path) | `string` | `"../../deploy/helm/ai-control-plane"` | no |
| chart_version | Chart version to deploy. If not set, uses the version from Chart.yaml | `string` | `null` | no |

### Release Configuration

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| release_name | Name of the Helm release | `string` | `"acp"` | no |
| namespace | Kubernetes namespace to deploy the Helm release into | `string` | `"acp"` | no |
| create_namespace | Whether to create the namespace if it doesn't exist | `bool` | `true` | no |
| description | Description of the Helm release | `string` | `"AI Control Plane - LiteLLM gateway with optional PostgreSQL"` | no |

### Values Configuration

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| values_files | List of paths to values files to use for the Helm release (applied in order) | `list(string)` | `[]` | no |
| values | Map of inline values to pass to the Helm chart. Deep merged with values_files | `any` | `{}` | no |

### Deployment Options

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| timeout | Timeout in seconds for the Helm release operation | `number` | `600` | no |
| atomic | If true, installation/upgrade rolls back on failure. Prevents partial deployments | `bool` | `true` | no |
| wait | If true, waits for all resources to be ready before marking release as successful | `bool` | `true` | no |
| wait_for_jobs | If true, waits for all Jobs to complete before marking release as successful | `bool` | `false` | no |
| cleanup_on_fail | If true, deletes newly created resources on failure during upgrade | `bool` | `false` | no |
| disable_openapi_validation | If true, skips OpenAPI schema validation of manifests | `bool` | `false` | no |
| disable_webhooks | If true, prevents Helm from running hooks | `bool` | `false` | no |
| force_update | If true, forces resource updates through delete/recreate when necessary | `bool` | `false` | no |
| recreate_pods | If true, performs pods restart for the resource if applicable | `bool` | `false` | no |
| replace | If true, reuses the given name even if that name is already used | `bool` | `false` | no |
| reuse_values | If true, reuses the last release's values and merges with new values | `bool` | `false` | no |
| reset_values | If true, resets values to the ones built into the chart | `bool` | `false` | no |

### Kubernetes Provider Configuration

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| kubeconfig_path | Path to kubeconfig file. If not set, uses KUBE_CONFIG_PATHS env var or default kubeconfig | `string` | `null` | no |
| kubeconfig_context | Kubernetes context to use. If not set, uses current context | `string` | `null` | no |

## Outputs

| Name | Description |
|------|-------------|
| release_name | Name of the Helm release |
| release_namespace | Namespace where the Helm release is deployed |
| release_status | Status of the Helm release (e.g., deployed, failed, pending-upgrade) |
| release_version | Version of the Helm release (incremented on each update) |
| chart_version | Version of the chart that was deployed |
| chart_name | Name of the chart that was deployed |
| chart_app_version | Application version of the deployed chart |
| release_id | Unique ID of the Helm release |

## Notes

### Helm Provider Configuration

This module requires the Helm provider to be configured by the caller. The module does not configure the provider itself to allow flexibility in authentication methods.

Example provider configurations:

**Using kubeconfig file:**
```hcl
provider "helm" {
  kubernetes {
    config_path    = "~/.kube/config"
    config_context = "my-cluster"
  }
}
```

**Using EKS (AWS):**
```hcl
provider "helm" {
  kubernetes {
    host                   = module.eks.cluster_endpoint
    cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)
    exec {
      api_version = "client.authentication.k8s.io/v1beta1"
      command     = "aws"
      args        = ["eks", "get-token", "--cluster-name", module.eks.cluster_name]
    }
  }
}
```

**Using GKE (GCP):**
```hcl
provider "helm" {
  kubernetes {
    host                   = "https://${module.gke.cluster_endpoint}"
    cluster_ca_certificate = base64decode(module.gke.cluster_ca_certificate)
    token                  = data.google_client_config.default.access_token
  }
}
```

### Values Precedence

Values are applied in the following order (later values override earlier ones):

1. Chart default values (`values.yaml` in the chart)
2. Values from `values_files` (in order provided)
3. Values from `values` map (inline)

### Namespace Handling

- When `create_namespace = true`, the namespace will be created if it doesn't exist
- When `create_namespace = false`, the namespace must already exist
- The module does not manage namespace deletion (Helm preserves namespaces)

### Deployment Safety

The default settings (`atomic = true`, `wait = true`, `timeout = 600`) are designed for safe deployments:

- **Atomic**: Failed deployments are automatically rolled back
- **Wait**: Terraform waits for all pods to be ready before continuing
- **Timeout**: Operations timeout after 10 minutes to prevent indefinite hangs

For development or CI/CD where faster feedback is needed, you may want to adjust these:

```hcl
module "ai_control_plane" {
  source = "./modules/common/helm-release"
  
  timeout = 300  # 5 minutes
  atomic  = false  # Don't rollback, easier to debug failures
  wait    = false  # Don't wait for pods (faster terraform apply)
  
  # ... other configuration
}
```

### Chart Path

The default `chart_path` assumes the module is called from a Terraform configuration located at `deploy/incubating/terraform/examples/*/`. Adjust the path based on your actual directory structure:

```
# If calling from deploy/incubating/terraform/modules/aws/eks/
chart_path = "../../../helm/ai-control-plane"

# If calling from root
cart_path = "./deploy/helm/ai-control-plane"

# Absolute path (not recommended for portability)
chart_path = "/opt/infrastructure/ai-control-plane/deploy/helm/ai-control-plane"
```

## References

- [Helm Provider Documentation](https://registry.terraform.io/providers/hashicorp/helm/latest/docs)
- [AI Control Plane Helm Chart](../../helm/ai-control-plane/)
- [Helm Best Practices](https://helm.sh/docs/chart_best_practices/)
