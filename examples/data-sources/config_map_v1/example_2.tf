data "kubernetes_config_map_v1" "example" {
  metadata {
    name = "my-config"
  }
}
