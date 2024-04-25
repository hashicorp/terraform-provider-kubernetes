# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_pod" "with_node_affinity" {
  metadata {
    name = "with-node-affinity"
  }

  spec {
    affinity {
      node_affinity {
        required_during_scheduling_ignored_during_execution {
          node_selector_term {
            match_expressions {
              key      = "kubernetes.io/e2e-az-name"
              operator = "In"
              values   = ["e2e-az1", "e2e-az2"]
            }
          }
        }

        preferred_during_scheduling_ignored_during_execution {
          weight = 1

          preference {
            match_expressions {
              key      = "another-node-label-key"
              operator = "In"
              values   = ["another-node-label-value"]
            }
          }
        }
      }
    }

    container {
      name  = "with-node-affinity"
      image = "k8s.gcr.io/pause:2.0"
    }
  }
}
