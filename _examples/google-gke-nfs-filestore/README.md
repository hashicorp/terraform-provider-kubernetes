# GKE (Google Kubernetes Engine) and Google Filestore (NFS)

This example demonstrates building a Google GKE cluster and Google Filestore (NFS) to create persistent volumes for use in applications in Kubernetes. There is an example application Deployment included to demonstrate mounting the NFS volume using a Persistent Volume Claim.

You will need the following environment variables to be set:

 - `GOOGLE_CREDENTIALS`
 - `GOOGLE_PROJECT`
 - `GOOGLE_REGION`

For example:
```
[myuser@linux ~]$ env | grep GOOGLE
GOOGLE_REGION=us-west1
GOOGLE_CREDENTIALS=/home/myuser/.config/gcloud/legacy_credentials/mygoogleuser/adc.json
GOOGLE_PROJECT=my-gcp-project
```

See [Google Cloud Provider docs](https://www.terraform.io/docs/providers/google/index.html#configuration-reference) for more details about these variables.

Install the example using the GKE default version of Kubernetes:
```
terraform init
terraform apply
```

Or optionally specify a version to use:
```
terraform apply -var=kubernetes_version=1.15
```

## Versions

Other available versions of Kubernetes can be found by running the [gcloud](https://cloud.google.com/sdk/docs#install_the_latest_cloud_tools_version_cloudsdk_current_version) tool. However, be aware that your chosen version must be present in the `validMasterVersions`, or else the GKE `defaultClusterVersion` will be used.

```
gcloud container get-server-config --region $GOOGLE_REGION
```

## Exporting K8S variables
To access the cluster you need to export the `KUBECONFIG` variable pointing to the `kubeconfig` file for the current cluster.
```
export KUBECONFIG="$(terraform output kubeconfig_path)"
export GOOGLE_ZONE=$(terraform output google_zone)
```

Now you can access the cluster via `kubectl`.
