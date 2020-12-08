---
layout: "kubernetes"
page_title: "Kubernetes: Upgrade Guide for Kubernetes Provider v2.0.0"
description: |-
  This guide covers the changes introduced in v2.0.0 of the Kubernetes provider and what you may need to do to upgrade your configuration.
---

# Upgrading to v2.0.0 of the Kubernetes provider

This guide covers the changes introduced in v2.0.0 of the Kubernetes provider and what you may need to do to upgrade your configuration.

## Changes in v2.0.0

### Changes to Kubernetes credentials supplied in the provider block

We have made several changes to the way access to Kubernetes is configured in the provider block.

1. The `load_config_file` attribute has been removed.
2. Support for the `KUBECONFIG` environment variable has been dropped.
3. The `config_path` attribute will no longer default to `~/.kube/config`.

The above changes have been made to encourage the best practice of configuring access to Kubernetes in the provider block explicitly, instead of relying upon default paths or `KUBECONFIG` being set. We have done this because allowing the provider to configure its access to Kubernetes implicitly caused confusion with a subset of our users. It also created risk for users who use Terraform to manage multiple clusters. Requiring explicit configuring for kubernetes in the provider block eliminates the possibility that the configuration will be applied to the wrong cluster.

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
          image = "nginx:1.7.8"
          name  = "example"
        }
        
        service_account_name = "default"
        automount_service_account_token = false
      }
    }
  }
}
```

### Normalize wait defaults across Deployment, DaemonSet, StatefulSet, Service, Ingress, and Job
https://github.com/hashicorp/terraform-provider-kubernetes/pull/1053

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
    type = "LoadBalancer"

    port {
      port        = 8080
      target_port = 80
    }
  }

  wait_for_load_balancer = "false"
}
```

### Changes to the `limits` and `requests` attributes on all resources that support a PodSpec
https://github.com/hashicorp/terraform-provider-kubernetes/pull/1065

The `limits` and `requests` attributes on all resources that include a PodSpec, are now a map.  This means that `limits {}` must be changed to `limits = {}`, and the same for `requests`. This change impacts the following resources: `kubernetes_deployment`, `kubernetes_daemonset`, `kubernetes_stateful_set`, `kubernetes_pod`, `kubernetes_job`, `kubernetes_cron_job`. 

This change was made to enable the use of extended resources, such as GPUs, in these fields.

```hcl
resource "kubernetes_pod" "test" {
  metadata {
    name = "terraform-example"
  }

  spec {
    container {
      image = "nginx:1.7.9"
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
