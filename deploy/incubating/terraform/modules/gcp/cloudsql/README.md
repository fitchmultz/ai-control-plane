# GCP Cloud SQL PostgreSQL Module

Terraform module for creating a Cloud SQL PostgreSQL instance for the AI Control Plane.

## Features

- PostgreSQL 16 (configurable)
- Private IP with VPC peering (optional)
- Public IP with authorized networks (when private IP not used)
- Automated backups with configurable retention
- Maintenance windows
- Query Insights (optional)
- Deletion protection

## Usage

### Basic Public IP Instance

```hcl
module "cloudsql" {
  source = "./modules/gcp/cloudsql"

  instance_name = "litellm-db"
  project_id    = "my-project"
  region        = "us-central1"
  user_password = var.db_password
}
```

### Private IP Instance

```hcl
module "cloudsql" {
  source = "./modules/gcp/cloudsql"

  instance_name = "litellm-db"
  project_id    = "my-project"
  region        = "us-central1"
  user_password = var.db_password
  vpc_network   = "my-vpc"

  tier              = "db-n1-standard-2"
  availability_type = "REGIONAL"
  backup_enabled    = true
}
```

### Production Configuration

```hcl
module "cloudsql" {
  source = "./modules/gcp/cloudsql"

  instance_name = "litellm-prod-db"
  project_id    = "my-project"
  region        = "us-central1"
  user_password = var.db_password

  database_version = "POSTGRES_16"
  tier             = "db-n1-standard-2"
  disk_size        = 100
  disk_autoresize  = true

  availability_type = "REGIONAL"

  backup_enabled       = true
  backup_start_time    = "03:00"
  backup_retention_count = 30

  maintenance_day   = 7
  maintenance_hour  = 4

  deletion_protection = true

  labels = {
    environment = "production"
    application = "litellm"
  }
}
```

## Requirements

| Name | Version |
|------|---------|
| terraform | >= 1.0 |
| google | >= 4.0 |
| google-beta | >= 4.0 |

## Providers

| Name | Version |
|------|---------|
| google | >= 4.0 |
| google-beta | >= 4.0 |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| instance_name | Name of the Cloud SQL instance | `string` | n/a | yes |
| project_id | GCP project ID | `string` | n/a | yes |
| region | GCP region for the Cloud SQL instance | `string` | n/a | yes |
| database_version | PostgreSQL version | `string` | `"POSTGRES_16"` | no |
| tier | Machine type tier | `string` | `"db-f1-micro"` | no |
| disk_size | Initial disk size in GB | `number` | `20` | no |
| disk_autoresize | Enable automatic disk resizing | `bool` | `true` | no |
| availability_type | ZONAL or REGIONAL | `string` | `"ZONAL"` | no |
| backup_enabled | Enable automated backups | `bool` | `true` | no |
| backup_start_time | Backup start time (HH:MM UTC) | `string` | `"03:00"` | no |
| backup_retention_count | Number of backups to retain | `number` | `7` | no |
| maintenance_day | Day of week for maintenance (1-7) | `number` | `7` | no |
| maintenance_hour | Hour for maintenance (0-23) | `number` | `4` | no |
| maintenance_track | Update track: stable or canary | `string` | `"stable"` | no |
| vpc_network | VPC network for private IP | `string` | `null` | no |
| authorized_networks | Authorized networks for public IP | `list(object)` | `[]` | no |
| database_name | Name of the database | `string` | `"litellm"` | no |
| user_name | Name of the database user | `string` | `"litellm"` | no |
| user_password | Password for the database user | `string` | n/a | yes |
| enable_insights | Enable Query Insights | `bool` | `false` | no |
| insights_query_length | Max query length for insights | `number` | `1024` | no |
| deletion_protection | Enable deletion protection | `bool` | `true` | no |
| labels | Labels for the instance | `map(string)` | `{}` | no |

## Outputs

| Name | Description |
|------|-------------|
| instance_id | The ID of the Cloud SQL instance |
| instance_name | The name of the Cloud SQL instance |
| connection_name | Connection name (for Cloud SQL Proxy) |
| private_ip_address | Private IP address (null if not enabled) |
| public_ip_address | Public IP address (null if not enabled) |
| database_name | The name of the created database |
| database_user | The name of the database user |
| database_url | PostgreSQL connection URL (sensitive) |
| database_url_proxy | Connection URL for Cloud SQL Proxy (sensitive) |

## Notes

- When `vpc_network` is specified, the module creates a private IP instance with VPC peering
- When `vpc_network` is null, the instance uses public IP with optional authorized networks
- Deletion protection prevents accidental destruction - set to `false` when destroying
- The `database_url` output includes the connection URL with the appropriate IP (private or public)
- Use `database_url_proxy` when connecting via Cloud SQL Proxy
