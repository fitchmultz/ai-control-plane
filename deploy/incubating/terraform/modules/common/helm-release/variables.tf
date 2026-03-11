# -----------------------------------------------------------------------------
# Helm Release Module - Variables
# -----------------------------------------------------------------------------

# ------------------------------------------------------------------------------
# Chart Configuration
# ------------------------------------------------------------------------------

variable "chart_path" {
  description = "Path to the Helm chart directory (relative to module caller or absolute path)"
  type        = string
  default     = "../../deploy/helm/ai-control-plane"
}

variable "chart_version" {
  description = "Chart version to deploy. If not set, uses the version from Chart.yaml"
  type        = string
  default     = null
}

# ------------------------------------------------------------------------------
# Release Configuration
# ------------------------------------------------------------------------------

variable "release_name" {
  description = "Name of the Helm release"
  type        = string
  default     = "acp"
}

variable "namespace" {
  description = "Kubernetes namespace to deploy the Helm release into"
  type        = string
  default     = "acp"
}

variable "create_namespace" {
  description = "Whether to create the namespace if it doesn't exist"
  type        = bool
  default     = true
}

variable "description" {
  description = "Description of the Helm release"
  type        = string
  default     = "AI Control Plane - LiteLLM gateway with optional PostgreSQL"
}

# ------------------------------------------------------------------------------
# Values Configuration
# ------------------------------------------------------------------------------

variable "values_files" {
  description = "List of paths to values files to use for the Helm release (applied in order)"
  type        = list(string)
  default     = []
}

variable "values" {
  description = "Map of inline values to pass to the Helm chart. Deep merged with values_files"
  type        = any
  default     = {}
}

# ------------------------------------------------------------------------------
# Deployment Options
# ------------------------------------------------------------------------------

variable "timeout" {
  description = "Timeout in seconds for the Helm release operation"
  type        = number
  default     = 600
}

variable "atomic" {
  description = "If true, installation/upgrade rolls back on failure. Prevents partial deployments"
  type        = bool
  default     = true
}

variable "wait" {
  description = "If true, waits for all resources to be ready before marking release as successful"
  type        = bool
  default     = true
}

variable "wait_for_jobs" {
  description = "If true, waits for all Jobs to complete before marking release as successful"
  type        = bool
  default     = false
}

variable "cleanup_on_fail" {
  description = "If true, deletes newly created resources on failure during upgrade"
  type        = bool
  default     = false
}

variable "disable_openapi_validation" {
  description = "If true, skips OpenAPI schema validation of manifests"
  type        = bool
  default     = false
}

variable "disable_webhooks" {
  description = "If true, prevents Helm from running hooks"
  type        = bool
  default     = false
}

variable "force_update" {
  description = "If true, forces resource updates through delete/recreate when necessary"
  type        = bool
  default     = false
}

variable "recreate_pods" {
  description = "If true, performs pods restart for the resource if applicable"
  type        = bool
  default     = false
}

variable "replace" {
  description = "If true, reuses the given name even if that name is already used"
  type        = bool
  default     = false
}

variable "reuse_values" {
  description = "If true, reuses the last release's values and merges with new values"
  type        = bool
  default     = false
}

variable "reset_values" {
  description = "If true, resets values to the ones built into the chart"
  type        = bool
  default     = false
}

# ------------------------------------------------------------------------------
# Kubernetes Provider Configuration
# ------------------------------------------------------------------------------

variable "kubeconfig_path" {
  description = "Path to kubeconfig file. If not set, uses KUBE_CONFIG_PATHS env var or default kubeconfig"
  type        = string
  default     = null
}

variable "kubeconfig_context" {
  description = "Kubernetes context to use. If not set, uses current context"
  type        = string
  default     = null
}
