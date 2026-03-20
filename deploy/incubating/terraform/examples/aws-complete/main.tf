#------------------------------------------------------------------------------
# AWS Complete Example - Main Configuration
#------------------------------------------------------------------------------
# This file defines the complete AI Control Plane infrastructure on AWS
# including VPC, EKS, RDS, IRSA, and the Helm deployment.
#------------------------------------------------------------------------------

#------------------------------------------------------------------------------
# Data Sources
#------------------------------------------------------------------------------

data "aws_availability_zones" "available" {
  count = var.validation_only ? 0 : 1
  state = "available"
}

#------------------------------------------------------------------------------
# Locals
#------------------------------------------------------------------------------

locals {
  validation_availability_zones = [for suffix in ["a", "b", "c"] : "${var.aws_region}${suffix}"]

  # Use provided AZs or default to first 3 available AZs
  azs = length(var.availability_zones) > 0 ? var.availability_zones : (
    var.validation_only ? local.validation_availability_zones : slice(data.aws_availability_zones.available[0].names, 0, 3)
  )

  validation_account_id = var.validation_only ? var.validation_account_id : data.aws_caller_identity.current[0].account_id

  # Common tags merged with user-provided tags
  common_tags = merge(
    {
      Environment = var.environment
      Project     = "ai-control-plane"
      ManagedBy   = "terraform"
    },
    var.tags
  )

  # Environment-specific sizing
  environment_config = {
    dev = {
      rds_instance_class = "db.t3.micro"
      rds_multi_az       = false
      single_nat_gateway = true
      node_groups = tomap({
        general = {
          desired_size               = 2
          min_size                   = 1
          max_size                   = 3
          instance_types             = tolist(["t3.medium"])
          capacity_type              = "ON_DEMAND"
          ami_type                   = "AL2_x86_64"
          disk_size                  = 50
          max_unavailable_percentage = 25
          labels = tomap({
            role = "general"
          })
          taints                  = []
          launch_template_id      = null
          launch_template_version = null
          remote_access           = null
          tags                    = tomap({})
        }
      })
    }
    staging = {
      rds_instance_class = "db.t3.small"
      rds_multi_az       = true
      single_nat_gateway = false
      node_groups = tomap({
        general = {
          desired_size               = 2
          min_size                   = 2
          max_size                   = 5
          instance_types             = tolist(["t3.medium"])
          capacity_type              = "ON_DEMAND"
          ami_type                   = "AL2_x86_64"
          disk_size                  = 50
          max_unavailable_percentage = 25
          labels = tomap({
            role = "general"
          })
          taints                  = []
          launch_template_id      = null
          launch_template_version = null
          remote_access           = null
          tags                    = tomap({})
        }
      })
    }
    production = {
      rds_instance_class = "db.t3.medium"
      rds_multi_az       = true
      single_nat_gateway = false
      node_groups = tomap({
        general = {
          desired_size               = 3
          min_size                   = 2
          max_size                   = 10
          instance_types             = tolist(["t3.large"])
          capacity_type              = "ON_DEMAND"
          ami_type                   = "AL2_x86_64"
          disk_size                  = 100
          max_unavailable_percentage = 25
          labels = tomap({
            role = "general"
          })
          taints                  = []
          launch_template_id      = null
          launch_template_version = null
          remote_access           = null
          tags                    = tomap({})
        }
      })
    }
  }

  # Determine node groups (user-provided or environment default)
  effective_node_groups = merge(local.environment_config[var.environment].node_groups, var.node_groups)

  # Terraform examples now target the production-safe Helm contract only.
  helm_profile = "production"
}

#------------------------------------------------------------------------------
# Random Password for RDS
#------------------------------------------------------------------------------

