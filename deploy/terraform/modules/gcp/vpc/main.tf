# GCP VPC Module
# Creates a VPC network with custom subnets, Cloud Router, and Cloud NAT

terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

#------------------------------------------------------------------------------
# VPC Network
#------------------------------------------------------------------------------

resource "google_compute_network" "main" {
  name                    = var.network_name
  project                 = var.project_id
  auto_create_subnetworks = false
  routing_mode            = "GLOBAL"

  delete_default_routes_on_create = false

  labels = merge(
    var.labels,
    {
      managed_by = "terraform"
    }
  )
}

#------------------------------------------------------------------------------
# Subnets
#------------------------------------------------------------------------------

resource "google_compute_subnetwork" "main" {
  for_each = { for subnet in var.subnets : subnet.name => subnet }

  name                     = each.value.name
  project                  = var.project_id
  network                  = google_compute_network.main.id
  ip_cidr_range            = each.value.ip_cidr_range
  region                   = coalesce(each.value.region, var.region)
  private_ip_google_access = coalesce(each.value.private_ip_google_access, true)

  dynamic "secondary_ip_range" {
    for_each = coalesce(each.value.secondary_ip_ranges, [])

    content {
      range_name    = secondary_ip_range.value.range_name
      ip_cidr_range = secondary_ip_range.value.ip_cidr_range
    }
  }

  labels = merge(
    var.labels,
    {
      managed_by = "terraform"
    }
  )
}

#------------------------------------------------------------------------------
# Cloud Router
#------------------------------------------------------------------------------

resource "google_compute_router" "main" {
  count   = var.create_nat_gateway ? 1 : 0
  name    = var.router_name
  project = var.project_id
  region  = var.region
  network = google_compute_network.main.id

  bgp {
    asn = 64514
  }

  labels = merge(
    var.labels,
    {
      managed_by = "terraform"
    }
  )
}

#------------------------------------------------------------------------------
# Cloud NAT
#------------------------------------------------------------------------------

resource "google_compute_address" "nat" {
  count   = var.create_nat_gateway ? 1 : 0
  name    = "${var.network_name}-nat-ip"
  project = var.project_id
  region  = var.region

  address_type = "EXTERNAL"
  network_tier = "PREMIUM"

  labels = merge(
    var.labels,
    {
      managed_by = "terraform"
    }
  )
}

resource "google_compute_router_nat" "main" {
  count                              = var.create_nat_gateway ? 1 : 0
  name                               = "${var.network_name}-nat"
  project                            = var.project_id
  router                             = google_compute_router.main[0].name
  region                             = var.region
  nat_ip_allocate_option             = "MANUAL_ONLY"
  nat_ips                            = google_compute_address.nat[*].self_link
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }
}

#------------------------------------------------------------------------------
# VPC Access Connector (for Cloud Run, etc.)
#------------------------------------------------------------------------------

# Note: VPC Access Connector is optional and can be enabled if needed
# for serverless services to access private resources

#------------------------------------------------------------------------------
# Firewall Rules
#------------------------------------------------------------------------------

# Allow internal traffic within the VPC
resource "google_compute_firewall" "allow_internal" {
  name        = "${var.network_name}-allow-internal"
  project     = var.project_id
  network     = google_compute_network.main.name
  description = "Allow internal traffic within the VPC"

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    ports    = ["0-65535"]
  }

  allow {
    protocol = "udp"
    ports    = ["0-65535"]
  }

  source_ranges = [for subnet in var.subnets : subnet.ip_cidr_range]
}

# Allow SSH from IAP (Identity-Aware Proxy)
resource "google_compute_firewall" "allow_iap_ssh" {
  name        = "${var.network_name}-allow-iap-ssh"
  project     = var.project_id
  network     = google_compute_network.main.name
  description = "Allow SSH from Identity-Aware Proxy"

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = ["35.235.240.0/20"]
}
