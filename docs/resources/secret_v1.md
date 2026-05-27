---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_secret_v1"
description: |-
  The resource provides mechanisms to inject containers with sensitive information while keeping containers agnostic of Kubernetes.
---

# <no value>

<no value>

~> Read more about security properties and risks involved with using Kubernetes secrets: [Kubernetes reference](https://kubernetes.io/docs/concepts/configuration/secret/#information-security-for-secrets)

~> **Note:** All arguments including the secret data will be stored in the raw state as plain-text. [Read more about sensitive data in state](/docs/state/sensitive-data.html).

<no value>

## Example Usage

```terraform
resource "kubernetes_secret_v1" "example" {
  metadata {
    name = "basic-auth"
  }

  data = {
    username = "admin"
    password = "P4ssw0rd"
  }

  type = "kubernetes.io/basic-auth"
}
```

## Example Usage (Docker config)

### Docker config file

```terraform
resource "kubernetes_secret_v1" "example" {
  metadata {
    name = "docker-cfg"
  }

  data = {
    ".dockerconfigjson" = "${file("${path.module}/.docker/config.json")}"
  }

  type = "kubernetes.io/dockerconfigjson"
}
```

### Username and password

```terraform
resource "kubernetes_secret_v1" "example" {
  metadata {
    name = "docker-cfg"
  }

  type = "kubernetes.io/dockerconfigjson"

  data = {
    ".dockerconfigjson" = jsonencode({
      auths = {
        "${var.registry_server}" = {
          "username" = var.registry_username
          "password" = var.registry_password
          "email"    = var.registry_email
          "auth"     = base64encode("${var.registry_username}:${var.registry_password}")
        }
      }
    })
  }
}
```

This is equivalent to the following kubectl command:

```sh
$ kubectl create secret docker-registry docker-cfg --docker-server=${registry_server} --docker-username=${registry_username} --docker-password=${registry_password} --docker-email=${registry_email}
```

## Example Usage (Service account token)

```terraform
resource "kubernetes_secret_v1" "example" {
  metadata {
    annotations = {
      "kubernetes.io/service-account.name" = "my-service-account"
    }

    generate_name = "my-service-account-"
  }

  type                           = "kubernetes.io/service-account-token"
  wait_for_service_account_token = true
}
```

## Import

Secret can be imported using its namespace and name, e.g.

```
$ terraform import kubernetes_secret_v1.example default/my-secret
```
