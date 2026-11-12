# Contributor Guide

Thank you for your interest in contributing to the Kubernetes provider. We welcome your contributions. Here, you'll find information to help you get started with provider development.

If you want to learn more about developing a Terraform provider, please refer to the [Plugin Development documentation](https://developer.hashicorp.com/terraform/plugin).

## Configuring Environment

1. Install Golang

    [Install](https://go.dev/doc/install) the version of Golang as indicated in the [go.mod](../go.mod) file.

2. Fork this repo

    [Fork](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/working-with-forks/fork-a-repo) the provider repository and clone it on your computer.

    Here is an example of how to clone this repository and switch to the directory:

    ```console
    $ git clone https://github.com/<YOUR-USERNAME>/terraform-provider-kubernetes.git
    $ cd terraform-provider-kubernetes
    ```

    From now on, we are going to assume that you have a copy of the repository on your computer and work within the `terraform-provider-kubernetes` directory.

3. Prepare a Kubernetes Cluster

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

### Adding a New Resource

This quick guide covers best practices for adding a new Resource. 

1. Ensure all dependncies are installed.
2. Add an SDK Client. 
3. Add Resource Schema and define attributes [see Kubernetes Documentation](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs). A best and recommended practice is reuse constants from the Kuberentes packages as a default value in an attribute or within a validation function. 
4. Scaffold an empty/new resource.
5. Add Acceptance Tests(s) for the resource.
6. Run Acceptance Tests(s) for this resource. 
7. Add documentation for this resource in the appropriate `docs/resources/<TYPE>_<VERSION>.go.md` by following [Documentation Generation](#documentation-generation).
8. Execute `make docs-lint` and `make tests-lint` commands 
9. Create a Pull Request for your changes. 

### Adding a New Data Source

1. Ensure all dependncies are installed.
2. Add an SDK Client. 
3. Add Data Source Schema and define attributes [see Kubernetes Documentation](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs).
A best and recommended practice is reuse constants from the Kuberentes packages as a default value in an attribute or within a validation function. 
4. Scaffold an empty/new resource.
5. Add Acceptance Tests(s) for the data source.
6. Run Acceptance Tests(s) for this data source. 
7. Add documentation for this data source in the appropriate `docs/data-sources/<TYPE>_<VERSION>.md` by following [Documentation Generation](#documentation-generation).
8. Execute `make docs-lint` and `make tests-lint` commands 
9. Create a Pull Request for your changes. 

### Documentation Generation

Use `make doc-generate` to generate documentation for a resource or data source
and `make doc-check` to verify that its published page is up to date.

Install the Go version from [go.mod](../go.mod), Terraform **1.16.4**, Git and `jq`. The helper runs tfplugindocs **v0.25.0**
as a versioned tool, without changing the provider's module dependencies.

1. Ensure the resource or data source is registered in the provider.
2. Create or update its template and the examples referenced by that template:

   | Kind | Template | Generated documentation |
   | --- | --- | --- |
   | Resource | `templates/resources/<TYPE>_<VERSION>.md.tmpl` | `docs/resources/<TYPE>_<VERSION>.md` |
   | Data source | `templates/data-sources/<TYPE>_<VERSION>.md.tmpl` | `docs/data-sources/<TYPE>_<VERSION>.md` |

   Edit schema descriptions in Go and explanatory text in the template. Use
   `{{ .SchemaMarkdown }}` for schema documentation and `tffile` to include examples.
   An explicit template is required. Before using an existing legacy template,
   compare it with the published page and preserve its examples, caveats, category
   and import guidance; legacy templates may be out of date.
3. Generate and check the page using the commands below.
4. Review the generated diff and include both source changes and the generated
   documentation in your pull request. Run `make docs-lint` and relevant code tests.

Replace `<TYPE>_<VERSION>` with the short registered name, without the
`kubernetes_` prefix or a file extension. Use the actual short name for resources
or data sources whose names have no version suffix or contain additional suffixes.
The quotes below keep the placeholder shell-safe; replace it before running.

```shell
make doc-generate KIND=resources NAME='<TYPE>_<VERSION>'
make doc-check KIND=resources NAME='<TYPE>_<VERSION>'

make doc-generate KIND=data-sources NAME='<TYPE>_<VERSION>'
make doc-check KIND=data-sources NAME='<TYPE>_<VERSION>'
```

`doc-check` reports differences and exits nonzero without changing published
files. `doc-generate` shows the diff before updating the selected page. It refuses
to replace differing uncommitted destination content unless you explicitly add
`OVERWRITE=1`; review or save those edits first. An already matching page is left
untouched, even if it is uncommitted.

## Testing

The Kubernetes provider includes two types of tests: [unit](https://developer.hashicorp.com/terraform/plugin/sdkv2/testing/unit-testing) tests and [acceptance](https://developer.hashicorp.com/terraform/plugin/sdkv2/testing/acceptance-tests) tests.

Before running any tests, make sure that the `KUBE_CONFIG_PATH` environment variable points to the Kubernetes configuration file:

```console
$ export KUBE_CONFIG_PATH=$HOME/.kube/config
```

The following commands demonstrate how to run unit and acceptance tests respectively.

```console
$ make test # unit tests
$ make testacc TESTARGS="-run ^TestAcc" # acceptance tests
```

1. Run existing tests
1. Write/Update tests
1. Run tests with new changes

## Updating changelog

A PR that is merged may or may not be added to the changelog. Not every change should be in the changelog since they don't affect users directly. Some instances of PRs that could be excluded are:

- unit and acceptance tests fixes
- minor documentation changes

However, PRs of the following categories should be added to the appropriate section:

* `FEATURES` 
* `ENHANCEMENTS`
* `MAJOR BUG FIXES`

Please refer to our [ChangeLog Guide](../CHANGELOG_GUIDE.md).

## Creating & Submiting a PR

Please refer to this [guide](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/creating-a-pull-request-from-a-fork).