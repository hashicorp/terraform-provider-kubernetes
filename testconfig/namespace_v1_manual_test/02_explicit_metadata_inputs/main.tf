# Scenario 02 — explicit annotations and labels in config (Fix 1)
#
# In the original PR, annotations and labels were Computed-only.
# Any existing config that set them would get:
#   "Cannot set value for this attribute as the provider has marked it as read-only"
#
# After the fix they are Optional+Computed, so this config must be accepted.
# The API response overwrites them in state — the output will show the real
# API values, not the empty maps configured here.

data "kubernetes_namespace_v1" "test" {
  metadata {
    name = "kube-system"

    # Explicitly setting annotations and labels was allowed in SDKv2 (Optional).
    # Must still be accepted after the Framework migration.
    annotations = {}
    labels      = {}
  }
}

output "id"               { value = data.kubernetes_namespace_v1.test.id }
output "name"             { value = data.kubernetes_namespace_v1.test.metadata[0].name }
# The API overwrites the empty maps — real cluster values will appear here:
output "annotations"      { value = data.kubernetes_namespace_v1.test.metadata[0].annotations }
output "labels"           { value = data.kubernetes_namespace_v1.test.metadata[0].labels }
output "spec"             { value = data.kubernetes_namespace_v1.test.spec }
