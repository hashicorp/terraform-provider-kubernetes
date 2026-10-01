// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podtemplate

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
			"args":                       podList(false, types.StringType),
			"command":                    podList(false, types.StringType),
			"image":                      podString(false, false, false, ""),
			"image_pull_policy":          podString(false, true, false, ""),
			"name":                       podString(true, false, false, ""),
			"restart_policy":             podString(false, true, false, "", stringvalidator.OneOf("Always")),
			"stdin":                      podBool(false, false),
			"stdin_once":                 podBool(false, false),
			"termination_message_path":   podString(false, false, false, "/dev/termination-log"),
			"termination_message_policy": podString(false, true, false, "", stringvalidator.OneOf("File", "FallbackToLogsOnError")),
			"tty":                        podBool(false, false),
			"working_dir":                podString(false, false, false, ""),
			"resources": schema.ListNestedAttribute{
				Description: "Compute resources required by this container. Omit or use null to retain API-populated values; a configured list must contain exactly one object. Use [{}] to leave limits and requests unset. An empty list is not omission.",
				Optional:    true, Computed: true,
				Validators:    []validator.List{listvalidator.SizeBetween(1, 1)},
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"limits":   podResourceQuantityMap(),
					"requests": podResourceQuantityMap(),
				}},
			},
		},
		Blocks: map[string]schema.Block{
			"env": podBlock(schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"name":  podString(true, false, false, ""),
					"value": podString(false, false, false, ""),
				},
				Blocks: map[string]schema.Block{
					"value_from": podBlock(podEnvValueFromObject(), 0, 1, false),
				},
			}, 0, 0, false),
			"env_from": podBlock(schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"prefix": podString(false, false, false, ""),
				},
				Blocks: map[string]schema.Block{
					"config_map_ref": podBlock(podEnvSourceObject(), 0, 1, false),
					"secret_ref":     podBlock(podEnvSourceObject(), 0, 1, false),
				},
			}, 0, 0, false),
			"lifecycle": podBlock(schema.NestedBlockObject{Blocks: map[string]schema.Block{
				"post_start": podBlock(podHandlerObject(), 0, 0, false),
				"pre_stop":   podBlock(podHandlerObject(), 0, 0, false),
			}}, 0, 1, false),
			"liveness_probe":  podBlock(podProbeObject(), 0, 1, false),
			"readiness_probe": podBlock(podProbeObject(), 0, 1, false),
			"startup_probe":   podBlock(podProbeObject(), 0, 1, false),
			"port": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"container_port": podInt(true, false, 0, int64validator.Between(1, 65535)),
				"host_ip":        podString(false, false, false, ""),
				"host_port":      podInt(false, false, 0, int64validator.Between(1, 65535)),
				"name":           podString(false, false, false, "", podStringRule("port")),
				"protocol":       podString(false, false, false, "TCP", stringvalidator.OneOf("TCP", "UDP")),
			}}, 0, 0, false),
			"security_context": podBlock(podContainerSecurityContextObject(), 0, 1, false),
			"volume_mount": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"mount_path":        podString(true, false, false, ""),
				"name":              podString(true, false, false, ""),
				"read_only":         podBool(false, false),
				"sub_path":          podString(false, false, false, ""),
				"sub_path_expr":     podString(false, false, false, ""),
				"mount_propagation": podString(false, false, false, "None", stringvalidator.OneOf("None", "HostToContainer", "Bidirectional")),
			}}, 0, 0, false),
			"volume_device": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"device_path": podString(true, false, false, ""),
				"name":        podString(true, false, false, ""),
			}}, 0, 0, false),
		},
	}
}

func podEnvSourceObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"name":     podString(true, false, false, ""),
		"optional": podBool(false, false),
	}}
}

func podEnvValueFromObject() schema.NestedBlockObject {
	keyReference := schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"key":      podString(false, false, false, ""),
		"name":     podString(false, false, false, ""),
		"optional": podBool(false, false),
	}}
	return schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"config_map_key_ref": podBlock(keyReference, 0, 1, false),
		"secret_key_ref":     podBlock(keyReference, 0, 1, false),
		"field_ref":          podBlock(podFieldReferenceObject(false), 0, 1, false),
		"resource_field_ref": podBlock(podResourceFieldReferenceObject(false, false), 0, 1, false),
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
		"divisor":        podQuantityString("1"),
		"resource":       podString(true, false, replace, ""),
	}}
}

func podHandlerObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"exec": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"command": podList(false, types.StringType),
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
		"failure_threshold":     podInt(false, false, 3, int64validator.AtLeast(1)),
		"initial_delay_seconds": podInt(false, false, 0),
		"period_seconds":        podInt(false, false, 10, int64validator.AtLeast(1)),
		"success_threshold":     podInt(false, false, 1, int64validator.AtLeast(1)),
		"timeout_seconds":       podInt(false, false, 1, int64validator.AtLeast(1)),
	}
	o.Blocks["grpc"] = podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"port":    podInt(true, false, 0, int64validator.Between(1, 65535)),
		"service": podString(false, false, false, ""),
	}}, 0, 0, false)
	return o
}

func podContainerSecurityContextObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"allow_privilege_escalation": podBool(false, true),
			"privileged":                 podBool(false, false),
			"read_only_root_filesystem":  podBool(false, false),
			"run_as_group":               podString(false, false, false, "", podStringRule("nullable-int")),
			"run_as_user":                podString(false, false, false, "", podStringRule("nullable-int")),
			"run_as_non_root":            podBool(false, false),
		},
		Blocks: map[string]schema.Block{
			"capabilities": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"add":  podList(false, types.StringType),
				"drop": podList(false, types.StringType),
			}}, 0, 1, false),
			"seccomp_profile":  podBlock(podSeccompObject(), 0, 1, false),
			"se_linux_options": podBlock(podSELinuxObject(), 0, 1, false),
		},
	}
}

func podSeccompObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"localhost_profile": podString(false, false, false, ""),
		"type":              podString(false, false, false, "Unconfined", stringvalidator.OneOf("Localhost", "RuntimeDefault", "Unconfined")),
	}}
}

func podSELinuxObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"level": podString(false, false, false, ""),
		"role":  podString(false, false, false, ""),
		"type":  podString(false, false, false, ""),
		"user":  podString(false, false, false, ""),
	}}
}

func podSecurityContextObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"fs_group":               podString(false, false, false, "", podStringRule("nullable-int")),
			"run_as_group":           podString(false, false, false, "", podStringRule("nullable-int")),
			"run_as_non_root":        podBool(false, false),
			"run_as_user":            podString(false, false, false, "", podStringRule("nullable-int")),
			"fs_group_change_policy": podString(false, false, false, "", stringvalidator.OneOf("Always", "OnRootMismatch")),
			"supplemental_groups":    podSet(false, types.Int64Type),
		},
		Blocks: map[string]schema.Block{
			"seccomp_profile":  podBlock(podSeccompObject(), 0, 1, false),
			"se_linux_options": podBlock(podSELinuxObject(), 0, 1, false),
			"windows_options": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"gmsa_credential_spec":      podString(false, false, false, ""),
				"gmsa_credential_spec_name": podString(false, false, false, ""),
				"host_process":              podBool(false, false),
				"run_as_username":           podString(false, false, false, ""),
			}}, 0, 1, false),
			"sysctl": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"name":  podString(true, false, false, ""),
				"value": podString(true, false, false, ""),
			}}, 0, 0, false),
		},
	}
}