resource "random_password" "rds" {
  length           = 24
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

resource "terraform_data" "deployment_guardrails" {
  input = {
    cluster_endpoint_public_access = var.cluster_endpoint_public_access
    cluster_public_access_cidrs    = var.cluster_public_access_cidrs
    public_ingress_enabled         = var.public_ingress_enabled
    alb_certificate_arn            = var.alb_certificate_arn
    enable_ingress                 = var.enable_ingress
    ingress_host                   = var.ingress_host
    rds_deletion_protection        = var.rds_deletion_protection
    rds_skip_final_snapshot        = var.rds_skip_final_snapshot
  }

  lifecycle {
    precondition {
      condition     = !var.cluster_endpoint_public_access || length(var.cluster_public_access_cidrs) > 0
      error_message = "Public EKS API access requires an explicit allowlist in cluster_public_access_cidrs."
    }

    precondition {
      condition     = !var.enable_ingress || var.alb_certificate_arn != ""
      error_message = "enable_ingress=true requires alb_certificate_arn so external access remains TLS-only."
    }

    precondition {
      condition     = !var.enable_ingress || var.ingress_host != ""
      error_message = "enable_ingress=true requires ingress_host."
    }

    precondition {
      condition     = var.public_ingress_enabled ? var.enable_ingress : true
      error_message = "public_ingress_enabled=true requires enable_ingress=true."
    }

    precondition {
      condition     = var.rds_deletion_protection
      error_message = "rds_deletion_protection must remain enabled."
    }

    precondition {
      condition     = !var.rds_skip_final_snapshot
      error_message = "rds_skip_final_snapshot must remain false."
    }
  }
}

#------------------------------------------------------------------------------
# VPC Module
#------------------------------------------------------------------------------

module "vpc" {
  source = "../../modules/aws/vpc"

  name_prefix = var.name_prefix
  vpc_cidr    = var.vpc_cidr

  availability_zones   = local.azs
  private_subnet_cidrs = slice(var.private_subnet_cidrs, 0, length(local.azs))
  public_subnet_cidrs  = slice(var.public_subnet_cidrs, 0, length(local.azs))

  enable_nat_gateway = true
  single_nat_gateway = var.single_nat_gateway || local.environment_config[var.environment].single_nat_gateway

  tags = local.common_tags
}

#------------------------------------------------------------------------------
# EKS Module
#------------------------------------------------------------------------------

module "eks" {
  source = "../../modules/aws/eks"

  cluster_name    = "${var.name_prefix}-${var.environment}"
  cluster_version = var.cluster_version

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnet_ids

  cluster_endpoint_public_access  = var.cluster_endpoint_public_access
  cluster_endpoint_private_access = var.cluster_endpoint_private_access
  cluster_public_access_cidrs     = var.cluster_public_access_cidrs

  node_groups               = local.effective_node_groups
  node_group_subnet_ids     = module.vpc.private_subnet_ids
  enable_cluster_autoscaler = var.enable_cluster_autoscaler
  enable_irsa               = true

  tags = local.common_tags
}

#------------------------------------------------------------------------------
# RDS Module
#------------------------------------------------------------------------------

module "rds" {
  source = "../../modules/aws/rds"

  identifier = "${var.name_prefix}-${var.environment}"
  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnet_ids

  # Engine settings
  engine_version = var.rds_engine_version

  # Instance settings (environment-specific or user-provided)
  instance_class = var.rds_instance_class != "db.t3.micro" ? var.rds_instance_class : local.environment_config[var.environment].rds_instance_class
  multi_az       = var.rds_multi_az || local.environment_config[var.environment].rds_multi_az

  # Storage settings
  allocated_storage     = var.rds_allocated_storage
  max_allocated_storage = var.rds_max_allocated_storage
  storage_encrypted     = true

  # Database settings
  db_name  = var.rds_database_name
  username = var.rds_username
  password = random_password.rds.result

  # Security settings
  allowed_security_groups = [module.eks.node_security_group_id]

  # Backup settings
  backup_retention_period = var.rds_backup_retention_period
  backup_window           = "03:00-04:00"
  maintenance_window      = "Mon:04:00-Mon:05:00"

  # Deletion protection
  deletion_protection = var.rds_deletion_protection
  skip_final_snapshot = var.rds_skip_final_snapshot

  # Monitoring
  performance_insights_enabled = var.rds_performance_insights_enabled

  tags = local.common_tags
}

#------------------------------------------------------------------------------
# Kubernetes Namespace Module
#------------------------------------------------------------------------------

module "namespace" {
  source = "../../modules/common/kubernetes-namespace"

  name = var.namespace
  labels = {
    "app.kubernetes.io/name"       = "ai-control-plane"
    "app.kubernetes.io/component"  = "namespace"
    "app.kubernetes.io/managed-by" = "terraform"
    environment                    = var.environment
  }

  depends_on = [module.eks]
}

#------------------------------------------------------------------------------
# IRSA (IAM Roles for Service Accounts) Module
#------------------------------------------------------------------------------

module "irsa" {
  source = "../../modules/aws/irsa"

  oidc_provider_arn = module.eks.oidc_provider_arn
  oidc_provider_url = replace(module.eks.oidc_provider_url, "https://", "")

  namespace            = var.namespace
  service_account_name = "${var.helm_release_name}-sa"
  role_name            = "${var.name_prefix}-${var.environment}-irsa"

  policy_statements = var.irsa_policy_statements

  tags = local.common_tags

  depends_on = [module.eks, module.namespace]
}

#------------------------------------------------------------------------------
# Kubernetes Secrets Module
#------------------------------------------------------------------------------

module "secrets" {
  source = "../../modules/common/secrets"

  namespace   = module.namespace.namespace_name
  secret_name = "${var.helm_release_name}-secrets"

  secret_data = {
    LITELLM_MASTER_KEY = var.litellm_master_key
    LITELLM_SALT_KEY   = var.litellm_salt_key
    DATABASE_URL       = "postgresql://${var.rds_username}:${random_password.rds.result}@${module.rds.db_instance_address}:${module.rds.db_instance_port}/${var.rds_database_name}?sslmode=require"
  }

  labels = {
    "app.kubernetes.io/name"       = "ai-control-plane"
    "app.kubernetes.io/component"  = "secrets"
    "app.kubernetes.io/managed-by" = "terraform"
  }

  depends_on = [module.namespace, module.rds]
}

#------------------------------------------------------------------------------
# Backup Replication to S3
#------------------------------------------------------------------------------

# S3 bucket for cross-region backup replication
data "aws_caller_identity" "current" {
  count = var.validation_only ? 0 : 1
}

resource "aws_s3_bucket" "backups" {
  count = var.backup_replication_enabled ? 1 : 0

  bucket = "${var.name_prefix}-${var.environment}-backups-${local.validation_account_id}"

  tags = merge(local.common_tags, {
    Name = "${var.name_prefix}-${var.environment}-backups"
  })
}

resource "aws_s3_bucket_versioning" "backups" {
  count = var.backup_replication_enabled ? 1 : 0

  bucket = aws_s3_bucket.backups[0].id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_public_access_block" "backups" {
  count = var.backup_replication_enabled ? 1 : 0

  bucket = aws_s3_bucket.backups[0].id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "backups" {
  count = var.backup_replication_enabled ? 1 : 0

  bucket = aws_s3_bucket.backups[0].id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "backups" {
  count = var.backup_replication_enabled ? 1 : 0

  bucket = aws_s3_bucket.backups[0].id

  rule {
    id     = "backup-retention"
    status = "Enabled"

    filter {}

    transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }

    transition {
      days          = 90
      storage_class = "GLACIER"
    }

    expiration {
      days = var.backup_retention_days
    }

    noncurrent_version_expiration {
      noncurrent_days = 30
    }
  }
}

# IAM policy for backup replication
resource "aws_iam_role_policy" "backup_replication" {
  count = var.backup_replication_enabled ? 1 : 0

  name = "${var.name_prefix}-${var.environment}-backup-replication"
  role = module.irsa.iam_role_name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.backups[0].arn,
          "${aws_s3_bucket.backups[0].arn}/*"
        ]
      }
    ]
  })
}

#------------------------------------------------------------------------------
# Helm Release Module
#------------------------------------------------------------------------------

module "helm_release" {
  source = "../../modules/common/helm-release"

  release_name = var.helm_release_name
  namespace    = module.namespace.namespace_name
  chart_path   = var.helm_chart_path

  # Values configuration
  values = {
    # Profile based on environment
    profile = local.helm_profile
    demo = {
      enabled = false
    }

    # LiteLLM configuration
    litellm = {
      replicaCount = var.litellm_replica_count
      resources    = var.litellm_resources
      service = {
        type = "ClusterIP"
        port = 4000
      }
    }

    # Secrets configuration (use existing secret created by Terraform)
    secrets = {
      create = false
      existingSecret = {
        name           = module.secrets.secret_name
        masterKeyKey   = "LITELLM_MASTER_KEY"
        saltKeyKey     = "LITELLM_SALT_KEY"
        databaseUrlKey = "DATABASE_URL"
      }
    }

    # Disable embedded PostgreSQL (using external RDS)
    postgres = {
      enabled = false
    }

    # Service account with IRSA annotation
    serviceAccount = {
      create = true
      name   = "${var.helm_release_name}-sa"
      annotations = length(var.irsa_policy_statements) > 0 || var.backup_replication_enabled ? {
        "eks.amazonaws.com/role-arn" = module.irsa.iam_role_arn
      } : {}
    }

    # Autoscaling configuration
    autoscaling = {
      enabled                        = var.enable_autoscaling
      minReplicas                    = 2
      maxReplicas                    = 10
      targetCPUUtilizationPercentage = 80
    }

    # Ingress configuration (optional)
    ingress = {
      enabled   = var.enable_ingress
      className = var.ingress_class_name
      annotations = var.enable_ingress ? {
        "alb.ingress.kubernetes.io/scheme"          = var.public_ingress_enabled ? "internet-facing" : "internal"
        "alb.ingress.kubernetes.io/target-type"     = "ip"
        "alb.ingress.kubernetes.io/listen-ports"    = "[{\"HTTP\":80},{\"HTTPS\":443}]"
        "alb.ingress.kubernetes.io/ssl-redirect"    = "443"
        "alb.ingress.kubernetes.io/certificate-arn" = var.alb_certificate_arn
      } : {}
      hosts = var.ingress_host != "" ? [{
        host = var.ingress_host
        paths = [{
          path     = "/"
          pathType = "Prefix"
        }]
      }] : []
      tls = []
    }

    # Pod disruption budget for high availability
    podDisruptionBudget = {
      enabled      = true
      minAvailable = 1
    }

    networkPolicy = {
      enabled = true
      ingress = var.enable_ingress ? [
        {
          from = [
            {
              ipBlock = {
                cidr = module.vpc.vpc_cidr_block
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

    # Monitoring
    monitoring = {
      serviceMonitor = {
        enabled = var.environment == "production"
      }
    }
  }

  timeout = 600
  atomic  = true
  wait    = true

  depends_on = [terraform_data.deployment_guardrails, module.eks, module.rds, module.irsa, module.secrets]
}
