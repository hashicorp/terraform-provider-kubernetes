# Scenario 01 — basic read
#
# Reads kube-system and my-test-ns (both real namespaces on the cluster).
# Verifies: id, name, uid, resource_version, generation, labels, annotations,
# and spec.finalizers are all populated correctly by the framework implementation.
# Also exercises the deferred-read path: the second data source's name comes from
# the first data source's id output (unknown at plan time).

# Direct lookup by literal name.
data "kubernetes_namespace_v1" "kube_system" {
  metadata {
    name = "kube-system"
  }
}

# Deferred read: name is known only after apply (Fix 2 — deferred spec).
# If spec were still a ListNestedBlock this would fail with:
#   "Provider produced inconsistent final plan / new element 0 has appeared"
data "kubernetes_namespace_v1" "my_test_ns" {
  metadata {
    name = data.kubernetes_namespace_v1.kube_system.id
  }
}

output "kube_system_id"               { value = data.kubernetes_namespace_v1.kube_system.id }
output "kube_system_name"             { value = data.kubernetes_namespace_v1.kube_system.metadata[0].name }
output "kube_system_uid"              { value = data.kubernetes_namespace_v1.kube_system.metadata[0].uid }
output "kube_system_resource_version" { value = data.kubernetes_namespace_v1.kube_system.metadata[0].resource_version }
output "kube_system_generation"       { value = data.kubernetes_namespace_v1.kube_system.metadata[0].generation }
output "kube_system_labels"           { value = data.kubernetes_namespace_v1.kube_system.metadata[0].labels }
output "kube_system_annotations"      { value = data.kubernetes_namespace_v1.kube_system.metadata[0].annotations }
output "kube_system_spec"             { value = data.kubernetes_namespace_v1.kube_system.spec }

output "my_test_ns_id"   { value = data.kubernetes_namespace_v1.my_test_ns.id }
output "my_test_ns_spec" { value = data.kubernetes_namespace_v1.my_test_ns.spec }
