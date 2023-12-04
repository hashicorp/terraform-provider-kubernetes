---
subcategory: "apps/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_stateful_set_v1"
description: |-
    StatefulSet is a Kubernetes resource used to manage stateful applications.
---

# kubernetes_stateful_set_v1

Manages the deployment and scaling of a set of Pods , and provides guarantees about the 
ordering and uniqueness of these Pods.

Like a Deployment , a StatefulSet manages Pods that are based on an identical container spec.
Unlike a Deployment, a StatefulSet maintains a sticky identity for each of their Pods.
These pods are created from the same spec, but are not interchangeable: each has a persistent 
identifier that it maintains across any rescheduling.

A StatefulSet operates under the same pattern as any other Controller.
You define your desired state in a StatefulSet object, and the StatefulSet controller makes any 
necessary updates to get there from the current state.

## Example Usage

```hcl
resource "kubernetes_stateful_set_v1" "prometheus" {
  metadata {
    annotations = {
      SomeAnnotation = "foobar"
    }

    labels = {
      k8s-app                           = "prometheus"
      "kubernetes.io/cluster-service"   = "true"
      "addonmanager.kubernetes.io/mode" = "Reconcile"
      version                           = "v2.2.1"
    }

    name = "prometheus"
  }

  spec {
    pod_management_policy  = "Parallel"
    replicas               = 1
    revision_history_limit = 5

    selector {
      match_labels = {
        k8s-app = "prometheus"
      }
    }

    service_name = "prometheus"

    template {
      metadata {
        labels = {
          k8s-app = "prometheus"
        }

        annotations = {}
      }

      spec {
        service_account_name = "prometheus"

        init_container {
          name              = "init-chown-data"
          image             = "busybox:latest"
          image_pull_policy = "IfNotPresent"
          command           = ["chown", "-R", "65534:65534", "/data"]

          volume_mount {
            name       = "prometheus-data"
            mount_path = "/data"
            sub_path   = ""
          }
        }

        container {
          name              = "prometheus-server-configmap-reload"
          image             = "jimmidyson/configmap-reload:v0.1"
          image_pull_policy = "IfNotPresent"

          args = [
            "--volume-dir=/etc/config",
            "--webhook-url=http://localhost:9090/-/reload",
          ]

          volume_mount {
            name       = "config-volume"
            mount_path = "/etc/config"
            read_only  = true
          }

          resources {
            limits = {
              cpu    = "10m"
              memory = "10Mi"
            }

            requests = {
              cpu    = "10m"
              memory = "10Mi"
            }
          }
        }

        container {
          name              = "prometheus-server"
          image             = "prom/prometheus:v2.2.1"
          image_pull_policy = "IfNotPresent"

          args = [
            "--config.file=/etc/config/prometheus.yml",
            "--storage.tsdb.path=/data",
            "--web.console.libraries=/etc/prometheus/console_libraries",
            "--web.console.templates=/etc/prometheus/consoles",
            "--web.enable-lifecycle",
          ]

          port {
            container_port = 9090
          }

          resources {
            limits = {
              cpu    = "200m"
              memory = "1000Mi"
            }

            requests = {
              cpu    = "200m"
              memory = "1000Mi"
            }
          }

          volume_mount {
            name       = "config-volume"
            mount_path = "/etc/config"
          }

          volume_mount {
            name       = "prometheus-data"
            mount_path = "/data"
            sub_path   = ""
          }

          readiness_probe {
            http_get {
              path = "/-/ready"
              port = 9090
            }

            initial_delay_seconds = 30
            timeout_seconds       = 30
          }

          liveness_probe {
            http_get {
              path   = "/-/healthy"
              port   = 9090
              scheme = "HTTPS"
            }

            initial_delay_seconds = 30
            timeout_seconds       = 30
          }
        }

        termination_grace_period_seconds = 300

        volume {
          name = "config-volume"

          config_map {
            name = "prometheus-config"
          }
        }
      }
    }

    update_strategy {
      type = "RollingUpdate"

      rolling_update {
        partition = 1
      }
    }

    volume_claim_template {
      metadata {
        name = "prometheus-data"
      }

      spec {
        access_modes       = ["ReadWriteOnce"]
        storage_class_name = "standard"

        resources {
          requests = {
            storage = "16Gi"
          }
        }
      }
    }

    persistent_volume_claim_retention_policy {
      when_deleted = "Delete"
      when_scaled  = "Delete"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard Kubernetes object metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)
* `spec` - (Required) Spec defines the specification of the desired behavior of the stateful set. For more info see [Kubernetes reference](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status)
* `wait_for_rollout` - (Optional) Wait for the StatefulSet to finish rolling out. Defaults to `true`.

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the stateful set that may be used to store arbitrary metadata. 

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the stateful set. **Must match `selector`**. 

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)

* `name` - (Optional) Name of the stateful set, must be unique. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
* `namespace` - (Optional) Namespace defines the space within which name of the stateful set must be unique.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this stateful set that can be used by clients to determine when stateful set has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this stateful set. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

### `spec`

#### Arguments

* `pod_management_policy` - (Optional) podManagementPolicy controls how pods are created during initial scale up, when replacing pods on nodes, or when scaling down. The default policy is `OrderedReady`, where pods are created in increasing order (pod-0, then pod-1, etc) and the controller will wait until each pod is ready before continuing. When scaling down, the pods are removed in the opposite order. The alternative policy is `Parallel` which will create pods in parallel to match the desired scale without waiting, and on scale down will delete all pods at once. *Changing this forces a new resource to be created.*

* `replicas` - (Optional) The desired number of replicas of the given Template. These are replicas in the sense that they are instantiations of the same Template, but individual replicas also have a consistent identity. If unspecified, defaults to 1. This attribute is a string to be able to distinguish between explicit zero and not specified.

* `revision_history_limit` - (Optional)  The maximum number of revisions that will be maintained in the StatefulSet's revision history. The revision history consists of all revisions not represented by a currently applied StatefulSetSpec version. The default value is 10. *Changing this forces a new resource to be created.*

* `selector` - (Required) A label query over pods that should match the replica count. **It must match the pod template's labels.** *Changing this forces a new resource to be created.* More info: [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors)

* `service_name` - (Required) The name of the service that governs this StatefulSet. This service must exist before the StatefulSet, and is responsible for the network identity of the set. Pods get DNS/hostnames that follow the pattern: pod-specific-string.serviceName.default.svc.cluster.local where "pod-specific-string" is managed by the StatefulSet controller. *Changing this forces a new resource to be created.*

* `template` - (Required) The object that describes the pod that will be created if insufficient replicas are detected. Each pod stamped out by the StatefulSet will fulfill this Template, but have a unique identity from the rest of the StatefulSet.

* `update_strategy` - (Optional) Indicates the StatefulSet update strategy that will be employed to update Pods in the StatefulSet when a revision is made to Template.

* `volume_claim_template` - (Optional) A list of volume claims that pods are allowed to reference. A claim in this list takes precedence over any volumes in the template, with the same name. *Changing this forces a new resource to be created.*

* `persistent_volume_claim_retention_policy` - (Optional) The object controls if and how PVCs are deleted during the lifecycle of a StatefulSet.

## Nested Blocks

### `spec.template`

#### Arguments

* `metadata` - (Required) Standard object's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata).

* `spec` - (Optional) Specification of the desired behavior of the pod. For more info see [Kubernetes reference](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status).

## Nested Blocks

### `spec.template.spec`

#### Arguments

These arguments are the same as the for the `spec` block of a Pod.

Please see the [Pod resource](pod.html#spec) for reference.

## Nested Blocks

### `spec.update_strategy`

#### Arguments

* `type` - (Optional) Indicates the type of the StatefulSetUpdateStrategy. There are two valid update strategies, RollingUpdate and OnDelete. Default is `RollingUpdate`.

* `rolling_update` - (Optional) The RollingUpdate update strategy will update all Pods in a StatefulSet, in reverse ordinal order, while respecting the StatefulSet guarantees.


### `spec.update_strategy.rolling_update`

#### Arguments

* `partition` - (Optional) Indicates the ordinal at which the StatefulSet should be partitioned. You can perform a phased roll out (e.g. a linear, geometric, or exponential roll out) using a partitioned rolling update in a similar manner to how you rolled out a canary. To perform a phased roll out, set the partition to the ordinal at which you want the controller to pause the update. By setting the partition to 0, you allow the StatefulSet controller to continue the update process. Default value is `0`.

## Nested Blocks

### `spec.volume_claim_template`

One or more `volume_claim_template` blocks can be specified.

#### Arguments

Each takes the same attibutes as a `kubernetes_persistent_volume_claim_v1` resource.

Please see its [documentation](persistent_volume_claim_v1.html#argument-reference) for reference.

### `spec.persistent_volume_claim_retention_policy`

#### Arguments

* `when_deleted` - (Optional) This field controls what happens when a Statefulset is deleted. Default is Retain.

* `when_scaled` - (Optional) This field controls what happens when a Statefulset is scaled. Default is Retain.

## Timeouts

The following [Timeout](/docs/configuration/resources.html#operation-timeouts) configuration options are available for the `kubernetes_stateful_set_v1` resource:

* `create` - (Default `10 minutes`) Used for creating new StatefulSet
* `read`   - (Default `10 minutes`) Used for reading a StatefulSet
* `update` - (Default `10 minutes`) Used for updating a StatefulSet
* `delete` - (Default `10 minutes`) Used for destroying a StatefulSet

## Import

kubernetes_stateful_set_v1 can be imported using its namespace and name, e.g.

```
$ terraform import kubernetes_stateful_set_v1.example default/terraform-example
```
