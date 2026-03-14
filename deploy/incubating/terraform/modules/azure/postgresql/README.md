# Azure Database for PostgreSQL - Flexible Server Terraform Module

Terraform module for creating an Azure Database for PostgreSQL Flexible Server optimized for the AI Control Plane's LiteLLM gateway.

## Features

- **PostgreSQL 16**: Optimized for LiteLLM with appropriate parameter settings
- **Flexible Server**: Azure's latest PostgreSQL offering with better performance and features
- **Private Endpoint**: Optional private network access via Azure Private Link
- **High Availability**: Zone redundancy support for production workloads
- **Backup**: Configurable backup retention with geo-redundancy options
- **LiteLLM Optimized**: Pre-configured with connection pooling and logging settings

## Usage

### Basic Example

```hcl
module "postgresql" {
  source = "./modules/azure/postgresql"

  server_name         = "ai-control-plane-db"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location

  administrator_password = var.db_password
}
```

### Private Endpoint Example

```hcl
module "postgresql" {
  source = "./modules/azure/postgresql"

  server_name         = "ai-control-plane-db"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location

  administrator_password = var.db_password

  # Private networking
  subnet_id           = azurerm_subnet.database.id
  private_dns_zone_id = azurerm_private_dns_zone.postgresql.id

  # SKU and storage
  sku_name   = "GP_Standard_D4s_v3"
  storage_mb = 65536

  # High availability
  high_availability_enabled = true

  # Backup
  backup_retention_days        = 30
  geo_redundant_backup_enabled = true

  tags = {
    Environment = "production"
    Project     = "ai-control-plane"
  }
}
```

### Public Access with Firewall Rules

```hcl
module "postgresql" {
  source = "./modules/azure/postgresql"

  server_name         = "ai-control-plane-db"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location

  administrator_password = var.db_password

  # Public access with firewall rules
  public_network_access_enabled = true
  allowed_ip_ranges = {
    "office" = "203.0.113.0/24"
    "vpn"    = "198.51.100.0/24"
  }
}
```

## Requirements

| Name | Version |
|------|---------|
| terraform | >= 1.0 |
| azurerm | >= 3.0 |

## Inputs

### Required Variables

| Name | Description | Type |
|------|-------------|------|
| `server_name` | Name of the Azure PostgreSQL Flexible Server | `string` |
| `resource_group_name` | Name of the resource group | `string` |
| `location` | Azure region for deployment | `string` |
| `administrator_password` | Password for the PostgreSQL administrator | `string` (sensitive) |

### Optional Variables

| Name | Description | Type | Default |
|------|-------------|------|---------|
| `postgresql_version` | PostgreSQL version (11-16) | `string` | `"16"` |
| `sku_name` | SKU name (e.g., B_Standard_B2s, GP_Standard_D4s_v3) | `string` | `"B_Standard_B2s"` |
| `storage_mb` | Storage size in MB (min 32768) | `number` | `32768` |
| `administrator_login` | Administrator login name | `string` | `"litellm"` |
| `database_name` | Name of the default database | `string` | `"litellm"` |
| `database_collation` | Database collation | `string` | `"en_US.utf8"` |
| `database_charset` | Database character set | `string` | `"UTF8"` |
| `subnet_id` | Subnet ID for private endpoint | `string` | `null` |
| `private_dns_zone_id` | Private DNS zone ID for private endpoint | `string` | `null` |
| `public_network_access_enabled` | Enable public network access | `bool` | `false` |
| `allowed_ip_ranges` | Map of allowed IP CIDR ranges | `map(string)` | `{}` |
| `backup_retention_days` | Backup retention in days (7-35) | `number` | `7` |
| `geo_redundant_backup_enabled` | Enable geo-redundant backups | `bool` | `false` |
| `high_availability_enabled` | Enable zone redundancy | `bool` | `false` |
| `high_availability_mode` | HA mode (ZoneRedundant or SameZone) | `string` | `"ZoneRedundant"` |
| `high_availability_standby_availability_zone` | Standby availability zone | `string` | `null` |
| `maintenance_window_day_of_week` | Maintenance day (0-6, Sunday=0) | `number` | `0` |
| `maintenance_window_start_hour` | Maintenance start hour (0-23) | `number` | `3` |
| `maintenance_window_start_minute` | Maintenance start minute (0-59) | `number` | `0` |
| `ssl_enforcement_enabled` | Enforce SSL connections | `bool` | `true` |
| `auto_grow_enabled` | Enable storage auto-grow | `bool` | `true` |
| `tags` | Tags to apply to all resources | `map(string)` | `{}` |

## Outputs

| Name | Description |
|------|-------------|
| `server_id` | The ID of the PostgreSQL Flexible Server |
| `server_name` | The name of the PostgreSQL Flexible Server |
| `fqdn` | The fully qualified domain name (FQDN) of the server |
| `private_fqdn` | The private FQDN (when using private endpoint) |
| `database_name` | The name of the PostgreSQL database |
| `database_id` | The ID of the PostgreSQL database |
| `administrator_login` | The administrator login name |
| `database_url` | PostgreSQL connection URL (sensitive) |
| `jdbc_connection_string` | JDBC connection string |
| `private_endpoint_id` | The ID of the private endpoint (if created) |
| `private_endpoint_private_ip` | The private IP of the private endpoint |
| `resource_group_name` | The resource group name |
| `location` | The Azure region |

## SKU Reference

| Tier | SKU Name | Description |
|------|----------|-------------|
| Burstable | B_Standard_B1ms | Dev/test, 1 vCore |
| Burstable | B_Standard_B2s | Dev/test, 2 vCores |
| General Purpose | GP_Standard_D2s_v3 | Production, 2 vCores |
| General Purpose | GP_Standard_D4s_v3 | Production, 4 vCores |
| General Purpose | GP_Standard_D8s_v3 | Production, 8 vCores |
| Memory Optimized | MO_Standard_E2s_v3 | Memory-intensive, 2 vCores |
| Memory Optimized | MO_Standard_E4s_v3 | Memory-intensive, 4 vCores |

**Note:** High availability and geo-redundant backups require General Purpose or higher SKUs.

## Security Considerations

1. **Private Endpoint**: Recommended for production - database is only accessible from within your VNet
2. **Firewall Rules**: If using public access, restrict to specific IP ranges
3. **SSL Enforcement**: Enabled by default with TLS 1.2 minimum
4. **Password**: Marked as sensitive and never logged in plain text

## LiteLLM Configuration

To use this database with LiteLLM, configure the `DATABASE_URL` environment variable:

```bash
# Using the sensitive output
export DATABASE_URL=$(terraform output -raw database_url)
```

Or construct manually:

```bash
export DATABASE_URL="postgresql://litellm:PASSWORD@SERVER_NAME.postgres.database.azure.com:5432/litellm?sslmode=require"
```

**Note:** Azure Database for PostgreSQL requires the username to be in the format `username@servername` for some connection methods, but the connection string output uses the standard format which works with most PostgreSQL clients.

## License

Apache-2.0 - See the main project [LICENSE](../../../../../../LICENSE) and [NOTICE](../../../../../../NOTICE) files for details.
