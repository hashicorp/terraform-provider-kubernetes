// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

func (r *JobV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID
	if req.Identity != nil && !req.Identity.Raw.IsNull() {
		var identity common.NamespacedResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.APIVersion.ValueString() != "batch/v1" || identity.Kind.ValueString() != "Job" {
			resp.Diagnostics.AddError("Invalid Job identity", "Expected api_version batch/v1 and kind Job.")
			return
		}
		namespace := identity.Namespace.ValueString()
		if namespace == "" {
			namespace = "default"
		}
		id = namespace + "/" + identity.Name.ValueString()
	}
	namespace, name, err := jobIDParts(id)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Job import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, jobIdentity(namespace, name))...)
	}
}

func jobIDParts(id string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected namespace/name, got %q", id)
	}
	return parts[0], parts[1], nil
}

// UpgradeState accepts the SDKv2 schema versions 0 and 1.
func (r *JobV1) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	upgrader := func(version int64) resource.StateUpgrader {
		return resource.StateUpgrader{
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				state, _, _, diags := r.jobLegacyState(ctx, req.RawState, version, "Unable to upgrade Job state")
				resp.Diagnostics.Append(diags...)
				if !resp.Diagnostics.HasError() {
					resp.State = state
				}
			},
		}
	}
	return map[int64]resource.StateUpgrader{0: upgrader(0), 1: upgrader(1)}
}

func (r *JobV1) MoveState(context.Context) []resource.StateMover {
	return []resource.StateMover{{
		StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
			if req.SourceTypeName != "kubernetes_job" ||
				(req.SourceSchemaVersion != 0 && req.SourceSchemaVersion != 1) ||
				!strings.HasSuffix(req.SourceProviderAddress, "/hashicorp/kubernetes") {
				return
			}
			state, namespace, name, diags := r.jobLegacyState(ctx, req.SourceRawState, req.SourceSchemaVersion, "Unable to move kubernetes_job state")
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.TargetState = state
			if resp.TargetIdentity != nil {
				resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, jobIdentity(namespace, name))...)
			}
		},
	}}
}

// jobLegacyState decodes state written by the SDKv2 kubernetes_job or
// kubernetes_job_v1 at the given schema version.
func (r *JobV1) jobLegacyState(ctx context.Context, raw *tfprotov6.RawState, version int64, summary string) (tfsdk.State, string, string, diag.Diagnostics) {
	var diags diag.Diagnostics
	var schemaResponse resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	diags.Append(schemaResponse.Diagnostics...)
	state := tfsdk.State{Schema: schemaResponse.Schema}
	var namespace, name string
	value, err := common.DecodeLegacyState(ctx, raw, schemaResponse.Schema, func(values map[string]any) error {
		var err error
		if namespace, name, err = common.LegacyStateName(values); err != nil {
			return err
		}
		spec, ok := jobLegacyObject(values["spec"])
		if !ok {
			return fmt.Errorf("expected exactly one spec element")
		}
		if version == 0 {
			return upgradeJobResourcesV0(spec)
		}
		return nil
	})
	if err != nil {
		diags.AddError(summary, err.Error())
		return state, "", "", diags
	}
	state.Raw = value
	return state, namespace, name, diags
}

func jobLegacyObject(value interface{}) (map[string]interface{}, bool) {
	list, ok := value.([]interface{})
	if !ok || len(list) != 1 {
		return nil, false
	}
	object, ok := list[0].(map[string]interface{})
	return object, ok
}

// Schema v0 stored resource requests/limits as singleton blocks. The Framework
// does not chain state upgraders, so this converts straight to maps.
func upgradeJobResourcesV0(spec map[string]interface{}) error {
	template, ok := jobLegacyObject(spec["template"])
	if !ok {
		return fmt.Errorf("expected exactly one pod template in Job v0 state")
	}
	pod, err := jobOptionalLegacyObject(template["spec"], "template.spec")
	if err != nil {
		return err
	}
	if pod == nil {
		return nil
	}
	for _, name := range []string{"container", "init_container"} {
		if pod[name] == nil {
			continue
		}
		containers, ok := pod[name].([]interface{})
		if !ok {
			return fmt.Errorf("expected %s to be a list in Job v0 state", name)
		}
		for _, value := range containers {
			container, ok := value.(map[string]interface{})
			if !ok || container == nil {
				return fmt.Errorf("invalid %s in Job v0 state", name)
			}
			resources, err := jobOptionalLegacyObject(container["resources"], name+".resources")
			if err != nil {
				return err
			}
			if resources == nil {
				continue
			}
			for _, field := range []string{"limits", "requests"} {
				list, ok := resources[field].([]interface{})
				if !ok && resources[field] != nil {
					return fmt.Errorf("expected v0 %s to be a list", field)
				}
				if len(list) == 0 {
					resources[field] = map[string]interface{}{}
				} else if len(list) == 1 {
					object, ok := list[0].(map[string]interface{})
					if !ok {
						return fmt.Errorf("invalid v0 %s object", field)
					}
					resources[field] = object
				} else {
					return fmt.Errorf("expected at most one v0 %s element", field)
				}
			}
		}
	}
	return nil
}

func jobOptionalLegacyObject(value interface{}, field string) (map[string]interface{}, error) {
	if value == nil {
		return nil, nil
	}
	list, ok := value.([]interface{})
	if !ok || len(list) > 1 {
		return nil, fmt.Errorf("expected %s to be a list with at most one element in Job v0 state", field)
	}
	if len(list) == 0 {
		return nil, nil
	}
	object, ok := list[0].(map[string]interface{})
	if !ok || object == nil {
		return nil, fmt.Errorf("invalid %s object in Job v0 state", field)
	}
	return object, nil
}
