# GCP Global HTTP(S) Load Balancer Module

Terraform module for creating a GCP Global HTTP(S) Load Balancer for the AI Control Plane.

## Features

- Global HTTP(S) Load Balancer with external IP
- Backend service for GKE services (supports container-native NEG)
- Health check configured for LiteLLM (port 4000, path `/health`)
- SSL certificate support (Google-managed or self-managed)
- IP address reservation
- Optional HTTP to HTTPS redirect
- Cloud Armor security policy support
- Identity-Aware Proxy (IAP) support
- Multi-domain and path-based routing support

## Usage

### Basic Usage (HTTP only)

```hcl
module "loadbalancer" {
  source = "./modules/gcp/loadbalancer"

  name       = "ai-control-plane-lb"
  project_id = "my-project-id"
  region     = "us-central1"

  enable_https = false

  tags = {
    Environment = "dev"
  }
}
```

### HTTPS with Google-Managed Certificate

```hcl
module "loadbalancer" {
  source = "./modules/gcp/loadbalancer"

  name       = "ai-control-plane-lb"
  project_id = "my-project-id"
  region     = "us-central1"

  enable_https                    = true
  enable_http                     = false
  enable_https_redirect          = true
  managed_ssl_certificate_domains = ["api.example.com", "ai.example.com"]

  tags = {
    Environment = "production"
  }
}
```

### HTTPS with Self-Managed Certificate

```hcl
resource "google_compute_ssl_certificate" "this" {
  name        = "ai-control-plane-cert"
  private_key = file("path/to/private.key")
  certificate = file("path/to/certificate.crt")
}

module "loadbalancer" {
  source = "./modules/gcp/loadbalancer"

  name            = "ai-control-plane-lb"
  project_id      = "my-project-id"
  region          = "us-central1"

  enable_https    = true
  ssl_certificate = google_compute_ssl_certificate.this.id

  tags = {
    Environment = "production"
  }
}
```

### Container-Native Load Balancer (GKE with NEG)

```hcl
module "loadbalancer" {
  source = "./modules/gcp/loadbalancer"

  name       = "ai-control-plane-lb"
  project_id = "my-project-id"
  region     = "us-central1"

  # Enable container-native NEG
  create_neg = true
  neg_name   = "litellm-neg"
  network    = "gke-network"
  subnetwork = "gke-subnet"

  enable_https                    = true
  enable_https_redirect          = true
  managed_ssl_certificate_domains = ["api.example.com"]

  # Health check for LiteLLM
  health_check_path = "/health"
  health_check_port = 4000

  tags = {
    Environment = "production"
  }
}

# Add endpoints to the NEG (typically done via GKE service annotations)
resource "google_compute_network_endpoint" "this" {
  network_endpoint_group = module.loadbalancer.neg_id

  instance   = "gke-node-name"
  port       = 4000
  ip_address = "10.0.0.1"
}
```

### GKE Integration with BackendConfig

```yaml
# GKE Service with NEG annotation
apiVersion: v1
kind: Service
metadata:
  name: litellm
  annotations:
    cloud.google.com/neg: '{"ingress": true}'
    beta.cloud.google.com/backend-config: '{"default": "litellm-backendconfig"}'
spec:
  type: ClusterIP
  selector:
    app: litellm
  ports:
    - port: 4000
      targetPort: 4000
---
apiVersion: cloud.google.com/v1
kind: BackendConfig
metadata:
  name: litellm-backendconfig
spec:
  healthCheck:
    checkIntervalSec: 10
    port: 4000
    type: HTTP
    requestPath: /health
  logging:
    enable: true
    sampleRate: 1.0
```

### With Cloud Armor Security Policy

```hcl
resource "google_compute_security_policy" "this" {
  name = "ai-control-plane-policy"

  rule {
    action   = "allow"
    priority = "1000"
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
    description = "Default allow"
  }
}

module "loadbalancer" {
  source = "./modules/gcp/loadbalancer"

  name       = "ai-control-plane-lb"
  project_id = "my-project-id"
  region     = "us-central1"

  enable_https                    = true
  managed_ssl_certificate_domains = ["api.example.com"]
  security_policy                 = google_compute_security_policy.this.name

  tags = {
    Environment = "production"
  }
}
```

## Requirements

