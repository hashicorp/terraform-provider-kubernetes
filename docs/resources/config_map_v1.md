---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_config_map_v1"
description: |-
  The resource provides mechanisms to inject containers with configuration data while keeping containers agnostic of Kubernetes.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_config_map_v1" "example" {
  metadata {
    name = "my-config"
  }

  data = {
    api_host             = "myhost:443"
    db_host              = "dbhost:5432"
    "my_config_file.yml" = "${file("${path.module}/my_config_file.yml")}"
  }

  binary_data = {
    "my_payload.bin" = "${filebase64("${path.module}/my_payload.bin")}"
  }
}
```

## Import

Config Map can be imported using its namespace and name, e.g.

```
$ terraform import kubernetes_config_map_v1.example default/my-config
```
