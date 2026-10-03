// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podspec

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

func (b builder) podContainerObject() schema.NestedBlockObject {
	resources := schema.ListNestedAttribute{
		Description: "Compute resources required by this container. Omit or use null to retain API-populated values; a configured list must contain exactly one object. Use [{}] to leave limits and requests unset. An empty list is not omission.",
		Optional:    true, Computed: true,
		Validators:    []validator.List{listvalidator.SizeBetween(1, 1)},
		PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
		NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"limits":   b.quantityMap(true, immutable),
			"requests": b.quantityMap(true, immutable),
		}},
	}
	if b.replace(immutable) {
		resources.PlanModifiers = append(resources.PlanModifiers, podListStructureRequiresReplace{absentZero: true})
	}
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"args":                       b.emptyCompatibleList(immutable, types.StringType),
			"command":                    b.emptyCompatibleList(immutable, types.StringType),
			"image":                      b.str(false, false, immutable, ""),
			"image_pull_policy":          b.str(false, true, immutable, ""),
			"name":                       b.str(true, false, immutable, ""),
			"restart_policy":             b.str(false, true, immutable, "", stringvalidator.OneOf("Always")),
			"stdin":                      b.boolean(false, immutable, false),
			"stdin_once":                 b.boolean(false, immutable, false),
			"termination_message_path":   b.str(false, false, immutable, "/dev/termination-log"),
			"termination_message_policy": b.str(false, true, immutable, "", stringvalidator.OneOf("File", "FallbackToLogsOnError")),
			"tty":                        b.boolean(false, immutable, false),
			"working_dir":                b.str(false, false, immutable, ""),
			"resources":                  resources,
		},
		Blocks: map[string]schema.Block{
			"env": b.block(schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"name":  b.str(true, false, immutable, ""),
					"value": b.str(false, false, immutable, ""),
				},
				Blocks: map[string]schema.Block{
					"value_from": b.block(b.podEnvValueFromObject(), 0, 1, updatable),
				},
			}, 0, 0, updatable),
			"env_from": b.block(schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"prefix": b.str(false, false, immutable, ""),
				},
				Blocks: map[string]schema.Block{
					"config_map_ref": b.block(b.podEnvSourceObject(), 0, 1, updatable),
					"secret_ref":     b.block(b.podEnvSourceObject(), 0, 1, updatable),
				},
			}, 0, 0, updatable),
			"lifecycle": b.block(schema.NestedBlockObject{Blocks: map[string]schema.Block{
				"post_start": b.block(b.podHandlerObject(), 0, 0, immutable),
				"pre_stop":   b.block(b.podHandlerObject(), 0, 0, immutable),
			}}, 0, 1, updatable),
			"liveness_probe":  b.block(b.podProbeObject(), 0, 1, immutable),
			"readiness_probe": b.block(b.podProbeObject(), 0, 1, immutable),
			"startup_probe":   b.block(b.podProbeObject(), 0, 1, immutable),
			"port": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"container_port": b.integer(true, false, immutable, 0, int64validator.Between(1, 65535)),
				"host_ip":        b.str(false, false, immutable, ""),
				"host_port":      b.integer(false, false, immutable, 0, int64validator.Between(1, 65535)),
				"name":           b.str(false, false, immutable, "", podStringRule("port")),
				"protocol":       b.str(false, false, immutable, "TCP", stringvalidator.OneOf("TCP", "UDP")),
			}}, 0, 0, updatable),
			"security_context": b.block(b.podContainerSecurityContextObject(), 0, 1, immutable),
			"volume_mount": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"mount_path":        b.str(true, false, updatable, ""),
				"name":              b.str(true, false, updatable, ""),
				"read_only":         b.boolean(false, updatable, false),
				"sub_path":          b.str(false, false, updatable, ""),
				"sub_path_expr":     b.str(false, false, updatable, ""),
				"mount_propagation": b.str(false, false, updatable, "None", stringvalidator.OneOf("None", "HostToContainer", "Bidirectional")),
			}}, 0, 0, immutable),
			"volume_device": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"device_path": b.str(true, false, updatable, ""),
				"name":        b.str(true, false, updatable, ""),
			}}, 0, 0, immutable),
		},
	}
}

func (b builder) podEnvSourceObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"name":     b.str(true, false, immutable, ""),
		"optional": b.boolean(false, immutable, false),
	}}
}

func (b builder) podEnvValueFromObject() schema.NestedBlockObject {
	keyReference := schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"key":      b.str(false, false, immutable, ""),
		"name":     b.str(false, false, immutable, ""),
		"optional": b.boolean(false, immutable, false),
	}}
	return schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"config_map_key_ref": b.block(keyReference, 0, 1, updatable),
		"secret_key_ref":     b.block(keyReference, 0, 1, updatable),
		"field_ref":          b.block(b.podFieldReferenceObject(immutable), 0, 1, updatable),
		"resource_field_ref": b.block(b.podResourceFieldReferenceObject(false, immutable), 0, 1, updatable),
	}}
}

