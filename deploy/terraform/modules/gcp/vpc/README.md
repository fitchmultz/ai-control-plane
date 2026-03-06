# GCP VPC Terraform Module

Terraform module for creating a Google Cloud VPC network with custom subnets, Cloud Router, and Cloud NAT for the AI Control Plane.

## Features

- VPC network with custom subnet configuration (auto-creation disabled)
- Subnets with configurable secondary IP ranges for GKE pods and services
- Cloud Router for dynamic routing
- Cloud NAT gateway for private subnet egress (configurable)
- Private Google Access enabled by default
- VPC Firewall rules for internal traffic and IAP SSH access
- Consistent labeling for all resources

## Requirements

| Name | Version |
|------|---------|
| terraform | >= 1.0 |
| google | ~> 5.0 |

## Providers

| Name | Version |
|------|---------|
| google | ~> 5.0 |

## Usage

### Basic Example with GKE Subnet

```hcl
module "vpc" {
  source = "./modules/gcp/vpc"

  project_id = "my-gcp-project"
  region     = "us-central1"

  network_name = "ai-control-plane-vpc"
  router_name  = "ai-control-plane-router"

  subnets = [
    {
      name                     = "gke-subnet"
      ip_cidr_range            = "10.0.0.0/24"
      region                   = "us-central1"
      private_ip_google_access = true
      secondary_ip_ranges = [
        {
          range_name    = "pods"
          ip_cidr_range = "10.4.0.0/14"
        },
        {
          range_name    = "services"
          ip_cidr_range = "10.0.32.0/20"
        }
      ]
    }
  ]

  create_nat_gateway = true

  labels = {
    environment = "production"
    project     = "ai-control-plane"
    team        = "platform"
  }
}
```

### Multiple Subnets

```hcl
module "vpc" {
  source = "./modules/gcp/vpc"

  project_id = "my-gcp-project"
  region     = "us-central1"

  network_name = "ai-control-plane-vpc"

  subnets = [
    {
      name                     = "gke-subnet"
      ip_cidr_range            = "10.0.0.0/24"
      private_ip_google_access = true
      secondary_ip_ranges = [
        {
          range_name    = "pods"
          ip_cidr_range = "10.4.0.0/14"
        },
        {
          range_name    = "services"
          ip_cidr_range = "10.0.32.0/20"
        }
      ]
    },
    {
      name                     = "database-subnet"
      ip_cidr_range            = "10.0.1.0/24"
      private_ip_google_access = true
    },
    {
      name                     = "services-subnet"
      ip_cidr_range            = "10.0.2.0/24"
      private_ip_google_access = true
    }
  ]

  create_nat_gateway = true

  labels = {
    environment = "production"
  }
}
```

### Without NAT Gateway (Private Subnets without Egress)

```hcl
module "vpc" {
  source = "./modules/gcp/vpc"

  project_id = "my-gcp-project"
  region     = "us-central1"

  network_name = "isolated-vpc"

  subnets = [
    {
      name                     = "private-subnet"
      ip_cidr_range            = "10.0.0.0/24"
      private_ip_google_access = true
    }
  ]

  create_nat_gateway = false

  labels = {
    environment = "restricted"
  }
}
```

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| network_name | Name of the VPC network | `string` | `"ai-control-plane-vpc"` | no |
| project_id | GCP project ID where resources will be created | `string` | n/a | yes |
| region | GCP region for regional resources | `string` | `"us-central1"` | no |
| subnets | List of subnet configurations | `list(object)` | See below | no |
| create_nat_gateway | Whether to create a Cloud NAT gateway for private subnet egress | `bool` | `true` | no |
| router_name | Name of the Cloud Router (used for NAT) | `string` | `"ai-control-plane-router"` | no |
| labels | Labels to apply to all resources | `map(string)` | See below | no |

### Default Subnet Configuration

```hcl
[
  {
    name                     = "gke-subnet"
    ip_cidr_range            = "10.0.0.0/24"
    private_ip_google_access = true
    secondary_ip_ranges = [
      {
        range_name    = "pods"
        ip_cidr_range = "10.4.0.0/14"
      },
      {
        range_name    = "services"
        ip_cidr_range = "10.0.32.0/20"
      }
    ]
  }
]
```

### Default Labels

```hcl
{
  environment = "production"
  project     = "ai-control-plane"
}
```

### Subnet Object Schema

| Name | Description | Type | Required |
|------|-------------|------|----------|
| name | Subnet name | `string` | yes |
| ip_cidr_range | Primary IP CIDR range | `string` | yes |
| region | Region (defaults to module region) | `string` | no |
| private_ip_google_access | Enable Private Google Access | `bool` | no (default: true) |
| secondary_ip_ranges | List of secondary IP ranges | `list(object)` | no |

### Secondary IP Range Schema

| Name | Description | Type |
|------|-------------|------|
| range_name | Name of the secondary range (e.g., "pods", "services") | `string` |
| ip_cidr_range | CIDR range for the secondary range | `string` |

## Outputs

| Name | Description |
|------|-------------|
| network_id | The ID of the VPC network |
| network_name | The name of the VPC network |
| network_self_link | The self_link of the VPC network |
| subnet_ids | Map of subnet names to their IDs |
| subnet_self_links | Map of subnet names to their self_links |
| subnet_secondary_ranges | Map of subnet names to their secondary IP ranges |
| router_name | The name of the Cloud Router (if created) |
| router_id | The ID of the Cloud Router (if created) |
| nat_ip | The external IP address of the Cloud NAT gateway (if created) |
| nat_ip_self_link | The self_link of the Cloud NAT IP address (if created) |
| nat_gateway_name | The name of the Cloud NAT gateway (if created) |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     VPC Network                             │
│              ai-control-plane-vpc                           │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                  GKE Subnet                         │   │
│  │                 10.0.0.0/24                         │   │
│  │                                                     │   │
│  │  Secondary Ranges:                                  │   │
│  │  - pods: 10.4.0.0/14                                │   │
│  │  - services: 10.0.32.0/20                           │   │
│  │                                                     │   │
│  │  Private Google Access: Enabled                     │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Cloud Router + NAT                     │   │
│  │                                                     │   │
│  │  - Provides egress for private subnets              │   │
│  │  - External IP for outbound internet access         │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │               Firewall Rules                        │   │
│  │                                                     │   │
│  │  - Allow internal traffic within VPC                │   │
│  │  - Allow SSH from IAP (35.235.240.0/20)             │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Notes

- Auto-creation of subnets is disabled; all subnets must be explicitly defined
- Private Google Access is enabled by default on all subnets
- Cloud NAT uses a single external IP address
- The firewall allows internal traffic between all subnets in the VPC
- IAP firewall rule allows SSH access from Google Identity-Aware Proxy

## Prerequisites

- GCP project with billing enabled
- Terraform configured with GCP credentials (service account or user credentials)
- Required GCP APIs enabled:
  - Compute Engine API
  - Cloud NAT API (if using NAT gateway)

## Security Notes

- Private Google Access allows VMs without external IPs to access Google APIs and services
- Cloud NAT provides outbound internet access without exposing VMs to inbound connections
- Firewall rules restrict SSH access to IAP only (35.235.240.0/20)
- Internal firewall allows all traffic between subnets within the VPC
