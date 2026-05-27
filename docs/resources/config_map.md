---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_config_map"
description: |-
  The resource provides mechanisms to inject containers with configuration data while keeping containers agnostic of Kubernetes.
---

# <no value>

The resource provides mechanisms to inject containers with configuration data while keeping containers agnostic of Kubernetes. Config Map can be used to store fine-grained information like individual properties or coarse-grained information like entire config files or JSON blobs.

<no value>

## Example Usage

```terraform
resource "kubernetes_config_map" "example" {
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
$ terraform import kubernetes_config_map.example default/my-config
```
