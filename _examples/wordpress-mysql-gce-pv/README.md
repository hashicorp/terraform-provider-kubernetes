# Example: WordPress + MySQL with Persistent Volumes

This example is heavily inspired by https://kubernetes.io/docs/tutorials/stateful-application/mysql-wordpress-persistent-volume/

It describes how to run a persistent installation of [WordPress](https://wordpress.org/)
and [MySQL](https://www.mysql.com/) on Kubernetes.

## Used resources

### Kubernetes Provider

 - `kubernetes_persistent_volume_claim`
 - `kubernetes_persistent_volume`
 - `kubernetes_replication_controller`
 - `kubernetes_secret`
 - `kubernetes_service`

### Google Cloud Provider

 - `google_compute_disk`

## Prerequsites

### Kubernetes

This example expects you to already have a running K8S cluster
and credentials set up in a config or environment variables.

See [related docs](../google-gke-cluster/README.md) if you don't have any of those.

### Google Cloud

We recommend configuring the Google Cloud provider by supplying
environment variables. Check out [related docs](https://www.terraform.io/docs/providers/google/index.html#configuration-reference)
on how to do so, specifically look at arguments `credentials` and `project`.

## Graph

Below is a graph of the all resources we're creating as part of this example
which also demonstrates how they depend on each other.

<img src="https://raw.githubusercontent.com/terraform-providers/terraform-provider-kubernetes/master/_examples/wordpress-mysql-gce-pv/graph.png">

## How to

### Create

First we make sure both providers are downloaded and available

```sh
terraform init
```

then we carry on by creating the real infrastructure which requires
password for the MySQL server and GCP project, region & zone
in which to create persistent disks. Both the region and zone
must match the location of your K8S cluster, otherwise K8S
won't be able to find those disks and claim the space.

```sh
terraform apply \
	-var 'mysql_password=MindTheWeakness' \
	-var 'gcp_region=us-west1' \
	-var 'gcp_zone=us-west1-b'
```

You may also specify version of WordPress and/or MySQL

```sh
terraform apply \
	-var 'mysql_version=5.6' \
	-var 'wordpress_version=4.7.3' \
	-var 'mysql_password=MindTheWeakness' \
	-var 'gcp_region=us-west1' \
	-var 'gcp_zone=us-west1-b'
```

After the `apply` operation has finished you should see output
in your console similar to the one below

```
...

Outputs:

lb_ip = 35.197.11.148
```

This is the IP address of your public load balancer
which exposes the Apache web server serving WordPress.
Open that IP in your browser to see the welcome page.

```sh
open "http://$(terraform output lb_ip)"
```

### Destroy

```
terraform destroy \
	-var 'gcp_region=us-west1' \
	-var 'gcp_zone=us-west1-b' \
	-var 'mysql_password='
```
