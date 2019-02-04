# GKE (Google Container Engine)

You will need the following environment variables to be set:

 - `GOOGLE_CREDENTIALS`
 - `GOOGLE_PROJECT`
 - `GOOGLE_REGION`

See [Google Cloud Provider docs](https://www.terraform.io/docs/providers/google/index.html#configuration-reference) for more details about these variables.

```
terraform init
terraform apply -var=kubernetes_version=1.7.12-gke.1
```

## Versions

See https://cloud.google.com/kubernetes-engine/versioning-and-upgrades#versions_available_for_new_cluster_masters for currently available versions.

## Exporting K8S variables
To access the cluster you need to export the `KUBECONFIG` variable pointing to the `kubeconfig` file for the current cluster.
```
export KUBECONFIG="$(terraform output kubeconfig_path)"
export GOOGLE_ZONE=$(terraform output google_zone)
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
