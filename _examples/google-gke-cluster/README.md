# Google GKE (Google Container Engine) cluster

In case you don't have a K8S cluster yet the easiest way
to create one from scratch is to use GKE (Google Container Service).

You can read more about GKE at https://cloud.google.com/container-engine/

## Prerequisites

*This example uses syntax elements specific to Terraform version 0.12+.
It will not work out-of-the-box with Terraform 0.11.x and lower.*

Configure the Google Cloud provider by supplying environment variables
and/or standard config files.
Check out [related docs](https://www.terraform.io/docs/providers/google/index.html#configuration-reference)
on how to do so, specifically look at arguments `credentials` and `project`.

## Creating cluster

First we make sure the Google provider is downloaded and available

```sh
terraform init
```

then we carry on by creating the real infrastructure which
requires GCP project, region, cluster username and password.

```sh
terraform apply \
	-var 'gcp_project=my-project' \
	-var 'gcp_region=us-west1' \
	-var 'username=MySecretUsername' \
	-var 'password=MySecretPassword'
```

You may also specify the minimal K8S version (see [available versions](https://cloud.google.com/kubernetes-engine/docs/release-notes))
and name of the cluster.

```sh
terraform apply \
	-var 'kubernetes_version=1.16.8' \
	-var 'cluster_name=terraform-example-cluster' \
	-var 'gcp_project=my-project' \
	-var 'gcp_region=us-west1' \
	-var 'username=MySecretUsername' \
	-var 'password=MySecretPassword'
```

Afterwards you should see output similar to this one in your console

```
...

Outputs:

cluster_name = terraform-example-cluster
endpoint = 102.186.121.2
node_version = 1.16.8
primary_zone = us-west1-a
```

## Credentials

It is generally a good practice not to hard-code credentials
in your source code and use environment variables and/or standard config instead.
Check out [the relevant docs](https://www.terraform.io/docs/providers/kubernetes/index.html#argument-reference)
for all supported provider arguments, most of which are related to authentication.

Once you have a cluster up and running on GKE the easiest way to supply
credentials to the provider is via the following set of steps:

```sh
gcloud container clusters get-credentials \
	$(terraform output cluster_name) \
	--zone=$(terraform output primary_zone)
```

Afterwards you should have a set of valid credentials stored
in the config at default location where the Kubernetes provider
can find them. The current context will also be automatically
pointed to those new credentials (in case you have any other
credentials there in an existing config).

You can verify this by running

```sh
kubectl cluster-info
```

which should provide output similar to the one below

```
Kubernetes master is running at https://102.186.121.2
GLBCDefaultBackend is running at https://102.186.121.2/api/v1/namespaces/kube-system/services/default-http-backend:http/proxy
Heapster is running at https://102.186.121.2/api/v1/namespaces/kube-system/services/heapster/proxy
KubeDNS is running at https://102.186.121.2/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy
Metrics-server is running at https://102.186.121.2/api/v1/namespaces/kube-system/services/https:metrics-server:/proxy

To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.
```

## Destroying cluster

```sh
terraform destroy \
	-var 'gcp_project=my-project' \
	-var 'gcp_region=us-west1' \
	-var 'username=' \
	-var 'password='
```
