
# Kubernetes Provider for Terraform [![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/hashicorp/terraform-provider-kubernetes?label=release)](https://github.com/hashicorp/terraform-provider-kubernetes/releases) [![license](https://img.shields.io/github/license/hashicorp/terraform-provider-kubernetes.svg)]()

<a href="https://terraform.io">
    <img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" alt="Terraform logo" title="Terrafpr," align="right" height="50" />
</a>

- [Getting Started](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs/guides/getting-started)
- [Interactive Tutorial](https://learn.hashicorp.com/tutorials/terraform/kubernetes-provider?in=terraform/kubernetes)
- Usage
  - [Documentation](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs)
  - [Examples](https://github.com/hashicorp/terraform-provider-kubernetes/tree/main/_examples)
  - [Kubernetes Provider 2.0 Upgrade guide](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs/guides/v2-upgrade-guide)
- Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)
- Chat: [#terraform-providers in Kubernetes](https://kubernetes.slack.com/messages/CJY6ATQH4) ([Sign up here](http://slack.k8s.io/))
- Roadmap: [Q3 2020](_about/ROADMAP.md)

The Kubernetes provider for Terraform is a plugin that enables full lifecycle management of Kubernetes resources. This provider is maintained internally by HashiCorp.

Please note: We take Terraform's security and our users' trust very seriously. If you believe you have found a security issue in the Terraform Kubernetes Provider, please responsibly disclose by contacting us at security@hashicorp.com.


## Requirements

-	[Terraform](https://www.terraform.io/downloads.html) 0.12.x
-	[Go](https://golang.org/doc/install) 1.16.x (to build the provider plugin)


## Kubernetes Alpha Provider

A new [experimental provider](https://registry.terraform.io/providers/hashicorp/kubernetes-alpha/latest) is now available that enables management of all Kubernetes resources, including CustomResourceDefinitions (CRDs). Our intent is to eventually merge these two providers.


## Contributing to the provider

The Kubernetes Provider for Terraform is the work of many contributors. We appreciate your help!

To contribute, please read the [contribution guidelines](_about/CONTRIBUTING.md). You may also [report an issue](https://github.com/hashicorp/terraform-provider-kubernetes/issues/new/choose). Once you've filed an issue, it will follow the [issue lifecycle](_about/ISSUES.md).

Also available are some answers to [Frequently Asked Questions](_about/FAQ.md).

