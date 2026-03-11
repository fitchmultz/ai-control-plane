output "instance_id" {
  description = "The ID of the Cloud SQL instance"
  value       = google_sql_database_instance.instance.id
}

output "instance_name" {
  description = "The name of the Cloud SQL instance"
  value       = google_sql_database_instance.instance.name
}

output "connection_name" {
  description = "The connection name of the Cloud SQL instance (used for Cloud SQL Proxy)"
  value       = google_sql_database_instance.instance.connection_name
}

output "private_ip_address" {
  description = "The private IP address of the Cloud SQL instance (null if private IP not enabled)"
  value       = var.vpc_network != null ? google_sql_database_instance.instance.private_ip_address : null
}

output "public_ip_address" {
  description = "The public IP address of the Cloud SQL instance (null if public IP disabled)"
  value       = var.vpc_network == null ? google_sql_database_instance.instance.public_ip_address : null
}

output "database_name" {
  description = "The name of the created database"
  value       = google_sql_database.database.name
}

output "database_user" {
  description = "The name of the database user"
  value       = google_sql_user.user.name
}

output "database_url" {
  description = "PostgreSQL connection URL (sensitive)"
  value = format(
    "postgresql://%s:%s@%s/%s?sslmode=require",
    urlencode(var.user_name),
    urlencode(var.user_password),
    var.vpc_network != null ? coalesce(google_sql_database_instance.instance.private_ip_address, "unknown") : coalesce(google_sql_database_instance.instance.public_ip_address, "unknown"),
    urlencode(var.database_name)
  )
  sensitive = true
}

output "database_url_proxy" {
  description = "PostgreSQL connection URL for Cloud SQL Proxy (sensitive)"
  value = format(
    "postgresql://%s:%s@localhost/%s?host=/cloudsql/%s",
    urlencode(var.user_name),
    urlencode(var.user_password),
    urlencode(var.database_name),
    google_sql_database_instance.instance.connection_name
  )
  sensitive = true
}
