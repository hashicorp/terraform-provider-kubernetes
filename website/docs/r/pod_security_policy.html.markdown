---
subcategory: "policy/v1beta1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_pod_security_policy"
description: |-
  A Pod Security Policy is a cluster-level resource that controls security sensitive aspects of the pod specification.
---

# kubernetes_pod_security_policy

A Pod Security Policy is a cluster-level resource that controls security sensitive aspects of the pod specification. The PodSecurityPolicy objects define a set of conditions that a pod must run with in order to be accepted into the system, as well as defaults for the related fields.

~> NOTE: With the release of Kubernetes v1.25, PodSecurityPolicy has been removed. You can read more information about the removal of PodSecurityPolicy in the [Kubernetes 1.25 release notes](https://kubernetes.io/blog/2022/08/23/kubernetes-v1-25-release/#pod-security-changes).

## Example Usage

```hcl
resource "kubernetes_pod_security_policy" "example" {
  metadata {
    name = "terraform-example"
  }
  spec {
    privileged                 = false
    allow_privilege_escalation = false

    volumes = [
      "configMap",
      "emptyDir",
      "projected",
      "secret",
      "downwardAPI",
      "persistentVolumeClaim",
    ]

    run_as_user {
      rule = "MustRunAsNonRoot"
    }

    se_linux {
      rule = "RunAsAny"
    }

    supplemental_groups {
      rule = "MustRunAs"
      range {
        min = 1
        max = 65535
      }
    }

    fs_group {
      rule = "MustRunAs"
      range {
        min = 1
        max = 65535
      }
    }

    read_only_root_filesystem = true
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard Pod Security Policy's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#metadata)
* `spec` - (Required) Spec contains information for locating and communicating with a server. [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#spec-and-status)

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the Pod Security Policy that may be used to store arbitrary metadata.

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the Pod Security Policy. 

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)

* `name` - (Optional) Name of the Pod Security Policy, must be unique. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)

#### Attributes


* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this Pod Security Policy that can be used by clients to determine when Pod Security Policy has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this Pod Security Policy. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

### `spec`

#### Arguments

* `allow_privilege_escalation` - (Optional) determines if a pod can request to allow privilege escalation. If unspecified, defaults to true.
* `allowed_capabilities` - (Optional) a list of capabilities that can be requested to add to the container. Capabilities in this field may be added at the pod author's discretion. You must not list a capability in both allowedCapabilities and requiredDropCapabilities.
* [`allowed_flex_volumes`](#allowed_flex_volumes) - (Optional) a whitelist of allowed Flexvolumes.  Empty or nil indicates that all Flexvolumes may be used.  This parameter is effective only when the usage of the Flexvolumes is allowed in the "volumes" field.
* [`allowed_host_paths`](#allowed_host_paths) - (Optional) a white list of allowed host paths. Empty indicates that all host paths may be used.
* `allowed_proc_mount_types` - (Optional) a whitelist of allowed ProcMountTypes. Empty or nil indicates that only the DefaultProcMountType may be used. This requires the ProcMountType feature flag to be enabled. Possible values are `"Default"` or `"Unmasked"`
* `allowed_unsafe_sysctls` - (Optional) a list of explicitly allowed unsafe sysctls, defaults to none. Each entry is either a plain sysctl name or ends in "*" in which case it is considered as a prefix of allowed sysctls. Single* means all unsafe sysctls are allowed. Kubelet has to whitelist all allowed unsafe sysctls explicitly to avoid rejection. Examples: "foo/*" allows "foo/bar", "foo/baz", etc. and "foo.*" allows "foo.bar", "foo.baz", etc.
* `default_add_capabilities` - (Optional) the default set of capabilities that will be added to the container unless the pod spec specifically drops the capability.  You may not list a capability in both defaultAddCapabilities and requiredDropCapabilities. Capabilities added here are implicitly allowed, and need not be included in the allowedCapabilities list.
* `default_allow_privilege_escalation` - (Optional) controls the default setting for whether a process can gain more privileges than its parent process.
* `forbidden_sysctls` - (Optional) forbiddenSysctls is a list of explicitly forbidden sysctls, defaults to none. Each entry is either a plain sysctl name or ends in "*" in which case it is considered as a prefix of forbidden sysctls. Single* means all sysctls are forbidden.
* [`fs_group`](#fs_group) - (Required) the strategy that will dictate what fs group is used by the SecurityContext.
* `host_ipc` - (Optional) determines if the policy allows the use of HostIPC in the pod spec.
* `host_network` - (Optional) determines if the policy allows the use of HostNetwork in the pod spec.
* `host_pid` - (Optional) determines if the policy allows the use of HostPID in the pod spec.
* `host_ports` - (Optional) determines which host port ranges are allowed to be exposed.
* `privileged` - (Optional) determines if a pod can request to be run as privileged.
* `read_only_root_filesystem` - (Optional) when set to true will force containers to run with a read only root file system.  If the container specifically requests to run with a non-read only root file system the PSP should deny the pod. If set to false the container may run with a read only root file system if it wishes but it will not be forced to.
* `required_drop_capabilities` - (Optional) the capabilities that will be dropped from the container.  These are required to be dropped and cannot be added.
* [`run_as_user`](#run_as_user) - (Required) the strategy that will dictate the allowable RunAsUser values that may be set.
* [`run_as_group`](#run_as_group) - (Optional) the strategy that will dictate the allowable RunAsGroup values that may be set. If this field is omitted, the pod's RunAsGroup can take any value. This field requires the RunAsGroup feature gate to be enabled.
* [`se_linux`](#se_linux) - (Required) the strategy that will dictate the allowable labels that may be set.
* [`supplemental_groups`](#supplemental_groups) - (Required) the strategy that will dictate what supplemental groups are used by the SecurityContext.
* `volumes` - (Optional) a white list of allowed volume plugins. Empty indicates that no volumes may be used. To allow all volumes you may use '*'.

### allowed_flex_volumes

### Arguments

* `driver` - (Required) the name of the Flexvolume driver.

### allowed_host_paths

### Arguments

* `path_prefix` - (Required) the path prefix that the host volume must match. It does not support `*`. Trailing slashes are trimmed when validating the path prefix with a host path. Examples: `/foo` would allow `/foo`, `/foo/` and `/foo/bar`. `/foo` would not allow `/food` or `/etc/foo`
* `read_only` - (Optional) when set to true, will allow host volumes matching the pathPrefix only if all volume mounts are readOnly.

### `fs_group`

#### Arguments

* `rule` - (Required) the strategy that will dictate what FSGroup is used in the SecurityContext.
* `range` - (Optional) the allowed ranges of fs groups.  If you would like to force a single fs group then supply a single range with the same start and end. Required for MustRunAs.

### `run_as_user`

#### Arguments

* `rule` - (Required) the strategy that will dictate the allowable RunAsUser values that may be set.
* `range` - (Optional) the allowed ranges of uids that may be used. If you would like to force a single uid then supply a single range with the same start and end. Required for MustRunAs.

### `run_as_group`

#### Arguments

* `rule` - (Required) the strategy that will dictate the allowable RunAsGroup values that may be set.
* `range` - (Optional) the allowed ranges of gids that may be used. If you would like to force a single gid then supply a single range with the same start and end. Required for MustRunAs.

### `se_linux`

#### Arguments

* `rule` - (Required) the strategy that will dictate the allowable labels that may be set.
* `se_linux_options` - (Optional) required to run as; required for MustRunAs. For more info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/

### `supplemental_groups`

#### Arguments

* `rule` - (Required) the strategy that will dictate what supplemental groups is used in the SecurityContext.
* `range` - (Optional) the allowed ranges of supplemental groups.  If you would like to force a single supplemental group then supply a single range with the same start and end. Required for MustRunAs.


### `range`

#### Arguments

* `min` - (Required) the start of the range, inclusive.
* `max` - (Required) the end of the range, inclusive.

## Import

Pod Security Policy can be imported using its name, e.g.

```
$ terraform import kubernetes_pod_security_policy.example terraform-example
```
