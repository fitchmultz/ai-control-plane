# Cloud SQL PostgreSQL Module for AI Control Plane
# Creates a PostgreSQL 16 instance with optional private IP

locals {
  require_ssl = false
}

# Get VPC network data if private IP is requested
data "google_compute_network" "vpc" {
  count   = var.vpc_network != null ? 1 : 0
  name    = var.vpc_network
  project = var.project_id
}

# Enable required APIs
resource "google_project_service" "sqladmin" {
  service            = "sqladmin.googleapis.com"
  project            = var.project_id
  disable_on_destroy = false
}

resource "google_project_service" "servicenetworking" {
  count              = var.vpc_network != null ? 1 : 0
  service            = "servicenetworking.googleapis.com"
  project            = var.project_id
  disable_on_destroy = false
}

# Reserve global address for VPC peering if private IP is requested
resource "google_compute_global_address" "private_ip_address" {
  count         = var.vpc_network != null ? 1 : 0
  provider      = google-beta
  name          = "${var.instance_name}-private-ip"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = data.google_compute_network.vpc[0].id
  project       = var.project_id
}

# Create VPC peering connection if private IP is requested
resource "google_service_networking_connection" "private_vpc_connection" {
  count                   = var.vpc_network != null ? 1 : 0
  provider                = google-beta
  network                 = data.google_compute_network.vpc[0].id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip_address[0].name]
  deletion_policy         = "ABANDON"
}

# Cloud SQL PostgreSQL instance
resource "google_sql_database_instance" "instance" {
  provider            = google-beta
  name                = var.instance_name
  database_version    = var.database_version
  region              = var.region
  project             = var.project_id
  deletion_protection = var.deletion_protection

  depends_on = [
    google_project_service.sqladmin,
    google_service_networking_connection.private_vpc_connection
  ]

  settings {
    tier              = var.tier
    disk_size         = var.disk_size
    disk_autoresize   = var.disk_autoresize
    availability_type = var.availability_type

    backup_configuration {
      enabled    = var.backup_enabled
      start_time = var.backup_start_time

      dynamic "backup_retention_settings" {
        for_each = var.backup_enabled ? [1] : []
        content {
          retained_backups = var.backup_retention_count
          retention_unit   = "COUNT"
        }
      }
    }

    maintenance_window {
      day          = var.maintenance_day
      hour         = var.maintenance_hour
      update_track = var.maintenance_track
    }

    ip_configuration {
      ipv4_enabled    = var.vpc_network == null
      private_network = var.vpc_network != null ? data.google_compute_network.vpc[0].id : null
      require_ssl     = local.require_ssl

      dynamic "authorized_networks" {
        for_each = var.authorized_networks
        content {
          name  = authorized_networks.value.name
          value = authorized_networks.value.cidr
        }
      }
    }

    dynamic "insights_config" {
      for_each = var.enable_insights ? [1] : []
      content {
        query_insights_enabled  = true
        query_string_length     = var.insights_query_length
        record_application_tags = true
        record_client_address   = true
      }
    }

    user_labels = var.labels
  }
}

# Create the database
resource "google_sql_database" "database" {
  provider = google-beta
  name     = var.database_name
  instance = google_sql_database_instance.instance.name
  project  = var.project_id
}

# Create the database user
resource "google_sql_user" "user" {
  provider = google-beta
  name     = var.user_name
  instance = google_sql_database_instance.instance.name
  password = var.user_password
  project  = var.project_id
}
