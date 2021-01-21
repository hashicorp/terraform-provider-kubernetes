# GKE (Google Container Engine)

This example shows how to use the Terraform Kubernetes Provider and Terraform Helm Provider to configure a GKE cluster. The example config builds the GKE cluster and applies the Kubernetes configurations in a single operation. This guide will also show you how to make changes to the underlying GKE cluster in such a way that Kuberntes/Helm resources are recreated after the underlying cluster is replaced.

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
export KUBECONFIG=$(terraform output -raw kubeconfig_path)
kubectl get pods -n test
```

Alternatively, a longer-lived configuration can be generated using the gcloud tool. Note: this command will overwrite the default kubeconfig at `$HOME/.kube/config`.

```
gcloud container clusters get-credentials $(terraform output -raw cluster_name) --zone $(terraform output -raw google_zone)
kubectl get pods -n test
```

## Replacing the GKE cluster and re-creating the Kubernetes / Helm resources

When the cluster is initially created, the Kubernetes and Helm providers will not be initialized until authentication details are created for the cluster. However, for future operations that may involve replacing the underlying cluster (for example, changing VM sizes), the GKE cluster will have to be targeted without the Kubernetes/Helm providers, as shown below. This is done by removing the `module.kubernetes-config` from Terraform State prior to replacing cluster credentials, to avoid passing outdated credentials into the providers.

This will create the new cluster and the Kubernetes resources in a single apply.

```
terraform state rm module.kubernetes-config
terraform apply
```
