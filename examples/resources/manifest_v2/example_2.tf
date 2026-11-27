terraform {
  required_providers {
    kubernetes = {
      source = "hashicorp/kubernetes"
    }
  }
  # Provider functions require Terraform 1.8 and later.
  required_version = ">= 1.8.0"
}

# One object from a YAML file.
resource "kubernetes_manifest_v2" "issuer" {
  manifest = provider::kubernetes::manifest_decode(file("${path.module}/issuer.yaml"))
}

# Every object of a multi-document YAML file, one resource instance each.
# The file can hold a Namespace, CustomResourceDefinitions and custom
# resources together. Split them when the order matters: see "Objects that
# depend on each other". A local value decodes the file once; in for_each,
# Terraform calls the function again for every object.
locals {
  operator = provider::kubernetes::manifest_decode_map(file("${path.module}/operator.yaml"))
}

resource "kubernetes_manifest_v2" "operator" {
  for_each = local.operator
  manifest = each.value
}

# A templated manifest.
resource "kubernetes_manifest_v2" "web" {
  manifest = provider::kubernetes::manifest_decode(templatefile("${path.module}/deployment.yaml.tftpl", {
    image = var.image
  }))
}
