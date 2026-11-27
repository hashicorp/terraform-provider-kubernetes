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

# Decode in a local value: in for_each, Terraform calls the function again
# for every resource instance.
locals {
  # One resource instance per object of a multi-document file.
  app = provider::kubernetes::manifest_decode_map(file("${path.module}/app.yaml"))
  # Every YAML file of a directory.
  all = provider::kubernetes::manifest_decode_map([
    for name in fileset("${path.module}/manifests", "**/*.yaml") : file("${path.module}/manifests/${name}")
  ]...)
}

resource "kubernetes_manifest_v2" "app" {
  for_each = local.app
  manifest = each.value
}

resource "kubernetes_manifest_v2" "all" {
  for_each = local.all
  manifest = each.value
}
