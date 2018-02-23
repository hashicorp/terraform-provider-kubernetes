# minikube @ Packet

You will need `PACKET_AUTH_TOKEN` to be set.

See [Packet Provider docs](https://www.terraform.io/docs/providers/packet/index.html#configuration-reference) for more details about configuration.

`route53_zone` has to be a valid domain (zone in Cloud DNS) which has correctly set and propagated NS records, i.e. it is reachable from outside.

```
terraform init
terraform apply -var=kubernetes_version=1.6.4
```

## Exporting K8S variables

```
export KUBE_HOST=https://localhost:$(terraform output local_tunnel_port)
export KUBE_USER=minikube
export KUBE_PASSWORD=minikube
export KUBE_CLIENT_CERT_DATA="$(cat $PWD/$(terraform output dotminikube_path)/client.crt)"
export KUBE_CLIENT_KEY_DATA="$(cat $PWD/$(terraform output dotminikube_path)/client.key)"
export KUBE_CLUSTER_CA_CERT_DATA="$(cat $PWD/$(terraform output dotminikube_path)/ca.crt)"
```
