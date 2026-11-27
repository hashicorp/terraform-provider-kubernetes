resource "kubernetes_manifest_v2" "service" {
  yaml_body = <<-YAML
    apiVersion: v1
    kind: Service
    metadata:
      name: web
      namespace: default
      labels:
        app.kubernetes.io/name: web
    spec:
      type: ClusterIP
      selector:
        app.kubernetes.io/name: web
      ports:
      - name: grpc
        port: 80
        targetPort: 4245
    YAML
}
