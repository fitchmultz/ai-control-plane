# Google Cloud Storage Backend Configuration for Terraform State
#
# Copy this file to your Terraform configuration and customize it.
# This backend stores state in a GCS bucket.
#
# Prerequisites:
#   1. Create a GCS bucket for state storage
#   2. Enable versioning (recommended)
#   3. Configure encryption (optional but recommended)
#   4. Set up authentication (Service Account or Application Default Credentials)
#
# Example setup commands:
#
#   # Create GCS bucket (must be globally unique)
#   gsutil mb -l us-central1 gs://ai-control-plane-tfstate
#
#   # Enable versioning
#   gsutil versioning set on gs://ai-control-plane-tfstate
#
#   # Set uniform bucket-level access (recommended)
#   gsutil uniformbucketlevelaccess set on gs://ai-control-plane-tfstate
#
#   # Configure encryption with CMEK (optional)
#   gsutil kms encryption -k projects/my-project/locations/us-central1/\
#     keyRings/my-keyring/cryptoKeys/my-key gs://ai-control-plane-tfstate
#
#   # Create folder structure (GCS doesn't have real folders, but this helps organize)
#   gsutil cp /dev/null gs://ai-control-plane-tfstate/gcp/terraform.tfstate
#
# ------------------------------------------------------------------------------

terraform {
  backend "gcs" {
    # GCS bucket name (must be globally unique)
    bucket = "ai-control-plane-tfstate"

    # Path prefix within the bucket for state file
    prefix = "gcp/terraform.tfstate"

    # Optional: Custom GCS endpoint (for testing or custom endpoints)
    # endpoint = "https://storage.googleapis.com/storage/v1/"

    # Optional: Use a specific service account for the backend
    # This is the email address of the service account
    # impersonate_service_account = "terraform@my-project.iam.gserviceaccount.com"

    # Optional: Chain of service accounts for impersonation
    # impersonate_service_account_delegates = [
    #   "service-account-1@my-project.iam.gserviceaccount.com",
    #   "service-account-2@my-project.iam.gserviceaccount.com"
    # ]

    # Optional: Access token (for short-lived tokens, CI/CD)
    # access_token = "ya29.a0ARrdaM..."

    # Optional: Encryption with Customer-Managed Encryption Key (CMEK)
    # kms_encryption_key = "projects/my-project/locations/us-central1/\
    #                       keyRings/my-keyring/cryptoKeys/my-key"

    # Optional: Storage class for the state file
    # storage_class = "STANDARD"  # Options: STANDARD, NEARLINE, COLDLINE, ARCHIVE
  }
}

# ------------------------------------------------------------------------------
# Alternative: Partial Configuration
# ------------------------------------------------------------------------------
#
# Use partial configuration with command line or environment variables:
#
#   terraform init \
#     -backend-config="bucket=ai-control-plane-tfstate" \
#     -backend-config="prefix=gcp/terraform.tfstate"
#
# Environment variables:
#   export GOOGLE_BACKEND_BUCKET="ai-control-plane-tfstate"
#   export GOOGLE_BACKEND_PREFIX="gcp/terraform.tfstate"
#
# For authentication, set one of:
#   export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"
#   export GOOGLE_ACCESS_TOKEN="your-access-token"
#
# ------------------------------------------------------------------------------
# Authentication Methods
# ------------------------------------------------------------------------------
#
# 1. Application Default Credentials (ADC) - Recommended for local development:
#    gcloud auth application-default login
#
# 2. Service Account Key File:
#    export GOOGLE_APPLICATION_CREDENTIALS="/path/to/key.json"
#
# 3. Service Account Impersonation:
#    export GOOGLE_IMPERSONATE_SERVICE_ACCOUNT="terraform@my-project.iam.gserviceaccount.com"
#
# 4. Workload Identity (for GKE, Cloud Build, etc.):
#    No additional configuration needed when running on GCP with Workload Identity
#
# 5. Access Token (for CI/CD):
#    export GOOGLE_ACCESS_TOKEN="$(gcloud auth print-access-token)"
#
# ------------------------------------------------------------------------------
