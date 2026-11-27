resource "kubernetes_manifest_patch" "cluster_dns" {
  yaml_body = <<-EOT
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: cluster-dns
      namespace: kube-system
    spec:
      template:
        spec:
          tolerations:
          - key: CriticalAddonsOnly
            operator: Exists
          - key: node-role.kubernetes.io/control-plane
            effect: NoSchedule
          - key: dedicated
            value: system
            effect: NoSchedule
  EOT

  # The add-on owns the list of tolerations.
  field_manager {
    force_conflicts = true
  }
}
