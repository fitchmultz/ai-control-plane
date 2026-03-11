# GCP GKE Terraform Module

A comprehensive Terraform module for creating Google Kubernetes Engine (GKE) clusters with production-ready defaults.

## Features

- **GKE Autopilot and Standard support**: Configurable node pools with autoscaling
- **Workload Identity**: Enabled by default for secure pod-to-GCP service authentication
- **Private Cluster**: Optional private nodes with authorized networks
- **Security**: Shielded nodes, Binary Authorization, network policies, and dataplane V2
- **Observability**: Cloud Monitoring, Cloud Logging, and Managed Prometheus
- **Networking**: VPC-native cluster with secondary IP ranges for pods and services
- **Maintenance**: Configurable maintenance windows
- **Cost Optimization**: Support for Spot/Preemptible VMs and cost allocation

## Usage

### Basic Example

```hcl
module "gke" {
  source = "./modules/gcp/gke"

  cluster_name  = "ai-control-plane"
  project_id    = "my-project-id"
  region        = "us-central1"
  network       = "projects/my-project-id/global/networks/my-vpc"
  subnetwork    = "projects/my-project-id/regions/us-central1/subnetworks/my-subnet"
  
  pods_secondary_range_name     = "pods"
  services_secondary_range_name = "services"
}
```

### Complete Example

```hcl
module "gke" {
  source = "./modules/gcp/gke"

  # Required parameters
  cluster_name  = "ai-control-plane"
  project_id    = "my-project-id"
  region        = "us-central1"
  network       = "projects/my-project-id/global/networks/my-vpc"
  subnetwork    = "projects/my-project-id/regions/us-central1/subnetworks/my-subnet"
  
  pods_secondary_range_name     = "pods"
  services_secondary_range_name = "services"

  # Cluster configuration
  kubernetes_version = "1.29"
  release_channel    = "REGULAR"

  # Node pools
  node_pools = {
    "system" = {
      machine_type       = "e2-standard-4"
      initial_node_count = 1
      min_count          = 1
      max_count          = 3
      disk_size_gb       = 100
      preemptible        = false
      labels = {
        "workload-type" = "system"
      }
      taints = []
    }
    
    "workloads" = {
      machine_type       = "e2-standard-8"
      initial_node_count = 2
      min_count          = 1
      max_count          = 10
      disk_size_gb       = 200
      spot               = true
      labels = {
        "workload-type" = "workloads"
      }
      taints = [
        {
          key    = "dedicated"
          value  = "workloads"
          effect = "NO_SCHEDULE"
        }
      ]
    }
    
    "gpu" = {
      machine_type       = "n1-standard-4"
      initial_node_count = 0
      min_count          = 0
      max_count          = 5
      disk_size_gb       = 100
      preemptible        = true
      labels = {
        "accelerator" = "nvidia-t4"
      }
      taints = [
        {
          key    = "nvidia.com/gpu"
          value  = "true"
          effect = "NO_SCHEDULE"
        }
      ]
    }
  }

  # Private cluster configuration
  enable_private_nodes       = true
  master_ipv4_cidr_block     = "172.16.0.0/28"
  master_authorized_networks = [
    {
      cidr_block   = "10.0.0.0/8"
      display_name = "Internal Network"
    },
    {
      cidr_block   = "203.0.113.0/24"
      display_name = "Office Network"
    }
  ]

  # Workload Identity
  enable_workload_identity = true

  # Maintenance window
  maintenance_start_time = "2024-01-01T06:00:00Z"
  maintenance_end_time   = "2024-01-01T12:00:00Z"
  maintenance_recurrence = "FREQ=WEEKLY;BYDAY=SA,SU"

  # Labels
  labels = {
    environment = "production"
    team        = "platform"
  }
}
```

### Workload Identity Binding

To bind a Kubernetes service account to a GCP service account:

```hcl
module "gke" {
  # ... other configuration ...

  enable_workload_identity = true

  workload_identity_bindings = {
    "app-storage" = {
      google_service_account = "app-storage@my-project-id.iam.gserviceaccount.com"
      namespace              = "default"
      k8s_service_account    = "app-storage-ksa"
    }
  }
}
```

