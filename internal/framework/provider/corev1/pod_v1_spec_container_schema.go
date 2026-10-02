// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func podContainerObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"args":                       podList(false, true, types.StringType),
			"command":                    podList(false, true, types.StringType),
			"image":                      podString(false, false, true, ""),
			"image_pull_policy":          podString(false, true, true, ""),
			"name":                       podString(true, false, true, ""),
			"restart_policy":             podString(false, true, true, "", stringvalidator.OneOf("Always")),
			"stdin":                      podBool(false, true, false),
			"stdin_once":                 podBool(false, true, false),
			"termination_message_path":   podString(false, false, true, "/dev/termination-log"),
			"termination_message_policy": podString(false, true, true, "", stringvalidator.OneOf("File", "FallbackToLogsOnError")),
			"tty":                        podBool(false, true, false),
			"working_dir":                podString(false, false, true, ""),
			"resources": schema.ListNestedAttribute{
				Description: "Compute resources required by this container. Cannot be updated. Omit or use null to allow API defaults; a configured list must contain exactly one object. Use [{}] to leave limits and requests unset for API defaults. An empty list is not omission.",
				Optional:    true, Computed: true,
				Validators:    []validator.List{listvalidator.SizeBetween(1, 1)},
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown(), podListStructureRequiresReplace{}},
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"limits":   podResourceQuantityMap(),
					"requests": podResourceQuantityMap(),
				}},
			},
		},
		Blocks: map[string]schema.Block{
			"env": podBlock(schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"name":  podString(true, false, true, ""),
					"value": podString(false, false, true, ""),
				},
				Blocks: map[string]schema.Block{
					"value_from": podBlock(podEnvValueFromObject(), 0, 1, false),
				},
			}, 0, 0, true),
			"env_from": podBlock(schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"prefix": podString(false, false, true, ""),
				},
				Blocks: map[string]schema.Block{
					"config_map_ref": podBlock(podEnvSourceObject(), 0, 1, false),
					"secret_ref":     podBlock(podEnvSourceObject(), 0, 1, false),
				},
			}, 0, 0, true),
			"lifecycle": podBlock(schema.NestedBlockObject{Blocks: map[string]schema.Block{
				"post_start": podBlock(podHandlerObject(), 0, 0, true),
				"pre_stop":   podBlock(podHandlerObject(), 0, 0, true),
			}}, 0, 1, false),
			"liveness_probe":  podBlock(podProbeObject(), 0, 1, true),
			"readiness_probe": podBlock(podProbeObject(), 0, 1, true),
			"startup_probe":   podBlock(podProbeObject(), 0, 1, true),
			"port": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"container_port": podInt(true, false, true, 0, int64validator.Between(1, 65535)),
				"host_ip":        podString(false, false, true, ""),
				"host_port":      podInt(false, false, true, 0, int64validator.Between(1, 65535)),
				"name":           podString(false, false, true, "", podStringRule("port")),
				"protocol":       podString(false, false, true, "TCP", stringvalidator.OneOf("TCP", "UDP")),
			}}, 0, 0, false),
			"security_context": podBlock(podContainerSecurityContextObject(), 0, 1, true),
			"volume_mount": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"mount_path":        podString(true, false, false, ""),
				"name":              podString(true, false, false, ""),
				"read_only":         podBool(false, false, false),
				"sub_path":          podString(false, false, false, ""),
				"sub_path_expr":     podString(false, false, false, ""),
				"mount_propagation": podString(false, false, false, "None", stringvalidator.OneOf("None", "HostToContainer", "Bidirectional")),
			}}, 0, 0, true),
			"volume_device": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"device_path": podString(true, false, false, ""),
				"name":        podString(true, false, false, ""),
			}}, 0, 0, true),
		},
	}
}

func podEnvSourceObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"name":     podString(true, false, true, ""),
		"optional": podBool(false, true, false),
	}}
}

func podEnvValueFromObject() schema.NestedBlockObject {
	keyReference := schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"key":      podString(false, false, true, ""),
		"name":     podString(false, false, true, ""),
		"optional": podBool(false, true, false),
	}}
	return schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"config_map_key_ref": podBlock(keyReference, 0, 1, false),
		"secret_key_ref":     podBlock(keyReference, 0, 1, false),
		"field_ref":          podBlock(podFieldReferenceObject(true), 0, 1, false),
		"resource_field_ref": podBlock(podResourceFieldReferenceObject(false, true), 0, 1, false),
	}}
}

