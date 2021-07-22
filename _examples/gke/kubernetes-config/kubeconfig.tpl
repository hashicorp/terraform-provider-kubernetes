apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: ${ca_cert}
    server: ${endpoint}
  name: gke_${project}_${zone}_${cluster_name}
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
        cmd-path: /var/lib/snapd/snap/bin/gcloud
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp
