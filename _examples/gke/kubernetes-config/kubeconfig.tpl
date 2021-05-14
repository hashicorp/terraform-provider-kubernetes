apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: ${ca_cert}
    server: ${endpoint}
  name: gke_terraform-strategic-providers_us-west1-a_k8s-acc-5642
contexts:
- context:
    cluster: gke_${project}_${zone}_${cluster_name}
    user: gke_${project}_${zone}_${cluster_name}
  name: gke_${project}_${zone}_${cluster_name}
current-context: gke_${project}_${zone}_${cluster_name}
kind: Config
preferences: {}
users:
- name: gke_${project}_${zone}_${cluster_name}
  user:
    auth-provider:
      config:
        cmd-args: config config-helper --format=json
        cmd-path: gcloud
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp
