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
  manifest = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata = {
      name = "example"
    }
    data = {
      EXAMPLE = "example"
    }
  }
}

output "example_output" {
  value = provider::kubernetes::manifest_encode(local.manifest)
}
