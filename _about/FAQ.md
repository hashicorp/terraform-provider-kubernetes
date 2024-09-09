# Frequently Asked Questions

### Who are the maintainers?

The HashiCorp Terraform Kubernetes provider team is :

* Vishnu Ravindra, Product Manager - [@vravind1](https://github.com/vravind1)
* Alex Somesan, Engineer - [@alexsomesan](https://github.com/alexsomesan)
* John Houston, Engineer - [@jrhouston](https://github.com/jrhouston)
* Sacha Rybolovlev, Engineer - [@arybolovlev](https://github.com/arybolovlev)
* Mauricio Alvarez Leon, Engineer - [@BBBmau](https://github.com/BBBmau) 
* Sheneska Williams, Engineer - [@sheneska](https://github.com/sheneska) 
* Brandy Jackson, Engineering Manager - [@ibrandyjackson](https://github.com/ibrandyjackson)

Our collaborators are:

* Patrick Decat - [@pdecat](https://github.com/pdecat)

### Why isn’t my PR merged yet?

Unfortunately, due to the volume of issues and new pull requests we receive, we are unable to give each one the full attention that we would like. We do our best to focus on the contributions that provide the greatest value to the most community members.

### How do you decide what gets merged for each release?

The number one factor we look at when deciding what issues to look at are your 👍 [reactions](https://blog.github.com/2016-03-10-add-reactions-to-pull-requests-issues-and-comments/) to the original issue/PR description as these can be easily discovered. Comments that further explain desired use cases or poor user experience are also heavily factored. The items with the most support are always on our radar, and we do our best to keep the community updated on their status and potential timelines.

We also are investing time to improve the contributing experience by improving documentation.


### Backward Compatibility Promise

Our policy is described on the Terraform website [here](https://www.terraform.io/docs/extend/best-practices/versioning.html). While we do our best to prevent breaking changes until major version releases of the provider, it is generally recommended to [pin the provider version in your configuration](https://www.terraform.io/docs/configuration/providers.html#provider-versions).

Due to the constant release pace of Kubernetes and the relatively infrequent major version releases of the provider, there can be cases where a minor version update may contain unexpected changes depending on your configuration or environment.

### Why is not recommended to create Kubernetes resources in the same apply as the cluster?

When using resource attributes to pass credentials to the provider block from resources such as `aws_eks_cluster` and `google_container_cluster`, these resources should not be created in the same Terraform apply operation as Kubernetes provider resources. This will lead to intermittent and unpredictable errors which are hard to debug and diagnose. The root issue lies with the order in which Terraform itself evaluates the provider blocks vs. resources. Please refer to the [Provider Configuratopm](https://developer.hashicorp.com/terraform/language/providers/configuration#provider-configuration) section of the Terraform docs for more information.

For the `kubernetes_manifest` resource specifically, this resource _requires_ a Kubernetes cluster to already be available, as it needs to be able to fetch the OpenAPI spec from the Kubernetes API to generate the Terraform schema information needed to create a plan. 

For this reason, the most reliable way to configure the Kubernetes provider is to ensure that the cluster itself and the Kubernetes provider resources can each be managed with separate apply operations. We recommend using the corresponding data sources to supply values to the provider block as needed.

### How can I help?

Check out the [Contributing Guide](CONTRIBUTING.md) for additional information.

### How can I become a maintainer?

This is an area under active research. Stay tuned!
