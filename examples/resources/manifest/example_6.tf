resource "kubernetes_manifest" "test" {
  provider = kubernetes-alpha

  manifest = {
    // ...
  }

  field_manager {
    # set the name of the field manager
    name = "myteam"

    # force field manager conflicts to be overridden
    force_conflicts = true
  }
}
