---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_config_map"
description: |-
  This data source reads configuration data from a config map.
---

# kubernetes_config map

Config Maps are key-value pairs containing configuration data. The Config Map data source provides a mechanism for extracting these key-value pairs.

~> **Note:** All arguments including the config map data will be stored in the raw state as plain-text. [Read more about sensitive data in state](/docs/state/sensitive-data.html).

## Example Usage

```hcl
data "kubernetes_config_map" "example" {
  metadata {
    name = "my-config"
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard config map's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)

## Nested Blocks

### `metadata`

#### Arguments

* `name` - (Required) Name of the config map, must be unique. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
* `namespace` - (Optional) Namespace defines the space within which name of the config map must be unique.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this config map that can be used by clients to determine when config map has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this config map. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

## Attribute Reference

* `data` - A map of the config map data.
* `binary_data` - A map of preserved non-UTF8 data. For more info see [Kubernetes API reference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#configmap-v1-core).
