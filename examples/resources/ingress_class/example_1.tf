# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_ingress_class" "example" {
  metadata {
    name = "example"
  }

  spec {
    controller = "example.com/ingress-controller"
    parameters {
      api_group = "k8s.example.com"
      kind      = "IngressParameters"
      name      = "external-lb"
    }
  }
}
