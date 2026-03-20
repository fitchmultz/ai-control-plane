# -----------------------------------------------------------------------------
# GCP Complete Example - Main Configuration
# -----------------------------------------------------------------------------
# This example demonstrates a complete AI Control Plane deployment on GCP
# using VPC, GKE, Cloud SQL, and Helm release modules.
#
# Architecture:
#   - VPC with private subnet and Cloud NAT
#   - GKE cluster with Workload Identity
#   - Cloud SQL PostgreSQL (private IP)
#   - Kubernetes namespace and secrets
#   - AI Control Plane Helm release with external database
# -----------------------------------------------------------------------------

locals {
  # Construct resource names using prefix and environment
  name = "${var.name_prefix}-${var.environment}"

  # Environment-specific Cloud SQL configuration
  cloudsql_tier_by_environment = {
    dev        = "db-f1-micro"
    staging    = "db-g1-small"
    production = "db-n1-standard-2"
  }

  cloudsql_availability_by_environment = {
    dev        = "ZONAL"
    staging    = "ZONAL"
    production = "REGIONAL"
  }

  cloudsql_tier         = var.cloudsql_tier != null ? var.cloudsql_tier : local.cloudsql_tier_by_environment[var.environment]
  cloudsql_availability = var.cloudsql_availability_type != null ? var.cloudsql_availability_type : local.cloudsql_availability_by_environment[var.environment]

  # Default node pools with environment-specific sizing
  default_node_pools = {
    default = {
      machine_type       = var.environment == "production" ? "e2-standard-2" : "e2-medium"
      initial_node_count = var.environment == "production" ? 2 : 1
      min_count          = var.environment == "production" ? 2 : 1
      max_count          = var.environment == "production" ? 5 : 3
      disk_size_gb       = 100
      spot               = var.environment == "dev"
      labels = {
        workload = "general"
      }
    }
  }

  node_pools = length(var.node_pools) > 0 ? var.node_pools : local.default_node_pools

  # Common labels for all resources
  common_labels = merge(
    {
      environment = var.environment
      project     = "ai-control-plane"
      managed_by  = "terraform"
    },
    var.common_labels
  )

}

resource "terraform_data" "deployment_guardrails" {
  input = {
    ingress_enabled            = var.ingress_enabled
    ingress_host               = var.ingress_host
    ingress_cluster_issuer     = var.ingress_cluster_issuer
    master_authorized_networks = var.master_authorized_networks
  }

  lifecycle {
    precondition {
      condition     = length(var.master_authorized_networks) > 0
      error_message = "master_authorized_networks must contain at least one explicit admin CIDR."
    }

    precondition {
      condition     = !var.ingress_enabled || var.ingress_host != ""
      error_message = "ingress_enabled=true requires ingress_host."
    }

    precondition {
      condition     = !var.ingress_enabled || var.ingress_cluster_issuer != ""
      error_message = "ingress_enabled=true requires ingress_cluster_issuer."
    }
  }
}

resource "random_password" "db_password" {
  length  = 24
  special = true
}

# -----------------------------------------------------------------------------
# VPC Module
# -----------------------------------------------------------------------------

module "vpc" {
  source = "../../modules/gcp/vpc"

  project_id = var.project_id
  region     = var.region

  network_name = "${local.name}-vpc"

  subnets = [
    {
      name                     = "${local.name}-gke-subnet"
      ip_cidr_range            = var.gke_subnet_cidr
      private_ip_google_access = true
      secondary_ip_ranges = [
        {
          range_name    = var.pods_ip_range.name
          ip_cidr_range = var.pods_ip_range.cidr
        },
        {
          range_name    = var.services_ip_range.name
          ip_cidr_range = var.services_ip_range.cidr
        }
      ]
    }
  ]

  create_nat_gateway = true
  router_name        = "${local.name}-router"

  labels = local.common_labels
}

# -----------------------------------------------------------------------------
# GKE Module
# -----------------------------------------------------------------------------

module "gke" {
  source = "../../modules/gcp/gke"

  project_id = var.project_id
  region     = var.region

  cluster_name = "${local.name}-cluster"
  description  = "GKE cluster for AI Control Plane - ${var.environment}"

  kubernetes_version = var.kubernetes_version
  release_channel    = var.release_channel

  network                       = module.vpc.network_self_link
  subnetwork                    = module.vpc.subnet_self_links["${local.name}-gke-subnet"]
  pods_secondary_range_name     = var.pods_ip_range.name
  services_secondary_range_name = var.services_ip_range.name

  enable_private_nodes        = var.enable_private_nodes
  master_ipv4_cidr_block      = var.master_ipv4_cidr_block
  master_authorized_networks  = var.master_authorized_networks
  enable_binary_authorization = true

  enable_workload_identity = var.enable_workload_identity

  node_pools = local.node_pools

  labels = local.common_labels

  depends_on = [terraform_data.deployment_guardrails, module.vpc]
}

# -----------------------------------------------------------------------------
# Cloud SQL Module
# -----------------------------------------------------------------------------

module "cloudsql" {
  source = "../../modules/gcp/cloudsql"

  project_id = var.project_id
  region     = var.region

  instance_name = "${local.name}-db"
  database_name = var.database_name
  user_name     = var.database_user
  user_password = random_password.db_password.result

  tier              = local.cloudsql_tier
  disk_size         = var.cloudsql_disk_size
  disk_autoresize   = var.cloudsql_disk_autoresize
  availability_type = local.cloudsql_availability

  backup_enabled         = var.cloudsql_backup_enabled
  backup_retention_count = var.cloudsql_backup_retention
  enable_insights        = true

