---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_deployment"
sidebar_current: "docs-kubernetes-resource-deployment"
description: |-
  A Deployment ensures that a specified number of pod “replicas” are running at any one time. In other words, a Deployment makes sure that a pod or homogeneous set of pods are always up and available. If there are too many pods, it will kill some. If there are too few, the Deployment will start more.
---

# kubernetes_deployment

A Deployment ensures that a specified number of pod “replicas” are running at any one time. In other words, a Deployment makes sure that a pod or homogeneous set of pods are always up and available. If there are too many pods, it will kill some. If there are too few, the Deployment will start more.


## Example Usage

```hcl
resource "kubernetes_deployment" "example" {
  metadata {
    name = "terraform-example"
    labels = {
      test = "MyExampleApp"
    }
  }

  spec {
    replicas = 3

    selector {
      match_labels = {
        test = "MyExampleApp"
      }
    }

    template {
      metadata {
        labels = {
          test = "MyExampleApp"
        }
      }

      spec {
        container {
          image = "nginx:1.7.8"
          name  = "example"

          resources {
            limits {
              cpu    = "0.5"
              memory = "512Mi"
            }
            requests {
              cpu    = "250m"
              memory = "50Mi"
            }
          }

          liveness_probe {
            http_get {
              path = "/nginx_status"
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
      }
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard deployment's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#metadata)
* `spec` - (Required) Spec defines the specification of the desired behavior of the deployment. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#spec-and-status)

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the deployment that may be used to store arbitrary metadata. 
**By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem).**
For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/annotations)
* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the deployment. 
**By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem).**
**Must match `selector`**. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/labels)
* `name` - (Optional) Name of the deployment, must be unique. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)
* `namespace` - (Optional) Namespace defines the space within which name of the deployment must be unique.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this deployment that can be used by clients to determine when deployment has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#concurrency-control-and-consistency)
* `self_link` - A URL representing this deployment.
* `uid` - The unique in time and space value for this deployment. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#uids)

### `spec`

#### Arguments

* `min_ready_seconds` - (Optional) Minimum number of seconds for which a newly created pod should be ready without any of its container crashing, for it to be considered available. Defaults to 0 (pod will be considered available as soon as it is ready)
* `paused` - (Optional) Indicates that the deployment is paused.
* `progress_deadline_seconds` - (Optional) The maximum time in seconds for a deployment to make progress before it is considered to be failed. The deployment controller will continue to process failed deployments and a condition with a ProgressDeadlineExceeded reason will be surfaced in the deployment status. Note that progress will not be estimated during the time a deployment is paused. Defaults to 600s.
* `replicas` - (Optional) The number of desired replicas. Defaults to 1. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#scaling-a-deployment)
* `revision_history_limit` - (Optional) The number of old ReplicaSets to retain to allow rollback. This is a pointer to distinguish between explicit zero and not specified. Defaults to 10.
* `strategy` - (Optional) The deployment strategy to use to replace existing pods with new ones.
* `selector` - (Optional) A label query over pods that should match the Replicas count. Label keys and values that must match in order to be controlled by this deployment. **Must match labels (`metadata.0.labels`)**. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/labels#label-selectors)
* `template` - (Required) Describes the pod that will be created if insufficient replicas are detected. This takes precedence over a TemplateRef. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#pod-template)

### `strategy`

#### Arguments

* `type` - Type of deployment. Can be 'Recreate' or 'RollingUpdate'. Default is RollingUpdate.
* `rolling_update` - Rolling update config params. Present only if type = RollingUpdate.

### `rolling_update`

#### Arguments

* `max_surge` - The maximum number of pods that can be scheduled above the desired number of pods. Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%). This can not be 0 if MaxUnavailable is 0. Absolute number is calculated from percentage by rounding up. Defaults to 25%. Example: when this is set to 30%, the new RC can be scaled up immediately when the rolling update starts, such that the total number of old and new pods do not exceed 130% of desired pods. Once old pods have been killed, new RC can be scaled up further, ensuring that total number of pods running at any time during the update is atmost 130% of desired pods.
* `max_unavailable` - The maximum number of pods that can be unavailable during the update. Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%). Absolute number is calculated from percentage by rounding down. This can not be 0 if MaxSurge is 0. Defaults to 25%. Example: when this is set to 30%, the old RC can be scaled down to 70% of desired pods immediately when the rolling update starts. Once new pods are ready, old RC can be scaled down further, followed by scaling up the new RC, ensuring that the total number of pods available at all times during the update is at least 70% of desired pods.

### `template`

#### Arguments

* `metadata` - (Required) Standard object's metadata. For more info see https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
* `spec` - (Required) Specification of the desired behavior of the pod. For more info see https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status

### template `spec`

#### Arguments

* `affinity` - (Optional) A group of affinity scheduling rules. If specified, the pod will be dispatched by specified scheduler. If not specified, the pod will be dispatched by default scheduler.
* `active_deadline_seconds` - (Optional) Optional duration in seconds the pod may be active on the node relative to StartTime before the system will actively try to mark it failed and kill associated containers. Value must be a positive integer.
* `automount_service_account_token` - (Optional) Indicates whether a service account token should be automatically mounted.
* `container` - (Optional) List of containers belonging to the pod. Containers cannot currently be added or removed. There must be at least one container in a Pod. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/containers)
* `init_container` - (Optional) List of init containers belonging to the pod. Init containers always run to completion and each must complete successfully before the next is started. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/workloads/pods/init-containers)/
* `dns_policy` - (Optional) Set DNS policy for containers within the pod. Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. To have DNS options set along with hostNetwork, you have to specify DNS policy explicitly to 'ClusterFirstWithHostNet'. Optional: Defaults to 'ClusterFirst', see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-dns-policy).
* `dns_config` - (Optional) Specifies the DNS parameters of a pod. Parameters specified here will be merged to the generated DNS configuration based on DNSPolicy. Defaults to empty. See `dns_config` block definition below.
* `host_alias` - (Optional) List of hosts and IPs that will be injected into the pod's hosts file if specified. Optional: Defaults to empty. See `host_alias` block definition below.
* `host_ipc` - (Optional) Use the host's ipc namespace. Optional: Defaults to false.
* `host_network` - (Optional) Host networking requested for this pod. Use the host's network namespace. If this option is set, the ports that will be used must be specified.
* `host_pid` - (Optional) Use the host's pid namespace.
* `hostname` - (Optional) Specifies the hostname of the Pod If not specified, the pod's hostname will be set to a system-defined value.
* `image_pull_secrets` - (Optional) ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec. If specified, these secrets will be passed to individual puller implementations for them to use. For example, in the case of docker, only DockerConfig type secrets are honored. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/images#specifying-imagepullsecrets-on-a-pod)
* `node_name` - (Optional) NodeName is a request to schedule this pod onto a specific node. If it is non-empty, the scheduler simply schedules this pod onto that node, assuming that it fits resource requirements.
* `node_selector` - (Optional) NodeSelector is a selector which must be true for the pod to fit on a node. Selector which must match a node's labels for the pod to be scheduled on that node. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/node-selection).
* `restart_policy` - (Optional) Restart policy for all containers within the pod. One of Always, OnFailure, Never. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/pod-states#restartpolicy).
* `security_context` - (Optional) SecurityContext holds pod-level security attributes and common container settings. Optional: Defaults to empty
* `service_account_name` - (Optional) ServiceAccountName is the name of the ServiceAccount to use to run this pod. For more info see http://releases.k8s.io/HEAD/docs/design/service_accounts.md.
* `share_process_namespace` - (Optional) Share a single process namespace between all of the containers in a pod. When this is set containers will be able to view and signal processes from other containers in the same pod, and the first process in each container will not be assigned PID 1. HostPID and ShareProcessNamespace cannot both be set.
* `subdomain` - (Optional) If specified, the fully qualified Pod hostname will be "...svc.". If not specified, the pod will not have a domainname at all..
* `termination_grace_period_seconds` - (Optional) Optional duration in seconds the pod needs to terminate gracefully. May be decreased in delete request. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period will be used instead. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process.
* `toleration` - (Optional) Optional pod node tolerations. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/)
* `volume` - (Optional) List of volumes that can be mounted by containers belonging to the pod. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes)

### `affinity`

#### Arguments

* `node_affinity` - (Optional) Node affinity scheduling rules for the pod. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#node-affinity-beta-feature)
* `pod_affinity` - (Optional) Inter-pod topological affinity. rules that specify that certain pods should be placed in the same topological domain (e.g. same node, same rack, same zone, same power domain, etc.) For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity-beta-feature)
* `pod_anti_affinity` - (Optional) Inter-pod topological affinity. rules that specify that certain pods should be placed in the same topological domain (e.g. same node, same rack, same zone, same power domain, etc.) For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity-beta-feature)

### `node_affinity`

#### Arguments

* `required_during_scheduling_ignored_during_execution` - (Optional) If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.

* `preferred_during_scheduling_ignored_during_execution` - (Optional) The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions.

### `required_during_scheduling_ignored_during_execution`

#### Arguments

* `node_selector_term` - (Required) A list of node selector terms. The terms are ORed.

## `node_selector_term`

#### Arguments

* `match_expressions` - (Optional) A list of node selector requirements by node's labels.

* `match_fields` - (Optional) A list of node selector requirements by node's fields.

### `match_expressions` / `match_fields`

#### Arguments

* `key` - (Required) The label key that the selector applies to.

* `operator` - (Required) Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.

* `values` - (Optional) An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer.

### `preferred_during_scheduling_ignored_during_execution`

#### Arguments

* `preference` - (Required) A node selector term, associated with the corresponding weight.

* `weight` - (Required) Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.

### `preference`

#### Arguments

* `match_expressions` - (Optional) A list of node selector requirements by node's labels.

* `match_fields` - (Optional) A list of node selector requirements by node's fields.

## `match_expressions` / `match_fields`

#### Arguments

* `key` - (Required) The label key that the selector applies to.

* `operator` - (Required) Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.

* `values` - (Optional) An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer.

### `pod_affinity`

#### Arguments

* `required_during_scheduling_ignored_during_execution` - (Optional) If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node.

* `preferred_during_scheduling_ignored_during_execution` - (Optional) The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions.

### `pod_anti_affinity`

#### Arguments

* `required_during_scheduling_ignored_during_execution` - (Optional) If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. 

* `preferred_during_scheduling_ignored_during_execution` - (Optional) The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions.

### `required_during_scheduling_ignored_during_execution` (pod_affinity_term)

#### Arguments

* `label_selector` - (Optional) A label query over a set of resources, in this case pods.
* `namespaces` - (Optional) Specifies which namespaces the `label_selector` applies to (matches against). Null or empty list means "this pod's namespace"
* `topology_key` - (Optional) This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the `label_selector` in the specified namespaces, where co-located is defined as running on a node whose value of the label with key `topology_key` matches that of any node on which any of the selected pods is running. Empty `topology_key` is not allowed.

### `preferred_during_scheduling_ignored_during_execution`

#### Arguments

* `pod_affinity_term` - (Required) A pod affinity term, associated with the corresponding weight.
* `weight` - (Required) Weight associated with matching the corresponding `pod_affinity_term`, in the range 1-100.

### `container`

#### Arguments

* `args` - (Optional) Arguments to the entrypoint. The docker image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/containers#containers-and-commands)
* `command` - (Optional) Entrypoint array. Not executed within a shell. The docker image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/containers#containers-and-commands)
* `env` - (Optional) Block of string name and value pairs to set in the container's environment. May be declared multiple times. Cannot be updated.
* `env_from` - (Optional) List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER. All invalid keys will be reported as an event when the container is starting. When a key exists in multiple sources, the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated.
* `image` - (Optional) Docker image name. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/images)
* `image_pull_policy` - (Optional) Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/images#updating-images)
* `lifecycle` - (Optional) Actions that the management system should take in response to container lifecycle events
* `liveness_probe` - (Optional) Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/pod-states#container-probes)
* `name` - (Required) Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated.
* `port` - (Optional) List of ports to expose from the container. Exposing a port here gives the system additional information about the network connections a container uses, but is primarily informational. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default "0.0.0.0" address inside a container will be accessible from the network. Cannot be updated.
* `readiness_probe` - (Optional) Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/pod-states#container-probes)
* `resources` - (Optional) Compute Resources required by this container. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/persistent-volumes#resources)
* `security_context` - (Optional) Security options the pod should run with. For more info see http://releases.k8s.io/HEAD/docs/design/security_context.md
* `stdin` - (Optional) Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF.
* `stdin_once` - (Optional) Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin, and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted. If this flag is false, a container processes that reads from stdin will never receive an EOF.
* `termination_message_path` - (Optional) Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem. Message written is intended to be brief final status, such as an assertion failure message. Defaults to /dev/termination-log. Cannot be updated.
* `tty` - (Optional) Whether this container should allocate a TTY for itself
* `volume_mount` - (Optional) Pod volumes to mount into the container's filesystem. Cannot be updated.
* `working_dir` - (Optional) Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated.

### `aws_elastic_block_store`

#### Arguments

* `fs_type` - (Optional) Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#awselasticblockstore)
* `partition` - (Optional) The partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as "1". Similarly, the volume partition for /dev/sda is "0" (or you can leave the property empty).
* `read_only` - (Optional) Whether to set the read-only property in VolumeMounts to "true". If omitted, the default is "false". For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#awselasticblockstore)
* `volume_id` - (Required) Unique ID of the persistent disk resource in AWS (Amazon EBS volume). For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#awselasticblockstore)

### `azure_disk`

#### Arguments

* `caching_mode` - (Required) Host Caching mode: None, Read Only, Read Write.
* `data_disk_uri` - (Required) The URI the data disk in the blob storage
* `disk_name` - (Required) The Name of the data disk in the blob storage
* `fs_type` - (Optional) Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.
* `read_only` - (Optional) Whether to force the read-only setting in VolumeMounts. Defaults to false (read/write).

### `azure_file`

#### Arguments

* `read_only` - (Optional) Whether to force the read-only setting in VolumeMounts. Defaults to false (read/write).
* `secret_name` - (Required) The name of secret that contains Azure Storage Account Name and Key
* `share_name` - (Required) Share Name

### `capabilities`

#### Arguments

* `add` - (Optional) Added capabilities
* `drop` - (Optional) Removed capabilities

### `ceph_fs`

#### Arguments

* `monitors` - (Required) Monitors is a collection of Ceph monitors For more info see http://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it
* `path` - (Optional) Used as the mounted root, rather than the full Ceph tree, default is /
* `read_only` - (Optional) Whether to force the read-only setting in VolumeMounts. Defaults to `false` (read/write). For more info see http://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it
* `secret_file` - (Optional) The path to key ring for User, default is /etc/ceph/user.secret For more info see http://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it
* `secret_ref` - (Optional) Reference to the authentication secret for User, default is empty. For more info see http://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it
* `user` - (Optional) User is the rados user name, default is admin. For more info see http://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it

### `cinder`

#### Arguments

* `fs_type` - (Optional) Filesystem type to mount. Must be a filesystem type supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. For more info see http://releases.k8s.io/HEAD/examples/mysql-cinder-pd/README.md
* `read_only` - (Optional) Whether to force the read-only setting in VolumeMounts. Defaults to false (read/write). For more info see http://releases.k8s.io/HEAD/examples/mysql-cinder-pd/README.md
* `volume_id` - (Required) Volume ID used to identify the volume in Cinder. For more info see http://releases.k8s.io/HEAD/examples/mysql-cinder-pd/README.md

### `config_map`

#### Arguments

* `default_mode` - (Optional) Optional: mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.
* `items` - (Optional) If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error. Paths must be relative and may not contain the '..' path or start with '..'.
* `name` - (Optional) Name of the referent. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)

### `config_map_ref`

#### Arguments

* `name` - (Required) Name of the referent. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)
* `optional` - (Optional) Specify whether the ConfigMap must be defined

### `config_map_key_ref`

#### Arguments

* `key` - (Optional) The key to select.
* `name` - (Optional) Name of the referent. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)

### `dns_config`

#### Arguments

* `nameservers` - (Optional) A list of DNS name server IP addresses specified as strings. This will be appended to the base nameservers generated from DNSPolicy. Duplicated nameservers will be removed. Optional: Defaults to empty.
* `option` - (Optional) A list of DNS resolver options specified as blocks with `name`/`value` pairs. This will be merged with the base options generated from DNSPolicy. Duplicated entries will be removed. Resolution options given in Options will override those that appear in the base DNSPolicy. Optional: Defaults to empty.
* `searches` - (Optional) A list of DNS search domains for host-name lookup specified as strings. This will be appended to the base search paths generated from DNSPolicy. Duplicated search paths will be removed. Optional: Defaults to empty.

The `option` block supports the following:

* `name` - (Required) Name of the option.
* `value` - (Optional) Value of the option. Optional: Defaults to empty.

### `downward_api`

#### Arguments

* `default_mode` - (Optional) Optional: mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.
* `items` - (Optional) If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error. Paths must be relative and may not contain the '..' path or start with '..'.

### `empty_dir`

#### Arguments

* `medium` - (Optional) What type of storage medium should back this directory. The default is "" which means to use the node's default medium. Must be an empty string (default) or Memory. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#emptydir)

### `env`

#### Arguments

* `name` - (Required) Name of the environment variable. Must be a C_IDENTIFIER
* `value` - (Optional) Variable references $(VAR_NAME) are expanded using the previous defined environment variables in the container and any service environment variables. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Defaults to "".
* `value_from` - (Optional) Source for the environment variable's value

### `env_from`

#### Arguments

* `config_map_ref` - (Optional) The ConfigMap to select from
* `prefix` - (Optional) An optional identifer to prepend to each key in the ConfigMap. Must be a C_IDENTIFIER..
* `secret_ref` - (Optional) The Secret to select from

### `exec`

#### Arguments

* `command` - (Optional) Command is the command line to execute inside the container, the working directory for the command is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.

### `fc`

#### Arguments

* `fs_type` - (Optional) Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.
* `lun` - (Required) FC target lun number
* `read_only` - (Optional) Whether to force the read-only setting in VolumeMounts. Defaults to false (read/write).
* `target_ww_ns` - (Required) FC target worldwide names (WWNs)

### `field_ref`

#### Arguments

* `api_version` - (Optional) Version of the schema the FieldPath is written in terms of, defaults to "v1".
* `field_path` - (Optional) Path of the field to select in the specified API version

### `flex_volume`

#### Arguments

* `driver` - (Required) Driver is the name of the driver to use for this volume.
* `fs_type` - (Optional) Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". The default filesystem depends on FlexVolume script.
* `options` - (Optional) Extra command options if any.
* `read_only` - (Optional) Whether to force the ReadOnly setting in VolumeMounts. Defaults to false (read/write).
* `secret_ref` - (Optional) Reference to the secret object containing sensitive information to pass to the plugin scripts. This may be empty if no secret object is specified. If the secret object contains more than one secret, all secrets are passed to the plugin scripts.

### `flocker`

#### Arguments

* `dataset_name` - (Optional) Name of the dataset stored as metadata -> name on the dataset for Flocker should be considered as deprecated
* `dataset_uuid` - (Optional) UUID of the dataset. This is unique identifier of a Flocker dataset

### `gce_persistent_disk`

#### Arguments

* `fs_type` - (Optional) Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#gcepersistentdisk)
* `partition` - (Optional) The partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as "1". Similarly, the volume partition for /dev/sda is "0" (or you can leave the property empty). For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#gcepersistentdisk)
* `pd_name` - (Required) Unique name of the PD resource in GCE. Used to identify the disk in GCE. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#gcepersistentdisk)
* `read_only` - (Optional) Whether to force the ReadOnly setting in VolumeMounts. Defaults to false. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#gcepersistentdisk)

### `git_repo`

#### Arguments

* `directory` - (Optional) Target directory name. Must not contain or start with '..'. If '.' is supplied, the volume directory will be the git repository. Otherwise, if specified, the volume will contain the git repository in the subdirectory with the given name.
* `repository` - (Optional) Repository URL
* `revision` - (Optional) Commit hash for the specified revision.

### `glusterfs`

#### Arguments

* `endpoints_name` - (Required) The endpoint name that details Glusterfs topology. For more info see http://releases.k8s.io/HEAD/examples/volumes/glusterfs/README.md#create-a-pod
* `path` - (Required) The Glusterfs volume path. For more info see http://releases.k8s.io/HEAD/examples/volumes/glusterfs/README.md#create-a-pod
* `read_only` - (Optional) Whether to force the Glusterfs volume to be mounted with read-only permissions. Defaults to false. For more info see http://releases.k8s.io/HEAD/examples/volumes/glusterfs/README.md#create-a-pod

### `host_alias`

#### Arguments

* `hostnames` - (Required) Hostnames for the IP address.
* `ip` - (Required) IP address of the host file entry.

### `host_path`

#### Arguments

* `path` - (Optional) Path of the directory on the host. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#hostpath)

### `http_get`

#### Arguments

* `host` - (Optional) Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.
* `http_header` - (Optional) Scheme to use for connecting to the host.
* `path` - (Optional) Path to access on the HTTP server.
* `port` - (Optional) Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
* `scheme` - (Optional) Scheme to use for connecting to the host.

### `http_header`

#### Arguments

* `name` - (Optional) The header field name
* `value` - (Optional) The header field value

### `image_pull_secrets`

#### Arguments

* `name` - (Required) Name of the referent. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)

### `iscsi`

#### Arguments

* `fs_type` - (Optional) Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#iscsi)
* `iqn` - (Required) Target iSCSI Qualified Name.
* `iscsi_interface` - (Optional) iSCSI interface name that uses an iSCSI transport. Defaults to 'default' (tcp).
* `lun` - (Optional) iSCSI target lun number.
* `read_only` - (Optional) Whether to force the read-only setting in VolumeMounts. Defaults to false.
* `target_portal` - (Required) iSCSI target portal. The portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260).

### `items`

#### Arguments

* `key` - (Optional) The key to project.
* `mode` - (Optional) Optional: mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.
* `path` - (Optional) The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.

### `lifecycle`

#### Arguments

* `post_start` - (Optional) post_start is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy. Other management of the container blocks until the hook completes. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/container-environment#hook-details)
* `pre_stop` - (Optional) pre_stop is called immediately before a container is terminated. The container is terminated after the handler completes. The reason for termination is passed to the handler. Regardless of the outcome of the handler, the container is eventually terminated. Other management of the container blocks until the hook completes. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/container-environment#hook-details)

### `limits`

#### Arguments

* `cpu` - (Optional) CPU
* `memory` - (Optional) Memory

### `liveness_probe`

#### Arguments

* `exec` - (Optional) exec specifies the action to take.
* `failure_threshold` - (Optional) Minimum consecutive failures for the probe to be considered failed after having succeeded.
* `http_get` - (Optional) Specifies the http request to perform.
* `initial_delay_seconds` - (Optional) Number of seconds after the container has started before liveness probes are initiated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/pod-states#container-probes)
* `period_seconds` - (Optional) How often (in seconds) to perform the probe
* `success_threshold` - (Optional) Minimum consecutive successes for the probe to be considered successful after having failed.
* `tcp_socket` - (Optional) TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported
* `timeout_seconds` - (Optional) Number of seconds after which the probe times out. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/pod-states#container-probes)

### `nfs`

#### Arguments

* `path` - (Required) Path that is exported by the NFS server. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#nfs)
* `read_only` - (Optional) Whether to force the NFS export to be mounted with read-only permissions. Defaults to false. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#nfs)
* `server` - (Required) Server is the hostname or IP address of the NFS server. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#nfs)

### `persistent_volume_claim`

#### Arguments

* `claim_name` - (Optional) ClaimName is the name of a PersistentVolumeClaim in the same
* `read_only` - (Optional) Will force the ReadOnly setting in VolumeMounts.

### `photon_persistent_disk`

#### Arguments

* `fs_type` - (Optional) Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.
* `pd_id` - (Required) ID that identifies Photon Controller persistent disk

### `port`

#### Arguments

* `container_port` - (Required) Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536.
* `host_ip` - (Optional) What host IP to bind the external port to.
* `host_port` - (Optional) Number of port to expose on the host. If specified, this must be a valid port number, 0 < x < 65536. If HostNetwork is specified, this must match ContainerPort. Most containers do not need this.
* `name` - (Optional) If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services
* `protocol` - (Optional) Protocol for port. Must be UDP or TCP. Defaults to "TCP".

### `post_start`

#### Arguments

* `exec` - (Optional) exec specifies the action to take.
* `http_get` - (Optional) Specifies the http request to perform.
* `tcp_socket` - (Optional) TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported

### `pre_stop`

#### Arguments

* `exec` - (Optional) exec specifies the action to take.
* `http_get` - (Optional) Specifies the http request to perform.
* `tcp_socket` - (Optional) TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported

### `quobyte`

#### Arguments

* `group` - (Optional) Group to map volume access to Default is no group
* `read_only` - (Optional) Whether to force the Quobyte volume to be mounted with read-only permissions. Defaults to false.
* `registry` - (Required) Registry represents a single or multiple Quobyte Registry services specified as a string as host:port pair (multiple entries are separated with commas) which acts as the central registry for volumes
* `user` - (Optional) User to map volume access to Defaults to serivceaccount user
* `volume` - (Required) Volume is a string that references an already created Quobyte volume by name.

### `rbd`

#### Arguments

* `ceph_monitors` - (Required) A collection of Ceph monitors. For more info see http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it
* `fs_type` - (Optional) Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#rbd)
* `keyring` - (Optional) Keyring is the path to key ring for RBDUser. Default is /etc/ceph/keyring. For more info see http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it
* `rados_user` - (Optional) The rados user name. Default is admin. For more info see http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it
* `rbd_image` - (Required) The rados image name. For more info see http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it
* `rbd_pool` - (Optional) The rados pool name. Default is rbd. For more info see http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it.
* `read_only` - (Optional) Whether to force the read-only setting in VolumeMounts. Defaults to false. For more info see http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it
* `secret_ref` - (Optional) Name of the authentication secret for RBDUser. If provided overrides keyring. Default is nil. For more info see http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it

### `readiness_probe`

#### Arguments

* `exec` - (Optional) exec specifies the action to take.
* `failure_threshold` - (Optional) Minimum consecutive failures for the probe to be considered failed after having succeeded.
* `http_get` - (Optional) Specifies the http request to perform.
* `initial_delay_seconds` - (Optional) Number of seconds after the container has started before liveness probes are initiated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/pod-states#container-probes)
* `period_seconds` - (Optional) How often (in seconds) to perform the probe
* `success_threshold` - (Optional) Minimum consecutive successes for the probe to be considered successful after having failed.
* `tcp_socket` - (Optional) TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported
* `timeout_seconds` - (Optional) Number of seconds after which the probe times out. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/pod-states#container-probes)

### `resources`

#### Arguments

* `limits` - (Optional) Describes the maximum amount of compute resources allowed. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/compute-resources)/
* `requests` - (Optional) Describes the minimum amount of compute resources required.

### `requests`

#### Arguments

* `cpu` - (Optional) CPU
* `memory` - (Optional) Memory

### `resource_field_ref`

#### Arguments

* `container_name` - (Optional) The name of the container
* `resource` - (Required) Resource to select

### `se_linux_options`

#### Arguments

* `level` - (Optional) Level is SELinux level label that applies to the container.
* `role` - (Optional) Role is a SELinux role label that applies to the container.
* `type` - (Optional) Type is a SELinux type label that applies to the container.
* `user` - (Optional) User is a SELinux user label that applies to the container.

### `secret`

#### Arguments

* `default_mode` - (Optional) Mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.
* `items` - (Optional) List of Secret Items to project into the volume. See `items` block definition below. If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked `optional`. Paths must be relative and may not contain the '..' path or start with '..'.
* `optional` - (Optional) Specify whether the Secret or it's keys must be defined.
* `secret_name` - (Optional) Name of the secret in the pod's namespace to use. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#secrets)

The `items` block supports the following:

* `key` - (Required) The key to project.
* `mode` - (Optional) Mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used.
* `path` - (Required) The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.

### `secret_ref`

#### Arguments

* `name` - (Required) Name of the referent. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)
* `optional` - (Optional) Specify whether the Secret must be defined

### `secret_key_ref`

#### Arguments

* `key` - (Optional) The key of the secret to select from. Must be a valid secret key.
* `name` - (Optional) Name of the referent. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)

### `secret_ref`

#### Arguments

* `name` - (Optional) Name of the referent. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)

### container `security_context`

#### Arguments

* `allow_privilege_escalation` - (Optional) AllowPrivilegeEscalation controls whether a process can gain more privileges than its parent process. This bool directly controls if the no_new_privs flag will be set on the container process. AllowPrivilegeEscalation is true always when the container is: 1) run as Privileged 2) has CAP_SYS_ADMIN
* `capabilities` - (Optional) The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime.
* `privileged` - (Optional) Run container in privileged mode. Processes in privileged containers are essentially equivalent to root on the host. Defaults to false.
* `read_only_root_filesystem` - (Optional) Whether this container has a read-only root filesystem. Default is false.
* `run_as_group` - (Optional) The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.
* `run_as_non_root` - (Optional) Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.
* `run_as_user` - (Optional) The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.
* `se_linux_options` - (Optional) The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.

### `capabilities`

#### Arguments

* `add` - (Optional) A list of added capabilities.
* `drop` - (Optional) A list of removed capabilities.

### pod `security_context`

#### Arguments

* `fs_group` - (Optional) A special supplemental group that applies to all containers in a pod. Some volume types allow the Kubelet to change the ownership of that volume to be owned by the pod: 1. The owning GID will be the FSGroup 2. The setgid bit is set (new files created in the volume will be owned by FSGroup) 3. The permission bits are OR'd with rw-rw---- If unset, the Kubelet will not modify the ownership and permissions of any volume.
* `run_as_group` - (Optional) The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container.
* `run_as_non_root` - (Optional) Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.
* `run_as_user` - (Optional) The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container.
* `se_linux_options` - (Optional) The SELinux context to be applied to all containers. If unspecified, the container runtime will allocate a random SELinux context for each container. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container.
* `supplemental_groups` - (Optional) A list of groups applied to the first process run in each container, in addition to the container's primary GID. If unspecified, no groups will be added to any container.

### `tcp_socket`

#### Arguments

* `port` - (Required) Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.

### `value_from`

#### Arguments

* `config_map_key_ref` - (Optional) Selects a key of a ConfigMap.
* `field_ref` - (Optional) Selects a field of the pod: supports metadata.name, metadata.namespace, metadata.labels, metadata.annotations, spec.nodeName, spec.serviceAccountName, status.podIP..
* `resource_field_ref` - (Optional) Selects a field of the pod: supports metadata.name, metadata.namespace, metadata.labels, metadata.annotations, spec.nodeName, spec.serviceAccountName, status.podIP..
* `secret_key_ref` - (Optional) Selects a field of the pod: supports metadata.name, metadata.namespace, metadata.labels, metadata.annotations, spec.nodeName, spec.serviceAccountName, status.podIP..

### `toleration`

#### Arguments

* `effect` - (Optional) Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.
* `key` - (Optional) Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.
* `operator` - (Optional) Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.
* `toleration_seconds` - (Optional) TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.
* `value` - (Optional) Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.

### `volume`

#### Arguments

* `aws_elastic_block_store` - (Optional) Represents an AWS Disk resource that is attached to a kubelet's host machine and then exposed to the pod. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#awselasticblockstore)
* `azure_disk` - (Optional) Represents an Azure Data Disk mount on the host and bind mount to the pod.
* `azure_file` - (Optional) Represents an Azure File Service mount on the host and bind mount to the pod.
* `ceph_fs` - (Optional) Represents a Ceph FS mount on the host that shares a pod's lifetime
* `cinder` - (Optional) Represents a cinder volume attached and mounted on kubelets host machine. For more info see http://releases.k8s.io/HEAD/examples/mysql-cinder-pd/README.md
* `config_map` - (Optional) ConfigMap represents a configMap that should populate this volume
* `downward_api` - (Optional) DownwardAPI represents downward API about the pod that should populate this volume
* `empty_dir` - (Optional) EmptyDir represents a temporary directory that shares a pod's lifetime. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#emptydir)
* `fc` - (Optional) Represents a Fibre Channel resource that is attached to a kubelet's host machine and then exposed to the pod.
* `flex_volume` - (Optional) Represents a generic volume resource that is provisioned/attached using an exec based plugin. This is an alpha feature and may change in future.
* `flocker` - (Optional) Represents a Flocker volume attached to a kubelet's host machine and exposed to the pod for its usage. This depends on the Flocker control service being running
* `gce_persistent_disk` - (Optional) Represents a GCE Disk resource that is attached to a kubelet's host machine and then exposed to the pod. Provisioned by an admin. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#gcepersistentdisk)
* `git_repo` - (Optional) GitRepo represents a git repository at a particular revision.
* `glusterfs` - (Optional) Represents a Glusterfs volume that is attached to a host and exposed to the pod. Provisioned by an admin. For more info see http://releases.k8s.io/HEAD/examples/volumes/glusterfs/README.md
* `host_path` - (Optional) Represents a directory on the host. Provisioned by a developer or tester. This is useful for single-node development and testing only! On-host storage is not supported in any way and WILL NOT WORK in a multi-node cluster. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#hostpath)
* `iscsi` - (Optional) Represents an ISCSI Disk resource that is attached to a kubelet's host machine and then exposed to the pod. Provisioned by an admin.
* `name` - (Optional) Volume's name. Must be a DNS_LABEL and unique within the pod. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)
* `nfs` - (Optional) Represents an NFS mount on the host. Provisioned by an admin. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#nfs)
* `persistent_volume_claim` - (Optional) The specification of a persistent volume.
* `photon_persistent_disk` - (Optional) Represents a PhotonController persistent disk attached and mounted on kubelets host machine
* `quobyte` - (Optional) Quobyte represents a Quobyte mount on the host that shares a pod's lifetime
* `rbd` - (Optional) Represents a Rados Block Device mount on the host that shares a pod's lifetime. For more info see http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md
* `secret` - (Optional) Secret represents a secret that should populate this volume. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/volumes#secrets)
* `vsphere_volume` - (Optional) Represents a vSphere volume attached and mounted on kubelets host machine

### `volume_mount`

#### Arguments

* `mount_path` - (Required) Path within the container at which the volume should be mounted. Must not contain ':'.
* `name` - (Required) This must match the Name of a Volume.
* `read_only` - (Optional) Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false.
* `sub_path` - (Optional) Path within the volume from which the container's volume should be mounted. Defaults to "" (volume's root).

### `vsphere_volume`

#### Arguments

* `fs_type` - (Optional) Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.
* `volume_path` - (Required) Path that identifies vSphere volume vmdk

## Timeouts

The following [Timeout](/docs/configuration/resources.html#operation-timeouts) configuration options are available for the `kubernetes_deployment` resource:

* `create` - (Default `10 minutes`) Used for creating new controller
* `update` - (Default `10 minutes`) Used for updating a controller
* `delete` - (Default `10 minutes`) Used for destroying a controller

## Import

Deployment can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_deployment.example default/terraform-example
```
