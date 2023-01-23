---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_config_map_v1_data"
description: |-
  This resource allows Terraform to manage the data for a ConfigMap that already exists
---

# kubernetes_config_map_v1_data

This resource allows Terraform to manage data within a pre-existing ConfigMap. This resource uses [field management](https://kubernetes.io/docs/reference/using-api/server-side-apply/#field-management) and [server-side apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/) to manage only the data that is defined in the Terraform configuration. Existing data not specified in the configuration will be ignored. If data specified in the config and is already managed by another client it will cause a conflict which can be overridden by setting `force` to true. 


## Example Usage

```hcl
resource "kubernetes_config_map_v1_data" "example" {
  metadata {
    name = "my-config"
  }
  data = {
    "owner" = "myteam"
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard metadata of the ConfigMap. 
* `data` - (Required) A map of data to apply to the ConfigMap.
* `force` - (Optional) Force management of the configured data if there is a conflict.
* `field_manager` - (Optional) The name of the [field manager](https://kubernetes.io/docs/reference/using-api/server-side-apply/#field-management). Defaults to `Terraform`.

## Nested Blocks

### `metadata`

#### Arguments

* `name` - (Required) Name of the ConfigMap.
* `namespace` - (Optional) Namespace of the ConfigMap.

## Import

This resource does not support the `import` command. As this resource operates on Kubernetes resources that already exist, creating the resource is equivalent to importing it. 