  # Use private IP with VPC
  vpc_network = module.vpc.network_name

  deletion_protection = var.environment == "production"

  labels = local.common_labels

  depends_on = [terraform_data.deployment_guardrails, module.vpc]
}

# -----------------------------------------------------------------------------
# Kubernetes Namespace Module
# -----------------------------------------------------------------------------

module "namespace" {
  source = "../../modules/common/kubernetes-namespace"

  name = var.namespace

  labels = merge(
    local.common_labels,
    {
      name = var.namespace
    }
  )

  depends_on = [terraform_data.deployment_guardrails, module.gke]
}

# -----------------------------------------------------------------------------
# Kubernetes Secrets Module
# -----------------------------------------------------------------------------

module "secrets" {
  source = "../../modules/common/secrets"

  namespace   = module.namespace.namespace_name
  secret_name = "${var.helm_release_name}-secrets"

  secret_data = {
    LITELLM_MASTER_KEY = var.litellm_master_key
    LITELLM_SALT_KEY   = var.litellm_salt_key
    DATABASE_URL       = "postgresql://${urlencode(var.database_user)}:${urlencode(random_password.db_password.result)}@localhost/${urlencode(var.database_name)}?host=/cloudsql/${module.cloudsql.connection_name}"
  }

  type = "Opaque"

  labels = local.common_labels

  depends_on = [terraform_data.deployment_guardrails, module.namespace, module.cloudsql]
}

# -----------------------------------------------------------------------------
# Service Account for Workload Identity
# -----------------------------------------------------------------------------

resource "google_service_account" "workload_identity" {
  count = var.enable_workload_identity ? 1 : 0

  account_id   = "${local.name}-workload"
  display_name = "Workload Identity SA for ${local.name}"
  description  = "Service account for AI Control Plane Workload Identity"
  project      = var.project_id
}

# Grant Cloud SQL Client role to the workload identity service account
resource "google_project_iam_member" "cloudsql_client" {
  count = var.enable_workload_identity ? 1 : 0

  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.workload_identity[0].email}"
}

# -----------------------------------------------------------------------------
# Helm Release Module
# -----------------------------------------------------------------------------

module "helm_release" {
  source = "../../modules/common/helm-release"

  release_name = var.helm_release_name
  namespace    = module.namespace.namespace_name
  description  = "AI Control Plane - ${var.environment}"

  # Path to the Helm chart (relative to this example)
  chart_path = "../../../helm/ai-control-plane"

  # Use production profile for production environment
  values = {
    profile = "production"
    demo = {
      enabled = false
    }

    # Use external database (Cloud SQL)
    postgres = {
      enabled = false
    }

    externalDatabase = {
      existingSecret    = module.secrets.secret_name
      existingSecretKey = "DATABASE_URL"
    }

    # Secret configuration
    secrets = {
      create = false
      existingSecret = {
        name           = module.secrets.secret_name
        masterKeyKey   = "LITELLM_MASTER_KEY"
        saltKeyKey     = "LITELLM_SALT_KEY"
        databaseUrlKey = "DATABASE_URL"
      }
    }

    # LiteLLM configuration
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

    # Ingress configuration
    ingress = {
      enabled   = var.ingress_enabled
      className = var.ingress_class_name
      annotations = var.ingress_enabled ? {
        "cert-manager.io/cluster-issuer"                 = var.ingress_cluster_issuer
        "nginx.ingress.kubernetes.io/ssl-redirect"       = "true"
        "nginx.ingress.kubernetes.io/force-ssl-redirect" = "true"
      } : {}
      hosts = var.ingress_enabled ? [{
        host = var.ingress_host
        paths = [{
          path     = "/"
          pathType = "Prefix"
        }]
      }] : []
      tls = var.ingress_enabled ? [{
        secretName = var.ingress_tls_secret_name
        hosts      = [var.ingress_host]
      }] : []
    }

    # Service account with Workload Identity annotation
    serviceAccount = {
      create = true
      annotations = var.enable_workload_identity ? {
        "iam.gke.io/gcp-service-account" = google_service_account.workload_identity[0].email
      } : {}
    }

    # Pod Disruption Budget
    podDisruptionBudget = {
      enabled      = true
      minAvailable = 1
    }

    # Autoscaling
    autoscaling = {
      enabled                        = true
      minReplicas                    = 2
      maxReplicas                    = 5
      targetCPUUtilizationPercentage = 70
    }

    networkPolicy = {
      enabled = true
      ingress = var.ingress_enabled ? [
        {
          from = [
            {
              namespaceSelector = {
                matchLabels = {
                  "kubernetes.io/metadata.name" = "ingress-nginx"
                }
              }
            }
          ]
          ports = [
            {
              protocol = "TCP"
              port     = 4000
            }
          ]
        }
      ] : []
    }

    # Common labels
    commonLabels = local.common_labels
  }

  timeout = 600
  atomic  = true
  wait    = true

  depends_on = [module.secrets, module.gke]
}

# -----------------------------------------------------------------------------
# Workload Identity Binding
# -----------------------------------------------------------------------------

resource "google_service_account_iam_binding" "workload_identity_binding" {
  count = var.enable_workload_identity ? 1 : 0

  service_account_id = google_service_account.workload_identity[0].name
  role               = "roles/iam.workloadIdentityUser"

  members = [
    "serviceAccount:${var.project_id}.svc.id.goog[${var.namespace}/${var.helm_release_name}]"
  ]

  depends_on = [module.helm_release]
}
