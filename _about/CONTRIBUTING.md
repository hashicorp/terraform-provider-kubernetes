# Contributor Guide

Thank you for your interest in contributing to the Kubernetes provider. We welcome your contributions. Here you'll find information to help you get started with provider development.

If you want to know more about how to develop a Terraform provider, please refer to the [Plugin Development documentation](https://developer.hashicorp.com/terraform/plugin).

## Configuring Environment

<!-- TODO:
- Add cluster name to the config
- Add an example of how to use Kustomize to tune the cluster config and provision different Kubernetes version
- Once we move on with more automation, we need to update this section too
- We might want to add an example of how to provision a KinD cluster with Terraform
- We might want to add a few words on how to use kubectl command to validate the cluster
- We might want to mention here or in a different place that some tests we can only run on a specific managed cluster, such as AKS, GKE, or AWS and how to do that
-->

1. Install Golang

    [Install](https://go.dev/doc/install) the version of Golang as indicated in the [go.mod](../go.mod) file.

1. Fork this repo

    [Fork](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/working-with-forks/fork-a-repo) the provider repository and clone it on your computer.

    Here is an example of how to clone this repository and switch to the directory.

    ```console
    $ git clone https://github.com/<YOUR-USERNAME>/terraform-provider-kubernetes.git
    $ cd terraform-provider-kubernetes
    ```

    From now on we are going to assume that you have a copy of the repository on your computer and work within the `terraform-provider-kubernetes` directory.

1. Prepare a Kubernetes Cluster

    While our preference is to use [kind](https://kind.sigs.k8s.io/) for setting up a Kubernetes cluster for development and test purposes, feel free to opt for the solution that best suits your preferences. Please, bear in mind that some acceptance tests might require specific cluster settings that we maintain in the KinD [configuration file](../.github/config/acceptance_tests_kind_config.yaml).

    Here is an example of how to provision a Kubernetes cluster with the configuration file:

    ```console
    $ kind create cluster --config=./.github/config/acceptance_tests_kind_config.yaml
    ```

    Once the Kubernetes cluster is up and running we strongly advise you to run acceptance tests before making any changes to make sure that they work with your setup. Please refer to the [Testing](#testing) section for more details.


## Making Changes

<!-- TODO:
- We need to mention here linters that we have and how to run them
- Break down changes into categories, such as adding, updating, removing(???) or fixing resource, data source, provider block, attribute, documentation or making a small change
- We might want to mention here some best practices that are specfic to the Kubernete provider, such as reuse constatns from the Kuberentes packages as a default value in an attribute or within a validation function
-->

## Testing

<!-- TODO:
- We need to explain here that the provider has unit and acceptance tests and when they need to be added or updated
- We need to explain here how to run a specific test or group of tests
- We need to explain here how to build a provider binary and run it
-->

```console
$ export KUBE_CONFIG_PATH=~/.kube/config
$ make testacc TESTARGS="-run ^TestAcc"
$ make test
```

1. Run existing tests
1. Write/Update tests
1. Run tests with new changes

## Updating changelog

<!-- TODO:
- We need to explain here when a change log is necessary to add
-->

Please refer to our [ChangeLog Guide](../CHANGELOG_GUIDE.md).

## Creating & Submiting a PR

<!--
- We need to explain here what do we expect to see in a PR, the same should be reflected in a PR template
-->

Please refer to this [guide](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/creating-a-pull-request-from-a-fork).

## Debug Guide

<!-- TODO THIS SECTION -->
