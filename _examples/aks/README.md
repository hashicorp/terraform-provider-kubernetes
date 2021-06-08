# AKS (Azure Kubernetes Service)

This example demonstrates the most reliable way to use the Kubernetes provider together with the Azurerm provider to create an AKS cluster. By keeping the two providers' resources in separate Terraform states (or separate workspaces using [Terraform Cloud](https://app.terraform.io/)), we can limit the scope of changes to either the AKS cluster or the Kubernetes resources. This will prevent dependency issues between the Azurerm and Kubernetes providers, since terraform's [provider configurations must be known before a configuration can be applied](https://www.terraform.io/docs/language/providers/configuration.html).

You will need the following environment variables to be set:

  - `ARM_SUBSCRIPTION_ID`
  - `ARM_TENANT_ID`
  - `ARM_CLIENT_ID`
  - `ARM_CLIENT_SECRET`

Ensure that `KUBE_CONFIG_FILE` and `KUBE_CONFIG_FILES` environment variables are NOT set, as they will interfere with the cluster build.

```
unset KUBE_CONFIG_FILE
unset KUBE_CONFIG_FILES
```

## Create AKS cluster

Choose a name for the cluster, or use the terraform config in the current directory to create a random name.

```
terraform init
terraform apply --auto-approve
export CLUSTERNAME=$(terraform output -raw cluster_name)
```

Change into the aks-cluster directory and create the AKS cluster infrastructure.

```
cd aks-cluster
terraform init
terraform apply -var=cluster_name=$CLUSTERNAME
cd -
```

### Optional: specify Kubernetes version

Choose a version of Kubernetes available in the location of your choice. For example:

```
$ az aks get-versions --location westus2 --output table
KubernetesVersion    Upgrades
-------------------  ------------------------
1.21.1(preview)      None available
1.20.7               1.21.1(preview)
1.20.5               1.20.7, 1.21.1(preview)
1.19.11              1.20.5, 1.20.7
1.19.9               1.19.11, 1.20.5, 1.20.7
1.18.19              1.19.9, 1.19.11
1.18.17              1.18.19, 1.19.9, 1.19.11
```

Then use the `kubernetes_version` variable to install the cluster:

```
terraform apply -var=kubernetes_version=1.21.1 -var=location=westus2
```

## Create Kubernetes resources

Change into the kubernetes-config directory to apply Kubernetes resources to the new cluster.

```
cd kubernetes-config
terraform init
terraform apply -var=cluster_name=$CLUSTERNAME
```

### Kubeconfig for manual CLI access

This example generates a kubeconfig file which can be used for manual CLI access to the cluster.

```
cd kubernetes-config
export KUBECONFIG=$(terraform output -raw kubeconfig)
kubectl get pods -n test
```

## Deleting the cluster

First, delete the Kubernetes resources as shown below. This will give Ingress and Service related Load Balancers a chance to delete before the other Azure resources are removed.

```
cd kubernetes-config
terraform destroy -var=cluster_name=$CLUSTERNAME
cd -
```

Then delete the AKS related resources:

```
cd aks-cluster
terraform destroy -var=cluster_name=$CLUSTERNAME
cd -
```