| Name | Version |
|------|---------|
| terraform | >= 1.0 |
| google | >= 4.0 |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| name | Name of the load balancer and related resources | `string` | `"ai-control-plane-lb"` | no |
| project_id | GCP project ID where the load balancer will be created | `string` | n/a | yes |
| region | GCP region for regional resources (NEG, etc.) | `string` | `"us-central1"` | no |
| tags | Labels to apply to all resources | `map(string)` | `{}` | no |
| network | VPC network for the NEG (if create_neg is true) | `string` | `"default"` | no |
| subnetwork | Subnetwork for the NEG (if create_neg is true) | `string` | `null` | no |
| backend_service_name | Name of the backend service | `string` | `"ai-control-plane-backend"` | no |
| backend_timeout_sec | Timeout for backend service in seconds | `number` | `60` | no |
| connection_draining_timeout_sec | Connection draining timeout in seconds | `number` | `300` | no |
| backend_logging_enabled | Enable logging for the backend service | `bool` | `false` | no |
| backend_logging_sample_rate | Sample rate for backend logging (0.0 to 1.0) | `number` | `1.0` | no |
| enable_cdn | Enable CDN for the backend service | `bool` | `false` | no |
| instance_group | Instance group to use as backend (if not using NEG) | `string` | `null` | no |
| create_neg | Create a zonal network endpoint group (container-native LB) | `bool` | `false` | no |
| create_serverless_neg | Create a serverless NEG for Cloud Run or GKE | `bool` | `false` | no |
| neg_name | Name of the NEG (optional, defaults to {name}-neg) | `string` | `null` | no |
| neg_zone | Zone for the NEG (defaults to {region}-a) | `string` | `null` | no |
| cloud_run_service | Cloud Run service name (for serverless NEG) | `string` | `null` | no |
| health_check_path | Path for health check requests | `string` | `"/health"` | no |
| health_check_port | Port for health check requests | `number` | `4000` | no |
| health_check_interval | Interval between health checks in seconds | `number` | `10` | no |
| health_check_timeout | Timeout for health check requests in seconds | `number` | `5` | no |
| health_check_healthy_threshold | Number of consecutive successful health checks required | `number` | `2` | no |
| health_check_unhealthy_threshold | Number of consecutive failed health checks required | `number` | `3` | no |
| health_check_logging_enabled | Enable logging for health checks | `bool` | `false` | no |
| enable_https | Enable HTTPS listener | `bool` | `true` | no |
| enable_http | Enable HTTP listener (disabled when enable_https_redirect is true) | `bool` | `true` | no |
| enable_https_redirect | Redirect HTTP to HTTPS | `bool` | `false` | no |
| ssl_certificate | Self-managed SSL certificate resource ID | `string` | `null` | no |
| managed_ssl_certificate_domains | List of domains for Google-managed SSL certificate | `list(string)` | `[]` | no |
| ssl_policy | SSL policy resource ID to apply to the HTTPS proxy | `string` | `null` | no |
| create_ssl_policy | Create a new SSL policy | `bool` | `false` | no |
| ssl_policy_profile | SSL policy profile (COMPATIBLE, MODERN, RESTRICTED, CUSTOM) | `string` | `"MODERN"` | no |
| ssl_policy_min_tls_version | Minimum TLS version for SSL policy | `string` | `"TLS_1_2"` | no |
| ssl_policy_custom_features | Custom SSL features (required when profile is CUSTOM) | `list(string)` | `[]` | no |
| quic_override | QUIC protocol override (DISABLE, ENABLE, or NONE) | `string` | `"NONE"` | no |
| host_rules | List of host rules for the URL map | `list(object)` | `[]` | no |
| path_matchers | List of path matchers for the URL map | `list(object)` | `[]` | no |
| security_policy | Cloud Armor security policy ID to attach to the backend service | `string` | `null` | no |
| iap_enabled | Enable Identity-Aware Proxy for the backend service | `bool` | `false` | no |
| iap_oauth2_client_id | OAuth2 client ID for IAP | `string` | `null` | no |
| iap_oauth2_client_secret | OAuth2 client secret for IAP | `string` | `null` | no |

## Outputs

| Name | Description |
|------|-------------|
| load_balancer_ip | Global IP address of the load balancer |
| load_balancer_ip_name | Name of the global IP address resource |
| backend_service_id | ID of the backend service |
| backend_service_name | Name of the backend service |
| url_map_id | ID of the URL map |
| url_map_name | Name of the URL map |
| target_proxy_id | ID of the target proxy (HTTP or HTTPS) |
| target_http_proxy_id | ID of the HTTP target proxy (null if HTTPS only) |
| target_https_proxy_id | ID of the HTTPS target proxy (null if HTTP only) |
| forwarding_rule_id | ID of the primary forwarding rule |
| http_forwarding_rule_id | ID of the HTTP forwarding rule (null if HTTPS only or redirect enabled) |
| https_forwarding_rule_id | ID of the HTTPS forwarding rule (null if HTTP only) |
| health_check_id | ID of the health check |
| health_check_name | Name of the health check |
| managed_ssl_certificate_id | ID of the managed SSL certificate (null if using self-managed or HTTP only) |
| managed_ssl_certificate_status | Status of the managed SSL certificate |
| neg_id | ID of the network endpoint group (null if not created) |
| serverless_neg_id | ID of the serverless NEG (null if not created) |
| ssl_policy_id | ID of the SSL policy (null if not created) |

## Notes

- When `enable_https` is `true` and `enable_https_redirect` is `true`, the HTTP forwarding rule redirects all traffic to HTTPS (301 redirect)
- Use either `ssl_certificate` (self-managed) OR `managed_ssl_certificate_domains` (Google-managed), not both
- The default health check is configured for LiteLLM on port 4000 with path `/health`
- Container-native load balancing requires the `create_neg` variable set to `true` and proper GKE service annotations
- Google-managed certificates may take 15-30 minutes to provision
- The module supports both zonal NEGs (GCE_VM_IP_PORT) and serverless NEGs (Cloud Run)
