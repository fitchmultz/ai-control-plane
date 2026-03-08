# Azure Complete Example - Main Configuration
# Complete Azure infrastructure with AKS, PostgreSQL, and AI Control Plane deployment

locals {
  # Combine default tags with user-provided tags
  common_tags = merge(
    {
      Environment = var.environment
      ManagedBy   = "Terraform"
      Project     = "AI Control Plane"
    },
    var.tags
  )

  # Environment-specific PostgreSQL SKU mapping
  postgresql_sku_map = {
    dev        = "B_Standard_B2s"
    staging    = "GP_Standard_D2s_v3"
    production = "GP_Standard_D4s_v3"
  }

  # Determine PostgreSQL SKU based on environment
  effective_postgresql_sku = var.postgresql_sku_name != "B_Standard_B2s" ? var.postgresql_sku_name : lookup(local.postgresql_sku_map, var.environment, "B_Standard_B2s")

  # Node pool configuration based on environment
  effective_node_pools = var.environment == "production" ? merge(
    var.node_pools,
    {
      "general" = merge(
        var.node_pools["general"],
        {
          vm_size    = "Standard_D4s_v3"
          min_count  = 2
          max_count  = 10
          node_count = 3
        }
      )
    }
  ) : var.node_pools

  # System node pool adjustments for production
  effective_system_node_pool = var.environment == "production" ? merge(
    var.system_node_pool,
    {
      vm_size    = "Standard_D2s_v3"
      node_count = 2
      min_count  = 2
      max_count  = 5
    }
  ) : var.system_node_pool

  # PostgreSQL HA enabled for production
  postgresql_ha_enabled = var.environment == "production" ? true : var.postgresql_high_availability_enabled

  # Geo-redundant backup for production
  postgresql_geo_backup = var.environment == "production" ? true : var.postgresql_geo_redundant_backup_enabled
}

resource "terraform_data" "deployment_guardrails" {
  input = {
    ingress_enabled            = var.ingress_enabled
    ingress_host               = var.ingress_host
    log_analytics_workspace_id = var.log_analytics_workspace_id
  }

  lifecycle {
    precondition {
      condition     = !var.ingress_enabled || var.ingress_host != ""
      error_message = "ingress_enabled=true requires ingress_host."
    }

    precondition {
      condition     = var.log_analytics_workspace_id != ""
      error_message = "log_analytics_workspace_id is required for the hardened Azure baseline."
    }
  }
}

#------------------------------------------------------------------------------
# Resource Group
#------------------------------------------------------------------------------

resource "azurerm_resource_group" "main" {
  name     = var.resource_group_name
  location = var.location
  tags     = local.common_tags
}

#------------------------------------------------------------------------------
# Random Password Generation
#------------------------------------------------------------------------------

