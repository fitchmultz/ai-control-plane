# AWS Application Load Balancer Module for AI Control Plane
# Creates an ALB with target group, listeners, and security group

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

locals {
  default_tags = {
    ManagedBy = "terraform"
    Module    = "alb"
  }

  all_tags = merge(local.default_tags, var.tags)
}

#------------------------------------------------------------------------------
# Security Group
#------------------------------------------------------------------------------

resource "aws_security_group" "this" {
  count = length(var.security_groups) == 0 ? 1 : 0

  name        = "${var.name}-sg"
  description = "Security group for ${var.name} ALB"
  vpc_id      = var.vpc_id

  tags = merge(
    local.all_tags,
    {
      Name = "${var.name}-sg"
    }
  )

  lifecycle {
    create_before_destroy = true
  }
}

# Ingress rule for HTTP
resource "aws_security_group_rule" "http_ingress" {
  count = length(var.security_groups) == 0 ? 1 : 0

  type              = "ingress"
  from_port         = 80
  to_port           = 80
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.this[0].id
  description       = "Allow HTTP inbound traffic"
}

# Ingress rule for HTTPS
resource "aws_security_group_rule" "https_ingress" {
  count = length(var.security_groups) == 0 && var.enable_https ? 1 : 0

  type              = "ingress"
  from_port         = 443
  to_port           = 443
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.this[0].id
  description       = "Allow HTTPS inbound traffic"
}

# Egress rule - allow all outbound
resource "aws_security_group_rule" "egress" {
  count = length(var.security_groups) == 0 ? 1 : 0

  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.this[0].id
  description       = "Allow all outbound traffic"
}

#------------------------------------------------------------------------------
# Application Load Balancer
#------------------------------------------------------------------------------

resource "aws_lb" "this" {
  name                       = var.name
  internal                   = var.internal
  load_balancer_type         = "application"
  security_groups            = length(var.security_groups) > 0 ? var.security_groups : [aws_security_group.this[0].id]
  subnets                    = var.subnet_ids
  enable_deletion_protection = var.enable_deletion_protection
  idle_timeout               = var.idle_timeout

  dynamic "access_logs" {
    for_each = var.access_logs_enabled && var.access_logs_bucket != "" ? [1] : []
    content {
      bucket  = var.access_logs_bucket
      prefix  = var.access_logs_prefix
      enabled = var.access_logs_enabled
    }
  }

  tags = merge(
    local.all_tags,
    {
      Name = var.name
    }
  )
}

#------------------------------------------------------------------------------
# Target Group
#------------------------------------------------------------------------------

resource "aws_lb_target_group" "this" {
  name                 = "${var.name}-tg"
  port                 = var.target_port
  protocol             = var.target_protocol
  vpc_id               = var.vpc_id
  target_type          = var.target_type
  deregistration_delay = var.deregistration_delay

  health_check {
    enabled             = var.health_check_enabled
    healthy_threshold   = var.health_check_healthy_threshold
    interval            = var.health_check_interval
    matcher             = var.health_check_matcher
    path                = var.health_check_path
    port                = var.health_check_port
    protocol            = var.health_check_protocol
    timeout             = var.health_check_timeout
    unhealthy_threshold = var.health_check_unhealthy_threshold
  }

  tags = merge(
    local.all_tags,
    {
      Name = "${var.name}-tg"
    }
  )

  lifecycle {
    create_before_destroy = true
  }
}

#------------------------------------------------------------------------------
# HTTP Listener
#------------------------------------------------------------------------------

resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.this.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type = var.enable_https ? "redirect" : "forward"

    dynamic "redirect" {
      for_each = var.enable_https ? [1] : []
      content {
        port        = "443"
        protocol    = "HTTPS"
        status_code = "HTTP_301"
      }
    }

    target_group_arn = var.enable_https ? null : aws_lb_target_group.this.arn
  }

  tags = merge(
    local.all_tags,
    {
      Name = "${var.name}-http"
    }
  )
}

#------------------------------------------------------------------------------
# HTTPS Listener (optional)
#------------------------------------------------------------------------------

resource "aws_lb_listener" "https" {
  count = var.enable_https ? 1 : 0

  load_balancer_arn = aws_lb.this.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = var.ssl_policy
  certificate_arn   = var.certificate_arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.this.arn
  }

  tags = merge(
    local.all_tags,
    {
      Name = "${var.name}-https"
    }
  )
}
