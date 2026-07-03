resource "kubernetes_manifest_yaml" "example" {
  yaml_body = <<-YAML
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx
      namespace: default
    spec:
      replicas: 2
      selector:
        matchLabels:
          app: nginx
      template:
        metadata:
          labels:
            app: nginx
        spec:
          containers:
            - name: nginx
              image: nginx:1.27
  YAML

  # Block until the rollout completes; surface pod errors (CrashLoop/ImagePull) on failure.
  wait {
    rollout = true
    timeout = "5m"
  }
}
