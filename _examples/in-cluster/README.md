# Example: In-cluster

Running terraform in a kubernetes cluster and using in-cluster config.

## Prerequisites

*This example uses syntax elements specific to Terraform version 0.12+.
It will not work out-of-the-box with Terraform 0.11.x and lower.*


Standard run:

```
# terraform apply \
  -var "minikube_host_ip=$(minikube --profile kubernetes-1.16 ip)"
```

With a custom build:

```
# terraform apply \
  -var "minikube_host_ip=$(minikube --profile kubernetes-1.16 ip)" \
  -var "in_cluster_provider_version=v1.10.1-dev" \
  -var "in_cluster_provider_url=https://storage.googleapis.com/my-custom-bucket/terraform-provider-kubernetes"
```
