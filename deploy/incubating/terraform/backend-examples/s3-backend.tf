# AWS S3 Backend Configuration for Terraform State
#
# Copy this file to your Terraform configuration and customize it.
# This backend stores state in S3 with DynamoDB for state locking.
#
# Prerequisites:
#   1. Create an S3 bucket for state storage
#   2. Create a DynamoDB table for state locking
#   3. Configure IAM permissions for Terraform
#
# Example setup commands:
#
#   # Create S3 bucket
#   aws s3 mb s3://ai-control-plane-tfstate --region us-east-1
#   aws s3api put-bucket-versioning \
#     --bucket ai-control-plane-tfstate \
#     --versioning-configuration Status=Enabled
#   aws s3api put-bucket-encryption \
#     --bucket ai-control-plane-tfstate \
#     --server-side-encryption-configuration '{
#       "Rules": [{"ApplyServerSideEncryptionByDefault": {"SSEAlgorithm": "AES256"}}]
#     }'
#
#   # Create DynamoDB table for locking
#   aws dynamodb create-table \
#     --table-name terraform-locks \
#     --attribute-definitions AttributeName=LockID,AttributeType=S \
#     --key-schema AttributeName=LockID,KeyType=HASH \
#     --billing-mode PAY_PER_REQUEST
#
# ------------------------------------------------------------------------------

terraform {
  backend "s3" {
    # S3 bucket for storing Terraform state
    bucket = "ai-control-plane-tfstate"

    # Path to the state file within the bucket
    key = "aws/terraform.tfstate"

    # AWS region where the bucket is located
    region = "us-east-1"

    # Enable server-side encryption
    encrypt = true

    # DynamoDB table for state locking
    dynamodb_table = "terraform-locks"

    # Optional: Use KMS key for encryption
    # kms_key_id = "arn:aws:kms:us-east-1:123456789012:key/my-key"

    # Optional: S3 bucket for access logging
    # accesslogging_bucket_name = "terraform-state-logs"
    # accesslogging_target_prefix = "logs/"

    # Optional: Assume role for backend operations
    # assume_role = {
    #   role_arn     = "arn:aws:iam::123456789012:role/TerraformStateAccess"
    #   session_name = "terraform-backend"
    # }
  }
}

# ------------------------------------------------------------------------------
# Alternative: Partial Configuration
# ------------------------------------------------------------------------------
#
# You can also use partial configuration and provide backend settings
# via command line or environment variables:
#
#   terraform init \
#     -backend-config="bucket=ai-control-plane-tfstate" \
#     -backend-config="key=aws/terraform.tfstate" \
#     -backend-config="region=us-east-1" \
#     -backend-config="dynamodb_table=terraform-locks"
#
# Or with environment variables:
#
#   export TF_VAR_backend_bucket="ai-control-plane-tfstate"
#   export TF_VAR_backend_key="aws/terraform.tfstate"
#
# ------------------------------------------------------------------------------
