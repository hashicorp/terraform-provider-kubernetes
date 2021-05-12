provider kubernetes {
  config_path = "~/.kube/config"
}

resource "kubernetes_manifest" "crd_test" {
  manifest = jsonencode(yamldecode(file("object_crd.yaml")))
}

resource "kubernetes_manifest" "jim" {
  depends_on = [
    kubernetes_manifest.crd_test
  ]

  manifest = jsonencode({
    apiVersion = "k8s.terraform.io/v1alpha1"
    kind = "Person"
    metadata = {
      name = "jim"
    }
    details = {
      firstName = "Jim"
      lastName  = "Jones"
      age       = 27
      org       = "Peoples Temple"
    }
  })
}

resource "kubernetes_manifest" "david" {
  depends_on = [
    kubernetes_manifest.crd_test
  ]

  manifest = jsonencode({
    apiVersion = "k8s.terraform.io/v1alpha1"
    kind = "Person"
    metadata = {
      name = "david"
    }
    details = {
      firstName = "David"
      lastName  = "Koresh"
      age       = 35
      org       = "Branch Davidians"
    }
  })
}

resource "kubernetes_manifest" "lron" {
  depends_on = [
    kubernetes_manifest.crd_test
  ]

  manifest = jsonencode({
    apiVersion = "k8s.terraform.io/v1alpha1"
    kind = "Person"
    metadata = {
      name = "lron"
    }
    details = {
      firstName = "L Ron"
      lastName  = "Hubbard"
      age       = 65
      status    = "active"
      org       = "Scientology"
    }
  })
}

resource "kubernetes_manifest" "configmap" {
  manifest = jsonencode({
    apiVersion = "v1"
    kind = "ConfigMap"
    metadata = {
      name = "demo"
    }
    data = {
      demo = "demo"
    }
  })

  force_apply = true
}

// trying to use Pod should cause an error
//
// resource "kubernetes_manifest" "pod" {
//   manifest = jsonencode({
//     apiVersion = "v1"
//     kind = "Pod"
//     metadata = {
//       name = "blorp"
//     }
//     spec = {}
//   })
// }



