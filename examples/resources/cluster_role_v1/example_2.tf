# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_cluster_role_v1" "example" {
  metadata {
    name = "terraform-example"
  }

  aggregation_rule {
    cluster_role_selectors {
      match_labels = {
        foo = "bar"
      }

      match_expressions {
        key      = "environment"
        operator = "In"
        values   = ["non-exists-12345"]
      }
    }
  }
}
