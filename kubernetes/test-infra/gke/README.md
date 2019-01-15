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

```
export KUBECONFIG="${KUBECONFIG}:$(terraform output kubeconfig_path)"
export GOOGLE_ZONE=$(terraform output google_zone)
```
