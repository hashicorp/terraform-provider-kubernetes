// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	api "k8s.io/api/core/v1"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func lifecycleHandlerFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"exec": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "exec specifies the action to take.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"command": {
						Type:        schema.TypeList,
						Description: `Command is the command line to execute inside the container, the working directory for the command is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.`,
						Optional:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
					},
				},
			},
		},
		"http_get": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "Specifies the http request to perform.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"host": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: `Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.`,
					},
					"path": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: `Path to access on the HTTP server.`,
					},
					"scheme": {
						Type:        schema.TypeString,
						Optional:    true,
						Default:     string(api.URISchemeHTTP),
						Description: `Scheme to use for connecting to the host.`,
						ValidateFunc: validation.StringInSlice([]string{
							string(api.URISchemeHTTP),
							string(api.URISchemeHTTPS),
						}, false),
					},
					"port": {
						Type:         schema.TypeString,
						Optional:     true,
						ValidateFunc: validatePortNumOrName,
						Description:  `Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
					},
					"http_header": {
						Type:        schema.TypeList,
						Optional:    true,
						Description: `Scheme to use for connecting to the host.`,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"name": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "The header field name",
								},
								"value": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "The header field value",
								},
							},
						},
					},
				},
			},
		},
		"tcp_socket": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"port": {
						Type:         schema.TypeString,
						Required:     true,
						ValidateFunc: validatePortNumOrName,
						Description:  "Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.",
					},
				},
			},
		},
	}
}

func resourcesFieldV1(isUpdatable bool) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"limits": {
			Type:        schema.TypeMap,
			Optional:    true,
			Computed:    true,
			ForceNew:    !isUpdatable,
			Description: "Describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			DiffSuppressFunc: suppressEquivalentResourceQuantity,
		},
		"requests": {
			Type:        schema.TypeMap,
			Optional:    true,
			Computed:    true,
			ForceNew:    !isUpdatable,
			Description: "Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			DiffSuppressFunc: suppressEquivalentResourceQuantity,
		},
	}
}

func resourcesFieldV0() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"limits": {
			Type:     schema.TypeList,
			Optional: true,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"cpu": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"memory": {
						Type:     schema.TypeString,
						Optional: true,
					},
				},
			},
		},
		"requests": {
			Type:     schema.TypeList,
			Optional: true,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"cpu": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"memory": {
						Type:     schema.TypeString,
						Optional: true,
					},
				},
			},
		},
	}
}

func seccompProfileField(isUpdatable bool) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"localhost_profile": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Default:     "",
			Description: "Localhost Profile indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work.",
		},
		"type": {
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: !isUpdatable,
			Default:  string(api.SeccompProfileTypeUnconfined),
			ValidateFunc: validation.StringInSlice([]string{
				string(api.SeccompProfileTypeLocalhost),
				string(api.SeccompProfileTypeRuntimeDefault),
				string(api.SeccompProfileTypeUnconfined),
			}, false),
			Description: "Type indicates which kind of seccomp profile will be applied. Valid options are: Localhost, RuntimeDefault, Unconfined.",
		},
	}
}

func seLinuxOptionsField(isUpdatable bool) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"level": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Description: "Level is SELinux level label that applies to the container.",
		},
		"role": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Description: "Role is a SELinux role label that applies to the container.",
		},
		"type": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Description: "Type is a SELinux type label that applies to the container.",
		},
		"user": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Description: "User is a SELinux user label that applies to the container.",
		},
	}
}

func volumeMountFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"mount_path": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Path within the container at which the volume should be mounted. Must not contain ':'.",
		},
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "This must match the Name of a Volume.",
		},
		"read_only": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false.",
		},
		"sub_path": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: `Path within the volume from which the container's volume should be mounted. Defaults to "" (volume's root).`,
		},
		"sub_path_expr": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: `Dynamic path within the volume from which the container's volume should be mounted. Defaults to "" (volume's root).`,
		},
		"mount_propagation": {
			Type:         schema.TypeString,
			Description:  "Mount propagation mode. mount_propagation determines how mounts are propagated from the host to container and the other way around. Valid values are None (default), HostToContainer and Bidirectional.",
			Optional:     true,
			Default:      "None",
			ValidateFunc: validation.StringInSlice([]string{"None", "HostToContainer", "Bidirectional"}, false),
		},
	}
}

func volumeDeviceFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"device_path": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Path within the container at which the volume device should be attached. For example '/dev/xvda'.",
		},
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "This must match the Name of a PersistentVolumeClaim.",
		},
	}
}

func containerFields(isUpdatable bool) map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"args": {
			Type:        schema.TypeList,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "Arguments to the entrypoint. The docker image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell",
		},
		"command": {
			Type:        schema.TypeList,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "Entrypoint array. Not executed within a shell. The docker image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell",
		},
		"env": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "List of environment variables to set in the container. Cannot be updated.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"name": {
						Type:        schema.TypeString,
						Required:    true,
						ForceNew:    !isUpdatable,
						Description: "Name of the environment variable. Must be a C_IDENTIFIER",
					},
					"value": {
						Type:        schema.TypeString,
						ForceNew:    !isUpdatable,
						Optional:    true,
						Description: `Variable references $(VAR_NAME) are expanded using the previous defined environment variables in the container and any service environment variables. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Defaults to "".`,
					},
					"value_from": {
						Type:        schema.TypeList,
						Optional:    true,
						MaxItems:    1,
						Description: "Source for the environment variable's value",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"config_map_key_ref": {
									Type:        schema.TypeList,
									Optional:    true,
									MaxItems:    1,
									Description: "Selects a key of a ConfigMap.",
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"key": {
												Type:        schema.TypeString,
												Optional:    true,
												ForceNew:    !isUpdatable,
												Description: "The key to select.",
											},
											"name": {
												Type:        schema.TypeString,
												Optional:    true,
												ForceNew:    !isUpdatable,
												Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
											},
											"optional": {
												Type:        schema.TypeBool,
												Optional:    true,
												ForceNew:    !isUpdatable,
												Description: "Specify whether the ConfigMap or its key must be defined.",
											},
										},
									},
								},
								"field_ref": {
									Type:        schema.TypeList,
									Optional:    true,
									MaxItems:    1,
									Description: "Selects a field of the pod: supports metadata.name, metadata.namespace, metadata.labels, metadata.annotations, spec.nodeName, spec.serviceAccountName, status.podIP.",
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"api_version": {
												Type:        schema.TypeString,
												Optional:    true,
												ForceNew:    !isUpdatable,
												Default:     "v1",
												Description: `Version of the schema the FieldPath is written in terms of, defaults to "v1".`,
											},
											"field_path": {
												Type:        schema.TypeString,
												Optional:    true,
												ForceNew:    !isUpdatable,
												Description: "Path of the field to select in the specified API version",
											},
										},
									},
								},
								"resource_field_ref": {
									Type:        schema.TypeList,
									Optional:    true,
									MaxItems:    1,
									Description: "Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.ephemeral-storage) are currently supported.",
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"container_name": {
												Type:     schema.TypeString,
												Optional: true,
												ForceNew: !isUpdatable,
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
												ForceNew:    !isUpdatable,
												Description: "Resource to select",
											},
										},
									},
								},
								"secret_key_ref": {
									Type:        schema.TypeList,
									Optional:    true,
									MaxItems:    1,
									Description: "Selects a key of a secret in the pod's namespace.",
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"key": {
												Type:        schema.TypeString,
												Optional:    true,
												ForceNew:    !isUpdatable,
												Description: "The key of the secret to select from. Must be a valid secret key.",
											},
											"name": {
												Type:        schema.TypeString,
												Optional:    true,
												ForceNew:    !isUpdatable,
												Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
											},
											"optional": {
												Type:        schema.TypeBool,
												Optional:    true,
												ForceNew:    !isUpdatable,
												Description: "Specify whether the Secret or its key must be defined.",
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
		"env_from": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER. All invalid keys will be reported as an event when the container is starting. When a key exists in multiple sources, the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"config_map_ref": {
						Type:        schema.TypeList,
						Optional:    true,
						MaxItems:    1,
						Description: "The ConfigMap to select from",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"name": {
									Type:        schema.TypeString,
									Required:    true,
									ForceNew:    !isUpdatable,
									Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
								},
								"optional": {
									Type:        schema.TypeBool,
									Optional:    true,
									ForceNew:    !isUpdatable,
									Description: "Specify whether the ConfigMap must be defined",
								},
							},
						},
					},
					"prefix": {
						Type:        schema.TypeString,
						Optional:    true,
						ForceNew:    !isUpdatable,
						Description: "An optional identifer to prepend to each key in the ConfigMap. Must be a C_IDENTIFIER.",
					},
					"secret_ref": {
						Type:        schema.TypeList,
						Optional:    true,
						MaxItems:    1,
						Description: "The Secret to select from",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"name": {
									Type:        schema.TypeString,
									Required:    true,
									ForceNew:    !isUpdatable,
									Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
								},
								"optional": {
									Type:        schema.TypeBool,
									Optional:    true,
									ForceNew:    !isUpdatable,
									Description: "Specify whether the Secret must be defined",
								},
							},
						},
					},
				},
			},
		},
		"image": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Description: "Docker image name. More info: https://kubernetes.io/docs/concepts/containers/images/",
		},
		"image_pull_policy": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			ForceNew:    !isUpdatable,
			Description: "Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/images/#updating-images",
		},
		"lifecycle": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "Actions that the management system should take in response to container lifecycle events",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"post_start": {
						Type:        schema.TypeList,
						Description: `post_start is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy. Other management of the container blocks until the hook completes. More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks`,
						Optional:    true,
						ForceNew:    !isUpdatable,
						Elem: &schema.Resource{
							Schema: lifecycleHandlerFields(),
						},
					},
					"pre_stop": {
						Type:        schema.TypeList,
						Description: `pre_stop is called immediately before a container is terminated. The container is terminated after the handler completes. The reason for termination is passed to the handler. Regardless of the outcome of the handler, the container is eventually terminated. Other management of the container blocks until the hook completes. More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks`,
						Optional:    true,
						ForceNew:    !isUpdatable,
						Elem: &schema.Resource{
							Schema: lifecycleHandlerFields(),
						},
					},
				},
			},
		},
		"liveness_probe": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			ForceNew:    !isUpdatable,
			Description: "Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes",
			Elem:        probeSchema(),
		},
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    !isUpdatable,
			Description: "Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated.",
		},
		"port": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: `List of ports to expose from the container. Exposing a port here gives the system additional information about the network connections a container uses, but is primarily informational. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default "0.0.0.0" address inside a container will be accessible from the network. Cannot be updated.`,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"container_port": {
						Type:         schema.TypeInt,
						Required:     true,
						ValidateFunc: validatePortNumOrName,
						ForceNew:     !isUpdatable,
						Description:  "Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536.",
					},
					"host_ip": {
						Type:        schema.TypeString,
						Optional:    true,
						ForceNew:    !isUpdatable,
						Description: "What host IP to bind the external port to.",
					},
					"host_port": {
						Type:         schema.TypeInt,
						Optional:     true,
						ForceNew:     !isUpdatable,
						Description:  "Number of port to expose on the host. If specified, this must be a valid port number, 0 < x < 65536. If HostNetwork is specified, this must match ContainerPort. Most containers do not need this.",
						ValidateFunc: validation.IsPortNumber,
					},
					"name": {
						Type:         schema.TypeString,
						Optional:     true,
						ForceNew:     !isUpdatable,
						ValidateFunc: validatePortNumOrName,
						Description:  "If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services",
					},
					"protocol": {
						Type:        schema.TypeString,
						Optional:    true,
						ForceNew:    !isUpdatable,
						Description: `Protocol for port. Must be UDP or TCP. Defaults to "TCP".`,
						Default:     string(api.ProtocolTCP),
						ValidateFunc: validation.StringInSlice([]string{
							string(api.ProtocolTCP),
							string(api.ProtocolUDP),
						}, false),
					},
				},
			},
		},
		"readiness_probe": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			ForceNew:    !isUpdatable,
			Description: "Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes",
			Elem:        probeSchema(),
		},
		"resources": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			ForceNew:    !isUpdatable,
			Computed:    true,
			Description: "Compute Resources required by this container. Cannot be updated. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources",
			Elem: &schema.Resource{
				Schema: resourcesFieldV1(isUpdatable),
			},
		},
		"security_context": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			ForceNew:    !isUpdatable,
			Description: "Security options the pod should run with. More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/",
			Elem:        securityContextSchema(isUpdatable),
		},
		"startup_probe": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			ForceNew:    !isUpdatable,
			Description: "StartupProbe indicates that the Pod has successfully initialized. If specified, no other probes are executed until this completes successfully. If this probe fails, the Pod will be restarted, just as if the livenessProbe failed. This can be used to provide different probe parameters at the beginning of a Pod's lifecycle, when it might take a long time to load data or warm a cache, than during steady-state operation. This cannot be updated. This is an alpha feature enabled by the StartupProbe feature flag. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes",
			Elem:        probeSchema(),
		},
		"stdin": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Default:     false,
			Description: "Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF. ",
		},
		"stdin_once": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Default:     false,
			Description: "Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin, and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted. If this flag is false, a container processes that reads from stdin will never receive an EOF.",
		},
		"termination_message_path": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Default:     "/dev/termination-log",
			Description: "Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem. Message written is intended to be brief final status, such as an assertion failure message. Defaults to /dev/termination-log. Cannot be updated.",
		},
		"termination_message_policy": {
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: !isUpdatable,
			ValidateFunc: validation.StringInSlice([]string{
				string(api.TerminationMessageReadFile),
				string(api.TerminationMessageFallbackToLogsOnError),
			}, false),
			Computed:    true,
			Description: "Optional: Indicate how the termination message should be populated. File will use the contents of terminationMessagePath to populate the container status message on both success and failure. FallbackToLogsOnError will use the last chunk of container log output if the termination message file is empty and the container exited with an error. The log output is limited to 2048 bytes or 80 lines, whichever is smaller. Defaults to File. Cannot be updated.",
		},
		"tty": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Default:     false,
			Description: "Whether this container should allocate a TTY for itself",
		},
		"volume_mount": {
			Type:        schema.TypeList,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Description: "Pod volumes to mount into the container's filesystem. Cannot be updated.",
			Elem: &schema.Resource{
				Schema: volumeMountFields(),
			},
		},
		"volume_device": {
			Type:        schema.TypeList,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Description: "Raw volume devices to attach into the container's filesystem as raw block devices. Cannot be updated.",
			Elem: &schema.Resource{
				Schema: volumeDeviceFields(),
			},
		},
		"working_dir": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Description: "Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated.",
		},
	}
	return s
}

func probeSchema() *schema.Resource {
	h := lifecycleHandlerFields()
	h["grpc"] = &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: "GRPC specifies an action involving a GRPC port.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"port": {
					Type:         schema.TypeInt,
					Required:     true,
					ValidateFunc: validatePortNum,
					Description:  "Number of the port to access on the container. Number must be in the range 1 to 65535.",
				},
				"service": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md). If this is not specified, the default behavior is defined by gRPC.",
				},
			},
		},
	}
	h["failure_threshold"] = &schema.Schema{
		Type:         schema.TypeInt,
		Optional:     true,
		Description:  "Minimum consecutive failures for the probe to be considered failed after having succeeded.",
		Default:      3,
		ValidateFunc: validatePositiveInteger,
	}
	h["initial_delay_seconds"] = &schema.Schema{
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes",
	}
	h["period_seconds"] = &schema.Schema{
		Type:         schema.TypeInt,
		Optional:     true,
		Default:      10,
		ValidateFunc: validatePositiveInteger,
		Description:  "How often (in seconds) to perform the probe",
	}
	h["success_threshold"] = &schema.Schema{
		Type:         schema.TypeInt,
		Optional:     true,
		Default:      1,
		ValidateFunc: validatePositiveInteger,
		Description:  "Minimum consecutive successes for the probe to be considered successful after having failed.",
	}

	h["timeout_seconds"] = &schema.Schema{
		Type:         schema.TypeInt,
		Optional:     true,
		Default:      1,
		ValidateFunc: validatePositiveInteger,
		Description:  "Number of seconds after which the probe times out. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes",
	}
	return &schema.Resource{
		Schema: h,
	}

}

func securityContextSchema(isUpdatable bool) *schema.Resource {
	m := map[string]*schema.Schema{
		"allow_privilege_escalation": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			ForceNew:    !isUpdatable,
			Description: `AllowPrivilegeEscalation controls whether a process can gain more privileges than its parent process. This bool directly controls if the no_new_privs flag will be set on the container process. AllowPrivilegeEscalation is true always when the container is: 1) run as Privileged 2) has CAP_SYS_ADMIN`,
		},
		"capabilities": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"add": {
						Type:        schema.TypeList,
						Optional:    true,
						ForceNew:    !isUpdatable,
						Elem:        &schema.Schema{Type: schema.TypeString},
						Description: "Added capabilities",
					},
					"drop": {
						Type:        schema.TypeList,
						Optional:    true,
						ForceNew:    !isUpdatable,
						Elem:        &schema.Schema{Type: schema.TypeString},
						Description: "Removed capabilities",
					},
				},
			},
		},
		"privileged": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Default:     false,
			Description: `Run container in privileged mode. Processes in privileged containers are essentially equivalent to root on the host. Defaults to false.`,
		},
		"read_only_root_filesystem": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    !isUpdatable,
			Default:     false,
			Description: "Whether this container has a read-only root filesystem. Default is false.",
		},
		"run_as_group": {
			Type:         schema.TypeString,
			Description:  "The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.",
			Optional:     true,
			ForceNew:     !isUpdatable,
			ValidateFunc: validateTypeStringNullableInt,
		},
		"run_as_non_root": {
			Type:        schema.TypeBool,
			Description: "Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.",
			ForceNew:    !isUpdatable,
			Optional:    true,
		},
		"run_as_user": {
			Type:         schema.TypeString,
			Description:  "The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.",
			Optional:     true,
			ForceNew:     !isUpdatable,
			ValidateFunc: validateTypeStringNullableInt,
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
			Description: "The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.",
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: seLinuxOptionsField(isUpdatable),
			},
		},
	}

	return &schema.Resource{
		Schema: m,
	}
}
