# GKE (Google Container Engine)

This example shows how to use the Terraform Kubernetes Provider and Terraform Helm Provider to configure a GKE cluster. The example builds the GKE cluster and applies the Kubernetes configurations in a single operation.

You will need the following environment variables to be set:

 - `GOOGLE_CREDENTIALS`
 - `GOOGLE_PROJECT`
 - `GOOGLE_REGION`

For example:
```
$ env | grep GOOGLE
GOOGLE_REGION=us-west1
GOOGLE_CREDENTIALS=/home/myuser/.config/gcloud/legacy_credentials/mygoogleuser/adc.json
GOOGLE_PROJECT=my-gcp-project
```

See [Google Provider docs](https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/provider_reference#full-reference) for more details about these variables.

Ensure that `KUBE_CONFIG_FILE` and `KUBE_CONFIG_FILES` environment variables are NOT set, as they will interfere with the cluster build.

```
unset KUBE_CONFIG_FILE
unset KUBE_CONFIG_FILES
```

To install the GKE cluster using default values, run terraform init and apply from the directory containing this README.

```
terraform init
terraform apply
```

Optionally, the Kubernetes version can also be specified:

```
terraform apply -var=kubernetes_version=1.18
```


## Versions

Valid versions for the GKE cluster can be found by using the gcloud tool.

```
gcloud container get-server-config --region $GOOGLE_REGION
```

## Kubeconfig for manual CLI access

This example generates a kubeconfig file in the current working directory. However, the token in this config will expire after 1 hour. The token can be refreshed by running `terraform apply` again.

```
terraform apply
export KUBECONFIG=$(terraform output kubeconfig_path|jq -r)
kubectl get pods -n test
```

Alternatively, a longer-lived configuration can be generated using the gcloud tool. Note: this command will overwrite the default kubeconfig at `$HOME/.kube/config`.

```
gcloud container clusters get-credentials $(terraform output cluster_name|jq -r) --zone $(terraform output google_zone |jq -r)
kubectl get pods -n test
```

