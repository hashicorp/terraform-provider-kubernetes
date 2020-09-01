# LKE (Linode Kubernetes Engine)

You will need to have the `LINODE_API_TOKEN` environment variable set to authenticate with the API. See the [Linode Terraform Provider docs](https://www.terraform.io/docs/providers/linode/index.html) for more information.

## Versions

Determine the supported Kubernetes versions via the Linode CLI.

```
linode-cli lke versions-list
```

Additionally, you can use the following API endpoint.

```sh
curl https://api.linode.com/v4/lke/versions
```

## Linode worker node types

Determine the supported Linode instance types to spin up as worker nodes via the following API endpoint.

```sh
curl https://api.linode.com/v4/linode/types
```

## Variables

The following variables can be set via their Environment variable bindings.

- `TF_VAR_kubernetes_version`
- `TF_VAR_workers_count` - amount of Linodes to spin up for cluster.
- `TF_VAR_workers_type` - type of Linodes to spin up for cluster.

Export values for them or pass them to the apply command line.

## Build the cluster

```
terraform init
LINODE_API_TOKEN="XXXXXXXXXXXXXXXX" \
    TF_VAR_kubernetes_version=1.17 \
    TF_VAR_workers_count=3 \
    TF_VAR_workers_type=g6-standard-2 \
    terraform apply --auto-approve
```

## Acceptance test usage

The path to the resulting kubeconfig file to access the provisioned cluster will be provided under the output name `kubeconfig_path`.

```sh
export KUBECONFIG="$(terraform output kubeconfig_path)"
```

Now you can access the cluster via `kubectl` and you can run acceptance tests against it.

To run acceptance tests, your the following command in the root of the repository.

```sh
TESTARGS="-run '^TestAcc'" make testacc
```

To run only a specific set of tests, you can replace ^TestAcc with any regular expression to filter tests by name. For example, to run tests for Pod resources, you can do:

```sh
TESTARGS="-run '^TestAccKubernetesPod_'" make testacc
```
