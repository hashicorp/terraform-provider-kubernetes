// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

const daemonSetSDKv2ProviderAddressSuffix = "/hashicorp/kubernetes"

func (d *DaemonSetV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != "" {
		namespace, name, err := kubernetes.IdParts(req.ID)
		if err != nil || namespace == "" || name == "" {
			resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected a non-empty namespace/name ID, got %q.", req.ID))
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("%s/%s", namespace, name))...)
		return
	}

	var identity common.NamespacedResourceIdentity
	resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if identity.Namespace.IsNull() || identity.Namespace.IsUnknown() || identity.Namespace.ValueString() == "" {
		resp.Diagnostics.AddError("Invalid import identity", "Import identity must include a namespace.")
		return
	}
	if identity.Name.IsNull() || identity.Name.IsUnknown() || identity.Name.ValueString() == "" {
		resp.Diagnostics.AddError("Invalid import identity", "Import identity must include a name.")
		return
	}
	if identity.APIVersion.IsNull() || identity.APIVersion.IsUnknown() || identity.APIVersion.ValueString() != daemonSetAPIVersion {
		resp.Diagnostics.AddError("Invalid import identity", fmt.Sprintf("Import identity api_version must be %q.", daemonSetAPIVersion))
		return
	}
	if identity.Kind.IsNull() || identity.Kind.IsUnknown() || identity.Kind.ValueString() != daemonSetKind {
		resp.Diagnostics.AddError("Invalid import identity", fmt.Sprintf("Import identity kind must be %q.", daemonSetKind))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(
		ctx,
		path.Root("id"),
		fmt.Sprintf("%s/%s", identity.Namespace.ValueString(), identity.Name.ValueString()),
	)...)
}

func (d *DaemonSetV1) MoveState(ctx context.Context) []resource.StateMover {
	resourceSchema, schemaDiags := d.schemasForStateMoves(ctx)
	if schemaDiags.HasError() {
		return []resource.StateMover{{
			StateMover: func(_ context.Context, _ resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				resp.Diagnostics.Append(schemaDiags...)
			},
		}}
	}

	return []resource.StateMover{
		{
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				if req.SourceTypeName != "kubernetes_daemonset" ||
					(req.SourceSchemaVersion != 0 && req.SourceSchemaVersion != 1) ||
					!strings.HasSuffix(req.SourceProviderAddress, daemonSetSDKv2ProviderAddressSuffix) {
					tflog.Debug(ctx, "MoveState: unsupported source, skipping", map[string]any{
						"source_type_name":        req.SourceTypeName,
						"source_schema_version":   req.SourceSchemaVersion,
						"source_provider_address": req.SourceProviderAddress,
					})
					return
				}

				decoded, ok := decodeDaemonSetRawState(req.SourceRawState, &resp.Diagnostics)
				if !ok {
					return
				}
				if req.SourceSchemaVersion == 0 {
					if err := validateDaemonSetV0State(decoded); err != nil {
						resp.Diagnostics.AddError("Unable to move daemon set state", err.Error())
						return
					}
					decoded = kubernetes.UpgradeTemplatePodSpecWithResourcesFieldV0ForFramework(ctx, decoded)
				}

				value, ok := encodeDaemonSetStateValue(ctx, decoded, resourceSchema, &resp.Diagnostics)
				if !ok {
					return
				}
				target := tfsdk.State{Schema: resourceSchema, Raw: value}
				resp.TargetState = target
				if resp.Diagnostics.HasError() || resp.TargetIdentity == nil {
					return
				}

				var model DaemonSetV1Model
				resp.Diagnostics.Append(target.Get(ctx, &model)...)
				if resp.Diagnostics.HasError() {
					return
				}
				namespace, name, ok := daemonSetIdentityFields(model, &resp.Diagnostics)
				if !ok {
					return
				}
				resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, daemonSetIdentity(namespace, name))...)
			},
		},
	}
}

func (d *DaemonSetV1) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	resourceSchema, schemaDiags := d.schemasForStateMoves(ctx)
	upgrader := resource.StateUpgrader{
		StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
			if schemaDiags.HasError() {
				resp.Diagnostics.Append(schemaDiags...)
				return
			}
			decoded, ok := decodeDaemonSetRawState(req.RawState, &resp.Diagnostics)
			if !ok {
				return
			}

			if err := validateDaemonSetV0State(decoded); err != nil {
				resp.Diagnostics.AddError("Unable to upgrade daemon set state", err.Error())
				return
			}
			upgraded := kubernetes.UpgradeTemplatePodSpecWithResourcesFieldV0ForFramework(ctx, decoded)
			value, ok := encodeDaemonSetStateValue(ctx, upgraded, resourceSchema, &resp.Diagnostics)
			if !ok {
				return
			}

			resp.State = tfsdk.State{Schema: resourceSchema, Raw: value}
		},
	}
	return map[int64]resource.StateUpgrader{
		0: upgrader,
	}
}

