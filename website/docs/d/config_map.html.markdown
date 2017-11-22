---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_config_map"
sidebar_current: "docs-kubernetes-data-source-config-map"
description: |-
  ConfigMap holds configuration data for pods to consume.
---

# kubernetes_config_map

ConfigMap holds configuration data for pods to consume.

Read more at https://kubernetes.io/docs/tasks/configure-pod-container/configmap/

## Example Usage

```
data "kubernetes_config_map" "example" {
  metadata {
    name = "terraform-example"
  }
}

output "config_map_value" {
  value = "${data.kubernetes_config_map.example.data.foo}
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard config map's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata


## Nested Blocks

### `metadata`

#### Arguments

* `name` - (Required) Name of the config map to retrieve.
* `namespace` - (Optional) Name of the namespace in which config map was created.

#### Attributes

The following attributes are exported:

* `metadata.0.generation` - A sequence number representing a specific generation of the desired state.
* `metadata.0.resource_version` - An opaque value that represents the internal version of this config map that can be used by clients to determine when config map has changed. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency
* `metadata.0.self_link` - A URL representing this config map.
* `metadata.0.uid` - The unique in time and space value for this config map. More info: http://kubernetes.io/docs/user-guide/identifiers#uids

* `data` - The map of key values pairs stored in the config map.