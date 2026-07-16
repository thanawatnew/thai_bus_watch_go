variable "tenancy_ocid" {
  description = "OCID of the OCI tenancy"
  type        = string
}

variable "compartment_ocid" {
  description = "Compartment in which resources will be created"
  type        = string
}

variable "region" {
  description = "OCI home region for Always Free resources"
  type        = string
}

variable "ssh_public_key" {
  description = "Contents of the .pub SSH key; never enter the private key"
  type        = string
  sensitive   = true
}

variable "ssh_allowed_cidr" {
  description = "Public source allowed to use SSH; your public IP followed by /32 is safest"
  type        = string
  default     = "0.0.0.0/0"
}

variable "availability_domain_index" {
  description = "Zero-based availability domain index; try another if A1 capacity is unavailable"
  type        = number
  default     = 0

  validation {
    condition     = var.availability_domain_index >= 0
    error_message = "Availability domain index must be zero or greater."
  }
}

variable "repository_url" {
  description = "Public Git repository deployed by cloud-init"
  type        = string
  default     = "https://github.com/thanawatnew/thai_bus_watch_go.git"
}

variable "repository_branch" {
  description = "Git branch to deploy"
  type        = string
  default     = "main"
}
