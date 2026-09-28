# Step 1 — establish SDKv2 state using the published 3.2.1 provider.
# This is the state a real user would have before upgrading.
# Uses kube-system so the test is self-contained (always exists).

data "kubernetes_namespace_v1" "test" {
  metadata {
    name = "kube-system"
  }
}

# terraform_data anchors every attribute value into managed resource state,
# making the diff visible if any value changes after the provider upgrade.
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
