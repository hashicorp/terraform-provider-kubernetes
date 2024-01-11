## Developing the provider

Thank you for your interest in contributing to the Kubernetes provider. We welcome your contributions. Here you'll find information to help you get started with provider development.

## Configuring Environment

1. Install Golang

    Install the version of [Golang](https://go.dev/) as indicated in the [go.mod](../go.mod) file. 

1. [Clone](https://docs.github.com/en/repositories/creating-and-managing-repositories/cloning-a-repository) this repo

    ```console
    $ git clone https://github.com/hashicorp/terraform-provider-kubernetes.git
    $ cd terraform-provider-kubernetes
     ```

1. Prepare a Kubernetes Cluster

    While our preference is to use [kind](https://kind.sigs.k8s.io/) for setting up a Kubernetes cluster for development and test purposes, feel free to opt for the solution that best suits your preferences.

    How to Provision a Cluster
    ```console
    $ kind create cluster --config=./.github/config/acceptance_tests_kind_config.yaml
     ```
    
    Validating Cluster 
        
    Please refer to the [Writing Tests](#writing-tests) section.


## Making Changes
//TO-DO

## Testing

```console
$ export KUBE_CONFIG_PATH=~/.kube/config
$ make testacc TESTARGS="-run ^TestAcc"
$ make test
```

1. Run existing tests
1. Write/Update tests
1. Run tests with new changes

## Updating changelog

Please refer to our [ChangeLog Guide](../CHANGELOG_GUIDE.md).

## Creating & Submiting a PR

Please refer to this [guide](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/creating-a-pull-request-from-a-fork).

## Debug Guide

//TO-DO