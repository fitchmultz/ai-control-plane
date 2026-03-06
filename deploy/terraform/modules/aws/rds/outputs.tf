#-------------------------------------------------------------------------------
# Connection Outputs
#-------------------------------------------------------------------------------

output "db_instance_address" {
  description = "The hostname of the RDS instance"
  value       = aws_db_instance.this.address
}

output "db_instance_endpoint" {
  description = "The connection endpoint of the RDS instance"
  value       = aws_db_instance.this.endpoint
}

output "db_instance_port" {
  description = "The port on which the RDS instance accepts connections"
  value       = aws_db_instance.this.port
}

output "db_instance_name" {
  description = "The database name"
  value       = aws_db_instance.this.db_name
}

output "db_instance_username" {
  description = "The master username for the database"
  value       = aws_db_instance.this.username
}

output "db_instance_arn" {
  description = "The ARN of the RDS instance"
  value       = aws_db_instance.this.arn
}

output "db_instance_id" {
  description = "The RDS instance identifier"
  value       = aws_db_instance.this.id
}

output "database_url" {
  description = "PostgreSQL connection URL (sensitive)"
  value       = "postgresql://${aws_db_instance.this.username}:${var.password}@${aws_db_instance.this.address}:${aws_db_instance.this.port}/${aws_db_instance.this.db_name}"
  sensitive   = true
}

#-------------------------------------------------------------------------------
# Security Group Output
#-------------------------------------------------------------------------------

output "security_group_id" {
  description = "The ID of the security group created for the RDS instance"
  value       = aws_security_group.this.id
}

output "security_group_arn" {
  description = "The ARN of the security group created for the RDS instance"
  value       = aws_security_group.this.arn
}

#-------------------------------------------------------------------------------
# Subnet Group Output
#-------------------------------------------------------------------------------

output "db_subnet_group_id" {
  description = "The ID of the DB subnet group"
  value       = aws_db_subnet_group.this.id
}

output "db_subnet_group_arn" {
  description = "The ARN of the DB subnet group"
  value       = aws_db_subnet_group.this.arn
}

#-------------------------------------------------------------------------------
# Resource References
#-------------------------------------------------------------------------------

output "db_instance_resource_id" {
  description = "The RDS Resource ID of this instance"
  value       = aws_db_instance.this.resource_id
}
