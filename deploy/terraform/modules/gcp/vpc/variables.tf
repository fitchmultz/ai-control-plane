#------------------------------------------------------------------------------
# GCP VPC Module - Variables
#------------------------------------------------------------------------------

variable "network_name" {
  description = "Name of the VPC network"
  type        = string
  default     = "ai-control-plane-vpc"
}

variable "project_id" {
  description = "GCP project ID where resources will be created"
  type        = string
}

variable "region" {
  description = "GCP region for regional resources"
  type        = string
  default     = "us-central1"
}

variable "subnets" {
  description = "List of subnet configurations"
  type = list(object({
    name                     = string
    ip_cidr_range            = string
    region                   = optional(string)
    private_ip_google_access = optional(bool)
    secondary_ip_ranges = optional(list(object({
      range_name    = string
      ip_cidr_range = string
    })))
  }))
  default = [
    {
      name                     = "gke-subnet"
      ip_cidr_range            = "10.0.0.0/24"
      private_ip_google_access = true
      secondary_ip_ranges = [
        {
          range_name    = "pods"
          ip_cidr_range = "10.4.0.0/14"
        },
        {
          range_name    = "services"
          ip_cidr_range = "10.0.32.0/20"
        }
      ]
    }
  ]
}

variable "create_nat_gateway" {
  description = "Whether to create a Cloud NAT gateway for private subnet egress"
  type        = bool
  default     = true
}

variable "router_name" {
  description = "Name of the Cloud Router (used for NAT)"
  type        = string
  default     = "ai-control-plane-router"
}

variable "labels" {
  description = "Labels to apply to all resources"
  type        = map(string)
  default = {
    environment = "production"
    project     = "ai-control-plane"
  }
}
