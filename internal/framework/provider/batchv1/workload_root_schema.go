// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

func workloadMetadataObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"annotations": schema.MapAttribute{
				Description: "An unstructured key value map stored with the job that may be used to store arbitrary metadata. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/",
				Optional:    true,
				ElementType: types.StringType,
				Validators:  []validator.Map{common.AnnotationsValidator()},
			},
			"generate_name": schema.StringAttribute{
				Description:   "Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency",
				Optional:      true,
				Validators:    []validator.String{common.DNSLabelPrefixValidator(), stringvalidator.ConflictsWith(path.MatchRoot("metadata").AtListIndex(0).AtName("name"))},
				PlanModifiers: []planmodifier.String{workloadStringRequiresReplace()},
			},
			"generation": schema.Int64Attribute{
				Description: "A sequence number representing a specific generation of the desired state.",
				Computed:    true,
			},
			"labels": schema.MapAttribute{
				Description: "Map of string keys and values that can be used to organize and categorize (scope and select) the job. May match selectors of replication controllers and services. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/",
				Optional:    true,
				ElementType: types.StringType,
				Validators:  []validator.Map{common.LabelsValidator()},
			},
			"name": schema.StringAttribute{
				Description:   "Name of the job, must be unique. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.String{common.DNSSubdomainNameValidator(), stringvalidator.ConflictsWith(path.MatchRoot("metadata").AtListIndex(0).AtName("generate_name"))},
				PlanModifiers: []planmodifier.String{workloadStringRequiresReplace()},
			},
			"resource_version": schema.StringAttribute{
				Description: "An opaque value that represents the internal version of this job that can be used by clients to determine when job has changed. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency",
				Computed:    true,
			},
			"uid": schema.StringAttribute{
				Description: "The unique in time and space value for this job. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids",
				Computed:    true,
			},
		},
	}
}

func workloadOptionObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description:   "Name of the option.",
				Required:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"value": schema.StringAttribute{
				Description:   "Value of the option. Optional: Defaults to empty.",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
		},
	}
}

func workloadDnsConfigObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"nameservers": schema.ListAttribute{
				Description:   "A list of DNS name server IP addresses. This will be appended to the base nameservers generated from DNSPolicy. Duplicated nameservers will be removed.",
				Optional:      true,
				ElementType:   types.StringType,
				Validators:    []validator.List{listvalidator.ValueStringsAre(workloadIPAddressValidator())},
				PlanModifiers: workloadListPlanModifiers(updatable),
			},
			"searches": schema.ListAttribute{
				Description: "A list of DNS search domains for host-name lookup. This will be appended to the base search paths generated from DNSPolicy. Duplicated search paths will be removed.",
				Optional:    true,
				ElementType: types.StringType,
				Validators:  []validator.List{listvalidator.ValueStringsAre(common.DNSSubdomainNameValidator())},
			},
		},
		Blocks: map[string]schema.Block{
			"option": schema.ListNestedBlock{
				Description:  "A list of DNS resolver options. This will be merged with the base options generated from DNSPolicy. Duplicated entries will be removed. Resolution options given in Options will override those that appear in the base DNSPolicy.",
				NestedObject: workloadOptionObject(updatable),
			},
		},
	}
}

func workloadHostAliasesObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"hostnames": schema.ListAttribute{
				Description:   "Hostnames for the IP address.",
				Required:      true,
				ElementType:   types.StringType,
				Validators:    []validator.List{listvalidator.SizeAtLeast(1)},
				PlanModifiers: workloadListPlanModifiers(updatable),
			},
			"ip": schema.StringAttribute{
				Description:   "IP address of the host file entry.",
				Required:      true,
				Validators:    []validator.String{workloadIPAddressValidator()},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
		},
	}
}

func workloadImagePullSecretsObject(updatable bool) schema.NestedAttributeObject {
	return schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description:   "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
				Required:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
		},
	}
}

func workloadOsObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name is the name of the operating system. The currently supported values are linux and windows.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.OneOf("linux", "windows")},
			},
		},
	}
}

