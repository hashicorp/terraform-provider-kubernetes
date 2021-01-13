# EKS (Amazon Elastic Kubernetes Service)

This example shows how to use the Terraform Kubernetes Provider and Terraform Helm Provider to configure an EKS cluster. The example builds the EKS cluster and applies the Kubernetes configurations in a single operation.

You will need the following environment variables to be set:

  - `AWS_ACCESS_KEY_ID`
  - `AWS_SECRET_ACCESS_KEY`

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


### Worker node count and instance type

The number of worker nodes, and the instance type, can be specified at apply time:

```
terraform apply -var=workers_count=4 -var=workers_type=m4.xlarge
```

## Additional configuration of EKS

To view all available configuration options for the EKS module used in this example, see [terraform-aws-modules/eks docs](https://registry.terraform.io/modules/terraform-aws-modules/eks/aws/latest).
