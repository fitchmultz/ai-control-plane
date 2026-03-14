# AWS RDS PostgreSQL Terraform Module

Terraform module for creating an AWS RDS PostgreSQL instance optimized for the AI Control Plane's LiteLLM gateway.

## Features

- **PostgreSQL 16**: Optimized for LiteLLM with appropriate parameter settings
- **High Availability**: Multi-AZ deployment support for production workloads
- **Security**: Encrypted storage, private subnets, security group-based access control
- **Storage Autoscaling**: Automatic storage scaling from initial to max allocation
- **Backup**: Configurable backup retention with automated snapshots
- **Monitoring**: CloudWatch Logs export for PostgreSQL and upgrade logs

## Usage

### Basic Example

```hcl
module "rds" {
  source = "./modules/aws/rds"

  identifier = "ai-control-plane-db"
  
  vpc_id     = aws_vpc.main.id
  subnet_ids = aws_subnet.private[*].id
  
  password = var.db_password
}
```

### Production Example

```hcl
module "rds" {
  source = "./modules/aws/rds"

  identifier = "ai-control-plane-db-prod"
  
  # Network
  vpc_id     = aws_vpc.main.id
  subnet_ids = aws_subnet.private[*].id
  
  # Credentials
  username = "litellm"
  password = var.db_password
  db_name  = "litellm"
  
  # Instance
  instance_class = "db.t3.medium"
  multi_az       = true
  
  # Storage
  allocated_storage     = 50
  max_allocated_storage = 500
  storage_encrypted     = true
  
  # Backup
  backup_retention_period = 30
  backup_window           = "03:00-04:00"
  
  # Security
  allowed_security_groups = [aws_security_group.litellm.id]
  deletion_protection     = true
  skip_final_snapshot     = false
  
  # Monitoring
  performance_insights_enabled = true
  
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
| aws | >= 5.0 |

## Inputs

### Required Variables

| Name | Description | Type |
|------|-------------|------|
| `vpc_id` | VPC ID where the RDS instance will be created | `string` |
| `subnet_ids` | List of private subnet IDs for the DB subnet group | `list(string)` |
| `password` | Master database password | `string` (sensitive) |

### Optional Variables

| Name | Description | Type | Default |
|------|-------------|------|---------|
| `identifier` | Unique identifier for the RDS instance | `string` | `"ai-control-plane-db"` |
| `engine_version` | PostgreSQL engine version | `string` | `"16.3"` |
| `instance_class` | RDS instance class | `string` | `"db.t3.micro"` |
| `allocated_storage` | Initial storage size in GB | `number` | `20` |
| `max_allocated_storage` | Maximum storage size in GB for autoscaling | `number` | `100` |
| `db_name` | Name of the default database to create | `string` | `"litellm"` |
| `username` | Master database username | `string` | `"litellm"` |
| `multi_az` | Enable Multi-AZ deployment for high availability | `bool` | `true` |
| `backup_retention_period` | Number of days to retain backups | `number` | `7` |
| `backup_window` | Preferred backup window (UTC) | `string` | `"03:00-04:00"` |
| `maintenance_window` | Preferred maintenance window (UTC) | `string` | `"Mon:04:00-Mon:05:00"` |
| `deletion_protection` | Enable deletion protection | `bool` | `true` |
| `storage_encrypted` | Enable storage encryption using AWS KMS | `bool` | `true` |
| `skip_final_snapshot` | Skip final snapshot when destroying | `bool` | `false` |
| `allowed_security_groups` | List of security group IDs allowed to connect | `list(string)` | `[]` |
| `auto_minor_version_upgrade` | Enable automatic minor version upgrades | `bool` | `true` |
| `performance_insights_enabled` | Enable Performance Insights | `bool` | `false` |
| `performance_insights_retention_period` | Performance Insights retention in days | `number` | `7` |
| `tags` | Tags to apply to all resources | `map(string)` | `{}` |

## Outputs

| Name | Description |
|------|-------------|
| `db_instance_address` | The hostname of the RDS instance |
| `db_instance_endpoint` | The connection endpoint (hostname:port) |
| `db_instance_port` | The port (5432) |
| `db_instance_name` | The database name |
| `db_instance_username` | The master username |
| `db_instance_arn` | The ARN of the RDS instance |
| `db_instance_id` | The RDS instance identifier |
| `database_url` | PostgreSQL connection URL (sensitive) |
| `security_group_id` | The security group ID |
| `security_group_arn` | The security group ARN |
| `db_subnet_group_id` | The DB subnet group ID |
| `db_subnet_group_arn` | The DB subnet group ARN |
| `db_instance_resource_id` | The RDS Resource ID |

## Security Considerations

1. **Private Subnets**: The database is always created in private subnets (no public access)
2. **Security Groups**: Access is controlled via security group references (not CIDR blocks)
3. **Encryption**: Storage encryption is enabled by default
4. **Deletion Protection**: Enabled by default to prevent accidental deletion
5. **Password**: Marked as sensitive and never logged in plain text

## LiteLLM Configuration

To use this database with LiteLLM, configure the `DATABASE_URL` environment variable:

```bash
# Using the sensitive output
export DATABASE_URL=$(terraform output -raw database_url)
```

Or construct manually:

```bash
export DATABASE_URL="postgresql://litellm:PASSWORD@ADDRESS:5432/litellm"
```

## License

Apache-2.0 - See the main project [LICENSE](../../../../../../LICENSE) and [NOTICE](../../../../../../NOTICE) files for details.
