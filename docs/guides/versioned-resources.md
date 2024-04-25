---
layout: "kubernetes"
page_title: "Versioned resource names"
description: |-
  This guide explains the naming conventions for resources and data sources in the Kubernetes provider. 
---

# Versioned resource names 

This guide explains the naming conventions for resources and data sources in the Kubernetes provider. 


## Version suffixes

From provider version v2.7.0 onwards Terraform resources and data sources that cover the [standard set of Kubernetes APIs](https://kubernetes.io/docs/reference/kubernetes-api/) will be suffixed with their corresponding Kubernetes API version (e.g `v1`, `v2`, `v2beta1`). The existing resources in the provider will continue to be maintained as is. 


## Motivation 

We are doing this to make it easier to use and maintain the provider, and to promote long-term stability and backwards compatibility with resources in the Kubernetes API as they reach maturity, and as the provider sees wider adoption. 

Because Terraform does not support configurable schema versions for individual resources in the same way that the Kubernetes API does, the user sees a simpler unversioned schema for the Terraform resource. This is sometimes a good thing as the user is not burdened by Kubernetes API groups and versions, but it has caused confusion as the Kubernetes API evolves while the Terraform provider still has to support older versions of API resources. This also burdens the user with having to version pin the provider if they still rely upon a specific API version in their configuration. 

In the past we have tried to support multiple Kubernetes API versions using a single Terraform resource with varying degrees of success. The [kubernetes_horizontal_pod_autoscaler](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs/resources/horizontal_pod_autoscaler) supports multiple versions of the autoscaling API by having a schema that includes attributes from both the `v1` and `v2beta2` APIs and then looks which attributes have been set to determine the appropriate Kubernetes API version to use. The [kubernetes_mutating_webhook_configuration](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs/resources/mutating_webhook_configuration) and [kubernetes_validating_webhook_configuration](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs/resources/validating_webhook_configuration) resources use the discovery client to determine which version of the `admissionregistration` API the cluster supports. These approaches seem reasonable but lead to Terraform resource schemas where it is not obvious which attributes are actually supported by the target cluster, and creates an unsustainable maintenance burden as a resource has to be cobbled together by hand to support multiple API versions. 

Ultimately, we plan to completely automate the generation of Terraform resources to cover the core Kubernetes API. Having a set of versioned schemas that more closely matches the Kubernetes API definition is going to make this easier to achieve and will enable us to add built-in support for new API versions much faster. 


## What will happen to the resources without versions in the name?

These resources will continue to be supported and maintained as is through to v3.0.0 of the provider, at which point they will be marked as deprecated and then subsequently removed in v4.0.0.


## `v1` and above resources

Resources suffixed with a major version number are considered to have stable APIs that will not change. These resources will be supported by the provider so long as the API version continues to be supported by the Kubernetes API, and likely for some time after it is deprecated and removed as there is often a long tail of migration as users of the provider continue to support legacy infrastructure. 

While the API contract for these resources is assumed to be concrete, we will still accept changes to add additional attributes to these resources for configuring convenience features such as the `wait_for_rollout` attribute seen on resources such as `kubernetes_deployment`. Changes to these attributes should always be accompanied by deprecation warnings, state upgraders, and follow our typical [semantic versioning](https://www.terraform.io/docs/extend/best-practices/versioning.html#versioning-specification) scheme.


## `beta` resources

We will continue to bring support for API resources which reach `beta` however it is expected that the API contract for these resources can still change and so they should be used with some caution. When a `beta` API changes we will provide a state upgrader for the resource where possible. Refer to the Kubernetes API documentation on the use of [beta resources](https://kubernetes.io/docs/reference/using-api/#api-versioning).


## `alpha` resources

We will continue our policy of not building support for `alpha` versioned resources into the provider. Please use the [kubernetes_manifest](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs/resources/manifest) resource to manage those resources. 


## How can I move a resource without a version to its versioned resource name?

The simplest, non-destructive way to do this is to modify the name of the resource to include the version suffix. Then remove the old resource from state and import the resource under the versioned resource like so:

```
terraform state rm kubernetes_config_map.example
terraform import kubernetes_config_map_v1.example default/example
```

Then run `terraform plan` to confirm that the import was successful. **NOTE: Do not run the plan after renaming the resource in the configuration until after the above steps have been carried out.** 

You can also skip this and just allow Terraform to destroy and recreate the resource, but this is not recommended for resources like `kubernetes_service` and `kubernetes_deployment`. 
