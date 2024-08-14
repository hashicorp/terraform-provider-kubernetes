resource "kubernetes_runtime_class_v1" "example" {
  metadata {
    name = "myclass"
  }
  handler = "abcdeagh"
}
