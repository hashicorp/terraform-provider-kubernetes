# AKS (Azure Kubernetes Service)

This example shows how to use the Terraform Kubernetes Provider and Terraform Helm Provider to configure an AKS cluster. The example config in this directory builds the AKS cluster and applies the Kubernetes configurations in a single operation. This guide will also show you how to make changes to the underlying AKS cluster in such a way that Kuberntes/Helm resources are recreated after the underlying cluster is replaced.

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

To install the AKS cluster using default values, run terraform init and apply from the directory containing this README.

```
terraform init
terraform apply
```

## Kubeconfig for manual CLI access

This example generates a kubeconfig file in the current working directory, which can be used for manual CLI access to the cluster.

```
export KUBECONFIG=$(terraform output -raw kubeconfig_path)
kubectl get pods -n test
```

However, in a real-world scenario, this config file would have to be replaced periodically as the AKS client certificates eventually expire (see the [Azure documentation](https://docs.microsoft.com/en-us/azure/aks/certificate-rotation) for the exact expiry dates). If the certificates (or other authentication attributes) are replaced, run a targeted `terraform apply` to save the new credentials into state.

```
terraform plan -target=module.aks-cluster
terraform apply -target=module.aks-cluster
```

Once the targeted apply is finished, the Kubernetes and Helm providers will be available for use again. Run `terraform apply` again (without targeting) to apply any updates to Kubernetes resources.

```
terraform plan
terraform apply
```

This approach prevents the Kubernetes and Helm providers from attempting to use cached, invalid credentials, which would cause provider configuration errors durring the plan and apply phases.

## Replacing the AKS cluster and re-creating the Kubernetes / Helm resources

When the cluster is initially created, the Kubernetes and Helm providers will not be initialized until authentication details are created for the cluster. However, for future operations that may involve replacing the underlying cluster (for example, changing VM sizes), the AKS cluster will have to be targeted without the Kubernetes/Helm providers, as shown below. This is done by removing the `module.kubernetes-config` from Terraform State prior to replacing cluster credentials, to avoid passing outdated credentials into the providers.

This will create the new cluster and the Kubernetes resources in a single apply.

```
terraform state rm module.kubernetes-config
terraform apply
```
