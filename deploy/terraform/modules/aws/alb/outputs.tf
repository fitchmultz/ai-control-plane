#------------------------------------------------------------------------------
# AWS ALB Module - Outputs
#------------------------------------------------------------------------------

output "alb_arn" {
  description = "ARN of the Application Load Balancer"
  value       = aws_lb.this.arn
}

output "alb_dns_name" {
  description = "DNS name of the Application Load Balancer"
  value       = aws_lb.this.dns_name
}

output "alb_zone_id" {
  description = "Zone ID of the Application Load Balancer (for Route 53 alias records)"
  value       = aws_lb.this.zone_id
}

output "alb_id" {
  description = "ID of the Application Load Balancer"
  value       = aws_lb.this.id
}

output "target_group_arn" {
  description = "ARN of the target group"
  value       = aws_lb_target_group.this.arn
}

output "target_group_name" {
  description = "Name of the target group"
  value       = aws_lb_target_group.this.name
}

output "http_listener_arn" {
  description = "ARN of the HTTP listener"
  value       = aws_lb_listener.http.arn
}

output "https_listener_arn" {
  description = "ARN of the HTTPS listener (null if HTTPS is disabled)"
  value       = var.enable_https ? aws_lb_listener.https[0].arn : null
}

output "security_group_id" {
  description = "ID of the security group created by the module (null if existing security groups used)"
  value       = length(var.security_groups) == 0 ? aws_security_group.this[0].id : null
}
