# Scenario 03 — not-found (Fix 3)
#
# In the original PR, the 404 path returned without writing state, leaving id=null.
# Any config that referenced .id in a string or downstream resource would fail:
#   "The expression result is null. Cannot include a null value in a string template."
#
# After the fix, id equals the requested name even when the namespace does not exist.
# Apply must succeed without errors; spec is empty.

data "kubernetes_namespace_v1" "missing" {
  metadata {
    name = "this-namespace-does-not-exist"
  }
}

# This interpolation would fail with a null id before the fix.
output "lookup_string" {
  value = "namespace:${data.kubernetes_namespace_v1.missing.id}"
}

output "id"   { value = data.kubernetes_namespace_v1.missing.id }
output "name" { value = data.kubernetes_namespace_v1.missing.metadata[0].name }
output "spec" { value = data.kubernetes_namespace_v1.missing.spec }
