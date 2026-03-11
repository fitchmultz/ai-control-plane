# AWS Application Load Balancer (ALB) Module

Terraform module for creating an AWS Application Load Balancer for the AI Control Plane.

## Features

- Application Load Balancer with configurable settings
- Target group for Kubernetes services (port 4000 for LiteLLM by default)
- HTTP listener (always enabled) - redirects to HTTPS when HTTPS is enabled
- HTTPS listener (optional, requires ACM certificate)
- Security group with HTTP/HTTPS inbound rules (auto-created or use existing)
- Configurable health check settings
- Access logs support (optional)

## Usage

### Basic Usage (HTTP only)

```hcl
module "alb" {
  source = "./modules/aws/alb"

  name       = "ai-control-plane-alb"
  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.public_subnet_ids

  enable_https = false

  tags = {
    Environment = "dev"
  }
}
```

### HTTPS with ACM Certificate

```hcl
module "alb" {
  source = "./modules/aws/alb"

  name            = "ai-control-plane-alb"
  vpc_id          = module.vpc.vpc_id
  subnet_ids      = module.vpc.public_subnet_ids
  certificate_arn = aws_acm_certificate.this.arn

  enable_https = true

  tags = {
    Environment = "production"
  }
}
```

### With Existing Security Group

```hcl
module "alb" {
  source = "./modules/aws/alb"

  name            = "ai-control-plane-alb"
  vpc_id          = module.vpc.vpc_id
  subnet_ids      = module.vpc.public_subnet_ids
  security_groups = [aws_security_group.alb.id]
  certificate_arn = aws_acm_certificate.this.arn

  enable_https = true

  tags = {
    Environment = "production"
  }
}
```

### EKS Integration

```hcl
module "alb" {
  source = "./modules/aws/alb"

  name            = "ai-control-plane-alb"
  vpc_id          = module.vpc.vpc_id
  subnet_ids      = module.vpc.public_subnet_ids
  certificate_arn = aws_acm_certificate.this.arn

  # Target type 'ip' is required for EKS/ Kubernetes
  target_type = "ip"

  # Health check for LiteLLM
  health_check_path = "/health"
  target_port       = 4000

  tags = {
    Environment = "production"
  }
}

# EKS service annotation to use the target group
# service.beta.kubernetes.io/aws-load-balancer-type: "external"
# service.beta.kubernetes.io/aws-load-balancer-target-group-arn: module.alb.target_group_arn
```

## Requirements

| Name | Version |
|------|---------|
| terraform | >= 1.0 |
| aws | ~> 5.0 |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| name | Name of the ALB and related resources | `string` | `"ai-control-plane-alb"` | no |
| vpc_id | ID of the VPC where the ALB will be created | `string` | n/a | yes |
| subnet_ids | List of public subnet IDs for the ALB | `list(string)` | n/a | yes |
| security_groups | List of security group IDs to attach to the ALB. If empty, module creates a security group | `list(string)` | `[]` | no |
| internal | Whether the ALB is internal (true) or internet-facing (false) | `bool` | `false` | no |
| enable_deletion_protection | Enable deletion protection for the ALB | `bool` | `false` | no |
| idle_timeout | Idle timeout in seconds for the ALB | `number` | `60` | no |
| tags | Tags to apply to all resources | `map(string)` | `{}` | no |
| enable_https | Enable HTTPS listener | `bool` | `true` | no |
| certificate_arn | ARN of the ACM certificate for HTTPS. Required if enable_https is true | `string` | `null` | no |
| ssl_policy | SSL policy for HTTPS listener | `string` | `"ELBSecurityPolicy-TLS13-1-2-2021-06"` | no |
| target_port | Port for the target group (LiteLLM port) | `number` | `4000` | no |
| target_protocol | Protocol for the target group | `string` | `"HTTP"` | no |
| target_type | Type of target (instance, ip, or alb) | `string` | `"ip"` | no |
| deregistration_delay | Deregistration delay in seconds | `number` | `30` | no |
| health_check_enabled | Enable health checks | `bool` | `true` | no |
| health_check_path | Path for health check requests | `string` | `"/health"` | no |
| health_check_port | Port for health check requests | `string` | `"traffic-port"` | no |
| health_check_protocol | Protocol for health check requests | `string` | `"HTTP"` | no |
| health_check_interval | Interval between health checks in seconds | `number` | `30` | no |
| health_check_timeout | Timeout for health check requests in seconds | `number` | `5` | no |
| health_check_healthy_threshold | Number of consecutive successful health checks required | `number` | `2` | no |
| health_check_unhealthy_threshold | Number of consecutive failed health checks required | `number` | `3` | no |
| health_check_matcher | HTTP codes to accept as healthy | `string` | `"200"` | no |
| access_logs_enabled | Enable access logs for the ALB | `bool` | `false` | no |
| access_logs_bucket | S3 bucket for access logs | `string` | `""` | no |
| access_logs_prefix | Prefix for access log files | `string` | `"alb-logs"` | no |

## Outputs

| Name | Description |
|------|-------------|
| alb_arn | ARN of the Application Load Balancer |
| alb_dns_name | DNS name of the Application Load Balancer |
| alb_zone_id | Zone ID of the Application Load Balancer (for Route 53 alias records) |
| alb_id | ID of the Application Load Balancer |
| target_group_arn | ARN of the target group |
| target_group_name | Name of the target group |
| http_listener_arn | ARN of the HTTP listener |
| https_listener_arn | ARN of the HTTPS listener (null if HTTPS is disabled) |
| security_group_id | ID of the security group created by the module (null if existing security groups used) |

## Notes

- When `enable_https` is `true`, the HTTP listener redirects all traffic to HTTPS (301 redirect)
- When `enable_https` is `false`, the HTTP listener forwards traffic directly to the target group
- The default `target_type` is `ip`, which is required for EKS/ Kubernetes services
- If `security_groups` is empty, the module creates a security group allowing HTTP (80) and HTTPS (443) from anywhere
- LiteLLM runs on port 4000 by default, which is the default target port
