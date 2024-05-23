# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test_cr" {

  manifest = {
    apiVersion = "${var.group}/${var.cr_version}"
    kind       = var.kind
    metadata = {
      name      = var.name
      namespace = var.namespace
    }
    spec = {
      teamId = "test"
      volume = {
        size = "1Gi"
      }
      numberOfInstances = 2
      users = {
        mike = [
          "superuser",
          "createdb"
        ]
        foo_user = [
          "superuser"
        ]
        bar_user = []
      }
      databases = {
        foo = "devdb"
      }
      postgresql = {
        version = "12"
      }
    }
  }
}