# Generate random password for PostgreSQL if not provided
resource "random_password" "postgresql" {
  length           = 32
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

#------------------------------------------------------------------------------
# Network Module
#------------------------------------------------------------------------------

module "network" {
  source = "../../modules/azure/network"

  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  name_prefix         = var.name_prefix
  vnet_cidr           = var.vnet_cidr
  subnet_cidrs        = var.subnet_cidrs
  tags                = local.common_tags
}

#------------------------------------------------------------------------------
# AKS Module
#------------------------------------------------------------------------------

module "aks" {
  source = "../../modules/azure/aks"

  # Basic configuration
  cluster_name        = var.cluster_name
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  kubernetes_version  = var.kubernetes_version
  sku_tier            = var.sku_tier

  # Network configuration
  subnet_id          = module.network.subnet_ids["aks"]
  availability_zones = var.availability_zones

  # Node pools
  system_node_pool = local.effective_system_node_pool
  node_pools       = local.effective_node_pools

  # Identity configuration
  enable_workload_identity = var.enable_workload_identity
  enable_oidc_issuer       = var.enable_oidc_issuer

  # Network settings
  network_plugin                    = "azure"
  network_policy                    = "calico"
  service_cidr                      = "10.1.0.0/16"
  dns_service_ip                    = "10.1.0.10"
  enable_microsoft_defender         = true
  enable_oms_agent                  = true
  enable_azure_policy               = true
  enable_key_vault_secrets_provider = true
  log_analytics_workspace_id        = var.log_analytics_workspace_id

  tags = local.common_tags

  depends_on = [terraform_data.deployment_guardrails, module.network]
}

#------------------------------------------------------------------------------
# PostgreSQL Module
#------------------------------------------------------------------------------

module "postgresql" {
  source = "../../modules/azure/postgresql"

  # Server configuration
  server_name         = "${var.name_prefix}-${var.postgresql_server_name}"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  postgresql_version  = var.postgresql_version

  # SKU and storage
  sku_name   = local.effective_postgresql_sku
  storage_mb = var.postgresql_storage_mb

  # Administrator credentials
  administrator_login    = var.postgresql_admin_username
  administrator_password = random_password.postgresql.result

  # Database configuration
  database_name = var.postgresql_database_name

  # Network configuration - use private endpoint via subnet
  subnet_id                     = module.network.subnet_ids["database"]
  public_network_access_enabled = false

  # Backup configuration
  backup_retention_days        = var.postgresql_backup_retention_days
  geo_redundant_backup_enabled = local.postgresql_geo_backup

  # High availability (only for Standard and Premium SKUs)
  high_availability_enabled = local.postgresql_ha_enabled

  tags = local.common_tags

  depends_on = [module.network, random_password.postgresql]
}

#------------------------------------------------------------------------------
# Kubernetes and Helm Provider Configuration
#------------------------------------------------------------------------------

# Data source for current Azure client config (provides tenant_id)
data "azurerm_client_config" "current" {}

# Data source for AKS cluster credentials
data "azurerm_kubernetes_cluster" "main" {
  name                = module.aks.cluster_name
  resource_group_name = azurerm_resource_group.main.name

  depends_on = [module.aks]
}

# Configure Kubernetes provider
provider "kubernetes" {
  host                   = module.aks.host
  cluster_ca_certificate = base64decode(module.aks.cluster_ca_certificate)

  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    command     = "kubelogin"
    args = [
      "get-token",
      "--login", "azurecli",
      "--server-id", "6dae42f8-4368-4678-94ff-3960e28e3637", # Azure AD server ID for AKS
      "--client-id", "80faf920-1908-4b52-b5ef-a8e7bedfc67a", # Azure AD client ID for AKS
      "--tenant-id", data.azurerm_client_config.current.tenant_id,
      "--environment", "AzurePublicCloud"
    ]
  }
}

# Alternative: Use client certificate authentication (simpler for initial setup)
provider "kubernetes" {
  alias = "cert_auth"

  host                   = module.aks.host
  client_certificate     = base64decode(module.aks.client_certificate)
  client_key             = base64decode(module.aks.client_key)
  cluster_ca_certificate = base64decode(module.aks.cluster_ca_certificate)
}

# Configure Helm provider
provider "helm" {
  kubernetes = {
    host                   = module.aks.host
    cluster_ca_certificate = base64decode(module.aks.cluster_ca_certificate)

    exec = {
      api_version = "client.authentication.k8s.io/v1beta1"
      command     = "kubelogin"
      args = [
        "get-token",
        "--login", "azurecli",
        "--server-id", "6dae42f8-4368-4678-94ff-3960e28e3637",
        "--client-id", "80faf920-1908-4b52-b5ef-a8e7bedfc67a"
      ]
    }
  }
}

# Alternative Helm provider with cert auth
provider "helm" {
  alias = "cert_auth"

  kubernetes = {
    host                   = module.aks.host
    client_certificate     = base64decode(module.aks.client_certificate)
    client_key             = base64decode(module.aks.client_key)
    cluster_ca_certificate = base64decode(module.aks.cluster_ca_certificate)
  }
}

#------------------------------------------------------------------------------
# Kubernetes Namespace
#------------------------------------------------------------------------------

module "namespace" {
  source = "../../modules/common/kubernetes-namespace"

  name = var.helm_namespace

  labels = {
    "app.kubernetes.io/managed-by" = "terraform"
    "environment"                  = var.environment
  }

  annotations = {
    "description" = "AI Control Plane namespace managed by Terraform"
  }

  # Use cert_auth provider for simplicity
  providers = {
    kubernetes = kubernetes.cert_auth
  }

  depends_on = [module.aks]
}

#------------------------------------------------------------------------------
# Kubernetes Secrets
#------------------------------------------------------------------------------

module "secrets" {
  source = "../../modules/common/secrets"

  namespace   = module.namespace.namespace_name
  secret_name = "ai-control-plane-secrets"

