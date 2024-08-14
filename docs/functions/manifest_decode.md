---
subcategory: ""
page_title: "manifest_decode function"
description: |-
  Decode a Kubernetes YAML manifest 
---

# function: manifest_decode

Given a YAML text containing a Kubernetes manifest, will decode and return an object representation of that resource.

## Example Usage

```terraform
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
  value = provider::kubernetes::manifest_decode(file("manifest.yaml"))
}
```

## Signature

```text
manifest_decode(manifest string) object
```

## Arguments

1. `manifest` (String) The YAML text for a Kubernetes manifest

## Return Type

The `object` returned from `manifest_decode` is dynamic and will mirror the structure of the YAML manifest supplied.
