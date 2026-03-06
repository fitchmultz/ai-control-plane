# AWS VPC Terraform Module

Terraform module for creating an AWS VPC with public and private subnets, Internet Gateway, and NAT Gateways.

## Features

- VPC with configurable CIDR block
- Public and private subnets across multiple Availability Zones
- Internet Gateway for public subnet internet access
- NAT Gateway(s) for private subnet egress (configurable: single or per-AZ)
- Route tables with proper associations
- Configurable tags for all resources

## Requirements

| Name | Version |
|------|---------|
| terraform | >= 1.0 |
| aws | ~> 5.0 |

## Providers

| Name | Version |
|------|---------|
| aws | ~> 5.0 |

## Usage

### Basic Example

```hcl
module "vpc" {
  source = "./modules/aws/vpc"

  name_prefix = "my-app"
  vpc_cidr    = "10.0.0.0/16"

  availability_zones   = ["us-east-1a", "us-east-1b", "us-east-1c"]
  private_subnet_cidrs = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  public_subnet_cidrs  = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]

  enable_nat_gateway = true
  single_nat_gateway = false

  tags = {
    Environment = "production"
    Project     = "my-project"
  }
}
```

### Single NAT Gateway (Cost-Optimized)

```hcl
module "vpc" {
  source = "./modules/aws/vpc"

  name_prefix = "dev-env"
  vpc_cidr    = "10.0.0.0/16"

  availability_zones   = ["us-east-1a", "us-east-1b"]
  private_subnet_cidrs = ["10.0.1.0/24", "10.0.2.0/24"]
  public_subnet_cidrs  = ["10.0.101.0/24", "10.0.102.0/24"]

  enable_nat_gateway = true
  single_nat_gateway = true  # Use one NAT gateway for all AZs

  tags = {
    Environment = "development"
  }
}
```

### No NAT Gateway (Private Subnets without Egress)

```hcl
module "vpc" {
  source = "./modules/aws/vpc"

  name_prefix = "isolated"
  vpc_cidr    = "10.0.0.0/16"

  availability_zones   = ["us-east-1a"]
  private_subnet_cidrs = ["10.0.1.0/24"]
  public_subnet_cidrs  = ["10.0.101.0/24"]

  enable_nat_gateway = false

  tags = {}
}
```

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| name_prefix | Prefix to be used for all resource names | `string` | `"ai-control-plane"` | no |
| vpc_cidr | CIDR block for the VPC | `string` | `"10.0.0.0/16"` | no |
| availability_zones | List of availability zones to use for subnets | `list(string)` | n/a | yes |
| private_subnet_cidrs | List of CIDR blocks for private subnets (one per AZ) | `list(string)` | n/a | yes |
| public_subnet_cidrs | List of CIDR blocks for public subnets (one per AZ) | `list(string)` | n/a | yes |
| enable_nat_gateway | Enable NAT gateway for private subnet egress | `bool` | `true` | no |
| single_nat_gateway | Use a single NAT gateway for all private subnets (cheaper but less HA) | `bool` | `false` | no |
| tags | Tags to apply to all resources | `map(string)` | `{}` | no |

## Outputs

| Name | Description |
|------|-------------|
| vpc_id | The ID of the VPC |
| vpc_cidr_block | The CIDR block of the VPC |
| private_subnet_ids | List of IDs of private subnets |
| public_subnet_ids | List of IDs of public subnets |
| nat_gateway_ids | List of IDs of NAT gateways (empty if NAT is disabled) |
| internet_gateway_id | The ID of the Internet Gateway |
| public_route_table_id | The ID of the public route table |
| private_route_table_ids | List of IDs of private route tables |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                           VPC                               │
│                    CIDR: 10.0.0.0/16                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────┐      ┌─────────────────┐              │
│  │  Public Subnet  │      │  Public Subnet  │              │
│  │  10.0.101.0/24  │      │  10.0.102.0/24  │              │
│  │   (AZ: 1a)      │      │   (AZ: 1b)      │              │
│  └────────┬────────┘      └────────┬────────┘              │
│           │                        │                        │
│           └────────────┬───────────┘                        │
│                        │                                    │
│              ┌─────────▼──────────┐                        │
│              │  Internet Gateway  │                        │
│              └────────────────────┘                        │
│                                                             │
│  ┌─────────────────┐      ┌─────────────────┐              │
│  │ Private Subnet  │      │ Private Subnet  │              │
│  │  10.0.1.0/24    │      │  10.0.2.0/24    │              │
│  │   (AZ: 1a)      │      │   (AZ: 1b)      │              │
│  └────────┬────────┘      └────────┬────────┘              │
│           │                        │                        │
│           └────────────┬───────────┘                        │
│                        │                                    │
│              ┌─────────▼──────────┐                        │
│              │   NAT Gateway(s)   │                        │
│              └────────────────────┘                        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Notes

- The number of AZs, private subnet CIDRs, and public subnet CIDRs must match
- Public subnets auto-assign public IPs on launch
- DNS hostnames and support are enabled on the VPC
- When `single_nat_gateway` is `true`, all private subnets route through the NAT gateway in the first AZ
- Elastic IPs are created for each NAT gateway