## Requirements

| Name | Version |
|------|---------|
| terraform | >= 1.0 |
| google | >= 5.0 |
| google-beta | >= 5.0 |

## Providers

| Name | Version |
|------|---------|
| google | >= 5.0 |
| google-beta | >= 5.0 |

## Resources

| Name | Type |
|------|------|
| google_container_cluster.primary | resource |
| google_container_node_pool.pools | resource |
| google_service_account.gke_nodes | resource |
| google_project_iam_member.gke_nodes_* | resource |
| google_service_account_iam_binding.workload_identity_binding | resource |

## Inputs

### Required Inputs

| Name | Description | Type |
|------|-------------|------|
| cluster_name | Name of the GKE cluster | `string` |
| project_id | GCP project ID where the cluster will be created | `string` |
| region | GCP region for the cluster | `string` |
| network | VPC network self_link where the cluster will be deployed | `string` |
| subnetwork | Subnetwork self_link where the cluster will be deployed | `string` |
| pods_secondary_range_name | Name of the secondary IP range for pods | `string` |
| services_secondary_range_name | Name of the secondary IP range for services | `string` |

### Optional Inputs

| Name | Description | Type | Default |
|------|-------------|------|---------|
| kubernetes_version | Kubernetes version for the cluster | `string` | `"1.29"` |
| release_channel | Release channel for the cluster | `string` | `"REGULAR"` |
| node_pools | Map of node pool configurations | `map(object)` | See description |
| enable_private_nodes | Enable private nodes | `bool` | `true` |
| master_ipv4_cidr_block | CIDR block for the master endpoint | `string` | `"172.16.0.0/28"` |
| master_authorized_networks | List of authorized networks for master access | `list(object)` | `[]` |
| enable_workload_identity | Enable Workload Identity | `bool` | `true` |
| workload_identity_bindings | Map of Workload Identity bindings | `map(object)` | `{}` |
| maintenance_start_time | Start time for maintenance window | `string` | `"2024-01-01T06:00:00Z"` |
| maintenance_end_time | End time for maintenance window | `string` | `"2024-01-01T12:00:00Z"` |
| maintenance_recurrence | Recurrence rule for maintenance | `string` | `"FREQ=WEEKLY;BYDAY=SA,SU"` |
| enable_cluster_autoscaling | Enable cluster autoscaling | `bool` | `false` |
| cluster_autoscaling_min_cpu | Minimum CPU cores for autoscaling | `number` | `2` |
| cluster_autoscaling_max_cpu | Maximum CPU cores for autoscaling | `number` | `100` |
| cluster_autoscaling_min_memory | Minimum memory for autoscaling | `number` | `4` |
| cluster_autoscaling_max_memory | Maximum memory for autoscaling | `number` | `400` |
| enable_binary_authorization | Enable Binary Authorization | `bool` | `false` |
| enable_vertical_pod_autoscaling | Enable Vertical Pod Autoscaling | `bool` | `true` |
| cluster_dns_provider | DNS provider for the cluster | `string` | `"PLATFORM_DEFAULT"` |
| cluster_dns_scope | DNS scope for the cluster | `string` | `"CLUSTER_SCOPE"` |
| cluster_dns_domain | DNS domain for the cluster | `string` | `"cluster.local"` |
| logging_components | List of logging components | `list(string)` | `["SYSTEM_COMPONENTS", "WORKLOADS"]` |
| monitoring_components | List of monitoring components | `list(string)` | `["SYSTEM_COMPONENTS", "APISERVER", "CONTROLLER_MANAGER", "SCHEDULER"]` |
| enable_managed_prometheus | Enable Managed Prometheus | `bool` | `true` |
| labels | Labels to apply to cluster resources | `map(string)` | `{}` |

### Node Pool Configuration Object

The `node_pools` variable accepts a map where each key is the node pool name and the value is an object with the following attributes:

