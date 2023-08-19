---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_pod"
description: |-
   A pod is a group of one or more containers, the shared storage for those containers, and options about how to run the containers. Pods are always co-located and co-scheduled, and run in a shared context.
---

# kubernetes_pod

A pod is a group of one or more containers, the shared storage for those containers, and options about how to run the containers. Pods are always co-located and co-scheduled, and run in a shared context.

Read more at [Kubernetes reference](https://kubernetes.io/docs/concepts/workloads/pods/pod/)

## Example Usage

```
data "kubernetes_pod" "test" {
  metadata {
    name = "terraform-example"
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard pod's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)


## Nested Blocks

### `metadata`

#### Arguments

* `name` - (Required) Name of the pod, must be unique. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
* `namespace` - (Optional) Namespace defines the space within which name of the pod must be unique.


#### Attributes


* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this pod that can be used by clients to determine when pod has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this pod. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

### `spec`

#### Attributes

* `affinity` - A group of affinity scheduling rules. If specified, the pod will be dispatched by specified scheduler. If not specified, the pod will be dispatched by default scheduler.
* `active_deadline_seconds` - Optional duration in seconds the pod may be active on the node relative to StartTime before the system will actively try to mark it failed and kill associated containers. Value must be a positive integer.
* `automount_service_account_token` - Indicates whether a service account token should be automatically mounted. Defaults to true for Pods.
* `container` - List of containers belonging to the pod. Containers cannot currently be added or removed. There must be at least one container in a Pod. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/containers/)
* `init_container` - List of init containers belonging to the pod. Init containers always run to completion and each must complete successfully before the next is started. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/)
* `dns_policy` - Set DNS policy for containers within the pod. Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. To have DNS options set along with hostNetwork, you have to specify DNS policy explicitly to 'ClusterFirstWithHostNet'. Optional: Defaults to 'ClusterFirst', see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-dns-policy).
* `dns_config` - Specifies the DNS parameters of a pod. Parameters specified here will be merged to the generated DNS configuration based on DNSPolicy. Defaults to empty. See `dns_config` block definition below.
* `host_alias` - List of hosts and IPs that will be injected into the pod's hosts file if specified. Optional: Defaults to empty. See `host_alias` block definition below.
* `host_ipc` -  Use the host's ipc namespace. Optional: Defaults to false.
* `host_network` - Host networking requested for this pod. Use the host's network namespace. If this option is set, the ports that will be used must be specified.
* `host_pid` - Use the host's pid namespace.
* `hostname` - Specifies the hostname of the Pod If not specified, the pod's hostname will be set to a system-defined value.
* `image_pull_secrets` - (Optional) ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec. If specified, these secrets will be passed to individual puller implementations for them to use. For example, in the case of docker, only DockerConfig type secrets are honored. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod)
* `node_name` - NodeName is a request to schedule this pod onto a specific node. If it is non-empty, the scheduler simply schedules this pod onto that node, assuming that it fits resource requirements.
* `node_selector` - NodeSelector is a selector which must be true for the pod to fit on a node. Selector which must match a node's labels for the pod to be scheduled on that node. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/).
* `priority_class_name` - If specified, indicates the pod's priority. 'system-node-critical' and 'system-cluster-critical' are two special keywords which indicate the highest priorities with the former being the highest priority. Any other name must be defined by creating a PriorityClass object with that name. If not specified, the pod priority will be default or zero if there is no default.
* `restart_policy` - Restart policy for all containers within the pod. One of Always, OnFailure, Never. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#restart-policy).
* `runtime_class_name` - (Optional) RuntimeClassName is a feature for selecting the container runtime configuration. The container runtime configuration is used to run a Pod's containers. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/containers/runtime-class)
* `security_context` - (SecurityContext holds pod-level security attributes and common container settings. Optional: Defaults to empty
* `service_account_name` - ServiceAccountName is the name of the ServiceAccount to use to run this pod. For more info see https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/.
* `share_process_namespace` - Share a single process namespace between all of the containers in a pod. When this is set containers will be able to view and signal processes from other containers in the same pod, and the first process in each container will not be assigned PID 1. HostPID and ShareProcessNamespace cannot both be set.
* `subdomain` - If specified, the fully qualified Pod hostname will be "...svc.". If not specified, the pod will not have a domainname at all..
* `termination_grace_period_seconds` - Optional duration in seconds the pod needs to terminate gracefully. May be decreased in delete request. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period will be used instead. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process.
* `toleration` - Optional pod node tolerations. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/)
* `volume` - (Optional) List of volumes that can be mounted by containers belonging to the pod. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/storage/volumes)

### `affinity`

#### Attributes

* `node_affinity` - Node affinity scheduling rules for the pod. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#node-affinity-beta-feature)
* `pod_affinity` - Inter-pod topological affinity. rules that specify that certain pods should be placed in the same topological domain (e.g. same node, same rack, same zone, same power domain, etc.) For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity-beta-feature)
* `pod_anti_affinity` - Inter-pod topological affinity. rules that specify that certain pods should be placed in the same topological domain (e.g. same node, same rack, same zone, same power domain, etc.) For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity-beta-feature)


### `container`

#### Attributes

* `args` - Arguments to the entrypoint. The docker image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell)
* `command` - Entrypoint array. Not executed within a shell. The docker image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell)
* `env` - Block of string name and value pairs to set in the container's environment. May be declared multiple times. Cannot be updated.
* `env_from` - List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER. All invalid keys will be reported as an event when the container is starting. When a key exists in multiple sources, the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated.
* `image` - Docker image name. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/containers/images/)
* `image_pull_policy` - Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/containers/images/#updating-images)
* `lifecycle` - Actions that the management system should take in response to container lifecycle events
* `liveness_probe` - Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes)
* `name` - Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated.
* `port` - List of ports to expose from the container. Exposing a port here gives the system additional information about the network connections a container uses, but is primarily informational. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default "0.0.0.0" address inside a container will be accessible from the network. Cannot be updated.
* `readiness_probe` - Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes)
* `resources` - Compute Resources required by this container. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources)
* `security_context` - Security options the pod should run with. For more info see https://kubernetes.io/docs/tasks/configure-pod-container/security-context/.
* `stdin` - Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF.
* `stdin_once` - Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin, and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted. If this flag is false, a container processes that reads from stdin will never receive an EOF.
* `termination_message_path` - Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem. Message written is intended to be brief final status, such as an assertion failure message. Defaults to /dev/termination-log. Cannot be updated.
* `tty` - Whether this container should allocate a TTY for itself
* `volume_mount` - Pod volumes to mount into the container's filesystem. Cannot be updated.
* `working_dir` - Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated.

### `config_map`

#### Attributes

* `default_mode` - Optional: mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.
* `items` - (Optional) If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked `optional`. Paths must be relative and may not contain the '..' path or start with '..'.
* `name` - Name of the referent. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)

### `config_map_ref`

#### Attributes

* `name` -  Name of the referent. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
* `optional` - Specify whether the ConfigMap must be defined

### `config_map_key_ref`

#### Attributes

* `key` - The key to select.
* `name` - Name of the referent. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)

### `dns_config`

#### Attributes

* `nameservers` - A list of DNS name server IP addresses specified as strings. This will be appended to the base nameservers generated from DNSPolicy. Duplicated nameservers will be removed. Optional: Defaults to empty.
* `option` - A list of DNS resolver options specified as blocks with `name`/`value` pairs. This will be merged with the base options generated from DNSPolicy. Duplicated entries will be removed. Resolution options given in Options will override those that appear in the base DNSPolicy. Optional: Defaults to empty.
* `searches` -  A list of DNS search domains for host-name lookup specified as strings. This will be appended to the base search paths generated from DNSPolicy. Duplicated search paths will be removed. Optional: Defaults to empty.

The `option` block supports the following:

* `name` -  Name of the option.
* `value` - Value of the option. Optional: Defaults to empty.

### `downward_api`

#### Attributes

* `default_mode` - Optional: mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.
* `items` - If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error. Paths must be relative and may not contain the '..' path or start with '..'.

### `empty_dir`

#### Attributes

* `medium` - What type of storage medium should back this directory. The default is "" which means to use the node's default medium. Must be an empty string (default) or Memory. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/storage/volumes#emptydir)
* `size_limit` - (Optional) Total amount of local storage required for this EmptyDir volume. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-units-in-kubernetes) and [Kubernetes Quantity type](https://pkg.go.dev/k8s.io/apimachinery/pkg/api/resource?tab=doc#Quantity).

### `env`

#### Attributes

* `name` - Name of the environment variable. Must be a C_IDENTIFIER
* `value` - Variable references $(VAR_NAME) are expanded using the previous defined environment variables in the container and any service environment variables. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Defaults to "".
* `value_from` - Source for the environment variable's value

### `env_from`

#### Attributes

* `config_map_ref` - The ConfigMap to select from
* `prefix` - An optional identifer to prepend to each key in the ConfigMap. Must be a C_IDENTIFIER..
* `secret_ref` - The Secret to select from

### `exec`

#### Attributes

* `command` - Command is the command line to execute inside the container, the working directory for the command is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.

### `grpc`

#### Arguments

* `port` - Number of the port to access on the container. Number must be in the range 1 to 65535.
* `service` - Name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md). If this is not specified, the default behavior is defined by gRPC.

### `image_pull_secrets`

#### Attributes

* `name` - Name of the referent. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)

### `lifecycle`

#### Attributes

* `post_start` - post_start is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy. Other management of the container blocks until the hook completes. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks)
* `pre_stop` - pre_stop is called immediately before a container is terminated. The container is terminated after the handler completes. The reason for termination is passed to the handler. Regardless of the outcome of the handler, the container is eventually terminated. Other management of the container blocks until the hook completes. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks)

### `limits`

#### Attributes

* `cpu` -  CPU
* `memory` -  Memory

### `liveness_probe`

#### Attributes

* `exec` -  exec specifies the action to take.
* `failure_threshold` -  Minimum consecutive failures for the probe to be considered failed after having succeeded.
* `http_get` -  Specifies the http request to perform.
* `grpc` -  GRPC specifies an action involving a GRPC port.
* `initial_delay_seconds` -  Number of seconds after the container has started before liveness probes are initiated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes)
* `period_seconds` -  How often (in seconds) to perform the probe
* `success_threshold` -  Minimum consecutive successes for the probe to be considered successful after having failed.
* `tcp_socket` -  TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported
* `timeout_seconds` -  Number of seconds after which the probe times out. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes)

### `nfs`

#### Attributes

* `path` -  Path that is exported by the NFS server. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/storage/volumes#nfs)
* `read_only` -  Whether to force the NFS export to be mounted with read-only permissions. Defaults to false. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/storage/volumes#nfs)
* `server` -  Server is the hostname or IP address of the NFS server. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/storage/volumes#nfs)

### `persistent_volume_claim`

#### Attributes

* `claim_name` -  ClaimName is the name of a PersistentVolumeClaim in the same
* `read_only` -  Will force the ReadOnly setting in VolumeMounts.

### `photon_persistent_disk`

#### Attributes

* `fs_type` -  Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.
* `pd_id` -  ID that identifies Photon Controller persistent disk

### `port`

#### Attributes

* `container_port` -  Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536.
* `host_ip` -  What host IP to bind the external port to.
* `host_port` -  Number of port to expose on the host. If specified, this must be a valid port number, 0 < x < 65536. If HostNetwork is specified, this must match ContainerPort. Most containers do not need this.
* `name` -  If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services
* `protocol` -  Protocol for port. Must be UDP or TCP. Defaults to "TCP".

### `post_start`

#### Attributes

* `exec` -  exec specifies the action to take.
* `http_get` -  Specifies the http request to perform.
* `tcp_socket` -  TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported

### `pre_stop`

#### Attributes

* `exec` -  exec specifies the action to take.
* `http_get` -  Specifies the http request to perform.
* `tcp_socket` -  TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported

### `quobyte`

#### Attributes

* `group` -  Group to map volume access to Default is no group
* `read_only` -  Whether to force the Quobyte volume to be mounted with read-only permissions. Defaults to false.
* `registry` -  Registry represents a single or multiple Quobyte Registry services specified as a string as host:port pair (multiple entries are separated with commas) which acts as the central registry for volumes
* `user` -  User to map volume access to Defaults to serivceaccount user
* `volume` -  Volume is a string that references an already created Quobyte volume by name.

### `rbd`

#### Attributes

* `ceph_monitors` -  A collection of Ceph monitors. For more info see https://kubernetes.io/docs/concepts/storage/volumes/#cephfs.
* `fs_type` -  Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/storage/volumes#rbd)
* `keyring` - Keyring is the path to key ring for RBDUser. Default is /etc/ceph/keyring. For more info see https://github.com/kubernetes/examples/tree/master/volumes/rbd#how-to-use-it.
* `rados_user` -  The rados user name. Default is admin. For more info see https://github.com/kubernetes/examples/tree/master/volumes/rbd#how-to-use-it.
* `rbd_image` -  The rados image name. For more info see https://github.com/kubernetes/examples/tree/master/volumes/rbd#how-to-use-it.
* `rbd_pool` -  The rados pool name. Default is rbd. For more info see https://github.com/kubernetes/examples/tree/master/volumes/rbd#how-to-use-it.
* `read_only` -  Whether to force the read-only setting in VolumeMounts. Defaults to false. For more info see https://github.com/kubernetes/examples/tree/master/volumes/rbd#how-to-use-it.
* `secret_ref` -  Name of the authentication secret for RBDUser. If provided overrides keyring. Default is nil. For more info see https://github.com/kubernetes/examples/tree/master/volumes/rbd#how-to-use-it.

### `readiness_probe`

#### Attributes

* `exec` -  exec specifies the action to take.
* `failure_threshold` -  Minimum consecutive failures for the probe to be considered failed after having succeeded.
* `grpc` -  GRPC specifies an action involving a GRPC port.
* `http_get` -  Specifies the http request to perform.
* `initial_delay_seconds` -  Number of seconds after the container has started before readiness probes are initiated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes)
* `period_seconds` -  How often (in seconds) to perform the probe
* `success_threshold` -  Minimum consecutive successes for the probe to be considered successful after having failed.
* `tcp_socket` -  TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported
* `timeout_seconds` -  Number of seconds after which the probe times out. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes)

### `resources`

#### Arguments

* `limits` - (Optional) Describes the maximum amount of compute resources allowed. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
* `requests` - (Optional) Describes the minimum amount of compute resources required.

`resources` is a computed attribute and thus if it is not configured in terraform code, the value will be computed from the returned Kubernetes object. That causes a situation when removing `resources` from terraform code does not update the Kubernetes object. In order to delete `resources` from the Kubernetes object, configure an empty attribute in your code.

Please, look at the example below:

```hcl
resources {
  limits   = {}
  requests = {}
}
```

### `requests`

#### Attributes

* `cpu` -  CPU
* `memory` -  Memory

### `resource_field_ref`

#### Attributes

* `container_name` -  The name of the container
* `resource` -  Resource to select

### `seccomp_profile`

#### Attributes

* `type` - Indicates which kind of seccomp profile will be applied. Valid options are:
    * `Localhost` - a profile defined in a file on the node should be used.
    * `RuntimeDefault` - the container runtime default profile should be used.
    * `Unconfined` - (Default) no profile should be applied.
* `localhost_profile` - Indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work. Must be a descending path, relative to the kubelet's configured seccomp profile location. Must only be set if `type` is `Localhost`.

### `se_linux_options`

#### Attributes

* `level` -  Level is SELinux level label that applies to the container.
* `role` -  Role is a SELinux role label that applies to the container.
* `type` -  Type is a SELinux type label that applies to the container.
* `user` -  User is a SELinux user label that applies to the container.

### `secret`

#### Attributes

* `default_mode` -  Mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.
* `items` -  List of Secret Items to project into the volume. See `items` block definition below. If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked `optional`. Paths must be relative and may not contain the '..' path or start with '..'.
* `optional` -  Specify whether the Secret or its keys must be defined.
* `secret_name` -  Name of the secret in the pod's namespace to use. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/storage/volumes#secrets)

The `items` block supports the following:

* `key` -  The key to project.
* `mode` -  Mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used.
* `path` -  The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.

### `secret_ref`

#### Attributes

* `name` -  Name of the referent. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
* `optional` -  Specify whether the Secret must be defined

### `secret_key_ref`

#### Attributes

* `key` -  The key of the secret to select from. Must be a valid secret key.
* `name` -  Name of the referent. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)

### `secret_ref`

#### Attributes

* `name` -  Name of the referent. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)

### container `security_context`

#### ArgumAttributesents

* `allow_privilege_escalation` -  AllowPrivilegeEscalation controls whether a process can gain more privileges than its parent process. This bool directly controls if the no_new_privs flag will be set on the container process. AllowPrivilegeEscalation is true always when the container is: 1) run as Privileged 2) has CAP_SYS_ADMIN
* `capabilities` -  The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime.
* `privileged` -  Run container in privileged mode. Processes in privileged containers are essentially equivalent to root on the host. Defaults to false.
* `read_only_root_filesystem` -  Whether this container has a read-only root filesystem. Default is false.
* `run_as_group` -  The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.
* `run_as_non_root` -  Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.
* `run_as_user` -  The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.
* `seccomp_profile` - The seccomp options to use by the containers in this pod. Note that this field cannot be set when spec.os.name is windows.
* `se_linux_options` -  The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.
* `fs_group_change_policy` - Defines behavior of changing ownership and permission of the volume before being exposed inside Pod. This field will only apply to volume types which support fsGroup based ownership(and permissions). It will have no effect on ephemeral volume types such as: secret, configmaps and emptydir. Valid values are "OnRootMismatch" and "Always". If not specified, "Always" is used. Note that this field cannot be set when spec.os.name is windows.

### `capabilities`

#### Attributes

* `add` -  A list of added capabilities.
* `drop` -  A list of removed capabilities.

### pod `security_context`

#### Attributes

* `fs_group` -  A special supplemental group that applies to all containers in a pod. Some volume types allow the Kubelet to change the ownership of that volume to be owned by the pod: 1. The owning GID will be the FSGroup 2. The setgid bit is set (new files created in the volume will be owned by FSGroup) 3. The permission bits are OR'd with rw-rw---- If unset, the Kubelet will not modify the ownership and permissions of any volume.
* `run_as_group` -  The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container.
* `run_as_non_root` -  Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.
* `run_as_user` -  The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container.
* `seccomp_profile` - The seccomp options to use by the containers in this pod. Note that this field cannot be set when spec.os.name is windows.
* `se_linux_options` -  The SELinux context to be applied to all containers. If unspecified, the container runtime will allocate a random SELinux context for each container. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container.
* `supplemental_groups` -  A list of groups applied to the first process run in each container, in addition to the container's primary GID. If unspecified, no groups will be added to any container.

### `tcp_socket`

#### Attributes

* `port` -  Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.


### `value_from`

#### Attributes

* `config_map_key_ref` -  Selects a key of a ConfigMap.
* `field_ref` -  (Optional) Selects a field of the pod: supports metadata.name, metadata.namespace, metadata.labels, metadata.annotations, spec.nodeName, spec.serviceAccountName, status.podIP.
* `resource_field_ref` -  (Optional) Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.ephemeral-storage) are currently supported.
* `secret_key_ref` -  (Optional) Selects a key of a secret in the pod's namespace.

### `volume_mount`

#### Attributes

* `mount_path` -  Path within the container at which the volume should be mounted. Must not contain ':'.
* `name` -  This must match the Name of a Volume.
* `read_only` -  Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false.
* `sub_path` -  Path within the volume from which the container's volume should be mounted. Defaults to "" (volume's root).
* `mount_propagation` -  Mount propagation mode. Defaults to "None". For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/storage/volumes/#mount-propagation)

## Argument Reference

The following attributes are exported:

* `status` - The current status of the pods.

## Import

Pod can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_pod.example default/terraform-example
```
