# GKE (Google Kubernetes Engine)

This example demonstrates the most reliable way to use the Kubernetes provider together with the Google provider to create a GKE cluster. By keeping the two providers' resources in separate Terraform states (or separate workspaces using [Terraform Cloud](https://app.terraform.io/)), we can limit the scope of changes to either the GKE cluster or the Kubernetes resources. This will prevent dependency issues between the Google and Kubernetes providers, since terraform's [provider configurations must be known before a configuration can be applied](https://www.terraform.io/docs/language/providers/configuration.html).

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

## Create GKE cluster

Choose a name for the cluster, or use the terraform config in the current directory to create a random name.

```
terraform init
terraform apply --auto-approve
export CLUSTERNAME=$(terraform output -raw cluster_name)
```

Change into the gke-cluster directory and create the GKE cluster infrastructure.

```
cd gke-cluster
terraform init
terraform apply -var=cluster_name=$CLUSTERNAME
cd -
```

Optionally, the Kubernetes version can be specified at apply time:

```
terraform apply -var=cluster_name=$CLUSTERNAME -var=kubernetes_version=1.18
```

A full list of versions is available per Zone or Region, using the gcloud tool:

```
$ gcloud container get-server-config --flatten="channels" --filter="channels.channel=STABLE" \
     --format="yaml(channels.channel,channels.validVersions)" --region=$GOOGLE_REGION

Fetching server config for us-west1
---
channels:
  channel: STABLE
  validVersions:
  - 1.19.11-gke.2101
  - 1.19.10-gke.1000
  - 1.18.19-gke.1701
  - 1.18.17-gke.1901
```

See Google's [GKE documentation](https://cloud.google.com/kubernetes-engine/versioning) for more information.


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

First, delete the Kubernetes resources as shown below. This will give Ingress and Service related Load Balancers a chance to delete before the other cluster infrastructure is removed.

```
cd kubernetes-config
terraform destroy -var=cluster_name=$CLUSTERNAME
cd -
```

Then delete the GKE related resources:

```
cd gke-cluster
terraform destroy -var=cluster_name=$CLUSTERNAME
cd -
```
