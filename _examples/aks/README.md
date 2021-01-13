# AKS (Azure Kubernetes Service)

This example shows how to use the Terraform Kubernetes Provider and Terraform Helm Provider to configure an AKS cluster. The example builds the AKS cluster and applies the Kubernetes configurations in a single operation.

You will need the following environment variables to be set:

  - `ARM_SUBSCRIPTION_ID`
  - `ARM_TENANT_ID`
  - `ARM_CLIENT_ID`
  - `ARM_CLIENT_SECRET`

See [AWS Provider docs](https://www.terraform.io/docs/providers/aws/index.html#configuration-reference) for more details about these variables and alternatives, like `AWS_PROFILE`.

To install the EKS cluster using default values, run terraform init and apply from the directory containing this README.

```
terraform init
terraform apply
```

## Kubeconfig for manual CLI access

This example generates a kubeconfig file in the current working directory. However, the token in this config expires in 15 minutes. The token can be refreshed by running `terraform apply` again. Export the KUBECONFIG to manually access the cluster:

```
terraform apply
export KUBECONFIG=$(terraform output kubeconfig_path|jq -r)
kubectl get pods -n test
```

## Optional variables

The Kubernetes version can be specified at apply time:

```
terraform apply -var=kubernetes_version=1.18
```

See https://docs.aws.amazon.com/eks/latest/userguide/platform-versions.html for currently available versions.

