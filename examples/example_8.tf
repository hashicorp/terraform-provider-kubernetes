# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

provider "kubernetes" {
  ignore_annotations = [
    "^service\\.beta\\.kubernetes\\.io\\/aws-load-balancer.*",
  ]
}
