# Step 2 — identical config, local framework binary.
# Copy of step1 config — must be byte-for-byte identical so that any diff
# is caused by the provider change, not the config.

data "kubernetes_namespace_v1" "test" {
  metadata {
    name = "kube-system"
  }
}

resource "terraform_data" "anchor" {
  input = {
    id               = data.kubernetes_namespace_v1.test.id
    name             = data.kubernetes_namespace_v1.test.metadata[0].name
    uid              = data.kubernetes_namespace_v1.test.metadata[0].uid
    resource_version = data.kubernetes_namespace_v1.test.metadata[0].resource_version
    generation       = data.kubernetes_namespace_v1.test.metadata[0].generation
    labels           = data.kubernetes_namespace_v1.test.metadata[0].labels
    annotations      = data.kubernetes_namespace_v1.test.metadata[0].annotations
    spec             = data.kubernetes_namespace_v1.test.spec
  }
}
