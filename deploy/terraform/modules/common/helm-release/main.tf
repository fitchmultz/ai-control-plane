# -----------------------------------------------------------------------------
# Helm Release Module - Main Configuration
# -----------------------------------------------------------------------------
# This module deploys the AI Control Plane Helm chart using the Helm provider.
# The Helm provider must be configured by the caller.
# -----------------------------------------------------------------------------

terraform {
  required_version = ">= 1.0"

  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = ">= 2.12.0"
    }
  }
}

# -----------------------------------------------------------------------------
# Helm Release Resource
# -----------------------------------------------------------------------------

resource "helm_release" "this" {
  name        = var.release_name
  namespace   = var.namespace
  description = var.description

  # Chart source - local path to the AI Control Plane chart
  chart = var.chart_path

  # Version constraint (null uses Chart.yaml version)
  version = var.chart_version

  # Namespace handling
  create_namespace = var.create_namespace

  # Values files (applied in order, later files override earlier ones)
  dynamic "values" {
    for_each = var.values_files
    content {
      value = file(values.value)
    }
  }

  # Inline values (deep merged with values files)
  values = [
    yamlencode(var.values)
  ]

  # -----------------------------------------------------------------------------
  # Deployment Safety Options
  # -----------------------------------------------------------------------------

  # Timeout for the operation (seconds)
  timeout = var.timeout

  # Atomic deployment - rolls back on failure
  atomic = var.atomic

  # Wait for all resources to be ready
  wait = var.wait

  # Wait for jobs to complete
  wait_for_jobs = var.wait_for_jobs

  # Cleanup resources on failed upgrade
  cleanup_on_fail = var.cleanup_on_fail

  # Validation and hook options
  disable_openapi_validation = var.disable_openapi_validation
  disable_webhooks           = var.disable_webhooks

  # Update behavior
  force_update  = var.force_update
  recreate_pods = var.recreate_pods
  replace       = var.replace
  reuse_values  = var.reuse_values
  reset_values  = var.reset_values
}
