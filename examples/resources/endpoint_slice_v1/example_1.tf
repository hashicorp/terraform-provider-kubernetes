# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_endpoint_slice_v1" "test" {
  metadata {
    name = "test"
  }

  endpoint {
    condition {
      ready = true
    }
    addresses = ["129.144.50.56"]
  }

  port {
    port = "9000"
    name = "first"
  }

  address_type = "IPv4"
}
