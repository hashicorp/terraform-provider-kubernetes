---
subcategory: ""
page_title: "manifest_decode_map function"
description: |-
  Decode Kubernetes YAML manifests into a map keyed by object
---

# function: manifest_decode_map

Decodes one or more YAML texts, each holding any number of Kubernetes objects, and returns the objects in a map. Use it with `for_each` to manage every object of a file or a directory as its own resource instance.

Each key names one object: `Kind[.group]/[namespace/]name`, for example `Deployment.apps/default/web`, `Namespace/prod` or `Widget.example.com/prod/web`. The namespace is part of the key only when the manifest sets it.

- The key does not depend on the position of an object or on the text it comes from. Reordering documents, or moving one to another file passed to the same call, keeps the resource address of every object.
- The key does not contain the API version, so moving an object to a newer version of its kind keeps its address.
- A list is replaced by its items: an object of a kind that ends in `List`, with `items` and without `metadata.name`. The items of a typed list, such as a `PodList`, take their kind and API version from the list when they have none. Empty documents are skipped.
- An object that appears twice, or that has no `apiVersion`, `kind` or `metadata.name`, is an error. So are two objects that differ only in that one has no namespace and the other is in `default`: for a namespaced kind they are the same object.
- Documents are split at `---` lines only, as `kubectl` splits them, and integers keep their exact value. A key that appears twice in one mapping is an error, and so is a key that overrides one merged with `<<`, which `kubectl` accepts.
- Call it in a local value and use the local value in `for_each`: in `for_each` itself, Terraform calls the function again for every resource instance.

Renaming an object, or adding or removing `metadata.namespace` in its manifest, changes its key. Terraform then plans to destroy one resource instance and create another. Add a `moved` block from the old key to the new one when the object itself stays the same.

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

# Decode in a local value: in for_each, Terraform calls the function again
# for every resource instance.
locals {
  # One resource instance per object of a multi-document file.
  app = provider::kubernetes::manifest_decode_map(file("${path.module}/app.yaml"))
  # Every YAML file of a directory.
  all = provider::kubernetes::manifest_decode_map([
    for name in fileset("${path.module}/manifests", "**/*.yaml") : file("${path.module}/manifests/${name}")
  ]...)
}

resource "kubernetes_manifest_v2" "app" {
  for_each = local.app
  manifest = each.value
}

resource "kubernetes_manifest_v2" "all" {
  for_each = local.all
  manifest = each.value
}
```

## Signature

```text
manifest_decode_map(manifests string...) object
```

## Arguments

1. `manifests` (Variadic, String) YAML texts of Kubernetes manifests

## Return Type

The `object` returned from `manifest_decode_map` has one attribute per Kubernetes object, named by the key described above. Each value mirrors the structure of the object in the YAML supplied, as [`manifest_decode`](manifest_decode.md) returns it.
