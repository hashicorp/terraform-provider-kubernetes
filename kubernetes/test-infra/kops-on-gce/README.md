# kops @ GCE

You will need the following environment variables to be set:

 - `GOOGLE_CREDENTIALS`
 - `GOOGLE_PROJECT`
 - `GOOGLE_REGION`

See [Google Cloud Provider docs](https://www.terraform.io/docs/providers/google/index.html#configuration-reference) for more details about these variables.

`route53_zone` has to be a valid domain (zone in Cloud DNS) which has correctly set and propagated NS records, i.e. it is reachable from outside.

```
terraform init
terraform apply -var=kubernetes_version=1.6.13 -var=zone_name=tfacc.testingdomain.com
```

## Exporting K8S variables

```
export CLUSTER_NAME=$(terraform output cluster_name)
export KUBE_HOST=$(kubectl config view -o jsonpath="{.clusters[?(@.name == \"${CLUSTER_NAME}\")].cluster.server}")
export KUBE_USER=$(kubectl config view -o jsonpath="{.users[?(@.name == \"${CLUSTER_NAME}\")].user.username}")
export KUBE_PASSWORD=$(kubectl config view -o jsonpath="{.users[?(@.name == \"${CLUSTER_NAME}\")].user.password}")
export KUBE_CLIENT_CERT_DATA="$(kubectl config view --raw=true -o json | jq -r ".users[] | select(.name==\"${CLUSTER_NAME}\") | .user[\"client-certificate-data\"]" | base64 -d -)"
export KUBE_CLIENT_KEY_DATA="$(kubectl config view --raw=true -o json | jq -r ".users[] | select(.name==\"${CLUSTER_NAME}\") | .user[\"client-key-data\"]" | base64 -d -)"
export KUBE_CLUSTER_CA_CERT_DATA="$(kubectl config --raw=true view -o json | jq -r ".clusters[] | select(.name==\"${CLUSTER_NAME}\") | .cluster[\"certificate-authority-data\"]" | base64 -d -)"
```
