# Azure Storage Backend Configuration for Terraform State
#
# Copy this file to your Terraform configuration and customize it.
# This backend stores state in Azure Blob Storage.
#
# Prerequisites:
#   1. Create a Resource Group for Terraform state
#   2. Create a Storage Account
#   3. Create a Container for state files
#   4. Configure access (Service Principal, Managed Identity, or OIDC)
#
# Example setup commands:
#
#   # Create resource group
#   az group create \
#     --name terraform-state \
#     --location eastus
#
#   # Create storage account (must be globally unique)
#   az storage account create \
#     --name aicptfstate \
#     --resource-group terraform-state \
#     --location eastus \
#     --sku Standard_GRS \
#     --encryption-services blob
#
#   # Create container
#   az storage container create \
#     --name tfstate \
#     --account-name aicptfstate \
#     --auth-mode login
#
#   # Enable versioning (optional but recommended)
#   az storage account blob-service-properties update \
#     --account-name aicptfstate \
#     --resource-group terraform-state \
#     --enable-versioning
#
# ------------------------------------------------------------------------------

terraform {
  backend "azurerm" {
    # Resource group containing the storage account
    resource_group_name = "terraform-state"

    # Storage account name (must be globally unique, lowercase alphanumeric)
    storage_account_name = "aicptfstate"

    # Container name for state files
    container_name = "tfstate"

    # Path to the state file within the container
    key = "azure/terraform.tfstate"

    # Authentication options (choose one):

    # Option 1: Service Principal (traditional)
    # client_id       = "00000000-0000-0000-0000-000000000000"
    # client_secret   = "client-secret"
    # subscription_id = "00000000-0000-0000-0000-000000000000"
    # tenant_id       = "00000000-0000-0000-0000-000000000000"

    # Option 2: Managed Identity (recommended for CI/CD)
    # use_msi = true
    # subscription_id = "00000000-0000-0000-0000-000000000000"
    # tenant_id       = "00000000-0000-0000-0000-000000000000"

    # Option 3: OIDC (recommended for GitHub Actions, GitLab CI)
    use_oidc        = true
    subscription_id = "00000000-0000-0000-0000-000000000000"
    tenant_id       = "00000000-0000-0000-0000-000000000000"
    client_id       = "00000000-0000-0000-0000-000000000000"
    # client_id_file_path = "/path/to/client-id"  # Alternative to client_id

    # Option 4: Azure CLI (for local development)
    # No additional config needed - uses `az login` context

    # Optional: Use Azure AD for storage authentication
    # use_azuread_auth = true

    # Optional: Environment (for Government or China clouds)
    # environment = "public"  # Options: public, usgovernment, china, german

    # Optional: Endpoint suffixes for custom clouds
    # endpoint = "https://custom.blob.core.usgovcloudapi.net/"
  }
}

# ------------------------------------------------------------------------------
# Alternative: Partial Configuration
# ------------------------------------------------------------------------------
#
# Use partial configuration with command line or environment variables:
#
#   terraform init \
#     -backend-config="resource_group_name=terraform-state" \
#     -backend-config="storage_account_name=aicptfstate" \
#     -backend-config="container_name=tfstate" \
#     -backend-config="key=azure/terraform.tfstate" \
#     -backend-config="subscription_id=$ARM_SUBSCRIPTION_ID" \
#     -backend-config="tenant_id=$ARM_TENANT_ID" \
#     -backend-config="use_oidc=true"
#
# Environment variables for OIDC:
#   export ARM_USE_OIDC=true
#   export ARM_SUBSCRIPTION_ID="00000000-0000-0000-0000-000000000000"
#   export ARM_TENANT_ID="00000000-0000-0000-0000-000000000000"
#   export ARM_CLIENT_ID="00000000-0000-0000-0000-000000000000"
#   export ARM_OIDC_TOKEN="your-oidc-token"
# ------------------------------------------------------------------------------
