# AKS (Azure Kubernetes Service)

You need to be logged into Azure using the `az` CLI tool.
To do that, follow the instructions in [this guide](https://www.terraform.io/docs/providers/azurerm/auth/service_principal_client_secret.html).

In addition, you will need the following environment variables to be set.

 - `TF_VAR_aks_client_id`
 - `TF_VAR_aks_client_secret`
 - `TF_VAR_location`

Obtaining the values for  ***client id*** and ***client secret*** is detailed in the documentation linked above.

## Versions

Determine the supported Kubernetes versions in a specific AKS location using the command:
```
az aks get-versions --location $TF_VAR_location --output table
```

You can set the desired version of Kubernetes via the `kubernetes_version` TF variable, like this:
```
export TF_VAR_kubernetes_version="1.12.4"
```

Alternatively you can pass it to the `apply` command line, like below.

## Worker node count and instance type

You can control the amount of worker nodes in the cluster as well as their machine type, using the following variables:

 - `TF_VAR_workers_count`
 - `TF_VAR_workers_type`

Export values for them or pass them to the apply command line.

## Build the cluster

```
terraform init
terraform apply -var=kubernetes_version=1.12.4
```

## Exporting K8S variables
To access the cluster you need to export the `KUBECONFIG` variable pointing to the `kubeconfig` file for the current cluster.
```
export KUBECONFIG="$(terraform output kubeconfig_path)"
```

Now you can access the cluster via `kubectl` and you can run acceptance tests against it.

To run acceptance tests, your the following command in the root of the repository.
```
TESTARGS="-run '^TestAcc'" make testacc
```

To run only a specific set of tests, you can replace `^TestAcc` with any regular expression to filter tests by name.
For example, to run tests for Pod resources, you can do:
```
TESTARGS="-run '^TestAccKubernetesPod_'" make testacc
```
