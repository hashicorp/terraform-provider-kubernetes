---
subcategory: "storage/v1"
page_title: "Kubernetes: kubernetes_storage_class"
description: |-
  Storage class is the foundation of dynamic provisioning, allowing cluster administrators to define abstractions for the underlying storage platform.
---

# <no value> 

Storage class is the foundation of dynamic provisioning, allowing cluster administrators to define abstractions for the underlying storage platform.

Read more at https://kubernetes.io/blog/2017/03/dynamic-provisioning-and-storage-classes-kubernetes/

<no value> 

## Example Usage

```terraform
resource "kubernetes_storage_class" "example" {
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

kubernetes_storage_class can be imported using its name, e.g.

```
$ terraform import kubernetes_storage_class.example terraform-example
```
