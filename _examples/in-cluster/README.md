# Example: In-cluster

Running terraform in a kubernetes cluster and using in-cluster config.

## Prerequisites

*This example uses syntax elements specific to Terraform version 0.12+.
It will not work out-of-the-box with Terraform 0.11.x and lower.*


Standard run:

```
# terraform apply \
  kubectl apply -f serviceAccount.yaml
  kubectl apply -f role.yaml
  kubectl apply -f clusterRoleBinding.yaml
```



```yaml 
apiVersion: v1
kind: ServiceAccount
metadata:
  name: terraform
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: terraform
rules:
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - create
  - get
  - delete
  - patch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: terraform
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: terraform
subjects:
- kind: ServiceAccount
  name: terraform

```

