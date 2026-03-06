# Azure Application Gateway Module

Terraform module for creating an Azure Application Gateway v2 for the AI Control Plane.

## Features

- Azure Application Gateway v2 (Standard_v2 or WAF_v2)
- Public IP address with Standard SKU
- Backend pool for AKS services
- HTTP listener (always enabled)
- HTTPS listener (optional, requires SSL certificate)
- HTTP to HTTPS redirect (when HTTPS is enabled)
- Health probe configuration for backend services
- Autoscaling support (optional)
- WAF configuration (WAF_v2 SKU only)
- SSL policy with recommended security settings
- Diagnostic settings support (optional)

## Usage

### Basic Usage (HTTP only)

```hcl
module "appgateway" {
  source = "./modules/azure/appgateway"

  name                = "ai-control-plane-appgw"
  resource_group_name = azurerm_resource_group.this.name
  location            = azurerm_resource_group.this.location
  subnet_id           = azurerm_subnet.appgateway.id

  backend_ip_addresses = ["10.0.1.10", "10.0.1.11"]

  tags = {
    Environment = "dev"
  }
}
```

### HTTPS with SSL Certificate

```hcl
module "appgateway" {
  source = "./modules/azure/appgateway"

  name                = "ai-control-plane-appgw"
  resource_group_name = azurerm_resource_group.this.name
  location            = azurerm_resource_group.this.location
  subnet_id           = azurerm_subnet.appgateway.id

  enable_https             = true
  ssl_certificate_path     = "./certs/certificate.pfx"
  ssl_certificate_password = var.ssl_password
  ssl_certificate_name     = "ai-control-plane-cert"

  backend_ip_addresses = ["10.0.1.10"]

  tags = {
    Environment = "production"
  }
}
```

### AKS Integration

```hcl
module "appgateway" {
  source = "./modules/azure/appgateway"

  name                = "ai-control-plane-appgw"
  resource_group_name = azurerm_resource_group.this.name
  location            = azurerm_resource_group.this.location
  subnet_id           = azurerm_subnet.appgateway.id

  # Health check for LiteLLM
  health_probe_path = "/health"
  backend_port      = 4000

  # Use AKS internal load balancer IPs or FQDNs
  backend_ip_addresses = ["10.0.1.100"]

  tags = {
    Environment = "production"
  }
}

# Note: Use the Application Gateway Ingress Controller (AGIC) or
# Service annotations to integrate with AKS
```

### WAF_v2 with Autoscaling

```hcl
module "appgateway" {
  source = "./modules/azure/appgateway"

  name                = "ai-control-plane-appgw"
  resource_group_name = azurerm_resource_group.this.name
  location            = azurerm_resource_group.this.location
  subnet_id           = azurerm_subnet.appgateway.id

  sku_tier = "WAF_v2"

  enable_autoscale     = true
  autoscale_min_capacity = 2
  autoscale_max_capacity = 10

  waf_enabled          = true
  waf_mode             = "Prevention"
  waf_rule_set_version = "3.2"

  backend_ip_addresses = ["10.0.1.10"]

  tags = {
    Environment = "production"
  }
}
```

### With Diagnostic Settings

```hcl
module "appgateway" {
  source = "./modules/azure/appgateway"

  name                = "ai-control-plane-appgw"
  resource_group_name = azurerm_resource_group.this.name
  location            = azurerm_resource_group.this.location
  subnet_id           = azurerm_subnet.appgateway.id

  enable_diagnostics         = true
  log_analytics_workspace_id = azurerm_log_analytics_workspace.this.id

  backend_ip_addresses = ["10.0.1.10"]

  tags = {
    Environment = "production"
  }
}
```

## Requirements

