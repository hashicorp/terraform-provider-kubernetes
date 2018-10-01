---
layout: "registry"
page_title: "Finding and Using Modules from the Terraform Registry"
sidebar_current: "docs-registry-use"
description: |-
  The Terraform Registry makes it simple to find and use modules.
---

# Finding and Using Modules

The [Terraform Registry](https://registry.terraform.io) makes it simple to
find and use modules.

## Finding Modules

Every page on the registry has a search field for finding
modules. Enter any type of module you're looking for (examples: "vault",
"vpc", "database") and resulting modules will be listed. The search query
will look at module name, provider, and description to match your search
terms. On the results page, filters can be used further refine search results.

By default, only [verified modules](/docs/registry/modules/verified.html)
are shown in search results. Verified modules are reviewed by HashiCorp to
ensure stability and compatibility. By using the filters, you can view unverified
modules as well.

## Using Modules

The Terraform Registry is integrated directly into Terraform. This makes
it easy to reference any module in the registry. The syntax for referencing
a registry module is `<NAMESPACE>/<NAME>/<PROVIDER>`. For example:
`hashicorp/consul/aws`.

~> **Note:** Module registry integration was added in Terraform v0.10.6, and full versioning support in v0.11.0.

When viewing a module on the registry on a tablet or desktop, usage instructions
are shown on the right side.
You can copy and paste this to get started with any module. Some modules
have required inputs you must set before being able to use the module.

```hcl
module "consul" {
  source = "hashicorp/consul/aws"
  version = "0.1.0"
}
```

The `terraform init` command will download and cache any modules referenced by
a configuration.

### Private Registry Module Sources

You can also use modules from a private registry, like the one provided by
Terraform Enterprise. Private registry modules have source strings of the form
`<HOSTNAME>/<NAMESPACE>/<NAME>/<PROVIDER>`. This is the same format as the
public registry, but with an added hostname prefix.

```hcl
module "vpc" {
  source = "app.terraform.io/example_corp/vpc/aws"
  version = "0.9.3"
}
```

Depending on the registry you're using, you might also need to configure
credentials to access modules. See your registry's documentation for details.
[Terraform Enterprise's private registry is documented here.](/docs/enterprise/registry/index.html)

Private registry module sources are supported in Terraform v0.11.0 and
newer.

## Module Versions

Each module in the registry is versioned. These versions syntactically must
follow [semantic versioning](http://semver.org/). In addition to pure syntax,
we encourage all modules to follow the full guidelines of semantic versioning.

Terraform since version 0.11 will resolve any provided
[module version constraints](/docs/modules/usage.html#module-versions) and
using them is highly recommended to avoid pulling in breaking changes.

Terraform versions after 0.10.6 but before 0.11 have partial support for the registry
protocol, but always download the latest version instead of honoring version
constraints.
