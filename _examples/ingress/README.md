# Example: Ingress

## Used resources

 - `kubernetes_deployment`
 - `kubernetes_service`
 - `kubernetes_ingress`

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

After the `apply` operation has finished you should see output
in your console similar to the one below

```
...

Outputs:

ingress_ip = 35.197.9.247
```

This is the IP address of your public load balancer
which exposes the web server. Open that IP in your
browser to see the nginx welcome page.

```sh
open "http://$(terraform output ingress_ip)"
```

### Destroy

```
terraform destroy
```