func workloadReadinessGateObject(updatable bool) schema.NestedAttributeObject {
	return schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"condition_type": schema.StringAttribute{
				Description:   "refers to a condition in the pod's condition list with matching type.",
				Required:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
		},
	}
}

func workloadSysctlObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description:   "Name of a property to set.",
				Required:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"value": schema.StringAttribute{
				Description:   "Value of a property to set.",
				Required:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
		},
	}
}

func workloadWindowsOptionsObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"gmsa_credential_spec": schema.StringAttribute{
				Description: "GMSACredentialSpec is where the GMSA admission webhook inlines the contents of the GMSA credential spec named by the GMSACredentialSpecName field",
				Optional:    true,
			},
			"gmsa_credential_spec_name": schema.StringAttribute{
				Description: "GMSACredentialSpecName is the name of the GMSA credential spec to use.",
				Optional:    true,
			},
			"host_process": schema.BoolAttribute{
				Description: "HostProcess determines if a container should be run as a 'Host Process' container. Default value is false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"run_as_username": schema.StringAttribute{
				Description: "The UserName in Windows to run the entrypoint of the container process. Defaults to the user specified in image metadata if unspecified. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.",
				Optional:    true,
			},
		},
	}
}

func workloadSecurityContextObject2(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"fs_group": schema.StringAttribute{
				Description:   "A special supplemental group that applies to all containers in a pod. Some volume types allow the Kubelet to change the ownership of that volume to be owned by the pod: 1. The owning GID will be the FSGroup 2. The setgid bit is set (new files created in the volume will be owned by FSGroup) 3. The permission bits are OR'd with rw-rw---- If unset, the Kubelet will not modify the ownership and permissions of any volume.",
				Optional:      true,
				Validators:    []validator.String{workloadNullableIntValidator()},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"fs_group_change_policy": schema.StringAttribute{
				Description:   "fsGroupChangePolicy defines behavior of changing ownership and permission of the volume before being exposed inside Pod. This field will only apply to volume types which support fsGroup based ownership(and permissions). It will have no effect on ephemeral volume types such as: secret, configmaps and emptydir.",
				Optional:      true,
				Validators:    []validator.String{stringvalidator.OneOf("Always", "OnRootMismatch")},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"run_as_group": schema.StringAttribute{
				Description:   "The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container.",
				Optional:      true,
				Validators:    []validator.String{workloadNullableIntValidator()},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"run_as_non_root": schema.BoolAttribute{
				Description:   "Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.",
				Optional:      true,
				PlanModifiers: workloadBoolPlanModifiers(updatable),
			},
			"run_as_user": schema.StringAttribute{
				Description:   "The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container.",
				Optional:      true,
				Validators:    []validator.String{workloadNullableIntValidator()},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"supplemental_groups": schema.SetAttribute{
				Description:   "A list of groups applied to the first process run in each container, in addition to the container's primary GID. If unspecified, no groups will be added to any container.",
				Optional:      true,
				ElementType:   types.Int64Type,
				PlanModifiers: workloadSetPlanModifiers(updatable),
			},
		},
		Blocks: map[string]schema.Block{
			"se_linux_options": schema.ListNestedBlock{
				Description:  "The SELinux context to be applied to all containers. If unspecified, the container runtime will allocate a random SELinux context for each container. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container.",
				NestedObject: workloadSeLinuxOptionsObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"seccomp_profile": schema.ListNestedBlock{
				Description:  "The seccomp options to use by the containers in this pod. Note that this field cannot be set when spec.os.name is windows.",
				NestedObject: workloadSeccompProfileObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"sysctl": schema.ListNestedBlock{
				Description:  "holds a list of namespaced sysctls used for the pod.",
				NestedObject: workloadSysctlObject(updatable),
			},
			"windows_options": schema.ListNestedBlock{
				Description:  "The Windows specific settings applied to all containers. If unspecified, the options within a container's SecurityContext will be used. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is linux.",
				NestedObject: workloadWindowsOptionsObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
		},
	}
}

func workloadTolerationObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"effect": schema.StringAttribute{
				Description:   "Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.",
				Optional:      true,
				Validators:    []validator.String{stringvalidator.OneOf("NoSchedule", "PreferNoSchedule", "NoExecute")},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"key": schema.StringAttribute{
				Description:   "Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"operator": schema.StringAttribute{
				Description:   "Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.",
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString("Equal"),
				Validators:    []validator.String{stringvalidator.OneOf("Exists", "Equal")},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"toleration_seconds": schema.StringAttribute{
				Description:   "TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.",
				Optional:      true,
				Validators:    []validator.String{workloadNullableIntValidator()},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"value": schema.StringAttribute{
				Description:   "Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
		},
	}
}

func workloadTopologySpreadConstraintObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"match_label_keys": schema.SetAttribute{
				Description:   "is a set of pod label keys to select the pods over which spreading will be calculated.",
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: workloadSetPlanModifiers(updatable),
			},
			"max_skew": schema.Int64Attribute{
				Description: "describes the degree to which pods may be unevenly distributed.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Validators:  []validator.Int64{int64validator.AtLeast(1)},
			},
			"min_domains": schema.Int64Attribute{
				Description:   "indicates a minimum number of eligible domains.",
				Optional:      true,
				Validators:    []validator.Int64{int64validator.AtLeast(1)},
				PlanModifiers: workloadInt64PlanModifiers(updatable),
			},
			"node_affinity_policy": schema.StringAttribute{
				Description:   "indicates how we will treat Pod's nodeAffinity/nodeSelector when calculating pod topology spread skew.",
				Optional:      true,
				Validators:    []validator.String{stringvalidator.OneOf("Honor", "Ignore")},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"node_taints_policy": schema.StringAttribute{
				Description:   "indicates how we will treat node taints when calculating pod topology spread skew.",
				Optional:      true,
				Validators:    []validator.String{stringvalidator.OneOf("Honor", "Ignore")},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"topology_key": schema.StringAttribute{
				Description: "the key of node labels. Nodes that have a label with this key and identical values are considered to be in the same topology.",
				Optional:    true,
			},
			"when_unsatisfiable": schema.StringAttribute{
				Description: "indicates how to deal with a pod if it doesn't satisfy the spread constraint.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("DoNotSchedule"),
				Validators:  []validator.String{stringvalidator.OneOf("DoNotSchedule", "ScheduleAnyway")},
			},
		},
		Blocks: map[string]schema.Block{
			"label_selector": schema.ListNestedBlock{
				Description:  "A label query over a set of resources, in this case pods.",
				NestedObject: workloadLabelSelectorObject(updatable),
			},
		},
	}
}

func workloadPodSpecObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"active_deadline_seconds": schema.Int64Attribute{
				Description: "Optional duration in seconds the pod may be active on the node relative to StartTime before the system will actively try to mark it failed and kill associated containers. Value must be a positive integer.",
				Optional:    true,
				Validators:  []validator.Int64{int64validator.AtLeast(1)},
			},
			"automount_service_account_token": schema.BoolAttribute{
				Description:   "AutomountServiceAccountToken indicates whether a service account token should be automatically mounted.",
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(true),
				PlanModifiers: workloadBoolPlanModifiers(updatable),
			},
			"dns_policy": schema.StringAttribute{
				Description:   "Set DNS policy for containers within the pod. Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. To have DNS options set along with hostNetwork, you have to specify DNS policy explicitly to 'ClusterFirstWithHostNet'. Defaults to 'ClusterFirst'. More info: https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-dns-policy",
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString("ClusterFirst"),
				Validators:    []validator.String{stringvalidator.OneOf("ClusterFirst", "ClusterFirstWithHostNet", "Default", "None")},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"enable_service_links": schema.BoolAttribute{
				Description:   "Enables generating environment variables for service discovery. Defaults to true.",
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(true),
				PlanModifiers: workloadBoolPlanModifiers(updatable),
			},
			"host_ipc": schema.BoolAttribute{
				Description:   "Use the host's ipc namespace. Optional: Defaults to false.",
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				PlanModifiers: workloadBoolPlanModifiers(updatable),
			},
			"host_network": schema.BoolAttribute{
				Description:   "Host networking requested for this pod. Use the host's network namespace. If this option is set, the ports that will be used must be specified.",
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				PlanModifiers: workloadBoolPlanModifiers(updatable),
			},
			"host_pid": schema.BoolAttribute{
				Description:   "Use the host's pid namespace.",
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				PlanModifiers: workloadBoolPlanModifiers(updatable),
			},
			"hostname": schema.StringAttribute{
				Description:   "Specifies the hostname of the Pod If not specified, the pod's hostname will be set to a system-defined value.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"image_pull_secrets": schema.ListNestedAttribute{
				Description:  "ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec. If specified, these secrets will be passed to individual puller implementations for them to use. For example, in the case of docker, only DockerConfig type secrets are honored. More info: https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod",
				Optional:     true,
				Computed:     true,
				NestedObject: workloadImagePullSecretsObject(updatable),
			},
			"node_name": schema.StringAttribute{
				Description:   "NodeName is a request to schedule this pod onto a specific node. If it is non-empty, the scheduler simply schedules this pod onto that node, assuming that it fits resource requirements.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"node_selector": schema.MapAttribute{
				Description:   "NodeSelector is a selector which must be true for the pod to fit on a node. Selector which must match a node's labels for the pod to be scheduled on that node. More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/.",
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: workloadMapPlanModifiers(updatable),
			},
			"priority_class_name": schema.StringAttribute{
				Description:   "If specified, indicates the pod's priority. \"system-node-critical\" and \"system-cluster-critical\" are two special keywords which indicate the highest priorities with the former being the highest priority. Any other name must be defined by creating a PriorityClass object with that name. If not specified, the pod priority will be default or zero if there is no default.",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"readiness_gate": schema.ListNestedAttribute{
				Description:  "If specified, all readiness gates will be evaluated for pod readiness. A pod is ready when all its containers are ready AND all conditions specified in the readiness gates have status equal to \"True\" More info: https://git.k8s.io/enhancements/keps/sig-network/0007-pod-ready%2B%2B.md",
				Optional:     true,
				Computed:     true,
				NestedObject: workloadReadinessGateObject(updatable),
			},
			"restart_policy": schema.StringAttribute{
				Description:   "Restart policy for all containers within the pod. One of Always, OnFailure, Never. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#restart-policy.",
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString("Never"),
				Validators:    []validator.String{stringvalidator.OneOf("Always", "OnFailure", "Never")},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"runtime_class_name": schema.StringAttribute{
				Description:   "RuntimeClassName is a feature for selecting the container runtime configuration. The container runtime configuration is used to run a Pod's containers. More info: https://kubernetes.io/docs/concepts/containers/runtime-class",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"scheduler_name": schema.StringAttribute{
				Description:   "If specified, the pod will be dispatched by specified scheduler. If not specified, the pod will be dispatched by default scheduler.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"service_account_name": schema.StringAttribute{
				Description:   "ServiceAccountName is the name of the ServiceAccount to use to run this pod. More info: http://releases.k8s.io/HEAD/docs/design/service_accounts.md.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"share_process_namespace": schema.BoolAttribute{
				Description:   "Share a single process namespace between all of the containers in a pod. When this is set containers will be able to view and signal processes from other containers in the same pod, and the first process in each container will not be assigned PID 1. HostPID and ShareProcessNamespace cannot both be set. Optional: Defaults to false.",
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				PlanModifiers: workloadBoolPlanModifiers(updatable),
			},
			"subdomain": schema.StringAttribute{
				Description:   "If specified, the fully qualified Pod hostname will be \"...svc.\". If not specified, the pod will not have a domainname at all..",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"termination_grace_period_seconds": schema.Int64Attribute{
				Description:   "Optional duration in seconds the pod needs to terminate gracefully. May be decreased in delete request. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period will be used instead. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process.",
				Optional:      true,
				Computed:      true,
				Default:       int64default.StaticInt64(30),
				Validators:    []validator.Int64{int64validator.AtLeast(0)},
				PlanModifiers: workloadInt64PlanModifiers(updatable),
			},
		},
		Blocks: map[string]schema.Block{
			"affinity": schema.ListNestedBlock{
				Description:   "Optional pod scheduling constraints.",
				NestedObject:  workloadAffinityObject(updatable),
				Validators:    []validator.List{listvalidator.SizeAtMost(1)},
				PlanModifiers: workloadObjectListPlanModifiers(updatable),
			},
			"container": schema.ListNestedBlock{
				Description:  "List of containers belonging to the pod. Containers cannot currently be added or removed. There must be at least one container in a Pod. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/",
				NestedObject: workloadContainerObject(updatable),
			},
			"dns_config": schema.ListNestedBlock{
				Description:  "Specifies the DNS parameters of a pod. Parameters specified here will be merged to the generated DNS configuration based on DNSPolicy. Optional: Defaults to empty",
				NestedObject: workloadDnsConfigObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"host_aliases": schema.ListNestedBlock{
				Description:   "List of hosts and IPs that will be injected into the pod's hosts file if specified. Optional: Defaults to empty.",
				NestedObject:  workloadHostAliasesObject(updatable),
				PlanModifiers: workloadObjectListPlanModifiers(updatable),
			},
			"init_container": schema.ListNestedBlock{
				Description:   "List of init containers belonging to the pod. Init containers always run to completion and each must complete successfully before the next is started. More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/",
				NestedObject:  workloadContainerObject(updatable),
				PlanModifiers: workloadObjectListPlanModifiers(updatable),
			},
			"os": schema.ListNestedBlock{
				Description:  "Specifies the OS of the containers in the pod.",
				NestedObject: workloadOsObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"security_context": schema.ListNestedBlock{
				Description:  "SecurityContext holds pod-level security attributes and common container settings. Optional: Defaults to empty",
				NestedObject: workloadSecurityContextObject2(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"toleration": schema.ListNestedBlock{
				Description:  "If specified, the pod's toleration. Optional: Defaults to empty",
				NestedObject: workloadTolerationObject(updatable),
			},
			"topology_spread_constraint": schema.ListNestedBlock{
				Description:  "describes how a group of pods ought to spread across topology domains. Scheduler will schedule pods in a way which abides by the constraints.",
				NestedObject: workloadTopologySpreadConstraintObject(updatable),
			},
			"volume": schema.ListNestedBlock{
				Description:  "List of volumes that can be mounted by containers belonging to the pod. More info: https://kubernetes.io/docs/concepts/storage/volumes",
				NestedObject: workloadVolumeObject(updatable),
			},
		},
	}
}

func workloadPodTemplateObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Blocks: map[string]schema.Block{
			"metadata": schema.ListNestedBlock{
				Description:  "Standard job's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata",
				NestedObject: workloadMetadataObject(updatable),
				Validators:   []validator.List{listvalidator.IsRequired(), listvalidator.SizeBetween(1, 1)},
			},
			"spec": schema.ListNestedBlock{
				Description:   "Spec of the pods owned by the job. Changing the number of spec blocks requires replacement; nested fields have their own replacement rules.",
				NestedObject:  workloadPodSpecObject(updatable),
				Validators:    []validator.List{listvalidator.SizeAtMost(1)},
				PlanModifiers: []planmodifier.List{workloadTemplateListRequiresReplace(updatable)},
			},
		},
	}
}

func podTemplateBlock(updatable bool) schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description:   "Describes the pod that will be created when executing a job. Nested fields retain their own replacement rules. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/",
		NestedObject:  workloadPodTemplateObject(updatable),
		Validators:    []validator.List{listvalidator.IsRequired(), listvalidator.SizeBetween(1, 1)},
		PlanModifiers: []planmodifier.List{workloadTemplateListRequiresReplace(updatable)},
	}
}
