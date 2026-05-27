---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_config_map_v1"
description: |-
  This data source reads configuration data from a config map.
---

# <no value>

<no value>

<no value>


~> **Note:** All arguments including the config map data will be stored in the raw state as plain-text. [Read more about sensitive data in state](/docs/state/sensitive-data.html).

## Example Usage

```terraform
data "kubernetes_config_map_v1" "example" {
  metadata {
    name = "my-config"
  }
}
```