  secret_data = {
    LITELLM_MASTER_KEY = var.litellm_master_key
    LITELLM_SALT_KEY   = var.litellm_salt_key
    DATABASE_URL       = "postgresql://${var.postgresql_admin_username}:${random_password.postgresql.result}@${module.postgresql.fqdn}:5432/${var.postgresql_database_name}?sslmode=require"
  }

  labels = {
    "app.kubernetes.io/part-of" = "ai-control-plane"
    "environment"               = var.environment
  }

  # Use cert_auth provider for simplicity
  providers = {
    kubernetes = kubernetes.cert_auth
  }

  depends_on = [module.namespace, module.postgresql]
}

#------------------------------------------------------------------------------
# Workload Identity Service Account (Optional)
#------------------------------------------------------------------------------

# Create service account with Azure Workload Identity annotations
resource "kubernetes_service_account_v1" "workload_identity" {
  provider = kubernetes.cert_auth

  count = var.enable_workload_identity ? 1 : 0

  metadata {
    name      = "ai-control-plane-sa"
    namespace = module.namespace.namespace_name
    annotations = {
      "azure.workload.identity/client-id" = module.aks.control_plane_identity.client_id
    }
    labels = {
      "azure.workload.identity/use" = "true"
    }
  }

  depends_on = [module.namespace]
}

#------------------------------------------------------------------------------
# Helm Release
#------------------------------------------------------------------------------

module "helm_release" {
  source = "../../modules/common/helm-release"

  release_name = var.helm_release_name
  namespace    = module.namespace.namespace_name
  chart_path   = var.helm_chart_path
  description  = "AI Control Plane - LiteLLM gateway on AKS"

  # Do not create namespace via Helm (we created it with the namespace module)
  create_namespace = false

  # Values configuration based on environment
  values = merge(
    # Base configuration
    {
      profile = "production"
      demo = {
        enabled = false
      }

      secrets = {
        create = false
        existingSecret = {
          name           = module.secrets.secret_name
          masterKeyKey   = "LITELLM_MASTER_KEY"
          saltKeyKey     = "LITELLM_SALT_KEY"
          databaseUrlKey = "DATABASE_URL"
        }
      }

      postgres = {
        enabled = false
      }

      externalDatabase = {
        existingSecret    = module.secrets.secret_name
        existingSecretKey = "DATABASE_URL"
      }

      litellm = {
        replicaCount = max(var.litellm_replica_count, 2)
        mode         = "online"
      }

      serviceAccount = {
        create = true
        name   = var.enable_workload_identity ? "ai-control-plane-sa" : ""
        annotations = var.enable_workload_identity ? {
          "azure.workload.identity/client-id" = module.aks.control_plane_identity.client_id
        } : {}
      }
    },

    # Production-safe defaults across the primary Azure path
    {
      litellm = {
        replicaCount = max(var.litellm_replica_count, 2)
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

      podDisruptionBudget = {
        enabled      = true
        minAvailable = 1
      }

      autoscaling = {
        enabled                           = var.enable_autoscaling
        minReplicas                       = 2
        maxReplicas                       = 10
        targetCPUUtilizationPercentage    = 70
        targetMemoryUtilizationPercentage = 80
      }

      monitoring = {
        serviceMonitor = {
          enabled = true
          labels = {
            release = "prometheus"
          }
        }
      }

      networkPolicy = {
        enabled = true
      }
    },

    # Ingress configuration (if enabled)
    var.ingress_enabled ? {
      ingress = {
        enabled   = true
        className = var.ingress_class_name
        annotations = {
          "cert-manager.io/cluster-issuer"                 = var.ingress_cluster_issuer
          "nginx.ingress.kubernetes.io/ssl-redirect"       = "true"
          "nginx.ingress.kubernetes.io/force-ssl-redirect" = "true"
        }
        hosts = [
          {
            host = var.ingress_host
            paths = [
              {
                path     = "/"
                pathType = "Prefix"
              }
            ]
          }
        ]
        tls = [
          {
            secretName = var.ingress_tls_secret_name
            hosts      = [var.ingress_host]
          }
        ]
      }
    } : {}
  )

  # Deployment safety
  atomic          = true
  wait            = true
  timeout         = 600
  cleanup_on_fail = true

  # Use cert_auth provider for simplicity
  providers = {
    helm = helm.cert_auth
  }

  depends_on = [terraform_data.deployment_guardrails, module.secrets, kubernetes_service_account_v1.workload_identity]
}
