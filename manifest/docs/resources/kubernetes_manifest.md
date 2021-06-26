---
page_title: "kubernetes_manifest Resource - terraform-provider-kubernetes-alpha"
subcategory: ""
description: |-
  A Kubernetes resource described in a manifest.
---

# Resource `kubernetes_manifest`

Represents one Kubernetes resource as described in the `manifest` attribute. The manifest value is the HCL transcription of a regular Kubernetes YAML manifest. To transcribe an existing manifest from YAML to HCL, we recommend using the Terrafrom built-in function [`yamldecode()`](https://www.terraform.io/docs/configuration/functions/yamldecode.html) or better yet [this purpose-built tool](https://github.com/jrhouston/tfk8s).

Once applied, the `object` attribute reflects the state of the resource as returned by the Kubernetes API, including all default values.


## Schema

### Required

- **manifest** (Dynamic, Required) A Kubernetes manifest describing the desired state of the resource in HCL format.

### Optional

- **object** (Dynamic, Optional) The resulting resource state, as returned by the API server after applying the desired state from `manifest`.
- **wait_for** (Object, Optional) (see [below for nested schema](#nestedatt--wait_for))

<a id="nestedatt--wait_for"></a>
### Nested Schema for `wait_for`

- **fields** (Map of String)


