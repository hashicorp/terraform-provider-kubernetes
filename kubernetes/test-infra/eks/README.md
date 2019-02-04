# Amazon EKS Clusters

You will need the standard AWS environment variables to be set, e.g.

  - `AWS_ACCESS_KEY_ID`
  - `AWS_SECRET_ACCESS_KEY`

See [AWS Provider docs](https://www.terraform.io/docs/providers/aws/index.html#configuration-reference) for more details about these variables
and alternatives, like `AWS_PROFILE`.

## Versions

You can set the desired version of Kubernetes via the `kubernetes_version` TF variable.

See https://docs.aws.amazon.com/eks/latest/userguide/platform-versions.html for currently available versions.

You can set the desired version of Kubernetes via the `kubernetes_version` TF variable, like this:
```
export TF_VAR_kubernetes_version="1.11"
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
terraform apply -var=kubernetes_version=1.11
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
