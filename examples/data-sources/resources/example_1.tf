# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "kubernetes_resources" "example" {
  api_version    = "v1"
  kind           = "Namespace"
  field_selector = "metadata.name!=kube-system"
}

output "test" {
  value = length(data.kubernetes_resources.example.objects)
}