| Name | Version |
|------|---------|
| terraform | >= 1.0 |
| azurerm | >= 3.0 |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| name | Name of the Application Gateway and related resources | `string` | n/a | yes |
| resource_group_name | Name of the resource group where the Application Gateway will be created | `string` | n/a | yes |
| location | Azure region where the Application Gateway will be created | `string` | n/a | yes |
| subnet_id | ID of the subnet where the Application Gateway will be deployed | `string` | n/a | yes |
| tags | Tags to apply to all resources | `map(string)` | `{}` | no |
| sku_tier | SKU tier for the Application Gateway (Standard_v2 or WAF_v2) | `string` | `"Standard_v2"` | no |
| sku_capacity | Number of instances for the Application Gateway (autoscale not configured) | `number` | `2` | no |
| enable_autoscale | Enable autoscaling for the Application Gateway | `bool` | `false` | no |
| autoscale_min_capacity | Minimum number of instances for autoscaling | `number` | `2` | no |
| autoscale_max_capacity | Maximum number of instances for autoscaling | `number` | `10` | no |
| frontend_port | Frontend port for HTTP traffic | `number` | `80` | no |
| frontend_https_port | Frontend port for HTTPS traffic | `number` | `443` | no |
| enable_https | Enable HTTPS listener | `bool` | `false` | no |
| ssl_certificate_path | Path to the SSL certificate file (PFX format). Required if enable_https is true | `string` | `null` | no |
| ssl_certificate_password | Password for the SSL certificate file | `string` | `null` | no |
| ssl_certificate_name | Name for the SSL certificate in Application Gateway | `string` | `"ssl-cert"` | no |
| backend_port | Backend port for the backend HTTP settings (LiteLLM port) | `number` | `4000` | no |
| backend_protocol | Protocol for backend communication | `string` | `"Http"` | no |
| backend_ip_addresses | List of backend IP addresses for the backend pool | `list(string)` | `[]` | no |
| backend_fqdns | List of backend FQDNs for the backend pool | `list(string)` | `[]` | no |
| health_probe_enabled | Enable health probe | `bool` | `true` | no |
| health_probe_path | Path for health probe requests | `string` | `"/health"` | no |
| health_probe_protocol | Protocol for health probe requests | `string` | `"Http"` | no |
| health_probe_interval | Interval between health probes in seconds | `number` | `30` | no |
| health_probe_timeout | Timeout for health probe requests in seconds | `number` | `30` | no |
| health_probe_unhealthy_threshold | Number of consecutive failed health probes before marking unhealthy | `number` | `3` | no |
| health_probe_match_status_codes | HTTP status codes to accept as healthy | `list(string)` | `["200-399"]` | no |
| waf_enabled | Enable WAF (only applicable when sku_tier is WAF_v2) | `bool` | `false` | no |
| waf_mode | WAF mode (Detection or Prevention) | `string` | `"Detection"` | no |
| waf_rule_set_type | Type of WAF rule set | `string` | `"OWASP"` | no |
| waf_rule_set_version | Version of WAF rule set | `string` | `"3.2"` | no |
| enable_diagnostics | Enable diagnostic settings | `bool` | `false` | no |
| log_analytics_workspace_id | Log Analytics Workspace ID for diagnostics | `string` | `null` | no |

## Outputs

| Name | Description |
|------|-------------|
| gateway_id | ID of the Application Gateway |
| gateway_name | Name of the Application Gateway |
| gateway_resource_group_name | Resource group name of the Application Gateway |
| gateway_location | Location of the Application Gateway |
| public_ip_address | Public IP address of the Application Gateway |
| public_ip_id | ID of the public IP address |
| public_ip_fqdn | FQDN of the public IP address |
| frontend_ip_configuration | Frontend IP configuration details |
| backend_address_pool_id | ID of the backend address pool |
| backend_address_pool_name | Name of the backend address pool |
| http_listener_id | ID of the HTTP listener |
| http_listener_name | Name of the HTTP listener |
| https_listener_id | ID of the HTTPS listener (null if HTTPS is disabled) |
| https_listener_name | Name of the HTTPS listener (null if HTTPS is disabled) |
| backend_http_settings_id | ID of the backend HTTP settings |
| backend_http_settings_name | Name of the backend HTTP settings |
| health_probe_id | ID of the health probe (null if health probe is disabled) |
| health_probe_name | Name of the health probe (null if health probe is disabled) |
| sku_tier | SKU tier of the Application Gateway |
| sku_capacity | SKU capacity of the Application Gateway |
| waf_enabled | Whether WAF is enabled |

## Notes

- When `enable_https` is `true`, the HTTP listener redirects all traffic to HTTPS (301 redirect)
- When `enable_https` is `false`, the HTTP listener forwards traffic directly to the backend
- The default `backend_port` is `4000`, which is the default port for LiteLLM
- The module ignores changes to `backend_address_pool` as it may be managed by Kubernetes (AGIC)
- The module ignores changes to `ssl_certificate` as it may be managed externally
- For WAF configuration, use `sku_tier = "WAF_v2"` and set `waf_enabled = true`
- Autoscaling is optional; when disabled, `sku_capacity` defines the fixed instance count
- The public IP is created with Standard SKU and zone-redundant configuration
- Health probe uses the backend HTTP settings hostname by default

## Important Considerations

### SSL Certificate Management

For production use, consider using Azure Key Vault for SSL certificate management instead of storing certificates in the Terraform configuration. You can integrate with Key Vault using the `azurerm_key_vault_certificate` data source.

### AKS Integration

When integrating with Azure Kubernetes Service (AKS), you have two options:

1. **Application Gateway Ingress Controller (AGIC)**: Automatically manages backend pool membership based on Kubernetes services
2. **Manual configuration**: Use `backend_ip_addresses` or `backend_fqdns` to point to the AKS internal load balancer

### Subnet Requirements

The Application Gateway requires a dedicated subnet with at least `/24` prefix (256 addresses) for Standard_v2 or WAF_v2 SKUs. The subnet cannot contain any other resources.

### NSG Requirements

When using WAF_v2, ensure your Network Security Groups (NSGs) allow traffic from the GatewayManager service tag and AzureLoadBalancer service tag.
