# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_deployment_v1" "this" {
  // omit the resource config
  lifecycle {
    ignore_changes = [
      spec[0].template[0].metadata[0].annotations["kubectl.kubernetes.io/restartedAt"],
    ]
  }
}
