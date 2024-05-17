---
layout: "kubernetes"
page_title: "Kubernetes: Upgrade Guide for Kubernetes Provider v2.0.0"
description: |-
  This guide covers the changes introduced in v2.0.0 of the Kubernetes provider and what you may need to do to upgrade your configuration.
---

# Upgrading to v2.0.0 of the Kubernetes provider

This guide covers the changes introduced in v2.0.0 of the Kubernetes provider and what you may need to do to upgrade your configuration.

Use `terraform init` to install version 2 of the provider. Then run `terraform plan` to determine if the upgrade will affect any existing resources. Some resources will have updated defaults and may be modified as a result. To opt out of this change, see the guide below and update your Terraform config file to match the existing resource settings (for example, set `automount_service_account_token=false`). Then run `terraform plan` again to ensure no resource updates will be applied.

NOTE: Even if there are no resource updates to apply, you may need to run `terraform refresh` to update your state to the newest version. Otherwise, some commands might fail with `Error: missing expected {`.

## Installing and testing this update

The `required_providers` block can be used to move between version 1.x and version 2.x of the Kubernetes provider, for testing purposes. Please note that this is only possible using `terraform plan`. Once you run `terraform apply` or `terraform refresh`, the changes to Terraform State become permanent, and rolling back is no longer an option. It may be possible to roll back the State by making a copy of `.terraform.tfstate` before running `apply` or `refresh`, but this configuration is unsupported.

### Using required_providers to test the update

The version of the Kubernetes provider can be controlled using the `required_providers` block:

```hcl
terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 2.0"
    }
  }
}
```

When the above code is in place, run `terraform init` to upgrade the provider version.

```
$ terraform init -upgrade
```

Ensure you have a valid provider block for 2.0 before proceeding with the `terraform plan` below. In version 2.0 of the provider, [provider configuration is now required](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs). A quick way to get up and running with the new provider configuration is to set `KUBE_CONFIG_PATH` to point to your existing kubeconfig.

```
export KUBE_CONFIG_PATH=$KUBECONFIG
```

Then run `terraform plan` to see what changes will be applied. This example shows the specific fields that would have been modified, and their effect on the resources, such as replacement or an in-place update. Some output is omitted for clarity.

```
$ export KUBE_CONFIG_PATH=$KUBECONFIG
$ terraform plan

kubernetes_pod.test: Refreshing state... [id=default/test]
kubernetes_job.test: Refreshing state... [id=default/test]
kubernetes_stateful_set.test: Refreshing state... [id=default/test]
kubernetes_deployment.test: Refreshing state... [id=default/test]
kubernetes_daemonset.test: Refreshing state... [id=default/test]
kubernetes_cron_job.test: Refreshing state... [id=default/test]

An execution plan has been generated and is shown below.
Resource actions are indicated with the following symbols:
  ~ update in-place
-/+ destroy and then create replacement

Terraform will perform the following actions:

  # kubernetes_cron_job.test must be replaced
-/+ resource "kubernetes_cron_job" "test" {
                          ~ enable_service_links             = false -> true # forces replacement

  # kubernetes_daemonset.test will be updated in-place
  ~ resource "kubernetes_daemonset" "test" {
      + wait_for_rollout = true
          ~ template {
              ~ spec {
                  ~ enable_service_links             = false -> true

  # kubernetes_deployment.test will be updated in-place
  ~ resource "kubernetes_deployment" "test" {
              ~ spec {
                  ~ enable_service_links             = false -> true

  # kubernetes_job.test must be replaced
-/+ resource "kubernetes_job" "test" {
                  ~ enable_service_links             = false -> true # forces replacement

  # kubernetes_stateful_set.test will be updated in-place
  ~ resource "kubernetes_stateful_set" "test" {
              ~ spec {
                  ~ enable_service_links             = false -> true

Plan: 2 to add, 3 to change, 2 to destroy.
```

Using the output from `terraform plan`, you can make modifications to your existing Terraform config, to avoid any unwanted resource changes. For example, in the above config, adding `enable_service_links = false` to the resources would prevent any changes from occurring to the existing resources.

#### Known limitation: Pod data sources need manual upgrade

During `terraform plan`, you might encounter the error below:

```
Error: .spec[0].container[0].resources[0].limits: missing expected {
```

This ocurrs when a Pod data source is present during upgrade. To work around this error, remove the data source from state and try the plan again.

```
$ terraform state rm data.kubernetes_pod.test
Removed data.kubernetes_pod.test
Successfully removed 1 resource instance(s).

$ terraform plan
```

The data source will automatically be added back to state with data from the upgraded schema.

### Rolling back to version 1.x

If you've run the above upgrade and plan, but you don't want to proceed with the 2.0 upgrade, you can roll back using the following steps. NOTE: this will only work if you haven't run `terraform apply` or `terraform refresh` while testing version 2 of the provider.

```
$ terraform version
Terraform v0.14.4
+ provider registry.terraform.io/hashicorp/kubernetes v2.0
```

Set the provider version back to 1.x.

```
terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "1.13"
    }
  }
}
```

Then run `terraform init -upgrade` to install the old provider version.

```
$ terraform init -upgrade

Initializing the backend...

Initializing provider plugins...
- Finding hashicorp/kubernetes versions matching "1.13.0"...
- Installing hashicorp/kubernetes v1.13.0...
- Installed hashicorp/kubernetes v1.13.0 (signed by HashiCorp)
```

The provider is now downgraded.