func validateDaemonSetV0State(raw map[string]interface{}) error {
	specs, ok := raw["spec"].([]interface{})
	if !ok || len(specs) == 0 {
		return nil
	}
	spec, ok := specs[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("stored spec element has unexpected type %T", specs[0])
	}
	templates, ok := spec["template"].([]interface{})
	if !ok || len(templates) == 0 {
		return nil
	}
	template, ok := templates[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("stored template element has unexpected type %T", templates[0])
	}
	podSpecs, ok := template["spec"].([]interface{})
	if !ok || len(podSpecs) == 0 {
		return nil
	}
	podSpec, ok := podSpecs[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("stored pod spec element has unexpected type %T", podSpecs[0])
	}
	for _, field := range []string{"container", "init_container"} {
		containers, ok := podSpec[field].([]interface{})
		if !ok {
			continue
		}
		for i, value := range containers {
			container, ok := value.(map[string]interface{})
			if !ok {
				return fmt.Errorf("stored %s element %d has unexpected type %T", field, i, value)
			}
			resources, ok := container["resources"].([]interface{})
			if !ok || len(resources) == 0 {
				continue
			}
			resource, ok := resources[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("stored %s element %d resources has unexpected type %T", field, i, resources[0])
			}
			for _, resourceField := range []string{"requests", "limits"} {
				values, ok := resource[resourceField].([]interface{})
				if !ok || len(values) == 0 {
					continue
				}
				if _, ok := values[0].(map[string]interface{}); !ok {
					return fmt.Errorf("stored %s element %d resources.%s has unexpected type %T", field, i, resourceField, values[0])
				}
			}
		}
	}
	return nil
}

func (d *DaemonSetV1) schemasForStateMoves(ctx context.Context) (schema.Schema, diag.Diagnostics) {
	var diags diag.Diagnostics

	var schemaResp resource.SchemaResponse
	d.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	diags.Append(schemaResp.Diagnostics...)
	if diags.HasError() {
		return schema.Schema{}, diags
	}

	return schemaResp.Schema, diags
}

func decodeDaemonSetRawState(raw *tfprotov6.RawState, diags *diag.Diagnostics) (map[string]interface{}, bool) {
	if raw == nil || len(raw.JSON) == 0 {
		diags.AddError("Unable to move daemon set state", "The source state has no JSON data.")
		return nil, false
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(raw.JSON, &decoded); err != nil {
		diags.AddError("Unable to move daemon set state", fmt.Sprintf("Could not decode source state JSON: %s", err))
		return nil, false
	}
	return decoded, true
}

func encodeDaemonSetStateValue(ctx context.Context, raw map[string]interface{}, targetSchema schema.Schema, diags *diag.Diagnostics) (tftypes.Value, bool) {
	stateJSON, err := json.Marshal(raw)
	if err != nil {
		diags.AddError("Unable to encode daemon set state", fmt.Sprintf("Could not encode upgraded state JSON: %s", err))
		return tftypes.Value{}, false
	}
	value, err := (&tfprotov6.RawState{JSON: stateJSON}).Unmarshal(targetSchema.Type().TerraformType(ctx))
	if err != nil {
		diags.AddError("Unable to decode daemon set state", fmt.Sprintf("Could not decode upgraded state: %s", err))
		return tftypes.Value{}, false
	}
	return value, true
}

func daemonSetIdentityFields(model DaemonSetV1Model, diags *diag.Diagnostics) (string, string, bool) {
	if len(model.Metadata) != 1 {
		diags.AddError("Unable to set daemon set identity", fmt.Sprintf("Expected exactly 1 metadata element, got %d.", len(model.Metadata)))
		return "", "", false
	}
	namespace := model.Metadata[0].Namespace
	name := model.Metadata[0].Name
	if namespace.IsNull() || namespace.IsUnknown() || namespace.ValueString() == "" {
		diags.AddError("Unable to set daemon set identity", "Metadata namespace is empty or unknown.")
		return "", "", false
	}
	if name.IsNull() || name.IsUnknown() || name.ValueString() == "" {
		diags.AddError("Unable to set daemon set identity", "Metadata name is empty or unknown.")
		return "", "", false
	}
	return namespace.ValueString(), name.ValueString(), true
}
