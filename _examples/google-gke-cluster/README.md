# Google GKE (Google Container Enginer) cluster

In case you don't have a K8S cluster yet the easiest way
to create one from scratch is to use GKE (Google Container Service).

You can read more about GKE at https://cloud.google.com/container-engine/

## Prerequsities

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
requires region, cluster username and password.

```sh
terraform apply \
	-var 'region=us-west1' \
	-var 'username=MySecretUsername' \
	-var 'password=MySecretPassword'
```

You may also specify the K8S version (see [available versions](https://cloud.google.com/container-engine/release-notes))
and name of the cluster.

```sh
terraform apply \
	-var 'kubernetes_version=1.6.7' \
	-var 'cluster_name=terraform-example-cluster' \
	-var 'region=us-west1' \
	-var 'username=MySecretUsername' \
	-var 'password=MySecretPassword'
```

Afterwards you should see output similar to this one in your console

```
...

Outputs:

additional_zones = [
    us-west1-b
]
cluster_name = terraform-example-cluster
endpoint = 102.186.121.2
node_version = 1.6.7
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
GLBCDefaultBackend is running at https://102.186.121.2/api/v1/namespaces/kube-system/services/default-http-backend/proxy
Heapster is running at https://102.186.121.2/api/v1/namespaces/kube-system/services/heapster/proxy
KubeDNS is running at https://102.186.121.2/api/v1/namespaces/kube-system/services/kube-dns/proxy
kubernetes-dashboard is running at https://102.186.121.2/api/v1/namespaces/kube-system/services/kubernetes-dashboard/proxy

To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.
```

You can also visit the Kubernetes dashboard by opening a proxy

```sh
kubectl proxy
```

which will print out the IP & port on which the proxy is listening

```
Starting to serve on 127.0.0.1:8001
```

then you can either copy & paste that into your browser
or just do this in a separate console session (while keeping the proxy running)

```
open http://127.0.0.1:8001/ui/
```

## Destroying cluster

```sh
terraform destroy \
	-var 'region=us-west1' \
	-var 'username=' \
	-var 'password='
```
