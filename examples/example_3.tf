# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

provider "kubernetes" {
  config_paths = [
    "/path/to/config_a.yaml",
    "/path/to/config_b.yaml"
  ]
}
