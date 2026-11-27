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
# Namespaces and CustomResourceDefinitions come first, in their own resource:
# the other objects need them, and Terraform applies the instances of one
# resource in no particular order. A local value decodes the file once; in
# for_each, Terraform calls the function again for every object.
locals {
  operator = provider::kubernetes::manifest_decode_map(file("${path.module}/operator.yaml"))
  first    = { for key, object in local.operator : key => object if contains(["Namespace", "CustomResourceDefinition"], object.kind) }
}

resource "kubernetes_manifest_v2" "operator_first" {
  for_each = local.first
  manifest = each.value
}

resource "kubernetes_manifest_v2" "operator" {
  for_each   = { for key, object in local.operator : key => object if !contains(keys(local.first), key) }
  manifest   = each.value
  depends_on = [kubernetes_manifest_v2.operator_first]
}

# A templated manifest.
resource "kubernetes_manifest_v2" "web" {
  manifest = provider::kubernetes::manifest_decode(templatefile("${path.module}/deployment.yaml.tftpl", {
    image = var.image
  }))
}