| Name | Description | Type | Default |
|------|-------------|------|---------|
| machine_type | Machine type for nodes | `string` | `"e2-medium"` |
| initial_node_count | Initial number of nodes | `number` | `1` |
| min_count | Minimum nodes for autoscaling | `number` | `null` |
| max_count | Maximum nodes for autoscaling | `number` | `null` |
| disk_size_gb | Disk size in GB | `number` | `100` |
| disk_type | Disk type (pd-standard, pd-balanced, pd-ssd) | `string` | `"pd-balanced"` |
| preemptible | Use preemptible VMs | `bool` | `false` |
| spot | Use spot VMs | `bool` | `false` |
| labels | Map of labels for nodes | `map(string)` | `{}` |
| taints | List of taint objects | `list(object)` | `[]` |
| max_surge | Maximum surge during upgrades | `number` | `1` |
| max_unavailable | Maximum unavailable during upgrades | `number` | `0` |
| enable_gcfs | Enable Google Container File System | `bool` | `false` |
| enable_gvnic | Enable gVNIC | `bool` | `true` |
| enable_confidential_nodes | Enable confidential nodes | `bool` | `false` |
| reservation_affinity_type | Reservation affinity type | `string` | `"NO_RESERVATION"` |
| network_tags | List of network tags | `list(string)` | `[]` |

## Outputs

| Name | Description | Sensitive |
|------|-------------|-----------|
| cluster_id | The unique identifier of the GKE cluster | no |
| cluster_name | The name of the GKE cluster | no |
| cluster_location | The location of the GKE cluster | no |
| endpoint | The endpoint IP address of the Kubernetes master | yes |
| ca_certificate | The base64-encoded cluster CA certificate | yes |
| master_version | The current Kubernetes master version | no |
| workload_identity_pool | The Workload Identity Pool | no |
| network | The VPC network self_link | no |
| subnetwork | The subnetwork self_link | no |
| pods_range_name | The secondary IP range name for pods | no |
| services_range_name | The secondary IP range name for services | no |
| private_endpoint | The internal IP of the master (private clusters) | yes |
| public_endpoint | The external IP of the master | yes |
| master_ipv4_cidr_block | The CIDR block for the master endpoint | no |
| node_pools | Map of node pool details | no |
| node_pool_names | List of node pool names | no |
| service_account_email | The email of the node service account | no |
| service_account_name | The fully-qualified name of the node service account | no |
| service_account_id | The account ID of the node service account | no |
| kubectl_connection_command | gcloud command to get kubectl credentials | no |
| release_channel | The release channel of the cluster | no |
| datapath_provider | The datapath provider | no |
| enable_workload_identity | Whether Workload Identity is enabled | no |
| enable_private_nodes | Whether private nodes are enabled | no |
| cluster_resource_labels | The resource labels on the cluster | no |
| maintenance_window | The maintenance window configuration | no |

## Notes

### Private Clusters

When `enable_private_nodes` is `true`:
- Nodes receive only private IP addresses
- The master endpoint is accessible via a public endpoint with authorized networks
- Configure `master_authorized_networks` to control access to the master endpoint
- Use `master_ipv4_cidr_block` to specify the CIDR for the master endpoint (must not overlap with VPC subnets)

### Workload Identity

When `enable_workload_identity` is `true`:
- Pods can authenticate to Google Cloud services without service account keys
- The node pool is configured with `workload_metadata_config.mode = "GKE_METADATA"`
- Use `workload_identity_bindings` to bind Kubernetes service accounts to GCP service accounts

### Node Pool Taints

Taints are specified as a list of objects with the following structure:

```hcl
taints = [
  {
    key    = "dedicated"
    value  = "gpu"
    effect = "NO_SCHEDULE"
  }
]
```

Valid effects: `NO_SCHEDULE`, `PREFER_NO_SCHEDULE`, `NO_EXECUTE`

### Maintenance Windows

The maintenance window uses RFC 3339 format for times and RFC 5545 RRULE format for recurrence:

- `FREQ=WEEKLY;BYDAY=SA,SU` - Weekends
- `FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR` - Weekdays
- `FREQ=DAILY` - Every day

## License

This module is part of the AI Control Plane project.
