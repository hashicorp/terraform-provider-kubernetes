# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test" {
  manifest = {
    // ...
  }

  wait {
    fields = {
      # Check the phase of a pod
      "status.phase" = "Running"

      # Check a container's status
      "status.containerStatuses[0].ready" = "true",

      # Check an ingress has an IP
      "status.loadBalancer.ingress[0].ip" = "^(\\d+(\\.|$)){4}"

      # Check the replica count of a Deployment
      "status.readyReplicas" = "2"
    }
  }

  timeouts {
    create = "10m"
    update = "10m"
    delete = "30s"
  }
}
