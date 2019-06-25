# Example: NGINX (HTTP)

This example is heavily inspired by https://github.com/kubernetes/examples/tree/master/staging/https-nginx

It shows how to spin up a basic HTTP server on Kubernetes using [nginx](https://www.nginx.com)
which is exposed to the internet through a load balancer (provisioned automatically by K8S).

## Used resources

 - `kubernetes_replication_controller`
 - `kubernetes_service`

## Prerequsites

*This example uses syntax elements specific to Terraform version 0.12+.
It will not work out-of-the-box with Terraform 0.11.x and lower.*

This example expects you to already have a running K8S cluster
and credentials set up in a config or environment variables.

See [related docs](../google-gke-cluster/README.md) if you don't have any of those.

## How to

### Create

First we make sure the Kubernetes provider is downloaded and available

```sh
terraform init
```

then we carry on by creating the resources

```sh
terraform apply
```

you may optionally specify the version of nginx like this

```sh
terraform apply -var 'nginx_version=1.7.8'
```

After the `apply` operation has finished you should see output
in your console similar to the one below

```
...

Outputs:

lb_ip = 35.197.9.247
```

This is the IP address of your public load balancer
which exposes the web server. Open that IP in your
browser to see the nginx welcome page.

```sh
open "http://$(terraform output lb_ip)"
```

### Destroy

```
terraform destroy
```
