---
subcategory: ""
page_title: "manifest_encode function"
description: |-
  Decode a Kubernetes YAML manifest containing multiple resources
---

# function: manifest_encode

Given an object representation of a Kubernetes manifest, will encode and return a YAML string for that resource.

## Example Usage

```terraform
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# Configuration using provider functions must include required_providers configuration.
terraform {
  required_providers {
    kubernetes = {
      source = "hashicorp/kubernetes"
      # Setting the provider version is a strongly recommended practice
      # version = "..."
    }
  }
  # Provider functions require Terraform 1.8 and later.
  required_version = ">= 1.8.0"
}

locals {
  manifest = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata = {
      name = "example"
    }
    data = {
      EXAMPLE = "example"
    }
  }
}

output "example_output" {
  value = provider::kubernetes::manifest_encode(local.manifest)
}
```

## Signature

```text
manifest_encode(manifest object) string
```

## Arguments

1. `manifest` (String) The object representation of a Kubernetes manifest

## Return Type

The `string` returned from `manifest_encode` will contain the YAML encoded Kubernetes manifest.
