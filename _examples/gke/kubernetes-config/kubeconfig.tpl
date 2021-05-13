apiVersion: v1
preferences: {}
kind: Config

clusters:
- cluster:
    server: ${endpoint}
    certificate-authority-data: ${ca_cert}
  name: ${cluster_name}

contexts:
- context:
    cluster: ${cluster_name}
    user: ${cluster_name}
  name: ${cluster_name}

current-context: ${cluster_name}

users:
- name: ${cluster_name}
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1alpha1
      # $KUBECONFIG is overwritten by gcloud tool.
      # So specify a local file to prevent overwriting the system's default kubeconfig.
      env:
        - name: "KUBECONFIG"
          value:  "./kubeconfig"
      command: gcloud
      args:
        - container
        - clusters
        - get-credentials
        - ${cluster_name}
        - --zone
        - ${zone}
        - --project
        - ${project}
