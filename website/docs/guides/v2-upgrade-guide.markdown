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
https://github.com/hashicorp/terraform-provider-kubernetes/pull/1071

### The `automount_service_account_token` attribute now defaults to `true` on Service, Deployment, StatefulSet, and DaemonSet 

This change was made to align with the Kubernetes API default.

Previously if `automount_service_account_token = true` was not set on the Service, Deployment, StatefulSet, or DaemonSet resources, the service account token was not mounted, even when a `service_account` was specified.  This lead to confusion for many users.

In practice, this means that the provider will update all Service, Deployment, StatefulSet, and DaemonSet resources that don't have `automount_service_account_token = false` set explicitly, to `automount_service_account_token = true`. See the documentation for the specific resource in question for more information on this attribute.

### Normalize wait defaults across Deployment, DaemonSet, StatefulSet, Service, Ingress, and Job
https://github.com/hashicorp/terraform-provider-kubernetes/pull/1053

### Changes to the `limits` and `requests` attributes to support extended resources
https://github.com/hashicorp/terraform-provider-kubernetes/pull/1065

### Dropped support for Terraform 0.11

All builds of the Kubernetes provider going forward will no longer work with Terraform 0.11. See [Upgrade Guides](https://www.terraform.io/upgrade-guides/index.html) for how to migrate your configurations to a newer version of Terraform.

### Upgrade to v2 of the Terraform Plugin SDK

Contributors to the provider will be interested to know this upgrade has brought the latest version of the [Terraform Plugin SDK](https://github.com/hashicorp/terraform-plugin-sdk) which introduced a number of enhancements to the developer experience. Details of the changes introduced can be found under [Extending Terraform](https://www.terraform.io/docs/extend/guides/v2-upgrade-guide.html).
