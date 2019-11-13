# Example: In-cluster

## Prerequisites

*This example uses syntax elements specific to Terraform version 0.12+.
It will not work out-of-the-box with Terraform 0.11.x and lower.*


```
# terraform apply -var "minikube_host_ip=$(minikube --profile kubernetes-1.16 ip)" -var "minikube_target_ip=$(minikube --profile kubernetes-1.15 ip)"
```