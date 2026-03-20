# GCP Complete Example

Complete AI Control Plane deployment on Google Cloud Platform (GCP) using Terraform.

## Overview

This example deploys a production-ready AI Control Plane on GCP with the following architecture:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              VPC Network                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                          GKE Cluster                                 │   │
│  │  ┌───────────────────────────────────────────────────────────────┐  │   │
│  │  │                    Kubernetes Namespace                        │  │   │
│  │  │  ┌──────────────────────────────────────────────────────────┐ │  │   │
│  │  │  │              AI Control Plane (Helm)                    │ │  │   │
│  │  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │ │  │   │
│  │  │  │  │  LiteLLM    │  │  Secrets    │  │   Ingress   │      │ │  │   │
│  │  │  │  │  Gateway    │  │  (K8s)      │  │   (opt)     │      │ │  │   │
│  │  │  │  └─────────────┘  └─────────────┘  └─────────────┘      │ │  │   │
│  │  │  └──────────────────────────────────────────────────────────┘ │  │   │
│  │  └───────────────────────────────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        Cloud SQL (Private IP)                        │   │
│  │                    PostgreSQL 16 - External DB                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        Cloud NAT Gateway                             │   │
│  │                 (Private subnet egress access)                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Features

- **VPC Network**: Private subnet with Cloud NAT for secure egress
- **GKE Cluster**: Workload Identity-enabled, private nodes, auto-upgrading
- **Cloud SQL**: PostgreSQL 16 with private IP, automated backups
- **Workload Identity**: Secure authentication between GKE and GCP services
- **Environment-specific Sizing**: Automatic resource sizing based on environment (dev/staging/production)
- **External Database**: LiteLLM configured to use Cloud SQL via Cloud SQL Proxy

## Prerequisites

1. **GCP Project** with billing enabled
2. **Required APIs enabled**:
   ```bash
   gcloud services enable compute.googleapis.com
   gcloud services enable container.googleapis.com
   gcloud services enable sqladmin.googleapis.com
   gcloud services enable servicenetworking.googleapis.com
   gcloud services enable iam.googleapis.com
   ```
3. **Terraform >= 1.5.0**
4. **gcloud CLI** (optional but recommended)
5. **kubectl** (optional but recommended)
6. **Cloud SQL Proxy** (optional, for database access)

## Quick Start

### 1. Clone and Navigate

```bash
cd deploy/incubating/terraform/examples/gcp-complete
```

### 2. Configure Variables

```bash
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars`:

```hcl
project_id = "your-gcp-project-id"
region     = "us-central1"
environment = "production"
litellm_master_key = "replace-with-32-character-minimum-master-key"
litellm_salt_key   = "replace-with-32-character-minimum-salt-key"
master_authorized_networks = [
  {
    cidr_block   = "YOUR_OFFICE_IP/32"
    display_name = "Office Network"
  }
]
```

### 3. Initialize Terraform

```bash
terraform init
```

### 4. Plan and Apply

```bash
terraform plan
terraform apply
```

### 5. Connect to the Cluster

```bash
# Configure kubectl (output from terraform)
gcloud container clusters get-credentials ai-cp-dev-cluster --region=us-central1 --project=your-gcp-project-id

# Verify pods
kubectl get pods -n acp
```

### 6. Access LiteLLM

```bash
# Port-forward for local access
kubectl port-forward -n acp svc/acp-litellm 4000:4000

# Keep access local-only when port-forwarding; shared access must use TLS ingress
```

## Module Structure

| Module | Source | Purpose |
|--------|--------|---------|
| `vpc` | `../../modules/gcp/vpc` | VPC network, subnet, Cloud NAT |
| `gke` | `../../modules/gcp/gke` | GKE cluster with Workload Identity |
| `cloudsql` | `../../modules/gcp/cloudsql` | Cloud SQL PostgreSQL instance |
| `namespace` | `../../modules/common/kubernetes-namespace` | Kubernetes namespace |
| `secrets` | `../../modules/common/secrets` | Kubernetes secrets for LiteLLM |
| `helm_release` | `../../modules/common/helm-release` | AI Control Plane Helm chart |

## Configuration

### Environment-Specific Sizing

The example automatically configures resources based on the `environment` variable:

| Environment | GKE Nodes | Cloud SQL Tier | Availability |
|-------------|-----------|----------------|--------------|
| `dev` | 1x e2-medium (spot) | db-f1-micro | ZONAL |
| `staging` | 1-3x e2-medium | db-g1-small | ZONAL |
| `production` | 2-5x e2-standard-2 | db-n1-standard-2 | REGIONAL |

### Node Pools

Default node pool configuration:

```hcl
node_pools = {
  default = {
    machine_type       = "e2-medium"  # e2-standard-2 for production
    initial_node_count = 1            # 2 for production
    min_count          = 1            # 2 for production
    max_count          = 3            # 5 for production
    spot               = true         # false for production
  }
}
```

Custom node pools can be specified via the `node_pools` variable.

### Ingress Configuration

Enable ingress for external access:

```hcl
ingress_enabled  = true
ingress_host     = "ai-control-plane.yourdomain.com"
ingress_class_name = "nginx"
ingress_tls_secret_name = "ai-control-plane-tls"
ingress_cluster_issuer  = "letsencrypt-prod"
```

Requires an ingress controller (e.g., NGINX Ingress Controller) to be installed in the cluster.

