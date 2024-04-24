# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

provider "kubernetes" {
  ignore_annotations = [
    "cni\\.projectcalico\\.org\\/podIP",
    "cni\\.projectcalico\\.org\\/podIPs",
  ]
}
