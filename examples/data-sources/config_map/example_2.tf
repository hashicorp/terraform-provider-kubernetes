data "kubernetes_config_map" "example" {
  metadata {
    name = "my-config"
  }
}
