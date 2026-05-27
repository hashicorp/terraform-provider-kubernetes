---
subcategory: "networking/v1"
page_title: "Kubernetes: kubernetes_network_policy"
description: |-
  Kubernetes supports network policies to specify how groups of pods are allowed to communicate with each other and with other network endpoints.
  NetworkPolicy resources use labels to select pods and define rules which specify what traffic is allowed to the selected pods.
---

# <no value>

Kubernetes supports network policies to specify how groups of pods are allowed to communicate with each other and with other network endpoints. NetworkPolicy resources use labels to select pods and define rules which specify what traffic is allowed to the selected pods. Read more about network policies at https://kubernetes.io/docs/concepts/services-networking/network-policies/

<no value>

## Example Usage

```terraform
resource "kubernetes_network_policy" "example" {
  metadata {
    name      = "terraform-example-network-policy"
    namespace = "default"
  }

  spec {
    pod_selector {
      match_expressions {
        key      = "name"
        operator = "In"
        values   = ["webfront", "api"]
      }
    }

    ingress {
      ports {
        port     = "http"
        protocol = "TCP"
      }
      ports {
        port     = "8125"
        protocol = "UDP"
      }

      from {
        namespace_selector {
          match_labels = {
            name = "default"
          }
        }
      }

      from {
        ip_block {
          cidr = "10.0.0.0/8"
          except = [
            "10.0.0.0/24",
            "10.0.1.0/24",
          ]
        }
      }
    }

    egress {} # single empty rule to allow all egress traffic

    policy_types = ["Ingress", "Egress"]
  }
}
```

## Import

Network policies can be imported using their identifier consisting of `<namespace-name>/<network-policy-name>`, e.g.:

```
$ terraform import kubernetes_network_policy.example default/terraform-example-network-policy
```
