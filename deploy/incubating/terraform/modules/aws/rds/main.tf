# AWS RDS PostgreSQL Module for AI Control Plane
# Creates a managed PostgreSQL database for LiteLLM

locals {
  default_tags = {
    ManagedBy = "terraform"
    Module    = "rds-postgresql"
  }
  
  all_tags = merge(local.default_tags, var.tags)
}

#-------------------------------------------------------------------------------
# DB Subnet Group
#-------------------------------------------------------------------------------
resource "aws_db_subnet_group" "this" {
  name        = "${var.identifier}-subnet-group"
  description = "Subnet group for ${var.identifier} RDS instance"
  subnet_ids  = var.subnet_ids

  tags = merge(
    local.all_tags,
    {
      Name = "${var.identifier}-subnet-group"
    }
  )
}

#-------------------------------------------------------------------------------
# Security Group
#-------------------------------------------------------------------------------
resource "aws_security_group" "this" {
  name        = "${var.identifier}-sg"
  description = "Security group for ${var.identifier} RDS instance"
  vpc_id      = var.vpc_id

  tags = merge(
    local.all_tags,
    {
      Name = "${var.identifier}-sg"
    }
  )

  lifecycle {
    create_before_destroy = true
  }
}

# Ingress rule for allowed security groups
resource "aws_security_group_rule" "ingress_from_security_groups" {
  count = length(var.allowed_security_groups) > 0 ? length(var.allowed_security_groups) : 0

  type                     = "ingress"
  from_port                = 5432
  to_port                  = 5432
  protocol                 = "tcp"
  source_security_group_id = var.allowed_security_groups[count.index]
  security_group_id        = aws_security_group.this.id
  description              = "Allow PostgreSQL access from ${var.allowed_security_groups[count.index]}"
}

# Ingress rule for allowed CIDR blocks
resource "aws_security_group_rule" "ingress_from_cidr" {
  count = length(var.allowed_cidr_blocks) > 0 ? length(var.allowed_cidr_blocks) : 0

  type              = "ingress"
  from_port         = 5432
  to_port           = 5432
  protocol          = "tcp"
  cidr_blocks       = [var.allowed_cidr_blocks[count.index]]
  security_group_id = aws_security_group.this.id
  description       = "Allow PostgreSQL access from ${var.allowed_cidr_blocks[count.index]}"
}

#-------------------------------------------------------------------------------
# RDS Parameter Group
#-------------------------------------------------------------------------------
resource "aws_db_parameter_group" "this" {
  name        = "${var.identifier}-params"
  family      = "postgres16"
  description = "Parameter group for ${var.identifier}"

  # LiteLLM-specific parameter settings
  parameter {
    name  = "log_connections"
    value = "1"
  }

  parameter {
    name  = "log_disconnections"
    value = "1"
  }

  parameter {
    name  = "log_checkpoints"
    value = "1"
  }

  tags = local.all_tags

  lifecycle {
    create_before_destroy = true
  }
}

#-------------------------------------------------------------------------------
# RDS Instance
#-------------------------------------------------------------------------------
resource "aws_db_instance" "this" {
  identifier = var.identifier

  # Engine settings
  engine         = "postgres"
  engine_version = var.engine_version
  instance_class = var.instance_class

  # Storage settings
  allocated_storage     = var.allocated_storage
  max_allocated_storage = var.max_allocated_storage
  storage_encrypted     = var.storage_encrypted
  storage_type          = "gp3"

  # Database settings
  db_name  = var.db_name
  username = var.username
  password = var.password

  # Network settings
  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = [aws_security_group.this.id]
  publicly_accessible    = false
  port                   = 5432

  # High availability
  multi_az = var.multi_az

  # Backup settings
  backup_retention_period = var.backup_retention_period
  backup_window           = var.backup_window
  maintenance_window      = var.maintenance_window

  # Deletion protection
  deletion_protection = var.deletion_protection
  skip_final_snapshot = var.skip_final_snapshot

  # Final snapshot
  final_snapshot_identifier = var.skip_final_snapshot ? null : "${var.identifier}-final-snapshot"

  # Parameter and option groups
  parameter_group_name = aws_db_parameter_group.this.name

  # Monitoring and logging
  enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]
  performance_insights_enabled    = var.performance_insights_enabled
  performance_insights_retention_period = var.performance_insights_enabled ? var.performance_insights_retention_period : null

  # Auto minor version upgrade
  auto_minor_version_upgrade = var.auto_minor_version_upgrade

  # Copy tags to snapshots
  copy_tags_to_snapshot = true

  # Tags
  tags = merge(
    local.all_tags,
    {
      Name = var.identifier
    }
  )
}
