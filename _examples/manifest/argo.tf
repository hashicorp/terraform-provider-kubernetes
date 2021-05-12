# use for_each to read each manifest in argo.yaml 
resource "kubernetes_manifest" "argo" {
  for_each = {
    for i, v in split("\n---\n", file("argo.yaml")) : i => yamldecode(v)
  }

  manifest = jsonencode(each.value)
}
