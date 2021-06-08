# EKS (Amazon Elastic Kubernetes Service)

This example demonstrates the most reliable way to use the Kubernetes provider together with the AWS provider to create an EKS cluster. By keeping the two providers' resources in separate Terraform states (or separate workspaces using [Terraform Cloud](https://app.terraform.io/)), we can limit the scope of changes to either the EKS cluster or the Kubernetes resources. This will prevent dependency issues between the AWS and Kubernetes providers, since terraform's [provider configurations must be known before a configuration can be applied](https://www.terraform.io/docs/language/providers/configuration.html).

You will need the following environment variables to be set:

  - `AWS_ACCESS_KEY_ID`
  - `AWS_SECRET_ACCESS_KEY`

See [AWS Provider docs](https://www.terraform.io/docs/providers/aws/index.html#configuration-reference) for more details about these variables and alternatives, like `AWS_PROFILE`.


## Create EKS cluster

Choose a name for the cluster, or use the terraform config in the current directory to create a random name.

```
terraform init
terraform apply --auto-approve
export CLUSTERNAME=$(terraform output -raw cluster_name)
```

Change into the eks-cluster directory and create the EKS cluster infrastructure.

```
cd eks-cluster
terraform init
terraform apply -var=cluster_name=$CLUSTERNAME
cd -
```

Optionally, the Kubernetes version can be specified at apply time:

```
terraform apply -var=cluster_name=$CLUSTERNAME -var=kubernetes_version=1.18
```

See https://docs.aws.amazon.com/eks/latest/userguide/platform-versions.html for currently available versions.


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

First, delete the Kubernetes resources as shown below. This will give Ingress and Service related Load Balancers a chance to delete before the other AWS resources are removed.

```
cd kubernetes-config
terraform destroy -var=cluster_name=$CLUSTERNAME
cd -
```

Then delete the EKS related resources:

```
cd eks-cluster
terraform destroy -var=cluster_name=$CLUSTERNAME
cd -
```
