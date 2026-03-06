#------------------------------------------------------------------------------
# GCP VPC Module - Outputs
#------------------------------------------------------------------------------

output "network_id" {
  description = "The ID of the VPC network"
  value       = google_compute_network.main.id
}

output "network_name" {
  description = "The name of the VPC network"
  value       = google_compute_network.main.name
}

output "network_self_link" {
  description = "The self_link of the VPC network"
  value       = google_compute_network.main.self_link
}

output "subnet_ids" {
  description = "Map of subnet names to their IDs"
  value       = { for name, subnet in google_compute_subnetwork.main : name => subnet.id }
}

output "subnet_self_links" {
  description = "Map of subnet names to their self_links"
  value       = { for name, subnet in google_compute_subnetwork.main : name => subnet.self_link }
}

output "subnet_secondary_ranges" {
  description = "Map of subnet names to their secondary IP ranges"
  value = {
    for name, subnet in google_compute_subnetwork.main : name => {
      for range in subnet.secondary_ip_range : range.range_name => range.ip_cidr_range
    }
  }
}

output "router_name" {
  description = "The name of the Cloud Router (if created)"
  value       = var.create_nat_gateway ? google_compute_router.main[0].name : null
}

output "router_id" {
  description = "The ID of the Cloud Router (if created)"
  value       = var.create_nat_gateway ? google_compute_router.main[0].id : null
}

output "nat_ip" {
  description = "The external IP address of the Cloud NAT gateway (if created)"
  value       = var.create_nat_gateway ? google_compute_address.nat[0].address : null
}

output "nat_ip_self_link" {
  description = "The self_link of the Cloud NAT IP address (if created)"
  value       = var.create_nat_gateway ? google_compute_address.nat[0].self_link : null
}

output "nat_gateway_name" {
  description = "The name of the Cloud NAT gateway (if created)"
  value       = var.create_nat_gateway ? google_compute_router_nat.main[0].name : null
}
