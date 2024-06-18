# Contributor Guide

Thank you for your interest in contributing to the Kubernetes provider. We welcome your contributions. Here, you'll find information to help you get started with provider development.

If you want to learn more about developing a Terraform provider, please refer to the [Plugin Development documentation](https://developer.hashicorp.com/terraform/plugin).

## Configuring Environment

<!-- TODO:
- Add cluster name to the config
- Once we move on with more automation, we need to update this section too
- We might want to add an example of how to provision a KinD cluster with Terraform
- We might want to add a few words on how to use kubectl command to validate the cluster
- We might want to mention here or in a different place that some tests we can only run on a specific managed cluster, such as AKS, GKE, or AWS and how to do that
-->

1. Install Golang

    [Install](https://go.dev/doc/install) the version of Golang as indicated in the [go.mod](../go.mod) file.

1. Fork this repo

    [Fork](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/working-with-forks/fork-a-repo) the provider repository and clone it on your computer.

    Here is an example of how to clone this repository and switch to the directory:

    ```console
    $ git clone https://github.com/<YOUR-USERNAME>/terraform-provider-kubernetes.git
    $ cd terraform-provider-kubernetes
    ```

    From now on, we are going to assume that you have a copy of the repository on your computer and work within the `terraform-provider-kubernetes` directory.

1. Prepare a Kubernetes Cluster

    While our preference is to use [KinD](https://kind.sigs.k8s.io/) for setting up a Kubernetes cluster for development and test purposes, feel free to opt for the solution that best suits your preferences. Please bear in mind that some acceptance tests might require specific cluster settings, which we maintain in the KinD [configuration file](../.github/config/acceptance_tests_kind_config.yaml).

    Here is an example of how to provision a Kubernetes cluster using the configuration file:

    ```console
    $ kind create cluster --config=./.github/config/acceptance_tests_kind_config.yaml
    ```

    KinD comes with a default Node image version that depends on the KinD version and thus might not be always the one you want to use. The above command can be extended with the `--image` option to spin up a particular Kubernetes version:

    ```console
    $ kind create cluster \
      --config=./.github/config/acceptance_tests_kind_config.yaml \
      --image kindest/node:v1.28.0@sha256:b7a4cad12c197af3ba43202d3efe03246b3f0793f162afb40a33c923952d5b31
    ```

    Refer to the KinD [releases](https://github.com/kubernetes-sigs/kind/releases) to get the right image.

    From now on, we are going to assume that the Kubernetes configuration is stored in the `$HOME/.kube/config` file, and the current context is set to a newly created KinD cluster.

    Once the Kubernetes cluster is up and running, we strongly advise you to run acceptance tests before making any changes to ensure they work with your setup. Please refer to the [Testing](#testing) section for more details.


## Making Changes

<!-- TODO:
- ✅We need to mention here linters that we have and how to run them
- ✅Break down changes into categories, such as adding, updating, removing(???) or fixing resource, data source, provider block, attribute, documentation or making a small change
- ✅We might want to mention here some best practices that are specfic to the Kubernete provider, such as reuse constatns from the Kuberentes packages as a default value in an attribute or within a validation function
-->

### Adding a New Resource

This quick guide covers best practices for adding a new Resource. 

1. Ensure all dependncies are installed.
1. Add an SDK Client. 
1. Add Resource Schema and define attributes [see Kubernetes Documentation](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs). A best and recommended practice is reuse constants from the Kuberentes packages as a default value in an attribute or within a validation function. 
1. Scaffold an empty/new resource.
1. Add Acceptance Tests(s) for the resource.
1. Run Acceptance Tests(s) for this resource. 
1. Add Documentation for this resource by editing the `.md.tmpl` file to include the appropriate [Data Fields](https://pkg.go.dev/text/template) and executing `tfplugindocs generate` command [see Terraform PluginDocs](https://github.com/hashicorp/terraform-plugin-docs#data-fields) then inspecting the corresponding `.md` file in the `/docs` to see all changes. The Data Fields that are currently apart of the templates are those for the Schema ({{ .SchemaMarkdown }}), Name ({{ .Name }}) and ({{ .Description }}).
1. Execute `make docs-lint` and `make tests-lint` commands 
1. Create a Pull Request for your changes. 

### Adding a New Data Source

1. Ensure all dependncies are installed.
1. Add an SDK Client. 
1. Add Data Source Schema and define attributes [see Kubernetes Documentation](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs).
A best and recommended practice is reuse constants from the Kuberentes packages as a default value in an attribute or within a validation function. 
1. Scaffold an empty/new resource.
1. Add Acceptance Tests(s) for the data source.
1. Run Acceptance Tests(s) for this data source. 
1. Add Documentation for this data source by editing the `.md.tmpl` file to include the appropriate [Data Fields](https://pkg.go.dev/text/template) and executing `tfplugindocs generate` command [see Terraform PluginDocs](https://github.com/hashicorp/terraform-plugin-docs#data-fields) then inspecting the corresponding `.md` file in the `/docs` to see all changes. The Data Fields that are currently apart of the templates are those for the Schema ({{ .SchemaMarkdown }}), Name ({{ .Name }}) and ({{ .Description }}).    
1. Execute `make docs-lint` and `make tests-lint` commands 
1. Create a Pull Request for your changes. 

### Adding/Editing Documentation
All Documentation is edited in the `.md.tmpl` file. Please note that the `tfplugindocs generate` command should be executed to ensure it is updated and reflected in the `.md` files. 

## Testing

The Kubernetes provider includes two types of tests: [unit](https://developer.hashicorp.com/terraform/plugin/sdkv2/testing/unit-testing) tests and [acceptance](https://developer.hashicorp.com/terraform/plugin/sdkv2/testing/acceptance-tests) tests.

Before running any tests, make sure that the `KUBE_CONFIG_PATH` environment variable points to the Kubernetes configuration file:

```console
$ export KUBE_CONFIG_PATH=$HOME/.kube/config
```

<!-- TODO:
- We need to explain here that the provider has unit and acceptance tests and when they need to be added or updated
- We need to explain here how to run a specific test or group of tests
- We need to explain here how to build a provider binary and run it
-->

The following commands demonstrate how to run unit and acceptance tests respectively.

```console
$ make test # unit tests
$ make testacc TESTARGS="-run ^TestAcc" # acceptance tests
```

1. Run existing tests
1. Write/Update tests
1. Run tests with new changes

## Updating changelog

<!-- TODO:
- We need to explain here when a change log is necessary to add
-->

A PR that is merged may or may not be added to the changelog. Not every change should be in the changelog since they don't affect users directly. Some instances of PRs that could be excluded are:

- unit and acceptance tests fixes
- minor documentation changes

However, PRs of the following categories should be added to the appropriate section:

* `FEATURES` 
* `ENHANCEMENTS`
* `MAJOR BUG FIXES`

Please refer to our [ChangeLog Guide](../CHANGELOG_GUIDE.md).

## Creating & Submiting a PR

<!--
- We need to explain here what do we expect to see in a PR, the same should be reflected in a PR template
-->

Please refer to this [guide](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/creating-a-pull-request-from-a-fork).

## Debug Guide

<!-- TODO THIS SECTION -->
