// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func podSpecFields(isUpdatable, isComputed bool) map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"affinity": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			ForceNew:    !isUpdatable,
			Description: "Optional pod scheduling constraints.",
			Elem: &schema.Resource{
				Schema: affinityFields(),
			},
		},
		"active_deadline_seconds": {
			Type:         schema.TypeInt,
			Optional:     true,
			Computed:     isComputed,
			ForceNew:     false, // always updatable
			ValidateFunc: validatePositiveInteger,
			Description:  "Optional duration in seconds the pod may be active on the node relative to StartTime before the system will actively try to mark it failed and kill associated containers. Value must be a positive integer.",
		},
		"automount_service_account_token": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			ForceNew:    !isUpdatable,
			Description: "AutomountServiceAccountToken indicates whether a service account token should be automatically mounted.",
		},
		"container": {
			Type:        schema.TypeList,
			Optional:    true,
			ForceNew:    false, // always updatable
			Description: "List of containers belonging to the pod. Containers cannot currently be added or removed. There must be at least one container in a Pod. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/",
			Elem: &schema.Resource{
				Schema: containerFields(isUpdatable),
			},
		},
		"readiness_gate": {
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			Description: "If specified, all readiness gates will be evaluated for pod readiness. A pod is ready when all its containers are ready AND all conditions specified in the readiness gates have status equal to \"True\" More info: https://git.k8s.io/enhancements/keps/sig-network/0007-pod-ready%2B%2B.md",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"condition_type": {
						Type:        schema.TypeString,
						Required:    true,
						ForceNew:    !isUpdatable,
						Description: "refers to a condition in the pod's condition list with matching type.",
					},
				},
			},
		},
		"init_container": {
			Type:        schema.TypeList,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Description: "List of init containers belonging to the pod. Init containers always run to completion and each must complete successfully before the next is started. More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/",
			Elem: &schema.Resource{
				Schema: containerFields(isUpdatable),
			},
		},
		"dns_policy": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    isComputed,
			ForceNew:    !isUpdatable,
			Default:     conditionalDefault(!isComputed, string(corev1.DNSClusterFirst)),
			Description: "Set DNS policy for containers within the pod. Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. To have DNS options set along with hostNetwork, you have to specify DNS policy explicitly to 'ClusterFirstWithHostNet'. Defaults to 'ClusterFirst'. More info: https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-dns-policy",
			ValidateFunc: validation.StringInSlice([]string{
				string(corev1.DNSClusterFirst),
				string(corev1.DNSClusterFirstWithHostNet),
				string(corev1.DNSDefault),
				string(corev1.DNSNone),
			}, false),
		},
		"dns_config": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "Specifies the DNS parameters of a pod. Parameters specified here will be merged to the generated DNS configuration based on DNSPolicy. Optional: Defaults to empty",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"nameservers": {
						Type:        schema.TypeList,
						Description: "A list of DNS name server IP addresses. This will be appended to the base nameservers generated from DNSPolicy. Duplicated nameservers will be removed.",
						Optional:    true,
						ForceNew:    !isUpdatable,
						Elem: &schema.Schema{
							Type:         schema.TypeString,
							ValidateFunc: validation.IsIPAddress,
						},
					},
					"option": {
						Type:        schema.TypeList,
						Description: "A list of DNS resolver options. This will be merged with the base options generated from DNSPolicy. Duplicated entries will be removed. Resolution options given in Options will override those that appear in the base DNSPolicy.",
						Optional:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"name": {
									Type:        schema.TypeString,
									Description: "Name of the option.",
									Required:    true,
									ForceNew:    !isUpdatable,
								},
								"value": {
									Type:        schema.TypeString,
									Description: "Value of the option. Optional: Defaults to empty.",
									Optional:    true,
									ForceNew:    !isUpdatable,
								},
							},
						},
					},
					"searches": {
						Type:        schema.TypeList,
						Description: "A list of DNS search domains for host-name lookup. This will be appended to the base search paths generated from DNSPolicy. Duplicated search paths will be removed.",
						Optional:    true,
						Elem: &schema.Schema{
							Type:         schema.TypeString,
							ValidateFunc: validateName,
							ForceNew:     !isUpdatable,
						},
					},
				},
			},
		},
		"enable_service_links": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Default:     true,
			Description: "Enables generating environment variables for service discovery. Defaults to true.",
		},
		"host_aliases": {
			Type:        schema.TypeList,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Computed:    isComputed,
			Description: "List of hosts and IPs that will be injected into the pod's hosts file if specified. Optional: Defaults to empty.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"hostnames": {
						Type:        schema.TypeList,
						Required:    true,
						ForceNew:    !isUpdatable,
						Description: "Hostnames for the IP address.",
						Elem:        &schema.Schema{Type: schema.TypeString},
					},
					"ip": {
						Type:         schema.TypeString,
						Required:     true,
						ForceNew:     !isUpdatable,
						Description:  "IP address of the host file entry.",
						ValidateFunc: validation.IsIPAddress,
					},
				},
			},
		},
		"host_ipc": {
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    isComputed,
			ForceNew:    !isUpdatable,
			Default:     conditionalDefault(!isComputed, false),
			Description: "Use the host's ipc namespace. Optional: Defaults to false.",
		},
		"host_network": {
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    isComputed,
			ForceNew:    !isUpdatable,
			Default:     conditionalDefault(!isComputed, false),
			Description: "Host networking requested for this pod. Use the host's network namespace. If this option is set, the ports that will be used must be specified.",
		},

		"host_pid": {
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    isComputed,
			ForceNew:    !isUpdatable,
			Default:     conditionalDefault(!isComputed, false),
			Description: "Use the host's pid namespace.",
		},

		"hostname": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			ForceNew:    !isUpdatable,
			Description: "Specifies the hostname of the Pod If not specified, the pod's hostname will be set to a system-defined value.",
		},
		"os": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "Specifies the OS of the containers in the pod.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"name": {
						Type:         schema.TypeString,
						Required:     true,
						ValidateFunc: validation.StringInSlice([]string{string(corev1.Linux), string(corev1.Windows)}, false),
						Description:  "Name is the name of the operating system. The currently supported values are linux and windows.",
					},
				},
			},
		},
		"image_pull_secrets": {
			Type:        schema.TypeList,
			Description: "ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec. If specified, these secrets will be passed to individual puller implementations for them to use. For example, in the case of docker, only DockerConfig type secrets are honored. More info: https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod",
			Optional:    true,
			Computed:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"name": {
						Type:        schema.TypeString,
						Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
						Required:    true,
						ForceNew:    !isUpdatable,
					},
				},
			},
		},
		"node_name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			ForceNew:    !isUpdatable,
			Description: "NodeName is a request to schedule this pod onto a specific node. If it is non-empty, the scheduler simply schedules this pod onto that node, assuming that it fits resource requirements.",
		},
		"node_selector": {
			Type:        schema.TypeMap,
			Optional:    true,
			Computed:    isComputed,
			ForceNew:    !isUpdatable,
			Description: "NodeSelector is a selector which must be true for the pod to fit on a node. Selector which must match a node's labels for the pod to be scheduled on that node. More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/.",
		},
		"runtime_class_name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    isComputed,
			ForceNew:    !isUpdatable,
			Description: "RuntimeClassName is a feature for selecting the container runtime configuration. The container runtime configuration is used to run a Pod's containers. More info: https://kubernetes.io/docs/concepts/containers/runtime-class",
		},
		"priority_class_name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    isComputed,
			ForceNew:    !isUpdatable,
			Description: `If specified, indicates the pod's priority. "system-node-critical" and "system-cluster-critical" are two special keywords which indicate the highest priorities with the former being the highest priority. Any other name must be defined by creating a PriorityClass object with that name. If not specified, the pod priority will be default or zero if there is no default.`,
		},
		"restart_policy": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    isComputed,
			ForceNew:    !isUpdatable,
			Default:     conditionalDefault(!isComputed, string(corev1.RestartPolicyAlways)),
			Description: "Restart policy for all containers within the pod. One of Always, OnFailure, Never. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#restart-policy.",
			ValidateFunc: validation.StringInSlice([]string{
				string(corev1.RestartPolicyAlways),
				string(corev1.RestartPolicyOnFailure),
				string(corev1.RestartPolicyNever),
			}, false),
		},
		"security_context": {
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    isComputed,
			MaxItems:    1,
			Description: "SecurityContext holds pod-level security attributes and common container settings. Optional: Defaults to empty",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"fs_group": {
						Type:         schema.TypeString,
						Description:  "A special supplemental group that applies to all containers in a pod. Some volume types allow the Kubelet to change the ownership of that volume to be owned by the pod: 1. The owning GID will be the FSGroup 2. The setgid bit is set (new files created in the volume will be owned by FSGroup) 3. The permission bits are OR'd with rw-rw---- If unset, the Kubelet will not modify the ownership and permissions of any volume.",
						Optional:     true,
						ValidateFunc: validateTypeStringNullableInt,
						ForceNew:     !isUpdatable,
					},
					"run_as_group": {
						Type:         schema.TypeString,
						Description:  "The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container.",
						Optional:     true,
						ValidateFunc: validateTypeStringNullableInt,
						ForceNew:     !isUpdatable,
					},
					"run_as_non_root": {
						Type:        schema.TypeBool,
						Description: "Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.",
						Optional:    true,
						ForceNew:    !isUpdatable,
					},
					"run_as_user": {
						Type:         schema.TypeString,
						Description:  "The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container.",
						Optional:     true,
						ValidateFunc: validateTypeStringNullableInt,
						ForceNew:     !isUpdatable,
					},
					"seccomp_profile": {
						Type:        schema.TypeList,
						Description: "The seccomp options to use by the containers in this pod. Note that this field cannot be set when spec.os.name is windows.",
						Optional:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: seccompProfileField(isUpdatable),
						},
					},
					"se_linux_options": {
						Type:        schema.TypeList,
						Description: "The SELinux context to be applied to all containers. If unspecified, the container runtime will allocate a random SELinux context for each container. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container.",
						Optional:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: seLinuxOptionsField(isUpdatable),
						},
					},
					"fs_group_change_policy": {
						Type:        schema.TypeString,
						Description: "fsGroupChangePolicy defines behavior of changing ownership and permission of the volume before being exposed inside Pod. This field will only apply to volume types which support fsGroup based ownership(and permissions). It will have no effect on ephemeral volume types such as: secret, configmaps and emptydir.",
						Optional:    true,
						ValidateFunc: validation.StringInSlice([]string{
							string(corev1.FSGroupChangeAlways),
							string(corev1.FSGroupChangeOnRootMismatch),
						}, false),
						ForceNew: !isUpdatable,
					},
					"supplemental_groups": {
						Type:        schema.TypeSet,
						Description: "A list of groups applied to the first process run in each container, in addition to the container's primary GID. If unspecified, no groups will be added to any container.",
						Optional:    true,
						ForceNew:    !isUpdatable,
						Elem: &schema.Schema{
							Type: schema.TypeInt,
						},
					},
					"windows_options": {
						Type:        schema.TypeList,
						MaxItems:    1,
						Description: "The Windows specific settings applied to all containers. If unspecified, the options within a container's SecurityContext will be used. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is linux.",
						Optional:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"gmsa_credential_spec": {
									Type:        schema.TypeString,
									Description: "GMSACredentialSpec is where the GMSA admission webhook inlines the contents of the GMSA credential spec named by the GMSACredentialSpecName field",
									Optional:    true,
								},
								"gmsa_credential_spec_name": {
									Type:        schema.TypeString,
									Description: "GMSACredentialSpecName is the name of the GMSA credential spec to use.",
									Optional:    true,
								},
								"host_process": {
									Type:        schema.TypeBool,
									Description: "HostProcess determines if a container should be run as a 'Host Process' container. Default value is false.",
									Default:     false,
									Optional:    true,
								},
								"run_as_username": {
									Type:        schema.TypeString,
									Description: "The UserName in Windows to run the entrypoint of the container process. Defaults to the user specified in image metadata if unspecified. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.",
									Optional:    true,
								},
							},
						},
					},
					"sysctl": {
						Type:        schema.TypeList,
						Optional:    true,
						Description: "holds a list of namespaced sysctls used for the pod.",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"name": {
									Type:        schema.TypeString,
									Description: "Name of a property to set.",
									Required:    true,
									ForceNew:    !isUpdatable,
								},
								"value": {
									Type:        schema.TypeString,
									Description: "Value of a property to set.",
									Required:    true,
									ForceNew:    !isUpdatable,
								},
							},
						},
					},
				},
			},
		},
		"scheduler_name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			ForceNew:    !isUpdatable,
			Description: "If specified, the pod will be dispatched by specified scheduler. If not specified, the pod will be dispatched by default scheduler.",
		},
		"service_account_name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			ForceNew:    !isUpdatable,
			Description: "ServiceAccountName is the name of the ServiceAccount to use to run this pod. More info: http://releases.k8s.io/HEAD/docs/design/service_accounts.md.",
		},
		"share_process_namespace": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			ForceNew:    !isUpdatable,
			Description: "Share a single process namespace between all of the containers in a pod. When this is set containers will be able to view and signal processes from other containers in the same pod, and the first process in each container will not be assigned PID 1. HostPID and ShareProcessNamespace cannot both be set. Optional: Defaults to false.",
		},
		"subdomain": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    isComputed,
			ForceNew:    !isUpdatable,
			Description: `If specified, the fully qualified Pod hostname will be "...svc.". If not specified, the pod will not have a domainname at all..`,
		},
		"termination_grace_period_seconds": {
			Type:         schema.TypeInt,
			Optional:     true,
			Computed:     isComputed,
			ForceNew:     !isUpdatable,
			Default:      conditionalDefault(!isComputed, 30),
			ValidateFunc: validateTerminationGracePeriodSeconds,
			Description:  "Optional duration in seconds the pod needs to terminate gracefully. May be decreased in delete request. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period will be used instead. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process.",
		},
		"toleration": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "If specified, the pod's toleration. Optional: Defaults to empty",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"effect": {
						Type:        schema.TypeString,
						Description: "Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.",
						Optional:    true,
						ForceNew:    !isUpdatable,
						ValidateFunc: validation.StringInSlice([]string{
							string(corev1.TaintEffectNoSchedule),
							string(corev1.TaintEffectPreferNoSchedule),
							string(corev1.TaintEffectNoExecute),
						}, false),
					},
					"key": {
						Type:        schema.TypeString,
						Description: "Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.",
						Optional:    true,
						ForceNew:    !isUpdatable,
					},
					"operator": {
						Type:        schema.TypeString,
						Description: "Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.",
						Default:     string(corev1.TolerationOpEqual),
						Optional:    true,
						ForceNew:    !isUpdatable,
						ValidateFunc: validation.StringInSlice([]string{
							string(corev1.TolerationOpExists),
							string(corev1.TolerationOpEqual),
						}, false),
					},
					"toleration_seconds": {
						// Use TypeString to allow an "unspecified" value,
						Type:         schema.TypeString,
						Description:  "TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.",
						Optional:     true,
						ForceNew:     !isUpdatable,
						ValidateFunc: validateTypeStringNullableInt,
					},
					"value": {
						Type:        schema.TypeString,
						Description: "Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.",
						Optional:    true,
						ForceNew:    !isUpdatable,
					},
				},
			},
		},
		"topology_spread_constraint": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "describes how a group of pods ought to spread across topology domains. Scheduler will schedule pods in a way which abides by the constraints.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"max_skew": {
						Type:         schema.TypeInt,
						Description:  "describes the degree to which pods may be unevenly distributed.",
						Optional:     true,
						Default:      1,
						ValidateFunc: validation.IntAtLeast(1),
					},
					"topology_key": {
						Type:        schema.TypeString,
						Description: "the key of node labels. Nodes that have a label with this key and identical values are considered to be in the same topology.",
						Optional:    true,
					},
					"when_unsatisfiable": {
						Type:        schema.TypeString,
						Description: "indicates how to deal with a pod if it doesn't satisfy the spread constraint.",
						Default:     string(corev1.DoNotSchedule),
						Optional:    true,
						ValidateFunc: validation.StringInSlice([]string{
							string(corev1.DoNotSchedule),
							string(corev1.ScheduleAnyway),
						}, false),
					},
					"label_selector": {
						Type:        schema.TypeList,
						Description: "A label query over a set of resources, in this case pods.",
						Optional:    true,
						Elem: &schema.Resource{
							Schema: labelSelectorFields(true),
						},
					},
				},
			},
		},
		"volume": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "List of volumes that can be mounted by containers belonging to the pod. More info: https://kubernetes.io/docs/concepts/storage/volumes",
			Elem:        volumeSchema(isUpdatable),
		},
	}
	return s
}

