
# Terraform Provider for Kubernetes [![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/terraform-providers/terraform-provider-kubernetes?label=release)](https://github.com/terraform-providers/terraform-provider-kubernetes/releases) [![license](https://img.shields.io/github/license/terraform-providers/terraform-provider-kubernetes.svg)]()

<a href="https://terraform.io">
    <img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" alt="Terraform logo" title="Terrafpr," align="right" height="50" />
</a>

- [Getting Started](https://learn.hashicorp.com/terraform?track=kubernetes#kubernetes)
- Usage 
  - [Documentation](https://www.terraform.io/docs/providers/kubernetes/index.html)
  - [Examples](https://github.com/terraform-providers/terraform-provider-kubernetes/tree/master/_examples)
- Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)
- Chat: [#terraform-providers in Kubernetes](https://kubernetes.slack.com/messages/CJY6ATQH4) ([Sign up here](http://slack.k8s.io/))

The Kubernetes provider for Terraform is a plugin that enables full lifecycle management of Kubernetes resources. This provider is maintained internally by HashiCorp.

Please note: We take Terraform's security and our users' trust very seriously. If you believe you have found a security issue in the Terraform Kubernetes Provider, please responsibly disclose by contacting us at security@hashicorp.com.


## Requirements

-	[Terraform](https://www.terraform.io/downloads.html) 0.12.x
    - Note that version 0.11.x currently works, but is [deprecated](https://www.hashicorp.com/blog/deprecating-terraform-0-11-support-in-terraform-providers/)
-	[Go](https://golang.org/doc/install) 1.14.x (to build the provider plugin)

## Contributing to the provider

The Terraform Kubernetes Provider is the work of many contributors. We appreciate your help!

To contribute, please read the [contribution guidelines](_about/CONTRIBUTING.md). You may also [report an issue](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/new/choose). Once you've filed an issue, it will follow the [issue lifecycle](_about/ISSUES.md).

Also available are some answers to [Frequently Asked Questions](_about/FAQ.md).

