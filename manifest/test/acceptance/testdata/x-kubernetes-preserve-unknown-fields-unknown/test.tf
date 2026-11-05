# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

# id is generated during apply, so plan sees an unknown string next to a known one.
resource "terraform_data" "client" {}

resource "kubernetes_manifest" "test" {
  manifest = {
    apiVersion = var.group_version
    kind       = var.kind
    metadata = {
      name      = var.name
      namespace = var.namespace
    }
    spec = {
      defaultValues = {
        known     = "already"
        client_id = terraform_data.client.id
      }
    }
  }
}
