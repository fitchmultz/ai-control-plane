#------------------------------------------------------------------------------
# GCP Load Balancer Module - Outputs
#------------------------------------------------------------------------------

output "load_balancer_ip" {
  description = "Global IP address of the load balancer"
  value       = google_compute_global_address.this.address
}

output "load_balancer_ip_name" {
  description = "Name of the global IP address resource"
  value       = google_compute_global_address.this.name
}

output "backend_service_id" {
  description = "ID of the backend service"
  value       = google_compute_backend_service.this.id
}

output "backend_service_name" {
  description = "Name of the backend service"
  value       = google_compute_backend_service.this.name
}

output "url_map_id" {
  description = "ID of the URL map"
  value       = google_compute_url_map.this.id
}

output "url_map_name" {
  description = "Name of the URL map"
  value       = google_compute_url_map.this.name
}

output "target_proxy_id" {
  description = "ID of the target proxy (HTTP or HTTPS)"
  value       = var.enable_https ? google_compute_target_https_proxy.this[0].id : google_compute_target_http_proxy.this[0].id
}

output "target_http_proxy_id" {
  description = "ID of the HTTP target proxy (null if HTTPS only)"
  value       = var.enable_https ? null : google_compute_target_http_proxy.this[0].id
}

output "target_https_proxy_id" {
  description = "ID of the HTTPS target proxy (null if HTTP only)"
  value       = var.enable_https ? google_compute_target_https_proxy.this[0].id : null
}

output "forwarding_rule_id" {
  description = "ID of the primary forwarding rule"
  value       = var.enable_https ? google_compute_global_forwarding_rule.https[0].id : google_compute_global_forwarding_rule.http[0].id
}

output "http_forwarding_rule_id" {
  description = "ID of the HTTP forwarding rule (null if HTTPS only or redirect enabled)"
  value       = var.enable_http && !var.enable_https_redirect ? google_compute_global_forwarding_rule.http[0].id : null
}

output "https_forwarding_rule_id" {
  description = "ID of the HTTPS forwarding rule (null if HTTP only)"
  value       = var.enable_https ? google_compute_global_forwarding_rule.https[0].id : null
}

output "health_check_id" {
  description = "ID of the health check"
  value       = google_compute_health_check.this.id
}

output "health_check_name" {
  description = "Name of the health check"
  value       = google_compute_health_check.this.name
}

output "managed_ssl_certificate_id" {
  description = "ID of the managed SSL certificate (null if using self-managed or HTTP only)"
  value       = local.use_managed_ssl ? google_compute_managed_ssl_certificate.this[0].id : null
}

output "managed_ssl_certificate_status" {
  description = "Status of the managed SSL certificate"
  value       = local.use_managed_ssl ? google_compute_managed_ssl_certificate.this[0].certificate_status : null
}

output "neg_id" {
  description = "ID of the network endpoint group (null if not created)"
  value       = var.create_neg ? google_compute_network_endpoint_group.this[0].id : null
}

output "serverless_neg_id" {
  description = "ID of the serverless NEG (null if not created)"
  value       = var.create_serverless_neg ? google_compute_region_network_endpoint_group.this[0].id : null
}

output "ssl_policy_id" {
  description = "ID of the SSL policy (null if not created)"
  value       = var.create_ssl_policy ? google_compute_ssl_policy.this[0].id : null
}
