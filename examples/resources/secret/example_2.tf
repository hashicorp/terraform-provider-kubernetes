resource "kubernetes_secret" "example" {
  metadata {
    name = "docker-cfg"
  }

  data = {
    ".dockerconfigjson" = "${file("${path.module}/.docker/config.json")}"
  }

  type = "kubernetes.io/dockerconfigjson"
}
