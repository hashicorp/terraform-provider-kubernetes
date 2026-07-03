# StatefulSets have immutable fields (volumeClaimTemplates, serviceName, selector,
# podManagementPolicy). Changing them requires deleting and recreating the object.
# `force_replace_on` turns such a change into a replacement, and
# `delete { propagation_policy = "Orphan" }` keeps the Pods/PVCs so the recreated
# StatefulSet re-adopts them (an in-place "upgrade" that preserves data).
resource "kubernetes_manifest_yaml" "database" {
  yaml_body = <<-YAML
    apiVersion: apps/v1
    kind: StatefulSet
    metadata:
      name: database
      namespace: default
    spec:
      serviceName: database
      replicas: 3
      selector:
        matchLabels:
          app: database
      template:
        metadata:
          labels:
            app: database
        spec:
          containers:
            - name: db
              image: postgres:16
      volumeClaimTemplates:
        - metadata:
            name: data
          spec:
            accessModes: ["ReadWriteOnce"]
            resources:
              requests:
                storage: 10Gi
  YAML

  force_replace_on = [
    "spec.volumeClaimTemplates",
    "spec.serviceName",
    "spec.selector",
    "spec.podManagementPolicy",
  ]

  delete {
    propagation_policy = "Orphan"
  }

  wait {
    rollout = true
    timeout = "10m"
  }
}
