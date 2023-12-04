# Example: In-cluster

Running Terraform in a Kubernetes cluster using in-cluster config.

## Steps

Executing Terraform in a Kubernetes cluster using an in-cluster config would require a service account with appropriate privileges attached to the Pod where Terraform is running.

Below are the necessary steps to create a new service account `terraform` and grant permissions to create a Pod in a `default` namespace using the `kubernetes_pod_v1` Terraform resource as a namespaced resource example.

1. Create a new service account:

    ```yaml
    apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: terraform
    ```

1. Create a Role to grant permissions that are enought to manage Pods via Terraform:

    ```yaml
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
    ```

1. Create a RoleBinding to attach service account `terraform` to the target Role:

    ```yaml
    apiVersion: rbac.authorization.k8s.io/v1
    kind: RoleBinding
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

1. Create a Pod that will initialize and apply Terraform code:

    ```yaml
    apiVersion: v1
    kind: Pod
    metadata:
      name: terraform
    spec:
      serviceAccount: terraform
      initContainers:
        - name: init
          image: "hashicorp/terraform"
          command: [ "terraform", "-chdir=/terraform", "init" ]
          volumeMounts:
          - name: terraform
            mountPath: /terraform
      containers:
        - name: apply
          image: "hashicorp/terraform"
          command: [ "terraform", "-chdir=/terraform", "apply", "-auto-approve" ]
          volumeMounts:
          - name: terraform
            mountPath: /terraform
      volumes:
        - name: terraform
          persistentVolumeClaim:
            claimName: terraform
      restartPolicy: Never
    ```

Terraform code example that will work with the above configuration resides in files [`provider.tf`](provider.tf) and [`pod.tf`](pod.tf). As you can see, the provider configuration block is empty. In this case, all the necessary privileges are granted via the service account.

Let's extend the previous example with privileges that are enough to create a Namespace using the `kubernetes_namespace_v1` Terraform resource as a cluster-level resource example.

1. Create a ClusterRole to grant permissions that are enought to manage Pods via Terraform:

    ```yaml
    apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      name: terraform
    rules:
    - apiGroups:
      - ""
      resources:
      - namespaces
      verbs:
      - create
      - get
      - delete
      - list
      - patch
      - update
    ```

1. Create a ClusterRoleBinding to attach service account `terraform` to the target ClusterRole:

    ```yaml
    apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      name: terraform
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: terraform
    subjects:
    - kind: ServiceAccount
      name: terraform
      namespace: default
    ```

Terraform code example can be extanded with [`namespace.tf`](namespace.tf) file. To apply changes restart the Pod where Terraform is running.

Please, always consult with the security team and follow the guidance accepted in your organization when granting RBAC privileges in a Kubernetes cluster.