func volumeSchema(isUpdatable bool) *schema.Resource {
	v := commonVolumeSources()

	v["config_map"] = &schema.Schema{
		Type:        schema.TypeList,
		Description: "ConfigMap represents a configMap that should populate this volume",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"items": {
					Type:        schema.TypeList,
					Description: `If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.`,
					Optional:    true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"key": {
								Type:        schema.TypeString,
								Optional:    true,
								Description: "The key to project.",
							},
							"mode": {
								Type:         schema.TypeString,
								Optional:     true,
								Description:  `Optional: mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.`,
								ValidateFunc: validateModeBits,
							},
							"path": {
								Type:         schema.TypeString,
								Optional:     true,
								ValidateFunc: validatePath,
								Description:  `The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.`,
							},
						},
					},
				},
				"default_mode": {
					Type:         schema.TypeString,
					Description:  "Optional: mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.",
					Optional:     true,
					Default:      "0644",
					ValidateFunc: validateModeBits,
				},
				"optional": {
					Type:        schema.TypeBool,
					Description: "Optional: Specify whether the ConfigMap or its keys must be defined.",
					Optional:    true,
				},
				"name": {
					Type:        schema.TypeString,
					Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
					Optional:    true,
				},
			},
		},
	}

	v["git_repo"] = &schema.Schema{
		Type:        schema.TypeList,
		Description: "GitRepo represents a git repository at a particular revision.",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"directory": {
					Type:         schema.TypeString,
					Description:  "Target directory name. Must not contain or start with '..'. If '.' is supplied, the volume directory will be the git repository. Otherwise, if specified, the volume will contain the git repository in the subdirectory with the given name.",
					Optional:     true,
					ValidateFunc: validatePath,
				},
				"repository": {
					Type:        schema.TypeString,
					Description: "Repository URL",
					Optional:    true,
				},
				"revision": {
					Type:        schema.TypeString,
					Description: "Commit hash for the specified revision.",
					Optional:    true,
				},
			},
		},
	}
	v["downward_api"] = &schema.Schema{
		Type:        schema.TypeList,
		Description: "DownwardAPI represents downward API about the pod that should populate this volume",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"default_mode": {
					Type:         schema.TypeString,
					Description:  "Optional: mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.",
					Optional:     true,
					Default:      "0644",
					ValidateFunc: validateModeBits,
				},
				"items": {
					Type:        schema.TypeList,
					Description: `If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error. Paths must be relative and may not contain the '..' path or start with '..'.`,
					Optional:    true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"field_ref": {
								Type:        schema.TypeList,
								Required:    true,
								MaxItems:    1,
								Description: "Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.",
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"api_version": {
											Type:        schema.TypeString,
											Optional:    true,
											Default:     "v1",
											Description: `Version of the schema the FieldPath is written in terms of, defaults to "v1".`,
										},
										"field_path": {
											Type:        schema.TypeString,
											Optional:    true,
											Description: "Path of the field to select in the specified API version",
										},
									},
								},
							},
							"mode": {
								Type:         schema.TypeString,
								Optional:     true,
								Description:  `Optional: mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.`,
								ValidateFunc: validateModeBits,
							},
							"path": {
								Type:         schema.TypeString,
								Required:     true,
								ValidateFunc: validatePath,
								Description:  `Path is the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'`,
							},
							"resource_field_ref": {
								Type:        schema.TypeList,
								Optional:    true,
								MaxItems:    1,
								Description: "Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.",
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"container_name": {
											Type:     schema.TypeString,
											Required: true,
										},
										"divisor": {
											Type:             schema.TypeString,
											Optional:         true,
											Default:          "1",
											ValidateFunc:     validateResourceQuantity,
											DiffSuppressFunc: suppressEquivalentResourceQuantity,
										},
										"resource": {
											Type:        schema.TypeString,
											Required:    true,
											Description: "Resource to select",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	v["empty_dir"] = &schema.Schema{
		Type:        schema.TypeList,
		Description: "EmptyDir represents a temporary directory that shares a pod's lifetime. More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"medium": {
					Type:        schema.TypeString,
					Description: `What type of storage medium should back this directory. The default is "" which means to use the node's default medium. Must be an empty string (default) or Memory. More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir`,
					Optional:    true,
					Default:     "",
					ForceNew:    !isUpdatable,
					ValidateFunc: validation.StringInSlice([]string{
						string(corev1.StorageMediumDefault),
						string(corev1.StorageMediumMemory),
					}, false),
				},
				"size_limit": {
					Type:             schema.TypeString,
					Description:      `Total amount of local storage required for this EmptyDir volume.`,
					Optional:         true,
					ForceNew:         !isUpdatable,
					ValidateFunc:     validateResourceQuantity,
					DiffSuppressFunc: suppressEquivalentResourceQuantity,
				},
			},
		},
	}

	v["ephemeral"] = &schema.Schema{
		Type:        schema.TypeList,
		Description: "Represents an ephemeral volume that is handled by a normal storage driver. More info: https://kubernetes.io/docs/concepts/storage/ephemeral-volumes/#generic-ephemeral-volumes",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"volume_claim_template": {
					Type:        schema.TypeList,
					Description: "Will be used to create a stand-alone PVC to provision the volume. The pod in which this EphemeralVolumeSource is embedded will be the owner of the PVC.",
					Required:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"metadata": {
								Type:        schema.TypeList,
								Description: "May contain labels and annotations that will be copied into the PVC when creating it. No other fields are allowed and will be rejected during validation.",
								Optional:    true,
								MaxItems:    1,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"annotations": {
											Type:         schema.TypeMap,
											Description:  "An unstructured key value map stored with the persistent volume claim that may be used to store arbitrary metadata. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/",
											Optional:     true,
											Elem:         &schema.Schema{Type: schema.TypeString},
											ValidateFunc: validateAnnotations,
										},
										"labels": {
											Type:         schema.TypeMap,
											Description:  "Map of string keys and values that can be used to organize and categorize (scope and select) the persistent volume claim. May match selectors of replication controllers and services. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/",
											Optional:     true,
											Elem:         &schema.Schema{Type: schema.TypeString},
											ValidateFunc: validateLabels,
										},
									},
								},
							},
							"spec": {
								Type:        schema.TypeList,
								Description: "The specification for the PersistentVolumeClaim. The entire content is copied unchanged into the PVC that gets created from this template. The same fields as in a PersistentVolumeClaim are also valid here.",
								Required:    true,
								MaxItems:    1,
								Elem: &schema.Resource{
									Schema: persistentVolumeClaimSpecFields(),
								},
							},
						},
					},
				},
			},
		},
	}

	v["persistent_volume_claim"] = &schema.Schema{
		Type:        schema.TypeList,
		Description: "The specification of a persistent volume.",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"claim_name": {
					Type:        schema.TypeString,
					Description: "ClaimName is the name of a PersistentVolumeClaim in the same ",
					Optional:    true,
				},
				"read_only": {
					Type:        schema.TypeBool,
					Description: "Will force the ReadOnly setting in VolumeMounts.",
					Optional:    true,
					Default:     false,
				},
			},
		},
	}

	v["secret"] = &schema.Schema{
		Type:        schema.TypeList,
		Description: "Secret represents a secret that should populate this volume. More info: https://kubernetes.io/docs/concepts/storage/volumes#secrets",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"default_mode": {
					Type:         schema.TypeString,
					Description:  "Optional: mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.",
					Optional:     true,
					Default:      "0644",
					ValidateFunc: validateModeBits,
				},
				"items": {
					Type:        schema.TypeList,
					Description: "If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.",
					Optional:    true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"key": {
								Type:        schema.TypeString,
								Optional:    true,
								Description: "The key to project.",
							},
							"mode": {
								Type:         schema.TypeString,
								Optional:     true,
								Description:  "Optional: mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.",
								ValidateFunc: validateModeBits,
							},
							"path": {
								Type:         schema.TypeString,
								Optional:     true,
								ValidateFunc: validatePath,
								Description:  "The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.",
							},
						},
					},
				},
				"optional": {
					Type:        schema.TypeBool,
					Description: "Optional: Specify whether the Secret or its keys must be defined.",
					Optional:    true,
				},
				"secret_name": {
					Type:        schema.TypeString,
					Description: "Name of the secret in the pod's namespace to use. More info: https://kubernetes.io/docs/concepts/storage/volumes#secrets",
					Optional:    true,
				},
			},
		},
	}
	v["name"] = &schema.Schema{
		Type:        schema.TypeString,
		Description: "Volume's name. Must be a DNS_LABEL and unique within the pod. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
		Optional:    true,
	}

	v["projected"] = &schema.Schema{
		Type:        schema.TypeList,
		Description: "Projected represents a single volume that projects several volume sources into the same directory. More info: https://kubernetes.io/docs/concepts/storage/volumes/#projected",
		Optional:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"default_mode": {
					Type:         schema.TypeString,
					Description:  "Optional: mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.",
					Optional:     true,
					Default:      "0644",
					ValidateFunc: validateModeBits,
				},
				"sources": {
					Type:        schema.TypeList,
					Description: "Source of the volume to project in the directory.",
					Required:    true,
					MinItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							// identical to SecretVolumeSource but without the default mode and uses a local object reference as name instead of a secret name.
							"secret": {
								Type:        schema.TypeList,
								Description: "Secret represents a secret that should populate this volume. More info: https://kubernetes.io/docs/concepts/storage/volumes#secrets",
								Optional:    true,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"name": {
											Type:        schema.TypeString,
											Description: "Name of the secret in the pod's namespace to use. More info: https://kubernetes.io/docs/concepts/storage/volumes#secrets",
											Optional:    true,
										},
										"items": {
											Type:        schema.TypeList,
											Description: "If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.",
											Optional:    true,
											Elem: &schema.Resource{
												Schema: map[string]*schema.Schema{
													"key": {
														Type:        schema.TypeString,
														Optional:    true,
														Description: "The key to project.",
													},
													"mode": {
														Type:         schema.TypeString,
														Optional:     true,
														Description:  "Optional: mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.",
														ValidateFunc: validateModeBits,
													},
													"path": {
														Type:         schema.TypeString,
														Optional:     true,
														ValidateFunc: validatePath,
														Description:  "The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.",
													},
												},
											},
										},
										"optional": {
											Type:        schema.TypeBool,
											Description: "Optional: Specify whether the Secret or it's keys must be defined.",
											Optional:    true,
										},
									},
								},
							},
							// identical to ConfigMapVolumeSource but without the default mode and uses a local object reference as name instead of a secret name.
							"config_map": {
								Type:        schema.TypeList,
								Description: "ConfigMap represents a configMap that should populate this volume",
								Optional:    true,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"name": {
											Type:        schema.TypeString,
											Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
											Optional:    true,
										},
										"items": {
											Type:        schema.TypeList,
											Description: "If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error. Paths must be relative and may not contain the '..' path or start with '..'.",
											Optional:    true,
											Elem: &schema.Resource{
												Schema: map[string]*schema.Schema{
													"key": {
														Type:        schema.TypeString,
														Optional:    true,
														Description: "The key to project.",
													},
													"mode": {
														Type:         schema.TypeString,
														Optional:     true,
														Description:  "Optional: mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.",
														ValidateFunc: validateModeBits,
													},
													"path": {
														Type:         schema.TypeString,
														Optional:     true,
														ValidateFunc: validatePath,
														Description:  "The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.",
													},
												},
											},
										},
										"optional": {
											Type:        schema.TypeBool,
											Description: "Optional: Specify whether the ConfigMap or it's keys must be defined.",
											Optional:    true,
										},
									},
								},
							},
							// identical to DownwardAPIVolumeSource but without the default mode.
							"downward_api": {
								Type:        schema.TypeList,
								Description: "DownwardAPI represents downward API about the pod that should populate this volume",
								Optional:    true,
								MaxItems:    1,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"items": {
											Type:        schema.TypeList,
											Description: "Represents a volume containing downward API info. Downward API volumes support ownership management and SELinux relabeling.",
											Optional:    true,
											Elem: &schema.Resource{
												Schema: map[string]*schema.Schema{
													"field_ref": {
														Type:        schema.TypeList,
														Optional:    true,
														MaxItems:    1,
														Description: "Selects a field of the pod: only annotations, labels, name and namespace are supported.",
														Elem: &schema.Resource{
															Schema: map[string]*schema.Schema{
																"api_version": {
																	Type:        schema.TypeString,
																	Optional:    true,
																	Default:     "v1",
																	Description: "Version of the schema the FieldPath is written in terms of, defaults to 'v1'.",
																},
																"field_path": {
																	Type:        schema.TypeString,
																	Optional:    true,
																	Description: "Path of the field to select in the specified API version",
																},
															},
														},
													},
													"mode": {
														Type:         schema.TypeString,
														Optional:     true,
														Description:  "Mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.",
														ValidateFunc: validateModeBits,
													},
													"path": {
														Type:         schema.TypeString,
														Required:     true,
														ValidateFunc: validatePath,
														Description:  "Path is the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'",
													},
													"resource_field_ref": {
														Type:        schema.TypeList,
														Optional:    true,
														MaxItems:    1,
														Description: "Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.",
														Elem: &schema.Resource{
															Schema: map[string]*schema.Schema{
																"container_name": {
																	Type:     schema.TypeString,
																	Required: true,
																},
																"divisor": {
																	Type:             schema.TypeString,
																	Optional:         true,
																	Default:          "1",
																	ValidateFunc:     validateResourceQuantity,
																	DiffSuppressFunc: suppressEquivalentResourceQuantity,
																},
																"resource": {
																	Type:        schema.TypeString,
																	Required:    true,
																	Description: "Resource to select",
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
							"service_account_token": {
								Type:        schema.TypeList,
								Description: "A projected service account token volume",
								Optional:    true,
								MaxItems:    1,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"audience": {
											Type:        schema.TypeString,
											Description: "Audience is the intended audience of the token",
											Optional:    true,
										},
										"expiration_seconds": {
											Type:         schema.TypeInt,
											Optional:     true,
											Default:      3600,
											Description:  "ExpirationSeconds is the expected duration of validity of the service account token. It defaults to 1 hour and must be at least 10 minutes (600 seconds).",
											ValidateFunc: validateIntGreaterThan(600),
										},
										"path": {
											Type:        schema.TypeString,
											Description: "Path specifies a relative path to the mount point of the projected volume.",
											Required:    true,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	v["csi"] = &schema.Schema{
		Type:        schema.TypeList,
		Description: "Represents a CSI Volume. More info: https://kubernetes.io/docs/concepts/storage/volumes#csi",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"driver": {
					Type:        schema.TypeString,
					Description: "the name of the volume driver to use. More info: https://kubernetes.io/docs/concepts/storage/volumes/#csi",
					Required:    true,
				},
				"volume_attributes": {
					Type:        schema.TypeMap,
					Description: "Attributes of the volume to publish.",
					Optional:    true,
				},
				"fs_type": {
					Type:        schema.TypeString,
					Description: "Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified.",
					Optional:    true,
				},
				"read_only": {
					Type:        schema.TypeBool,
					Description: "Whether to set the read-only property in VolumeMounts to \"true\". If omitted, the default is \"false\". More info: https://kubernetes.io/docs/concepts/storage/volumes#csi",
					Optional:    true,
				},
				"node_publish_secret_ref": {
					Type:        schema.TypeList,
					Description: "A reference to the secret object containing sensitive information to pass to the CSI driver to complete the CSI NodePublishVolume and NodeUnpublishVolume calls.",
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"name": {
								Type:        schema.TypeString,
								Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
								Optional:    true,
							},
						},
					},
				},
			},
		},
	}
	return &schema.Resource{
		Schema: v,
	}
}
