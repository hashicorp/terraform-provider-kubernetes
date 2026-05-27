---
subcategory: "storage/v1"
page_title: "Kubernetes: kubernetes_storage_class_v1"
description: |-
  Storage class is the foundation of dynamic provisioning, allowing cluster administrators to define abstractions for the underlying storage platform.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_storage_class_v1" "example" {
  metadata {
    name = "terraform-example"
  }
  storage_provisioner = "kubernetes.io/gce-pd"
  reclaim_policy      = "Retain"
  parameters = {
    type = "pd-standard"
  }
  mount_options = ["file_mode=0700", "dir_mode=0777", "mfsymlinks", "uid=1000", "gid=1000", "nobrl", "cache=none"]
}
```

## Import

kubernetes_storage_class_v1 can be imported using its name, e.g.

```
$ terraform import kubernetes_storage_class_v1.example terraform-example
```
