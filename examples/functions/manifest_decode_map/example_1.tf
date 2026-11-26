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

# One resource instance per object of a multi-document file.
resource "kubernetes_manifest_v2" "app" {
  for_each = provider::kubernetes::manifest_decode_map(file("${path.module}/app.yaml"))
  manifest = each.value
}

# Every YAML file of a directory.
resource "kubernetes_manifest_v2" "all" {
  for_each = provider::kubernetes::manifest_decode_map([
    for name in fileset("${path.module}/manifests", "**/*.yaml") : file("${path.module}/manifests/${name}")
  ]...)
  manifest = each.value
}
