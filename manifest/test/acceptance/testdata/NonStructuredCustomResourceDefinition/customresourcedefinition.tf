# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


resource "kubernetes_manifest" "test" {

  manifest = {
    apiVersion = "apiextensions.k8s.io/v1beta1"
    kind       = "CustomResourceDefinition"
    metadata = {
      name = "${var.plural}.${var.group}"
    }
    spec = {
      preserveUnknownFields = true
      group                 = var.group
      names = {
        kind   = var.kind
        plural = var.plural
      }
      scope = "Namespaced"
      versions = [{
        name    = var.cr_version
        served  = true
        storage = true
      }]
    }
  }
}
