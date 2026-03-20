# Terraform Backend Configuration Examples

This directory contains example backend configurations for storing Terraform state remotely.

## Overview

Terraform state contains sensitive information about your infrastructure. Storing state locally:
- Makes collaboration difficult
- Risks losing state if your machine fails
- Makes CI/CD integration challenging

Remote backends solve these problems by:
- Storing state securely in cloud storage
- Enabling state locking to prevent concurrent modifications
- Supporting encryption at rest
- Enabling team collaboration

## Available Backends

| File | Provider | Storage | Locking | Best For |
|------|----------|---------|---------|----------|
| `s3-backend.tf` | AWS | S3 | DynamoDB | AWS deployments |
| `azurerm-backend.tf` | Azure | Blob Storage | Native | Azure deployments |
| `gcs-backend.tf` | GCP | Cloud Storage | Native | GCP deployments |

## Quick Start

### AWS (S3 + DynamoDB)

1. Create the S3 bucket and DynamoDB table:

```bash
# Create S3 bucket
aws s3 mb s3://ai-control-plane-tfstate --region us-east-1

# Enable versioning
aws s3api put-bucket-versioning \
  --bucket ai-control-plane-tfstate \
  --versioning-configuration Status=Enabled

# Create DynamoDB table for locking
aws dynamodb create-table \
  --table-name terraform-locks \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST
```

2. Copy `s3-backend.tf` to your example directory
3. Update the bucket name and region in the file
4. Initialize Terraform:

```bash
cd deploy/incubating/terraform/examples/aws-complete
cp ../../backend-examples/s3-backend.tf .
# Edit s3-backend.tf with your bucket name
terraform init
```

### Azure (Blob Storage)

1. Create the storage account:

```bash
# Create resource group
az group create --name terraform-state --location eastus

# Create storage account (must be globally unique)
az storage account create \
  --name aicptfstate \
  --resource-group terraform-state \
  --location eastus \
  --sku Standard_GRS

# Create container
az storage container create \
  --name tfstate \
  --account-name aicptfstate \
  --auth-mode login
```

2. Copy `azurerm-backend.tf` to your example directory
3. Update the storage account name (must be globally unique)
4. Initialize Terraform:

```bash
cd deploy/incubating/terraform/examples/azure-complete
cp ../../backend-examples/azurerm-backend.tf .
# Edit azurerm-backend.tf with your values
terraform init
```

### GCP (Cloud Storage)

1. Create the GCS bucket:

```bash
# Create bucket (must be globally unique)
gsutil mb -l us-central1 gs://ai-control-plane-tfstate

# Enable versioning
gsutil versioning set on gs://ai-control-plane-tfstate
```

2. Copy `gcs-backend.tf` to your example directory
3. Update the bucket name (must be globally unique)
4. Initialize Terraform:

```bash
cd deploy/incubating/terraform/examples/gcp-complete
cp ../../backend-examples/gcs-backend.tf .
# Edit gcs-backend.tf with your bucket name
gcloud auth application-default login  # If not already done
terraform init
```

## Backend Security Best Practices

### 1. Encryption

All backends support encryption at rest:
- **AWS**: S3 server-side encryption (SSE-S3 or SSE-KMS)
- **Azure**: Storage encryption enabled by default
- **GCP**: GCS encryption enabled by default, CMEK supported

### 2. Versioning

Enable versioning to recover from accidental deletion or corruption:
- **AWS**: `aws s3api put-bucket-versioning`
- **Azure**: `az storage account blob-service-properties update --enable-versioning`
- **GCP**: `gsutil versioning set on`

### 3. Access Control

Limit access to state files:

**AWS IAM Policy:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject"
      ],
      "Resource": "arn:aws:s3:::ai-control-plane-tfstate/*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:DeleteItem"
      ],
      "Resource": "arn:aws:dynamodb:*:*:table/terraform-locks"
    }
  ]
}
```

**Azure RBAC:**
Assign "Storage Blob Data Contributor" role to the Terraform service principal.

**GCP IAM:**
Grant `roles/storage.objectAdmin` on the bucket to the Terraform service account.

### 4. State Locking

Always enable state locking to prevent concurrent modifications:
- **AWS**: DynamoDB table
- **Azure**: Native blob storage locking
- **GCP**: Native GCS locking

## Migrating to Remote Backend

If you have existing local state:

```bash
# 1. Copy backend configuration to your working directory
cp backend-examples/s3-backend.tf .

# 2. Edit the configuration with your bucket details

# 3. Initialize with migration
terraform init -migrate-state

# 4. Verify state was migrated
terraform state list

# 5. (Optional) Delete local state file
rm terraform.tfstate
```

## Partial Configuration

For CI/CD pipelines, use partial configuration to avoid hardcoding secrets:

```bash
terraform init \
  -backend-config="bucket=$TF_STATE_BUCKET" \
  -backend-config="key=$TF_STATE_KEY" \
  -backend-config="region=$TF_STATE_REGION"
```

## Troubleshooting

### "Error: Failed to get existing workspaces"

The bucket/container doesn't exist or you don't have access. Verify:
- Resource exists
- IAM permissions are correct
- Credentials are configured

### "Error: Error acquiring the state lock"

Another process is holding the lock. To force unlock (use with caution):

```bash
terraform force-unlock <LOCK_ID>
```

### "Error: No valid credential sources found"

Authentication is not configured. Set up credentials:
- **AWS**: `aws configure` or `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY`
- **Azure**: `az login` or service principal env vars
- **GCP**: `gcloud auth application-default login` or `GOOGLE_APPLICATION_CREDENTIALS`

## Additional Resources

- [Terraform Backends Documentation](https://developer.hashicorp.com/terraform/language/settings/backends)
- [AWS S3 Backend](https://developer.hashicorp.com/terraform/language/settings/backends/s3)
- [Azure Blob Storage Backend](https://developer.hashicorp.com/terraform/language/settings/backends/azurerm)
- [GCS Backend](https://developer.hashicorp.com/terraform/language/settings/backends/gcs)
