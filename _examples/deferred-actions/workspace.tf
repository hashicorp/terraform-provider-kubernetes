# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_namespace_v1" "demo_ns" {
  metadata {
    name = "demo-ns"
  }
}

resource "kubernetes_manifest" "demo_workspace" {
  manifest = {
    apiVersion = "app.terraform.io/v1alpha2"
    kind       = kubernetes_manifest.crd_workspaces.object.spec.names.kind
    metadata = {
      name      = "deferred-demo"
      namespace = kubernetes_namespace_v1.demo_ns.id
    }
    spec = {
      name         = "demo-ws"
      organization = "demo-org"
      token = {
        secretKeyRef = {
          name = "demo-token"
          key  = "token"
        }
      }
    }
  }
}