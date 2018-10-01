---
layout: "registry"
page_title: "Terraform Registry - Publishing Modules"
sidebar_current: "docs-registry-publish"
description: |-
  Anyone can publish and share modules on the Terraform Registry.
---

# Publishing Modules

Anyone can publish and share modules on the [Terraform Registry](https://registry.terraform.io).

Published modules support versioning, automatically generate documentation,
allow browsing version histories, show examples and READMEs, and more. We
recommend publishing reusable modules to a registry.

Public modules are managed via Git and GitHub. Publishing a module takes only
a few minutes. Once a module is published, you can release a new version of
a module by simply pushing a properly formed Git tag.

The registry extracts information about the module from the module's source.
The module name, provider, documentation, inputs/outputs, and dependencies are
all parsed and available via the UI or API, as well as the same information for
any submodules or examples in the module's source repository.

## Requirements

The list below contains all the requirements for publishing a module.
Meeting the requirements for publishing a module is extremely easy. The
list may appear long only to ensure we're detailed, but adhering to the
requirements should happen naturally.

- **GitHub.** The module must be on GitHub and must be a public repo.
This is only a requirement for the [public registry](https://registry.terraform.io).
If you're using a private registry, you may ignore this requirement.

- **Named `terraform-<PROVIDER>-<NAME>`.** Module repositories must use this
three-part name format, where `<NAME>` reflects the type of infrastructure the
module manages and `<PROVIDER>` is the main provider where it creates that
infrastructure. The `<NAME>` segment can contain additional hyphens. Examples:
`terraform-google-vault` or `terraform-aws-ec2-instance`.

- **Repository description.** The GitHub repository description is used
to populate the short description of the module. This should be a simple
one sentence description of the module.

- **Standard module structure.** The module must adhere to the
[standard module structure](/docs/modules/create.html#standard-module-structure).
This allows the registry to inspect your module and generate documentation,
track resource usage, parse submodules and examples, and more.

- **`x.y.z` tags for releases.** The registry uses tags to identify module
versions. Release tag names must be a [semantic version](http://semver.org),
which can optionally be prefixed with a `v`. For example, `v1.0.4` and `0.9.2`.
To publish a module initially, at least one release tag must be present. Tags
that don't look like version numbers are ignored.

## Publishing a Public Module

With the requirements met, you can publish a public module by going to
the [Terraform Registry](https://registry.terraform.io) and clicking the
"Upload" link in the top navigation.

If you're not signed in, this will ask you to connect with GitHub. We only
ask for access to public repositories, since the public registry may only
publish public modules. We require access to hooks so we can register a webhook
with your repository. We require access to your email address so that we can
email you alerts about your module. We will not spam you.

The upload page will list your available repositories, filtered to those that
match the [naming convention described above](#Requirements). This is shown in
the screenshot below. Select the repository of the module you want to add and
click "Publish Module."

In a few seconds, your module will be created.

![Publish Module flow animation](/assets/images/docs/registry-publish.gif)

## Releasing New Versions

The Terraform Registry uses tags to detect releases.

Tag names must be a valid [semantic version](http://semver.org), optionally
prefixed with a `v`. Example of valid tags are: `v1.0.1` and `0.9.4`. To publish
a new module, you must already have at least one tag created.

To release a new version, create and push a new tag with the proper format.
The webhook will notify the registry of the new version and it will appear
on the registry usually in less than a minute.

If your version doesn't appear properly, you may force a sync with GitHub
by viewing your module on the registry and clicking "Force GitHub Sync"
under the "Manage Module" dropdown. This process may take a few minutes.
Please only do this if you do not see the version appear, since it will
cause the registry to resync _all versions_ of your module.
