# -----------------------------------------------------------------------------
# GKE Cluster Module
# -----------------------------------------------------------------------------

terraform {
  required_version = ">= 1.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 5.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = ">= 5.0"
    }
  }
}

# -----------------------------------------------------------------------------
# Service Account for GKE Nodes
# -----------------------------------------------------------------------------

resource "google_service_account" "gke_nodes" {
  account_id   = "${var.cluster_name}-nodes"
  display_name = "GKE Node Service Account for ${var.cluster_name}"
  description  = "Service account used by GKE nodes"
  project      = var.project_id
}

# Grant minimal required roles to the node service account
resource "google_project_iam_member" "gke_nodes_log_writer" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.gke_nodes.email}"
}

resource "google_project_iam_member" "gke_nodes_metric_writer" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.gke_nodes.email}"
}

resource "google_project_iam_member" "gke_nodes_monitoring_viewer" {
  project = var.project_id
  role    = "roles/monitoring.viewer"
  member  = "serviceAccount:${google_service_account.gke_nodes.email}"
}

resource "google_project_iam_member" "gke_nodes_stackdriver_writer" {
  project = var.project_id
  role    = "roles/stackdriver.resourceMetadata.writer"
  member  = "serviceAccount:${google_service_account.gke_nodes.email}"
}

# -----------------------------------------------------------------------------
# GKE Cluster
# -----------------------------------------------------------------------------

resource "google_container_cluster" "primary" {
  provider = google-beta

  name        = var.cluster_name
  location    = var.region
  project     = var.project_id
  description = "GKE cluster for AI Control Plane"

  # Kubernetes version configuration
  min_master_version = var.kubernetes_version
  release_channel {
    channel = var.release_channel
  }

  # Network configuration
  network    = var.network
  subnetwork = var.subnetwork

  # IP allocation for pods and services using secondary ranges
  ip_allocation_policy {
    cluster_secondary_range_name  = var.pods_secondary_range_name
    services_secondary_range_name = var.services_secondary_range_name
  }

  # Private cluster configuration
  private_cluster_config {
    enable_private_nodes    = var.enable_private_nodes
    enable_private_endpoint = false
    master_ipv4_cidr_block  = var.master_ipv4_cidr_block

    dynamic "master_global_access_config" {
      for_each = var.enable_private_nodes ? [1] : []
      content {
        enabled = true
      }
    }
  }

  # Master authorized networks
  dynamic "master_authorized_networks_config" {
    for_each = length(var.master_authorized_networks) > 0 ? [1] : []
    content {
      dynamic "cidr_blocks" {
        for_each = var.master_authorized_networks
        content {
          cidr_block   = cidr_blocks.value.cidr_block
          display_name = cidr_blocks.value.display_name
        }
      }
    }
  }

  # Workload Identity configuration
  workload_identity_config {
    workload_pool = var.enable_workload_identity ? "${var.project_id}.svc.id.goog" : null
  }

  # Enable Shielded Nodes for enhanced security
  shielded_nodes {
    enabled = true
  }

  # Network policy (enabled by default)
  network_policy {
    enabled  = true
    provider = "CALICO"
  }

  # Enable intranode visibility for better network monitoring
  enable_intranode_visibility = true

  # Enable dataplane V2 for enhanced networking and security
  datapath_provider = "ADVANCED_DATAPATH"

  # Cost management configuration
  resource_usage_export_config {
    enable_network_egress_metering       = true
    enable_resource_consumption_metering = true
  }

  # Maintenance policy
  maintenance_policy {
    recurring_window {
      start_time = var.maintenance_start_time
      end_time   = var.maintenance_end_time
      recurrence = var.maintenance_recurrence
    }
  }

  # Cluster-level labels
  resource_labels = merge(
    var.labels,
    {
      managed_by = "terraform"
      cluster    = var.cluster_name
    }
  )

  # Remove default node pool - we'll create custom node pools
  remove_default_node_pool = true
  initial_node_count       = 1

  # Cluster autoscaling (for node auto-provisioning)
  dynamic "cluster_autoscaling" {
    for_each = var.enable_cluster_autoscaling ? [1] : []
    content {
      enabled = true

      resource_limits {
        resource_type = "cpu"
        minimum       = var.cluster_autoscaling_min_cpu
        maximum       = var.cluster_autoscaling_max_cpu
      }

      resource_limits {
        resource_type = "memory"
        minimum       = var.cluster_autoscaling_min_memory
        maximum       = var.cluster_autoscaling_max_memory
      }

      auto_provisioning_defaults {
        service_account = google_service_account.gke_nodes.email
        oauth_scopes = [
          "https://www.googleapis.com/auth/cloud-platform"
        ]

        management {
          auto_repair  = true
          auto_upgrade = true
        }

        upgrade_settings {
          max_surge       = 1
          max_unavailable = 0
        }

        disk_size_gb = 100
        disk_type    = "pd-balanced"
      }
    }
  }

  # Binary authorization (optional)
  dynamic "binary_authorization" {
    for_each = var.enable_binary_authorization ? [1] : []
    content {
      evaluation_mode = "PROJECT_SINGLETON_POLICY_ENFORCE"
    }
  }

  # Cost allocation for better visibility
  cost_management_config {
    enabled = true
  }

  # Vertical Pod Autoscaling
  vertical_pod_autoscaling {
    enabled = var.enable_vertical_pod_autoscaling
  }

  # DNS configuration
  dns_config {
    cluster_dns        = var.cluster_dns_provider
    cluster_dns_scope  = var.cluster_dns_scope
    cluster_dns_domain = var.cluster_dns_domain
  }

  # Logging and monitoring
  logging_config {
    enable_components = var.logging_components
  }

  monitoring_config {
    enable_components = var.monitoring_components
    managed_prometheus {
      enabled = var.enable_managed_prometheus
    }
  }

  # Depends on service account IAM bindings
  depends_on = [
    google_project_iam_member.gke_nodes_log_writer,
    google_project_iam_member.gke_nodes_metric_writer,
    google_project_iam_member.gke_nodes_monitoring_viewer,
    google_project_iam_member.gke_nodes_stackdriver_writer,
  ]

  lifecycle {
    ignore_changes = [
      # Ignore changes to node pools, as they're managed separately
      node_pool,
      initial_node_count,
    ]
  }
}

