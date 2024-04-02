---
page_title: "manifest_decode_multi function"
subcategory: ""
description: |-
  Decode a Kubernetes YAML manifest containing multiple resources
---

# function: manifest_decode_multi

Given a YAML text containing a Kubernetes manifest with multiple resources, will decode the manifest and return a tuple of object representations for each resource.

## Example Usage

```hcl
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

output "example_output" {
  value = provider::kubernetes::manifest_decode_multi(file("manifest.yaml"))
}
```

## Signature

```text
manifest_decode_multi(manifest string) tuple
```

## Arguments


1. `manifest` (String) The YAML text for a Kubernetes manifest containing multiple resources


## Return Type

The `tuple` returned from `manifest_decode_multi` will contain dynamic objects that mirror the structure of the resources in YAML manifest supplied. 
