---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_pod_v1"
description: |-
  A pod is a group of one or more containers, the shared storage for those containers, and options about how to run the containers. Pods are always co-located and co-scheduled, and run in a shared context.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_pod_v1" "test" {
  metadata {
    name = "terraform-example"
  }

  spec {
    container {
      image = "nginx:1.21.6"
      name  = "example"

      env {
        name  = "environment"
        value = "test"
      }

      port {
        container_port = 80
      }

      liveness_probe {
        http_get {
          path = "/"
          port = 80

          http_header {
            name  = "X-Custom-Header"
            value = "Awesome"
          }
        }

        initial_delay_seconds = 3
        period_seconds        = 3
      }
    }

    dns_config {
      nameservers = ["1.1.1.1", "8.8.8.8", "9.9.9.9"]
      searches    = ["example.com"]

      option {
        name  = "ndots"
        value = 1
      }

      option {
        name = "use-vc"
      }
    }

    dns_policy = "None"
  }
}
```

terraform version of the [pods/pod-with-node-affinity.yaml](https://raw.githubusercontent.com/kubernetes/website/master/content/en/examples/pods/pod-with-node-affinity.yaml) example.

```terraform
resource "kubernetes_pod_v1" "with_node_affinity" {
  metadata {
    name = "with-node-affinity"
  }

  spec {
    affinity {
      node_affinity {
        required_during_scheduling_ignored_during_execution {
          node_selector_term {
            match_expressions {
              key      = "kubernetes.io/e2e-az-name"
              operator = "In"
              values   = ["e2e-az1", "e2e-az2"]
            }
          }
        }

        preferred_during_scheduling_ignored_during_execution {
          weight = 1

          preference {
            match_expressions {
              key      = "another-node-label-key"
              operator = "In"
              values   = ["another-node-label-value"]
            }
          }
        }
      }
    }

    container {
      name  = "with-node-affinity"
      image = "k8s.gcr.io/pause:2.0"
    }
  }
}
```

terraform version of the [pods/pod-with-pod-affinity.yaml](https://raw.githubusercontent.com/kubernetes/website/master/content/en/examples/pods/pod-with-pod-affinity.yaml) example.

```terraform
resource "kubernetes_pod_v1" "with_pod_affinity" {
  metadata {
    name = "with-pod-affinity"
  }

  spec {
    affinity {
      pod_affinity {
        required_during_scheduling_ignored_during_execution {
          label_selector {
            match_expressions {
              key      = "security"
              operator = "In"
              values   = ["S1"]
            }
          }

          topology_key = "failure-domain.beta.kubernetes.io/zone"
        }
      }

      pod_anti_affinity {
        preferred_during_scheduling_ignored_during_execution {
          weight = 100

          pod_affinity_term {
            label_selector {
              match_expressions {
                key      = "security"
                operator = "In"
                values   = ["S2"]
              }
            }

            topology_key = "failure-domain.beta.kubernetes.io/zone"
          }
        }
      }
    }

    container {
      name  = "with-pod-affinity"
      image = "k8s.gcr.io/pause:2.0"
    }
  }
}
```

## Import

Pod can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_pod_v1.example default/terraform-example
```
