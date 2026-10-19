# Configuration using provider functions must include required_providers configuration.
terraform {
  required_providers {
    kubernetes = {
      source = "hashicorp/kubernetes"
      # Setting the provider version is a strongly recommended practice
      # version = "..."
    }
  }
  # Provider functions require Terraform 1.8 and later.
  required_version = ">= 1.8.0"
}

locals {
  env = provider::kubernetes::env_from_map({
    GREETING = "Hello from the environment"
    NAME     = "Kubernetes"
  })
}

# Use the result to populate the env block of a container in a manifest.
output "env" {
  value = local.env
}
