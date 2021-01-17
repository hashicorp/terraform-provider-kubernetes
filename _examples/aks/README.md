# AKS (Azure Kubernetes Service)

This example shows how to use the Terraform Kubernetes Provider and Terraform Helm Provider to configure an AKS cluster. The example config in this directory builds the AKS cluster and applies the Kubernetes configurations in a single operation. This guide will also show you how to make changes to the underlying AKS cluster in such a way that Kuberntes/Helm resources are recreated after the underlying cluster is replaced.

You will need the following environment variables to be set:

  - `ARM_SUBSCRIPTION_ID`
  - `ARM_TENANT_ID`
  - `ARM_CLIENT_ID`
  - `ARM_CLIENT_SECRET`

See [AWS Provider docs](https://www.terraform.io/docs/providers/aws/index.html#configuration-reference) for more details about these variables and alternatives, like `AWS_PROFILE`.

To install the AKS cluster using default values, run terraform init and apply from the directory containing this README.

```
terraform init
terraform apply
```

## Kubeconfig for manual CLI access

This example generates a kubeconfig file in the current working directory, which can be used for manual CLI access to the cluster.

```
export KUBECONFIG=$(terraform output kubeconfig_path|jq -r)
kubectl get pods -n test
```

However, in a real-world scenario, this config file would have to be replaced periodically as the AKS client certificates eventually expire (see the [Azure documentation](https://docs.microsoft.com/en-us/azure/aks/certificate-rotation) for the exact expiry dates). If the certificates are replaced, the AKS module will have to be targeted to pull in the new credentials before they can be passed into the Kubernetes or Helm providers.

```
terraform state rm module.kubernetes-config
terraform plan
terraform apply
export KUBECONFIG=$(terraform output kubeconfig_path|jq -r)
kubectl get pods -n test
```

This approach prevents the Kubernetes and Helm provider from using cached, invalid credentials, which would cause provider configuration errors durring the plan and apply phases. (The resources that were previously deployed will not be affected by the `state rm`).

## Replacing the AKS cluster, or its authentication credentials

When the cluster is initially created, the Kubernetes and Helm providers will not be initialized until authentication details are created for the cluster. However, for future operations that may involve replacing the underlying cluster (for example, changing VM sizes), the AKS cluster will have to be targeted without the Kubernetes/Helm providers, as shown below. This is done by removing the `module.kubernetes-config` from Terraform State prior to replacing cluster credentials, to avoid passing outdated credentials into the providers.

This will create the new cluster and the Kubernetes resources in a single apply. If this is being applied to an existing cluster (such as in the case of credential rotation), the existing Kubernetes/Helm resources will continue running and simply undergo a credential refresh.

```
terraform state rm module.kubernetes-config
terraform apply
```
