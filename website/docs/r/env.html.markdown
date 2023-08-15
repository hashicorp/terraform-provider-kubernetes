---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_env"
description: |-
  This resource provides a way to manage environment variables in resources that were created outside of Terraform.
---

# kubernetes_env

This resource provides a way to manage environment variables in resources that were created outside of Terraform. This resource provides functionality similar to the `kubectl set env` command. 

## Example Usage

```hcl
resource "kubernetes_env" "example" {
  container = "nginx"
  metadata {
    name = "nginx-deployment"
  }

  api_version = "apps/v1"
  kind        = "Deployment"

  env {
    name  = "NGINX_HOST"
    value = "google.com"
  }

  env {
    name  = "NGINX_PORT"
    value = "90"
  }
}
```

## Argument Reference

The following arguments are supported:

* `api_version` - (Required) The apiVersion of the resource to add environment variables to.
* `kind` - (Required) The kind of the resource to add environment variables to.
* `metadata` - (Required) Standard metadata of the resource to add environment variables to. 
* `container` - (Optional) Name of the container for which we are updating the environment variables.
* `init_container` - (Optional) Name of the initContainer for which we are updating the environment variables.
* `env` - (Required) Value block with custom values used to represent environment variables
* `force` - (Optional) Force management of environment variables if there is a conflict.
* `field_manager` - (Optional) The name of the [field manager](https://kubernetes.io/docs/reference/using-api/server-side-apply/#field-management). Defaults to `Terraform`.

## Nested Blocks

### `metadata`

#### Arguments

* `name` - (Required) Name of the resource to add environment variables to.
* `namespace` - (Optional) Namespace of the resource to add environment variables to.

### `env`

#### Arguments

* `name` - (Required) Name of the environment variable. Must be a C_IDENTIFIER
* `value` - (Optional) Variable references $(VAR_NAME) are expanded using the previous defined environment variables in the container and any service environment variables. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Defaults to "".
* `value_from` - (Optional) Source for the environment variable's value

### `value_from`

#### Arguments

* `config_map_key_ref` - (Optional) Selects a key of a ConfigMap.
* `field_ref` - (Optional) Selects a field of the pod: supports metadata.name, metadata.namespace, metadata.labels, metadata.annotations, spec.nodeName, spec.serviceAccountName, status.podIP.
* `resource_field_ref` - (Optional) Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.ephemeral-storage) are currently supported.
* `secret_key_ref` - (Optional) Selects a key of a secret in the pod's namespace.

### `config_map_key_ref`

#### Arguments

* `key` - (Optional) The key to select.
* `name` - (Optional) Name of the referent. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
* `optional` - (Optional) Specify whether the Secret or its key must be defined

### `field_ref`

#### Arguments

* `api_version` - (Optional) Version of the schema the FieldPath is written in terms of, defaults to "v1".
* `field_path` - (Optional) Path of the field to select in the specified API version

### `resource_field_ref`

#### Arguments

* `container_name` - (Optional) The name of the container
* `resource` - (Required) Resource to select
* `divisor` - (Optional) Specifies the output format of the exposed resources, defaults to "1".

### `secret_key_ref`

#### Arguments

* `key` - (Optional) The key of the secret to select from. Must be a valid secret key.
* `name` - (Optional) Name of the referent. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
* `optional` - (Optional) Specify whether the Secret or its key must be defined


## Import

This resource does not support the `import` command. As this resource operates on Kubernetes resources that already exist, creating the resource is equivalent to importing it. 
