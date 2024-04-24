# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_secret" "example" {
  metadata {
    name = "docker-cfg"
  }

  data = {
    ".dockerconfigjson" = "${file("${path.module}/.docker/config.json")}"
  }

  type = "kubernetes.io/dockerconfigjson"
}
