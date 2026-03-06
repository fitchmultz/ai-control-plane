#------------------------------------------------------------------------------
# AWS ALB Module - Variables
#------------------------------------------------------------------------------

#------------------------------------------------------------------------------
# General
#------------------------------------------------------------------------------

variable "name" {
  description = "Name of the ALB and related resources"
  type        = string
  default     = "ai-control-plane-alb"
}

variable "vpc_id" {
  description = "ID of the VPC where the ALB will be created"
  type        = string
}

variable "subnet_ids" {
  description = "List of public subnet IDs for the ALB"
  type        = list(string)
}

variable "security_groups" {
  description = "List of security group IDs to attach to the ALB. If empty, module creates a security group"
  type        = list(string)
  default     = []
}

variable "internal" {
  description = "Whether the ALB is internal (true) or internet-facing (false)"
  type        = bool
  default     = false
}

variable "enable_deletion_protection" {
  description = "Enable deletion protection for the ALB"
  type        = bool
  default     = false
}

variable "idle_timeout" {
  description = "Idle timeout in seconds for the ALB"
  type        = number
  default     = 60
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}

#------------------------------------------------------------------------------
# HTTPS
#------------------------------------------------------------------------------

variable "enable_https" {
  description = "Enable HTTPS listener"
  type        = bool
  default     = true
}

variable "certificate_arn" {
  description = "ARN of the ACM certificate for HTTPS. Required if enable_https is true"
  type        = string
  default     = null
}

variable "ssl_policy" {
  description = "SSL policy for HTTPS listener"
  type        = string
  default     = "ELBSecurityPolicy-TLS13-1-2-2021-06"
}

#------------------------------------------------------------------------------
# Target Group
#------------------------------------------------------------------------------

variable "target_port" {
  description = "Port for the target group (LiteLLM port)"
  type        = number
  default     = 4000
}

variable "target_protocol" {
  description = "Protocol for the target group"
  type        = string
  default     = "HTTP"
}

variable "target_type" {
  description = "Type of target (instance, ip, or alb)"
  type        = string
  default     = "ip"
}

variable "deregistration_delay" {
  description = "Deregistration delay in seconds"
  type        = number
  default     = 30
}

#------------------------------------------------------------------------------
# Health Check
#------------------------------------------------------------------------------

variable "health_check_enabled" {
  description = "Enable health checks"
  type        = bool
  default     = true
}

variable "health_check_path" {
  description = "Path for health check requests"
  type        = string
  default     = "/health"
}

variable "health_check_port" {
  description = "Port for health check requests"
  type        = string
  default     = "traffic-port"
}

variable "health_check_protocol" {
  description = "Protocol for health check requests"
  type        = string
  default     = "HTTP"
}

variable "health_check_interval" {
  description = "Interval between health checks in seconds"
  type        = number
  default     = 30
}

variable "health_check_timeout" {
  description = "Timeout for health check requests in seconds"
  type        = number
  default     = 5
}

variable "health_check_healthy_threshold" {
  description = "Number of consecutive successful health checks required"
  type        = number
  default     = 2
}

variable "health_check_unhealthy_threshold" {
  description = "Number of consecutive failed health checks required"
  type        = number
  default     = 3
}

variable "health_check_matcher" {
  description = "HTTP codes to accept as healthy"
  type        = string
  default     = "200"
}

#------------------------------------------------------------------------------
# Access Logs
#------------------------------------------------------------------------------

variable "access_logs_enabled" {
  description = "Enable access logs for the ALB"
  type        = bool
  default     = false
}

variable "access_logs_bucket" {
  description = "S3 bucket for access logs"
  type        = string
  default     = ""
}

variable "access_logs_prefix" {
  description = "Prefix for access log files"
  type        = string
  default     = "alb-logs"
}