func podFieldReferenceObject(replace bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"api_version": podString(false, false, replace, "v1"),
		"field_path":  podString(false, false, replace, ""),
	}}
}

func podResourceFieldReferenceObject(requiredContainer, replace bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"container_name": podString(requiredContainer, false, replace, ""),
		"divisor":        podQuantityString(false, false, "1"),
		"resource":       podString(true, false, replace, ""),
	}}
}

func podHandlerObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"exec": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"command": podList(false, false, types.StringType),
		}}, 0, 1, false),
		"http_get": podBlock(schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"host":   podString(false, false, false, ""),
				"path":   podString(false, false, false, ""),
				"scheme": podString(false, false, false, "HTTP", stringvalidator.OneOf("HTTP", "HTTPS")),
				"port":   podString(false, false, false, "", podStringRule("port")),
			},
			Blocks: map[string]schema.Block{
				"http_header": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"name":  podString(false, false, false, ""),
					"value": podString(false, false, false, ""),
				}}, 0, 0, false),
			},
		}, 0, 1, false),
		"tcp_socket": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"port": podString(true, false, false, "", podStringRule("port")),
		}}, 0, 0, false),
	}}
}

func podProbeObject() schema.NestedBlockObject {
	o := podHandlerObject()
	o.Attributes = map[string]schema.Attribute{
		"failure_threshold":     podInt(false, false, false, 3, int64validator.AtLeast(1)),
		"initial_delay_seconds": podInt(false, false, false, 0),
		"period_seconds":        podInt(false, false, false, 10, int64validator.AtLeast(1)),
		"success_threshold":     podInt(false, false, false, 1, int64validator.AtLeast(1)),
		"timeout_seconds":       podInt(false, false, false, 1, int64validator.AtLeast(1)),
	}
	o.Blocks["grpc"] = podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"port":    podInt(true, false, false, 0, int64validator.Between(1, 65535)),
		"service": podString(false, false, false, ""),
	}}, 0, 0, false)
	return o
}

func podContainerSecurityContextObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"allow_privilege_escalation": podBool(false, true, true),
			"privileged":                 podBool(false, true, false),
			"read_only_root_filesystem":  podBool(false, true, false),
			"run_as_group":               podString(false, false, true, "", podStringRule("nullable-int")),
			"run_as_user":                podString(false, false, true, "", podStringRule("nullable-int")),
			"run_as_non_root":            podBool(false, true, false),
		},
		Blocks: map[string]schema.Block{
			"capabilities": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"add":  podList(false, true, types.StringType),
				"drop": podList(false, true, types.StringType),
			}}, 0, 1, false),
			"seccomp_profile":  podBlock(podSeccompObject(), 0, 1, false),
			"se_linux_options": podBlock(podSELinuxObject(), 0, 1, false),
		},
	}
}

func podSeccompObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"localhost_profile": podString(false, false, true, ""),
		"type":              podString(false, false, true, "Unconfined", stringvalidator.OneOf("Localhost", "RuntimeDefault", "Unconfined")),
	}}
}

func podSELinuxObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"level": podString(false, false, true, ""),
		"role":  podString(false, false, true, ""),
		"type":  podString(false, false, true, ""),
		"user":  podString(false, false, true, ""),
	}}
}

func podSecurityContextObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"fs_group":               podString(false, false, true, "", podStringRule("nullable-int")),
			"run_as_group":           podString(false, false, true, "", podStringRule("nullable-int")),
			"run_as_non_root":        podBool(false, true, false),
			"run_as_user":            podString(false, false, true, "", podStringRule("nullable-int")),
			"fs_group_change_policy": podString(false, false, true, "", stringvalidator.OneOf("Always", "OnRootMismatch")),
			"supplemental_groups":    podSet(true, types.Int64Type),
		},
		Blocks: map[string]schema.Block{
			"seccomp_profile":  podBlock(podSeccompObject(), 0, 1, false),
			"se_linux_options": podBlock(podSELinuxObject(), 0, 1, false),
			"windows_options": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"gmsa_credential_spec":      podString(false, false, false, ""),
				"gmsa_credential_spec_name": podString(false, false, false, ""),
				"host_process":              podBool(false, false, false),
				"run_as_username":           podString(false, false, false, ""),
			}}, 0, 1, false),
			"sysctl": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"name":  podString(true, false, true, ""),
				"value": podString(true, false, true, ""),
			}}, 0, 0, false),
		},
	}
}
