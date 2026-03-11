# Input Variables for Kubernetes Secrets Module

variable "namespace" {
  description = "Kubernetes namespace where the secret will be created"
  type        = string
}

variable "secret_name" {
  description = "Name of the Kubernetes secret"
  type        = string
  default     = "ai-control-plane-secrets"
}

variable "secret_data" {
  description = "Map of secret key-value pairs. Values should be base64-encoded if using 'data' field, or plain text if using 'stringData' (handled internally)"
  type        = map(string)
  sensitive   = true
  default     = {}
}

variable "type" {
  description = "Type of Kubernetes secret (e.g., Opaque, kubernetes.io/tls, kubernetes.io/dockerconfigjson, kubernetes.io/basic-auth)"
  type        = string
  default     = "Opaque"
}

variable "labels" {
  description = "Labels to apply to the secret"
  type        = map(string)
  default     = {}
}

variable "annotations" {
  description = "Annotations to apply to the secret"
  type        = map(string)
  default     = {}
}
