# Input Variables for Kubernetes Namespace Module

variable "name" {
  description = "Name of the Kubernetes namespace"
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9]([-a-z0-9]*[a-z0-9])?$", var.name))
    error_message = "Namespace name must be lowercase alphanumeric characters or '-', and must start and end with an alphanumeric character (e.g., 'my-namespace', or 'myname')."
  }
}

variable "labels" {
  description = "Labels to apply to the namespace"
  type        = map(string)
  default     = {}
}

variable "annotations" {
  description = "Annotations to apply to the namespace"
  type        = map(string)
  default     = {}
}
