provider "kubernetes" {}

# This example shows how to use an external YAML file.
# Something to note here is that we have to decode the YAML
# into an object so that we can encode it to JSON
resource "kubernetes_custom" "cats_crd" {
  json = jsonencode(yamldecode(file("cats_crd.yaml")))

  # NOTE can't do stuff like this :(
  # lifecycle {
  #   ignore_changes = [
  #     jsondecode(json).spec.version
  #   ]
  # }
}

# This example shows how to use the the terraform object
# syntax to define your custom object
resource "kubernetes_custom" "garfield" {
  depends_on = [
    kubernetes_custom.cats_crd
  ]

  json = jsonencode({
    apiVersion = "app.terraform.io/v1alpha1"
    kind = "Cat"
    metadata = {
      name = "garfield"
      namespace = "test"
    }
    data = {
      test = "this-is-a-test"
      blah = true
      tests = "hello"
    }
  })
}

# This example shows that you can embed the YAML
# for your custom resource inside terraform
resource "kubernetes_custom" "tom" {
  depends_on = [
    kubernetes_custom.cats_crd
  ]

  json = jsonencode(yamldecode(<<YAML
---
apiVersion: app.terraform.io/v1alpha1
kind: Cat
metadata:
  name: tom
  namespace: test
data:
  test: something
  or: other
  blah: blah
  YAML
  ))
}
