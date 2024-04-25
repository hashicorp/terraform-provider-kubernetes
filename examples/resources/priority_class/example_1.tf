resource "kubernetes_priority_class" "example" {
  metadata {
    name = "terraform-example"
  }

  value = 100
}