func (b builder) podFieldReferenceObject(f forceNew) schema.NestedBlockObject {
	return schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"api_version": b.str(false, false, f, "v1"),
		"field_path":  b.str(false, false, f, ""),
	}}
}

func (b builder) podResourceFieldReferenceObject(requiredContainer bool, f forceNew) schema.NestedBlockObject {
	return schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"container_name": b.str(requiredContainer, false, f, ""),
		"divisor":        b.quantityString(updatable, "1"),
		"resource":       b.str(true, false, f, ""),
	}}
}

func (b builder) podHandlerObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"exec": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"command": b.list(false, updatable, types.StringType),
		}}, 0, 1, updatable),
		"http_get": b.block(schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"host":   b.str(false, false, updatable, ""),
				"path":   b.httpGetPath(),
				"scheme": b.str(false, false, updatable, "HTTP", stringvalidator.OneOf("HTTP", "HTTPS")),
				"port":   b.str(false, false, updatable, "", podStringRule("port")),
			},
			Blocks: map[string]schema.Block{
				"http_header": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"name":  b.str(false, false, updatable, ""),
					"value": b.str(false, false, updatable, ""),
				}}, 0, 0, updatable),
			},
		}, 0, 1, updatable),
		"tcp_socket": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"port": b.str(true, false, updatable, "", podStringRule("port")),
		}}, 0, 0, updatable),
	}}
}

func (b builder) podProbeObject() schema.NestedBlockObject {
	o := b.podHandlerObject()
	o.Attributes = map[string]schema.Attribute{
		"failure_threshold":     b.integer(false, false, updatable, 3, int64validator.AtLeast(1)),
		"initial_delay_seconds": b.integer(false, false, updatable, 0),
		"period_seconds":        b.integer(false, false, updatable, 10, int64validator.AtLeast(1)),
		"success_threshold":     b.integer(false, false, updatable, 1, int64validator.AtLeast(1)),
		"timeout_seconds":       b.integer(false, false, updatable, 1, int64validator.AtLeast(1)),
	}
	o.Blocks["grpc"] = b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"port":    b.integer(true, false, updatable, 0, int64validator.Between(1, 65535)),
		"service": b.str(false, false, updatable, ""),
	}}, 0, 0, updatable)
	return o
}

func (b builder) podContainerSecurityContextObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"allow_privilege_escalation": b.boolean(false, immutable, true),
			"privileged":                 b.boolean(false, immutable, false),
			"read_only_root_filesystem":  b.boolean(false, immutable, false),
			"run_as_group":               b.str(false, false, immutable, "", podStringRule("nullable-int")),
			"run_as_user":                b.str(false, false, immutable, "", podStringRule("nullable-int")),
			"run_as_non_root":            b.boolean(false, immutable, false),
		},
		Blocks: map[string]schema.Block{
			"capabilities": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"add":  b.list(false, immutable, types.StringType),
				"drop": b.list(false, immutable, types.StringType),
			}}, 0, 1, updatable),
			"seccomp_profile":  b.block(b.podSeccompObject(), 0, 1, updatable),
			"se_linux_options": b.block(b.podSELinuxObject(), 0, 1, updatable),
		},
	}
}

func (b builder) podSeccompObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"localhost_profile": b.str(false, false, immutable, ""),
		"type":              b.str(false, false, immutable, "Unconfined", stringvalidator.OneOf("Localhost", "RuntimeDefault", "Unconfined")),
	}}
}

func (b builder) podSELinuxObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"level": b.str(false, false, immutable, ""),
		"role":  b.str(false, false, immutable, ""),
		"type":  b.str(false, false, immutable, ""),
		"user":  b.str(false, false, immutable, ""),
	}}
}

func (b builder) podSecurityContextObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"fs_group":               b.str(false, false, immutable, "", podStringRule("nullable-int")),
			"run_as_group":           b.str(false, false, immutable, "", podStringRule("nullable-int")),
			"run_as_non_root":        b.boolean(false, immutable, false),
			"run_as_user":            b.str(false, false, immutable, "", podStringRule("nullable-int")),
			"fs_group_change_policy": b.str(false, false, immutable, "", stringvalidator.OneOf("Always", "OnRootMismatch")),
			"supplemental_groups":    b.set(immutable, types.Int64Type),
		},
		Blocks: map[string]schema.Block{
			"seccomp_profile":  b.block(b.podSeccompObject(), 0, 1, updatable),
			"se_linux_options": b.block(b.podSELinuxObject(), 0, 1, updatable),
			"windows_options": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"gmsa_credential_spec":      b.str(false, false, updatable, ""),
				"gmsa_credential_spec_name": b.str(false, false, updatable, ""),
				"host_process":              b.boolean(false, updatable, false),
				"run_as_username":           b.str(false, false, updatable, ""),
			}}, 0, 1, updatable),
			"sysctl": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"name":  b.str(true, false, immutable, ""),
				"value": b.str(true, false, immutable, ""),
			}}, 0, 0, updatable),
		},
	}
}
