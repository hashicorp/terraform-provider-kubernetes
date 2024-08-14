resource "kubernetes_config_map_v1_data" "example" {
  metadata {
    name = "my-config"
  }
  data = {
    "owner" = "myteam"
  }
}