# -----------------------------------------------------------------------------
# Node Pools
# -----------------------------------------------------------------------------

resource "google_container_node_pool" "pools" {
  for_each = var.node_pools

  provider = google-beta

  name     = each.key
  location = var.region
  cluster  = google_container_cluster.primary.name
  project  = var.project_id

  # Node count and autoscaling
  initial_node_count = each.value.initial_node_count

  dynamic "autoscaling" {
    for_each = each.value.min_count != null && each.value.max_count != null ? [1] : []
    content {
      min_node_count  = each.value.min_count
      max_node_count  = each.value.max_count
      location_policy = "BALANCED"
    }
  }

  # Node configuration
  node_config {
    machine_type    = each.value.machine_type
    disk_size_gb    = each.value.disk_size_gb
    disk_type       = each.value.disk_type
    preemptible     = each.value.preemptible
    spot            = each.value.spot
    service_account = google_service_account.gke_nodes.email

    # Workload Identity on nodes
    workload_metadata_config {
      mode = var.enable_workload_identity ? "GKE_METADATA" : "GCE_METADATA"
    }

    # OAuth scopes - use cloud-platform for Workload Identity
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]

    # Node labels
    labels = merge(
      each.value.labels,
      {
        "node-pool" = each.key
      }
    )

    # Node taints
    dynamic "taint" {
      for_each = each.value.taints
      content {
        key    = taint.value.key
        value  = taint.value.value
        effect = taint.value.effect
      }
    }

    # Shielded instance config
    shielded_instance_config {
      enable_secure_boot          = true
      enable_integrity_monitoring = true
    }

    # Containerd runtime (default for GKE 1.24+)
    gcfs_config {
      enabled = each.value.enable_gcfs
    }

    # gvnic for better networking performance
    gvnic {
      enabled = each.value.enable_gvnic
    }

    # Confidential nodes (optional)
    dynamic "confidential_nodes" {
      for_each = each.value.enable_confidential_nodes ? [1] : []
      content {
        enabled = true
      }
    }

    # Reservation affinity
    reservation_affinity {
      consume_reservation_type = each.value.reservation_affinity_type
    }

    tags = each.value.network_tags
  }

  # Management settings
  management {
    auto_repair  = true
    auto_upgrade = true
  }

  # Upgrade settings
  upgrade_settings {
    max_surge       = each.value.max_surge
    max_unavailable = each.value.max_unavailable
    strategy        = "SURGE"
  }

  # Node pool lifecycle
  lifecycle {
    create_before_destroy = true
    ignore_changes = [
      initial_node_count,
    ]
  }

  depends_on = [google_container_cluster.primary]
}

# -----------------------------------------------------------------------------
# Workload Identity Service Account Binding (for default namespace)
# -----------------------------------------------------------------------------

resource "google_service_account_iam_binding" "workload_identity_binding" {
  for_each = var.enable_workload_identity ? var.workload_identity_bindings : {}

  service_account_id = each.value.google_service_account
  role               = "roles/iam.workloadIdentityUser"

  members = [
    "serviceAccount:${var.project_id}.svc.id.goog[${each.value.namespace}/${each.value.k8s_service_account}]"
  ]
}