```
$ terraform version
Terraform v0.14.4
+ provider registry.terraform.io/hashicorp/kubernetes v1.13.0
```


## Changes in v2.0.0

### Changes to Kubernetes credentials supplied in the provider block

We have made several changes to the way access to Kubernetes is configured in the provider block.

1. The `load_config_file` attribute has been removed.
2. Support for the `KUBECONFIG` environment variable has been dropped. (Use `KUBE_CONFIG_PATH` or `KUBE_CONFIG_PATHS` instead).
3. The `config_path` attribute will no longer default to `~/.kube/config`.

The above changes have been made to encourage the best practice of configuring access to Kubernetes in the provider block explicitly, instead of relying upon default paths or `KUBECONFIG` being set. We have done this because allowing the provider to configure its access to Kubernetes implicitly caused confusion with a subset of our users. It also created risk for users who use Terraform to manage multiple clusters. Requiring explicit configuration for Kubernetes in the provider block eliminates the possibility that the configuration will be applied to the wrong cluster.

You will therefore need to explicitly configure access to your Kubernetes cluster in the provider block going forward. For many users this will simply mean specifying the `config_path` attribute in the provider block. Users already explicitly configuring the provider should not be affected by this change, but will need to remove the `load_config_file` attribute if they are currently using it.

### Changes to the `load_balancers_ingress` block on Service and Ingress

We changed the `load_balancers_ingress` block on the Service and Ingress resources and data sources to align with the upstream Kubernetes API. `load_balancers_ingress` was a computed attribute that allowed users to obtain the `ip` or `hostname` of a `load_balancer`. Instead of `load_balancers_ingress`, users should use `status[].load_balancer[].ingress[]` to obtain the `ip` or `hostname` attributes.

```hcl
output "ingress_hostname" {
  value = kubernetes_ingress.example_ingress.status[0].load_balancer[0].ingress[0].hostname
}
```

### The `automount_service_account_token` attribute now defaults to `true` on Service, Deployment, StatefulSet, and DaemonSet 

Previously if `automount_service_account_token = true` was not set on the Service, Deployment, StatefulSet, or DaemonSet resources, the service account token was not mounted, even when a `service_account_name` was specified.  This lead to confusion for many users, because our implementation did not align with the default behavior of the Kubernetes API, which defaults to `true` for this attribute.

```hcl
resource "kubernetes_deployment" "example" {
  metadata {
    name = "terraform-example"
    labels = {
      test = "MyExampleApp"
    }
  }

  spec {
    replicas = 1

    selector {
      match_labels = {
        test = "MyExampleApp"
      }
    }

    template {
      metadata {
        labels = {
          test = "MyExampleApp"
        }
      }

      spec {
        container {
          image = "nginx:1.21.6"
          name  = "example"
        }

        service_account_name            = "default"
        automount_service_account_token = false
      }
    }
  }
}
```

### Normalize wait defaults across Deployment, DaemonSet, StatefulSet, Service, Ingress, and Job

All of the `wait_for` attributes now default to `true`, including:

- `wait_for_rollout` on the `kubernetes_deployment`, `kubernetes_daemonset`, and `kubernetes_stateful_set` resources
- `wait_for_loadbalancer` on the `kubernetes_service` and `kubernetes_ingress` resources
- `wait_for_completion` on the `kubernetes_job` resource

Previously some of them defaulted to `false` while others defaulted to `true`, causing an inconsistent user experience. If you don't want Terraform to wait for the specified condition before moving on, you must now always set the appropriate attribute to `false`

```hcl
resource "kubernetes_service" "myapp1" {
  metadata {
    name = "myapp1"
  }

  spec {
    selector = {
      app = kubernetes_pod.example.metadata[0].labels.app
    }

    session_affinity = "ClientIP"
    type             = "LoadBalancer"

    port {
      port        = 8080
      target_port = 80
    }
  }

  wait_for_load_balancer = "false"
}
```

### Changes to the `limits` and `requests` attributes on all resources that support a PodSpec

The `limits` and `requests` attributes on all resources that include a PodSpec, are now a map.  This means that `limits {}` must be changed to `limits = {}`, and the same for `requests`. This change impacts the following resources: `kubernetes_deployment`, `kubernetes_daemonset`, `kubernetes_stateful_set`, `kubernetes_pod`, `kubernetes_job`, `kubernetes_cron_job`. 

This change was made to enable the use of extended resources, such as GPUs, in these fields.

```hcl
resource "kubernetes_pod" "test" {
  metadata {
    name = "terraform-example"
  }

  spec {
    container {
      image = "nginx:1.21.6"
      name  = "example"

      resources {
        limits = {
          cpu          = "0.5"
          memory       = "512Mi"
          "nvidia/gpu" = "1"
        }

        requests = {
          cpu          = "250m"
          memory       = "50Mi"
          "nvidia/gpu" = "1"
        }
      }
    }
  }
}
```


### Dropped support for Terraform 0.11

All builds of the Kubernetes provider going forward will no longer work with Terraform 0.11. See [Upgrade Guides](https://www.terraform.io/upgrade-guides/index.html) for how to migrate your configurations to a newer version of Terraform.

### Upgrade to v2 of the Terraform Plugin SDK

Contributors to the provider will be interested to know this upgrade has brought the latest version of the [Terraform Plugin SDK](https://github.com/hashicorp/terraform-plugin-sdk) which introduced a number of enhancements to the developer experience. Details of the changes introduced can be found under [Extending Terraform](https://www.terraform.io/docs/extend/guides/v2-upgrade-guide.html).
