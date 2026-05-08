resource "kubernetes_priority_class_v1" "example" {
  metadata {
    name = "terraform-example"
  }

  value = 100
}
