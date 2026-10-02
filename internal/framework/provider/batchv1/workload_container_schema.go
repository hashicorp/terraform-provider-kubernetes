// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func workloadConfigMapKeyRefObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				Description:   "The key to select.",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"name": schema.StringAttribute{
				Description:   "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"optional": schema.BoolAttribute{
				Description:   "Specify whether the ConfigMap or its key must be defined.",
				Optional:      true,
				PlanModifiers: workloadBoolPlanModifiers(updatable),
			},
		},
	}
}

func workloadFieldRefObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				Description:   "Version of the schema the FieldPath is written in terms of, defaults to \"v1\".",
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString("v1"),
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"field_path": schema.StringAttribute{
				Description:   "Path of the field to select in the specified API version",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
		},
	}
}

func workloadResourceFieldRefObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"container_name": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"divisor": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString("1"),
				CustomType:    workloadQuantityType{},
				Validators:    []validator.String{workloadQuantityValidator()},
				PlanModifiers: []planmodifier.String{workloadQuantityStringPlanModifier{}},
			},
			"resource": schema.StringAttribute{
				Description:   "Resource to select",
				Required:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
		},
	}
}

func workloadSecretKeyRefObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				Description:   "The key of the secret to select from. Must be a valid secret key.",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"name": schema.StringAttribute{
				Description:   "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"optional": schema.BoolAttribute{
				Description:   "Specify whether the Secret or its key must be defined.",
				Optional:      true,
				PlanModifiers: workloadBoolPlanModifiers(updatable),
			},
		},
	}
}

func workloadValueFromObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Blocks: map[string]schema.Block{
			"config_map_key_ref": schema.ListNestedBlock{
				Description:  "Selects a key of a ConfigMap.",
				NestedObject: workloadConfigMapKeyRefObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"field_ref": schema.ListNestedBlock{
				Description:  "Selects a field of the pod: supports metadata.name, metadata.namespace, metadata.labels, metadata.annotations, spec.nodeName, spec.serviceAccountName, status.podIP.",
				NestedObject: workloadFieldRefObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"resource_field_ref": schema.ListNestedBlock{
				Description:  "Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.ephemeral-storage) are currently supported.",
				NestedObject: workloadResourceFieldRefObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"secret_key_ref": schema.ListNestedBlock{
				Description:  "Selects a key of a secret in the pod's namespace.",
				NestedObject: workloadSecretKeyRefObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
		},
	}
}

func workloadEnvObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description:   "Name of the environment variable. Must be a C_IDENTIFIER",
				Required:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"value": schema.StringAttribute{
				Description:   "Variable references $(VAR_NAME) are expanded using the previous defined environment variables in the container and any service environment variables. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Defaults to \"\".",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
		},
		Blocks: map[string]schema.Block{
			"value_from": schema.ListNestedBlock{
				Description:  "Source for the environment variable's value",
				NestedObject: workloadValueFromObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
		},
	}
}

func workloadConfigMapRefObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description:   "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
				Required:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"optional": schema.BoolAttribute{
				Description:   "Specify whether the ConfigMap must be defined",
				Optional:      true,
				PlanModifiers: workloadBoolPlanModifiers(updatable),
			},
		},
	}
}

func workloadSecretRefObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description:   "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
				Required:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"optional": schema.BoolAttribute{
				Description:   "Specify whether the Secret must be defined",
				Optional:      true,
				PlanModifiers: workloadBoolPlanModifiers(updatable),
			},
		},
	}
}

func workloadEnvFromObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"prefix": schema.StringAttribute{
				Description:   "An optional identifer to prepend to each key in the ConfigMap. Must be a C_IDENTIFIER.",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
		},
		Blocks: map[string]schema.Block{
			"config_map_ref": schema.ListNestedBlock{
				Description:  "The ConfigMap to select from",
				NestedObject: workloadConfigMapRefObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"secret_ref": schema.ListNestedBlock{
				Description:  "The Secret to select from",
				NestedObject: workloadSecretRefObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
		},
	}
}

func workloadExecObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"command": schema.ListAttribute{
				Description: "Command is the command line to execute inside the container, the working directory for the command is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func workloadHttpHeaderObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The header field name",
				Optional:    true,
			},
			"value": schema.StringAttribute{
				Description: "The header field value",
				Optional:    true,
			},
		},
	}
}

func workloadHttpGetObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "Host name to connect to, defaults to the pod IP. You probably want to set \"Host\" in httpHeaders instead.",
				Optional:    true,
			},
			"path": schema.StringAttribute{
				Description: "Path to access on the HTTP server.",
				Optional:    true,
			},
			"port": schema.StringAttribute{
				Description: "Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.",
				Optional:    true,
				Validators:  []validator.String{workloadPortValidator()},
			},
			"scheme": schema.StringAttribute{
				Description: "Scheme to use for connecting to the host.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("HTTP"),
				Validators:  []validator.String{stringvalidator.OneOf("HTTP", "HTTPS")},
			},
		},
		Blocks: map[string]schema.Block{
			"http_header": schema.ListNestedBlock{
				Description:  "Scheme to use for connecting to the host.",
				NestedObject: workloadHttpHeaderObject(updatable),
			},
		},
	}
}

func workloadTcpSocketObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"port": schema.StringAttribute{
				Description: "Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.",
				Required:    true,
				Validators:  []validator.String{workloadPortValidator()},
			},
		},
	}
}

func workloadPostStartObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Blocks: map[string]schema.Block{
			"exec": schema.ListNestedBlock{
				Description:  "exec specifies the action to take.",
				NestedObject: workloadExecObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"http_get": schema.ListNestedBlock{
				Description:  "Specifies the http request to perform.",
				NestedObject: workloadHttpGetObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"tcp_socket": schema.ListNestedBlock{
				Description:  "TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported",
				NestedObject: workloadTcpSocketObject(updatable),
			},
		},
	}
}

func workloadLifecycleObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Blocks: map[string]schema.Block{
			"post_start": schema.ListNestedBlock{
				Description:   "post_start is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy. Other management of the container blocks until the hook completes. More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks",
				NestedObject:  workloadPostStartObject(updatable),
				PlanModifiers: workloadObjectListPlanModifiers(updatable),
			},
			"pre_stop": schema.ListNestedBlock{
				Description:   "pre_stop is called immediately before a container is terminated. The container is terminated after the handler completes. The reason for termination is passed to the handler. Regardless of the outcome of the handler, the container is eventually terminated. Other management of the container blocks until the hook completes. More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks",
				NestedObject:  workloadPostStartObject(updatable),
				PlanModifiers: workloadObjectListPlanModifiers(updatable),
			},
		},
	}
}

func workloadGrpcObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"port": schema.Int64Attribute{
				Description: "Number of the port to access on the container. Number must be in the range 1 to 65535.",
				Required:    true,
				Validators:  []validator.Int64{int64validator.Between(1, 65535)},
			},
			"service": schema.StringAttribute{
				Description: "Name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md). If this is not specified, the default behavior is defined by gRPC.",
				Optional:    true,
			},
		},
	}
}

func workloadLivenessProbeObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"failure_threshold": schema.Int64Attribute{
				Description: "Minimum consecutive failures for the probe to be considered failed after having succeeded.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(3),
				Validators:  []validator.Int64{int64validator.AtLeast(1)},
			},
			"initial_delay_seconds": schema.Int64Attribute{
				Description: "Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes",
				Optional:    true,
			},
			"period_seconds": schema.Int64Attribute{
				Description: "How often (in seconds) to perform the probe",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(10),
				Validators:  []validator.Int64{int64validator.AtLeast(1)},
			},
			"success_threshold": schema.Int64Attribute{
				Description: "Minimum consecutive successes for the probe to be considered successful after having failed.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Validators:  []validator.Int64{int64validator.AtLeast(1)},
			},
			"timeout_seconds": schema.Int64Attribute{
				Description: "Number of seconds after which the probe times out. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Validators:  []validator.Int64{int64validator.AtLeast(1)},
			},
		},
		Blocks: map[string]schema.Block{
			"exec": schema.ListNestedBlock{
				Description:  "exec specifies the action to take.",
				NestedObject: workloadExecObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"grpc": schema.ListNestedBlock{
				Description:  "GRPC specifies an action involving a GRPC port.",
				NestedObject: workloadGrpcObject(updatable),
			},
			"http_get": schema.ListNestedBlock{
				Description:  "Specifies the http request to perform.",
				NestedObject: workloadHttpGetObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"tcp_socket": schema.ListNestedBlock{
				Description:  "TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported",
				NestedObject: workloadTcpSocketObject(updatable),
			},
		},
	}
}

func workloadPortObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"container_port": schema.Int64Attribute{
				Description:   "Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536.",
				Required:      true,
				Validators:    []validator.Int64{int64validator.Between(1, 65535)},
				PlanModifiers: workloadInt64PlanModifiers(updatable),
			},
			"host_ip": schema.StringAttribute{
				Description:   "What host IP to bind the external port to.",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"host_port": schema.Int64Attribute{
				Description:   "Number of port to expose on the host. If specified, this must be a valid port number, 0 < x < 65536. If HostNetwork is specified, this must match ContainerPort. Most containers do not need this.",
				Optional:      true,
				Validators:    []validator.Int64{int64validator.Between(1, 65535)},
				PlanModifiers: workloadInt64PlanModifiers(updatable),
			},
			"name": schema.StringAttribute{
				Description:   "If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services",
				Optional:      true,
				Validators:    []validator.String{workloadPortValidator()},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"protocol": schema.StringAttribute{
				Description:   "Protocol for port. Must be UDP or TCP. Defaults to \"TCP\".",
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString("TCP"),
				Validators:    []validator.String{stringvalidator.OneOf("TCP", "UDP")},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
		},
	}
}

func workloadResourcesObject(updatable bool) schema.NestedAttributeObject {
	return schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"limits": schema.MapAttribute{
				Description:   "Describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/",
				Optional:      true,
				Computed:      true,
				ElementType:   workloadQuantityType{},
				PlanModifiers: []planmodifier.Map{workloadQuantityMapPlanModifier{requiresReplace: !updatable}},
			},
			"requests": schema.MapAttribute{
				Description:   "Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/",
				Optional:      true,
				Computed:      true,
				ElementType:   workloadQuantityType{},
				PlanModifiers: []planmodifier.Map{workloadQuantityMapPlanModifier{requiresReplace: !updatable}},
			},
		},
	}
}

func workloadCapabilitiesObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"add": schema.ListAttribute{
				Description:   "Added capabilities",
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: workloadListPlanModifiers(updatable),
			},
			"drop": schema.ListAttribute{
				Description:   "Removed capabilities",
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: workloadListPlanModifiers(updatable),
			},
		},
	}
}

func workloadSeLinuxOptionsObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"level": schema.StringAttribute{
				Description:   "Level is SELinux level label that applies to the container.",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"role": schema.StringAttribute{
				Description:   "Role is a SELinux role label that applies to the container.",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"type": schema.StringAttribute{
				Description:   "Type is a SELinux type label that applies to the container.",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"user": schema.StringAttribute{
				Description:   "User is a SELinux user label that applies to the container.",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
		},
	}
}

func workloadSeccompProfileObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"localhost_profile": schema.StringAttribute{
				Description:   "Localhost Profile indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work.",
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString(""),
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"type": schema.StringAttribute{
				Description:   "Type indicates which kind of seccomp profile will be applied. Valid options are: Localhost, RuntimeDefault, Unconfined.",
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString("Unconfined"),
				Validators:    []validator.String{stringvalidator.OneOf("Localhost", "RuntimeDefault", "Unconfined")},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
		},
	}
}

func workloadSecurityContextObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"allow_privilege_escalation": schema.BoolAttribute{
				Description:   "AllowPrivilegeEscalation controls whether a process can gain more privileges than its parent process. This bool directly controls if the no_new_privs flag will be set on the container process. AllowPrivilegeEscalation is true always when the container is: 1) run as Privileged 2) has CAP_SYS_ADMIN",
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(true),
				PlanModifiers: workloadBoolPlanModifiers(updatable),
			},
			"privileged": schema.BoolAttribute{
				Description:   "Run container in privileged mode. Processes in privileged containers are essentially equivalent to root on the host. Defaults to false.",
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				PlanModifiers: workloadBoolPlanModifiers(updatable),
			},
			"read_only_root_filesystem": schema.BoolAttribute{
				Description:   "Whether this container has a read-only root filesystem. Default is false.",
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				PlanModifiers: workloadBoolPlanModifiers(updatable),
			},
			"run_as_group": schema.StringAttribute{
				Description:   "The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.",
				Optional:      true,
				Validators:    []validator.String{workloadNullableIntValidator()},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"run_as_non_root": schema.BoolAttribute{
				Description:   "Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.",
				Optional:      true,
				PlanModifiers: workloadBoolPlanModifiers(updatable),
			},
			"run_as_user": schema.StringAttribute{
				Description:   "The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.",
				Optional:      true,
				Validators:    []validator.String{workloadNullableIntValidator()},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
		},
		Blocks: map[string]schema.Block{
			"capabilities": schema.ListNestedBlock{
				Description:  "The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime.",
				NestedObject: workloadCapabilitiesObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"se_linux_options": schema.ListNestedBlock{
				Description:  "The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.",
				NestedObject: workloadSeLinuxOptionsObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"seccomp_profile": schema.ListNestedBlock{
				Description:  "The seccomp options to use by the containers in this pod. Note that this field cannot be set when spec.os.name is windows.",
				NestedObject: workloadSeccompProfileObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
		},
	}
}

func workloadVolumeDeviceObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"device_path": schema.StringAttribute{
				Description: "Path within the container at which the volume device should be attached. For example '/dev/xvda'.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "This must match the Name of a PersistentVolumeClaim.",
				Required:    true,
			},
		},
	}
}

func workloadVolumeMountObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"mount_path": schema.StringAttribute{
				Description: "Path within the container at which the volume should be mounted. Must not contain ':'.",
				Required:    true,
			},
			"mount_propagation": schema.StringAttribute{
				Description: "Mount propagation mode. mount_propagation determines how mounts are propagated from the host to container and the other way around. Valid values are None (default), HostToContainer and Bidirectional.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("None"),
				Validators:  []validator.String{stringvalidator.OneOf("None", "HostToContainer", "Bidirectional")},
			},
			"name": schema.StringAttribute{
				Description: "This must match the Name of a Volume.",
				Required:    true,
			},
			"read_only": schema.BoolAttribute{
				Description: "Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"sub_path": schema.StringAttribute{
				Description: "Path within the volume from which the container's volume should be mounted. Defaults to \"\" (volume's root).",
				Optional:    true,
			},
			"sub_path_expr": schema.StringAttribute{
				Description: "Dynamic path within the volume from which the container's volume should be mounted. Defaults to \"\" (volume's root).",
				Optional:    true,
			},
		},
	}
}

func workloadContainerObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"args": schema.ListAttribute{
				Description:   "Arguments to the entrypoint. The docker image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell",
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: workloadListPlanModifiers(updatable),
			},
			"command": schema.ListAttribute{
				Description:   "Entrypoint array. Not executed within a shell. The docker image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell",
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: workloadListPlanModifiers(updatable),
			},
			"image": schema.StringAttribute{
				Description:   "Docker image name. More info: https://kubernetes.io/docs/concepts/containers/images/",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"image_pull_policy": schema.StringAttribute{
				Description:   "Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/images/#updating-images",
				Optional:      true,
				Computed:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"name": schema.StringAttribute{
				Description:   "Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated.",
				Required:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"resources": schema.ListNestedAttribute{
				Description:   "Compute resources required by this container, configured as a list of objects. Changing limits or requests requires Job replacement but updates CronJob templates in place. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/",
				Optional:      true,
				Computed:      true,
				NestedObject:  workloadResourcesObject(updatable),
				Validators:    []validator.List{listvalidator.SizeAtMost(1)},
				PlanModifiers: workloadObjectListPlanModifiers(updatable),
			},
			"restart_policy": schema.StringAttribute{
				Description:   "Restart policy for designating init container as a sidecar. Can only be `Always`. More info: https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/#pod-sidecar-containers.",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.String{stringvalidator.OneOf("Always")},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"stdin": schema.BoolAttribute{
				Description:   "Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF. ",
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				PlanModifiers: workloadBoolPlanModifiers(updatable),
			},
			"stdin_once": schema.BoolAttribute{
				Description:   "Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin, and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted. If this flag is false, a container processes that reads from stdin will never receive an EOF.",
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				PlanModifiers: workloadBoolPlanModifiers(updatable),
			},
			"termination_message_path": schema.StringAttribute{
				Description:   "Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem. Message written is intended to be brief final status, such as an assertion failure message. Defaults to /dev/termination-log. Cannot be updated.",
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString("/dev/termination-log"),
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"termination_message_policy": schema.StringAttribute{
				Description:   "Optional: Indicate how the termination message should be populated. File will use the contents of terminationMessagePath to populate the container status message on both success and failure. FallbackToLogsOnError will use the last chunk of container log output if the termination message file is empty and the container exited with an error. The log output is limited to 2048 bytes or 80 lines, whichever is smaller. Defaults to File. Cannot be updated.",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.String{stringvalidator.OneOf("File", "FallbackToLogsOnError")},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"tty": schema.BoolAttribute{
				Description:   "Whether this container should allocate a TTY for itself",
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				PlanModifiers: workloadBoolPlanModifiers(updatable),
			},
			"working_dir": schema.StringAttribute{
				Description:   "Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated.",
				Optional:      true,
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
		},
		Blocks: map[string]schema.Block{
			"env": schema.ListNestedBlock{
				Description:  "List of environment variables to set in the container. Cannot be updated.",
				NestedObject: workloadEnvObject(updatable),
			},
			"env_from": schema.ListNestedBlock{
				Description:  "List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER. All invalid keys will be reported as an event when the container is starting. When a key exists in multiple sources, the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated.",
				NestedObject: workloadEnvFromObject(updatable),
			},
			"lifecycle": schema.ListNestedBlock{
				Description:  "Actions that the management system should take in response to container lifecycle events",
				NestedObject: workloadLifecycleObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"liveness_probe": schema.ListNestedBlock{
				Description:   "Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes",
				NestedObject:  workloadLivenessProbeObject(updatable),
				Validators:    []validator.List{listvalidator.SizeAtMost(1)},
				PlanModifiers: workloadObjectListPlanModifiers(updatable),
			},
			"port": schema.ListNestedBlock{
				Description:  "List of ports to expose from the container. Exposing a port here gives the system additional information about the network connections a container uses, but is primarily informational. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default \"0.0.0.0\" address inside a container will be accessible from the network. Cannot be updated.",
				NestedObject: workloadPortObject(updatable),
			},
			"readiness_probe": schema.ListNestedBlock{
				Description:   "Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes",
				NestedObject:  workloadLivenessProbeObject(updatable),
				Validators:    []validator.List{listvalidator.SizeAtMost(1)},
				PlanModifiers: workloadObjectListPlanModifiers(updatable),
			},
			"security_context": schema.ListNestedBlock{
				Description:   "Security options the pod should run with. More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/",
				NestedObject:  workloadSecurityContextObject(updatable),
				Validators:    []validator.List{listvalidator.SizeAtMost(1)},
				PlanModifiers: workloadObjectListPlanModifiers(updatable),
			},
			"startup_probe": schema.ListNestedBlock{
				Description:   "StartupProbe indicates that the Pod has successfully initialized. If specified, no other probes are executed until this completes successfully. If this probe fails, the Pod will be restarted, just as if the livenessProbe failed. This can be used to provide different probe parameters at the beginning of a Pod's lifecycle, when it might take a long time to load data or warm a cache, than during steady-state operation. This cannot be updated. This is an alpha feature enabled by the StartupProbe feature flag. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes",
				NestedObject:  workloadLivenessProbeObject(updatable),
				Validators:    []validator.List{listvalidator.SizeAtMost(1)},
				PlanModifiers: workloadObjectListPlanModifiers(updatable),
			},
			"volume_device": schema.ListNestedBlock{
				Description:   "Raw volume devices to attach into the container's filesystem as raw block devices. Cannot be updated.",
				NestedObject:  workloadVolumeDeviceObject(updatable),
				PlanModifiers: workloadObjectListPlanModifiers(updatable),
			},
			"volume_mount": schema.ListNestedBlock{
				Description:   "Pod volumes to mount into the container's filesystem. Cannot be updated.",
				NestedObject:  workloadVolumeMountObject(updatable),
				PlanModifiers: workloadObjectListPlanModifiers(updatable),
			},
		},
	}
}
