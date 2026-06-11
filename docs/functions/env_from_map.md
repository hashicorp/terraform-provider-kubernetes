---
subcategory: ""
page_title: "env_from_map function"
description: |-
  Convert a map of strings into a list of name/value objects
---

# function: env_from_map

Given a map of strings, returns a list of objects with `name` and `value` attributes, sorted by key. This is useful for populating the `env` field of a container in a `kubernetes_manifest` resource without repeating the `name`/`value` boilerplate for every variable.

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

locals {
  env = provider::kubernetes::env_from_map({
    GREETING = "Hello from the environment"
    NAME     = "Kubernetes"
  })
}

# Use the result to populate the env block of a container in a manifest.
output "env" {
  value = local.env
}
```

## Signature

```text
env_from_map(env map of string) list of object
```

## Arguments

1. `env` (Map of String) A map of environment variable names to values

## Return Type

The `list of object` returned from `env_from_map` contains one object per map entry, each with a `name` and `value` attribute, ordered by `name`.
