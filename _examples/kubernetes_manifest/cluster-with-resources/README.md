# Provision a cluster on AKS and install Kubernetes manifests

This example demonstrates how to use the provider together with a cluster provisioned on Azure's AKS.

**It's important to be aware that this example will not work as expected when applied with Terraform as a single operation.**

This is because the provider requires access to the Kubernetes API during the planning phase. Initially, before any resources have been created, there is no available API endpoint and the provider will throw an error.

This example demonstrates how to group resources related to the cluster in their own module and the Kubernetes manifests in another separate module, to facilitate building in distinct stages.

It is still possible to build this configuration and obtain a single state file. For this, two Terraform operations are necessary.

Before anything, make sure your environment is configured with valid Azure credentials.
In paricular, have these environment variables set to relevant values in your Azure account:
```
ARM_SUBSCRIPTION_ID￼￼
ARM_TENANT_ID￼￼
ARM_CLIENT_ID (a.k.a. App ID)
ARM_CLIENT_SECRET (a.k.a. password)
```

Initialize the workspace:

```shell
 » terraform init
```

The first Terraform apply operation will build just the AKS cluster and other resources it may require. For this, we use the `-target` argument to Terraform to limit the scope of the apply.

```shell
 » terraform apply -target module.cluster
```

Once this operation succeeds, the AKS cluster is available and the state file contains its attribute values.

Running `terraform state list` should show the following resources present in the state:

```shell
 » terraform state list
module.cluster.azurerm_kubernetes_cluster.test
module.cluster.azurerm_resource_group.test
module.cluster.local_file.kubeconfig
```

At this point the Kubernetes provider is able to access the cluster API. The second apply operation will build the rest of the resources successfully.

Run the second apply operation:

```shell
terraform apply
```

At this point all the resources should be successfully created and present in the state file.

```shell
 » terraform state list
module.cluster.azurerm_kubernetes_cluster.test
module.cluster.azurerm_resource_group.test
module.cluster.local_file.kubeconfig
module.manifests.kubernetes_manifest.test-cfm
```

The provider will also produce a credentials file called `kubeconfig.test` for use with `kubectl` to facilitate validation of the created resources. You can run commands against the cluster like this:

```shell
 » kubectl --kubeconfig kubeconfig.test ...
```
