provider kubernetes {
  config_path = "~/.kube/config"
}


# NOTE run kubectl create cm patch-demo first 
resource kubernetes_patch example {
  kind = "ConfigMap"

  metadata {
    name = "patch-demo"
  }

  patch = jsonencode({
    metadata = {
      labels = {
        demo = "314"
      }
    },
    data = {
      demo = "456"
    }
  })

  patch_type = "strategic"
}