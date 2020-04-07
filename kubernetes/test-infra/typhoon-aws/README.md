# Typhoon Kubernetes clusters on AWS

This environment deploys a Kubernetes cluster using the Typhoon distribution. See here for details: https://github.com/poseidon/typhoon

You will need the standard AWS environment variables to be set, e.g.

  - `AWS_ACCESS_KEY_ID`
  - `AWS_SECRET_ACCESS_KEY`

See [AWS Provider docs](https://www.terraform.io/docs/providers/aws/index.html#configuration-reference) for more details about these variables
and alternatives, like `AWS_PROFILE`.

Additionally, a publicly accesible DNS domain registered as a Route53 managed zone is required.
The name of the domain should be passed to terraform via the `base_domain` input variable.

Example:

```export TF_VAR_base_domain=k8s.myself.com```
## Versions

You can set the desired version of Kubernetes by editing the `main.tf` configuration file and replacing the version in the source URL of the `typhoon-acc` module.

Example:
```

module "typhoon-acc" {
  source = "git::https://github.com/poseidon/typhoon//aws/fedora-coreos/kubernetes?ref=v1.18.0" # set the desired Kubernetes version here
...
```
## Worker node count and instance type

You can control the amount of worker nodes in the cluster as well as their machine type, using the following variables:

 - `TF_VAR_controller_count`
 - `TF_VAR_controller_type`
 - `TF_VAR_workers_count`
 - `TF_VAR_workers_type`

Export values for them or pass them to the apply command line.

## Build the cluster

```
terraform init
terraform apply -var=cluster_name=typhoon
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