### Master Authorized Networks

Restrict GKE control plane access:

```hcl
master_authorized_networks = [
  {
    cidr_block   = "YOUR_OFFICE_IP/32"
    display_name = "Office Network"
  }
]
```

## Workload Identity

This example uses [Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) to allow the AI Control Plane pods to authenticate to Google Cloud services without service account keys.

The following resources are created:

1. **GCP Service Account**: `${name_prefix}-${environment}-workload`
2. **IAM Binding**: Grants `roles/cloudsql.client` to the service account
3. **Workload Identity Binding**: Links K8s service account to GCP service account

The Helm chart is configured with the Workload Identity annotation:

```yaml
serviceAccount:
  annotations:
    iam.gke.io/gcp-service-account: ai-cp-dev-workload@project-id.iam.gserviceaccount.com
```

## Database Connection

Cloud SQL is accessed via the [Cloud SQL Proxy](https://cloud.google.com/sql/docs/postgres/sql-proxy) sidecar pattern. The connection uses Unix sockets for secure communication.

Connection string format:
```
postgresql://user:pass@localhost/litellm?host=/cloudsql/PROJECT:REGION:INSTANCE
```

## Outputs

### Infrastructure Information

| Output | Description |
|--------|-------------|
| `cluster_endpoint` | GKE cluster endpoint IP |
| `cluster_name` | GKE cluster name |
| `database_connection_name` | Cloud SQL connection name |
| `database_private_ip` | Cloud SQL private IP |

### Connection Commands

| Output | Description |
|--------|-------------|
| `kubectl_connection_command` | Command to configure kubectl |
| `port_forward_command` | Command to port-forward to LiteLLM |
| `cloud_sql_proxy_command` | Command to start Cloud SQL Proxy |
| `get_master_key_command` | Command to retrieve LiteLLM master key |

### Application URLs

| Output | Description |
|--------|-------------|
| `application_url` | URL to access AI Control Plane |

## Maintenance

### Scaling

```bash
# Update node pool size
gcloud container clusters resize ai-cp-dev-cluster \
  --node-pool=default \
  --num-nodes=3 \
  --region=us-central1

# Or update terraform
cat > terraform.tfvars <<EOF
node_pools = {
  default = {
    machine_type       = "e2-standard-2"
    initial_node_count = 3
    min_count          = 3
    max_count          = 5
    spot               = false
    labels             = {}
  }
}
EOF

terraform apply
```

### Database Backups

Automated backups are enabled by default. To create a manual backup:

```bash
gcloud sql backups create --instance=ai-cp-dev-db
```

### Viewing Logs

```bash
# LiteLLM logs
kubectl logs -n acp -l app.kubernetes.io/name=litellm -f

# All pods
kubectl logs -n acp --all-containers -f
```

## Cleanup

⚠️ **Warning**: This will delete all resources including the database.

```bash
terraform destroy
```

To preserve the database, set `deletion_protection = true` (default for production) and manually delete other resources.

## Troubleshooting

### GKE Connection Issues

```bash
# Verify cluster access
gcloud container clusters get-credentials ai-cp-dev-cluster --region=us-central1

# Check node status
kubectl get nodes
```

### Database Connection Issues

```bash
# Check Cloud SQL Proxy logs
kubectl logs -n acp -l app.kubernetes.io/name=litellm -c cloud-sql-proxy

# Verify database is accessible
kubectl run -it --rm debug --image=postgres:16 --restart=Never -- \
  psql "postgresql://litellm@localhost/litellm?host=/cloudsql/PROJECT:REGION:ai-cp-dev-db"
```

### Pod Not Starting

```bash
# Check pod events
kubectl describe pod -n acp -l app.kubernetes.io/name=litellm

# Check secret exists
kubectl get secret -n acp ai-cp-secrets
```

## Security Considerations

1. **Private Nodes**: Nodes have no external IP addresses
2. **Cloud NAT**: Provides secure egress for private nodes
3. **Private Cloud SQL**: Database is only accessible via private IP
4. **Workload Identity**: No service account keys stored in Kubernetes
5. **Master Authorized Networks**: GKE control plane access can be restricted
6. **Secrets**: Auto-generated strong passwords for database and LiteLLM

## Cost Estimation

Approximate monthly costs (us-central1):

| Component | Dev | Production |
|-----------|-----|------------|
| GKE (e2-medium × 1) | ~$25 | ~$200 (e2-standard-2 × 2) |
| Cloud SQL (db-f1-micro) | ~$7 | ~$100 (db-n1-standard-2) |
| VPC + NAT | ~$30 | ~$30 |
| **Total** | **~$62** | **~$330** |

*Note: Actual costs vary based on usage and data transfer.*

## Contributing

See the main project [CONTRIBUTING.md](../../../CONTRIBUTING.md) for guidelines.

## License and Compliance

- **Project License**: Apache-2.0. See the main project [`LICENSE`](../../../../../LICENSE) and [`NOTICE`](../../../../../NOTICE) files.
- **Third-Party License Policy**: `docs/policy/THIRD_PARTY_LICENSE_MATRIX.md` defines the complete third-party license boundary.
- **License Summary**: `docs/deployment/THIRD_PARTY_LICENSE_SUMMARY.md` — Generated compliance report for customer handoff
- **Compliance Check**: Run `make license-check` to verify no restricted components are included
