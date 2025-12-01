---
subcategory: ""
page_title: "Kubernetes: Upgrade Guide for Kubernetes Provider v3.0.0"
description: |-
  This guide covers the changes introduced in v3.0.0 of the Kubernetes provider and what you may need to do to upgrade your configuration.
---

# Upgrading to v3.0.0 of the Kubernetes provider

This guide covers the changes introduced in v2.0.0 of the Kubernetes provider and what you may need to do to upgrade your configuration.

Use `terraform init` to install version 3 of the provider. Then run `terraform plan` to determine if the upgrade will affect any existing resources. Some resources will have updated defaults and may be modified as a result. Update your config files to match the existing resource settings Then run `terraform plan` again to ensure no resource updates will be applied.

NOTE: Even if there are no resource updates to apply, you may need to run `terraform refresh` to update your state to the newest version.

## Installing and testing this update

The `required_providers` block can be used to move between version 2.x and version 3.x of the Kubernetes provider, for testing purposes. Please note that this is only possible using `terraform plan`. Once you run `terraform apply` or `terraform refresh`, the changes to Terraform State become permanent, and rolling back is no longer an option. It may be possible to roll back the State by making a copy of `.terraform.tfstate` before running `apply` or `refresh`, but this configuration is unsupported.

### Using required_providers to test the update

The version of the Kubernetes provider can be controlled using the `required_providers` block:

```hcl
terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 3.0"
    }
  }
}
```

When the above code is in place, run `terraform init` to upgrade the provider version.

```
$ terraform init -upgrade
```

Then run `terraform plan` to see what changes will be applied. 

```
$ export KUBE_CONFIG_PATH=$KUBECONFIG
$ terraform plan
```

Using the output from `terraform plan`, you can make modifications to your existing Terraform config, to avoid any unwanted resource changes. 

### Rolling back to version 2.x.x.

If you've run the above upgrade and plan, but you don't want to proceed with the 3.0 upgrade, you can roll back using the following steps. NOTE: this will only work if you haven't run `terraform apply` or `terraform refresh` while testing version 3 of the provider.

Set the provider version back to 2.x.x.

```
terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 2.38.0"
    }
  }
}
```

Then run `terraform init -upgrade` to install the old provider version.

```
$ terraform init -upgrade

Initializing the backend...

Initializing provider plugins...
- Finding hashicorp/kubernetes versions matching "2.38.0"...
- Installing hashicorp/kubernetes v2.38.0...
- Installed hashicorp/kubernetes v2.38.0 (signed by HashiCorp)
```

The provider is now downgraded.

## Changes in v3.0.0

## Dropping support for Terraform versions before `v1.0.0`

This release upgrades the Terraform protocol version used by the provider to the [latest version](https://developer.hashicorp.com/terraform/plugin/terraform-plugin-protocol#protocol-version-6). This protocol version only supports Terraform versions from `v1.0.0`. Configurations dependent on Terraform versions before `v1.0.0` should pin this provider to `v2.x.x`. 

### Changes to Kubernetes credentials for `kubernetes_manifest` resource

We have made changes to the way the `kubernetes_manifest` resolves credentials to make it consistent with the other resources. In `v2.x.x` this resource would allow environment variables such as `KUBE_CONFIG_PATH` to override values present in the configuration file. This was inconsistent with how all other resources resolve environent variables, which is to use the value from the environment if no value is specified in the configuration.  

### Deprecation of non-versioned resource names

This version of the provider marks all resources which do not have a version suffix as deprecated. A table of unversioned resources and the version suffixed resource that should be used is below.

| Unversioned resource                          | Version Suffixed Resource                               |
|-----------------------------------------------|---------------------------------------------------------|
| kubernetes_namespace                          | kubernetes_namespace_v1                                 |
| kubernetes_service                            | kubernetes_service_v1                                   |
| kubernetes_service_account                    | kubernetes_service_account_v1                           |
| kubernetes_default_service_account            | kubernetes_default_service_account_v1                   |
| kubernetes_config_map                         | kubernetes_config_map_v1                                |
| kubernetes_secret                             | kubernetes_secret_v1                                    |
| kubernetes_pod                                | kubernetes_pod_v1                                       |
| kubernetes_endpoints                          | kubernetes_endpoints_v1                                 |
| kubernetes_limit_range                        | kubernetes_limit_range_v1                               |
| kubernetes_persistent_volume                  | kubernetes_persistent_volume_v1                         |
| kubernetes_persistent_volume_claim            | kubernetes_persistent_volume_claim_v1                   |
| kubernetes_replication_controller             | kubernetes_replication_controller_v1                    |
| kubernetes_resource_quota                     | kubernetes_resource_quota_v1                            |
| kubernetes_api_service                        | kubernetes_api_service_v1                               |
| kubernetes_deployment                         | kubernetes_deployment_v1                                |
| kubernetes_daemonset                          | kubernetes_daemon_set_v1                                |
| kubernetes_stateful_set                       | kubernetes_stateful_set_v1                              |
| kubernetes_job                                | kubernetes_job_v1                                       |
| kubernetes_cron_job                           | kubernetes_cron_job_v1                                  |
| kubernetes_horizontal_pod_autoscaler          | kubernetes_horizontal_pod_autoscaler_v1 or kubernetes_horizontal_pod_autoscaler_v2 |
| kubernetes_certificate_signing_request        | kubernetes_certificate_signing_request_v1               |
| kubernetes_role                               | kubernetes_role_v1                                      |
| kubernetes_role_binding                       | kubernetes_role_binding_v1                              |
| kubernetes_cluster_role                       | kubernetes_cluster_role_v1                              |
| kubernetes_cluster_role_binding               | kubernetes_cluster_role_binding_v1                      |
| kubernetes_ingress                            | kubernetes_ingress_v1                                   |
| kubernetes_ingress_class                      | kubernetes_ingress_class_v1                             |
| kubernetes_network_policy                     | kubernetes_network_policy_v1                            |
| kubernetes_pod_disruption_budget              | kubernetes_pod_disruption_budget_v1                     |
| kubernetes_pod_security_policy                | *Removed from upstream*                                 |
| kubernetes_priority_class                     | kubernetes_priority_class_v1                            |
| kubernetes_validating_webhook_configuration   | kubernetes_validating_webhook_configuration_v1          |
| kubernetes_mutating_webhook_configuration     | kubernetes_mutating_webhook_configuration_v1            |
| kubernetes_storage_class                      | kubernetes_storage_class_v1                             |
| kubernetes_csi_driver                         | kubernetes_csi_driver_v1                                |

## Removal of `kubernetes_pod_security_policy`

This resource was removed from the Kubernetes API v1.25 and has been removed from this provider. 

## Behaviour of the `wait_for_rollout` attribute of `kubernetes_daemon_set_v1`

A [long standing bug in the wait logic for this resource has been fixed](https://github.com/hashicorp/terraform-provider-kubernetes/issues/2092) where despite this flag being set to true, the resource would not wait for the daemonset to become ready. As this attribute is set to false users may now notice that this resource takes longer to create as it will wait by default.  


