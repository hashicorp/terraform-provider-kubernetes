// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// MoveState lets a moved block bring an object here from kubernetes_manifest.
// The move does not contact the cluster or change the object.
func (r *ManifestV2) MoveState(context.Context) []resource.StateMover {
	return []resource.StateMover{{StateMover: moveFromManifest}}
}

type timeoutsModel struct {
	Create types.String `tfsdk:"create"`
	Update types.String `tfsdk:"update"`
	Delete types.String `tfsdk:"delete"`
}

// moved is the target state of a move. Nil blocks stay null.
type moved struct {
	manifest, object attr.Value
	computedFields   []string
	fieldManager     *fieldManagerModel
	wait             *waitModel
	timeouts         *timeoutsModel
	namespace        string
}

// moveFromManifest keeps every argument, so a configuration that only
// changes the resource type plans no change.
func moveFromManifest(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
	if req.SourceTypeName != "kubernetes_manifest" || !strings.HasSuffix(req.SourceProviderAddress, "/hashicorp/kubernetes") {
		return
	}
	const summary = "Unable to move kubernetes_manifest state"
	if req.SourceRawState == nil || len(req.SourceRawState.JSON) == 0 {
		resp.Diagnostics.AddError(summary, "The source state has no JSON data.")
		return
	}
	dynamic := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"manifest": tftypes.DynamicPseudoType, "object": tftypes.DynamicPseudoType,
	}}
	raw, err := req.SourceRawState.UnmarshalWithOpts(dynamic, tfprotov6.UnmarshalOpts{
		ValueFromJSONOpts: tftypes.ValueFromJSONOpts{IgnoreUndefinedAttributes: true},
	})
	var values map[string]tftypes.Value
	if err == nil {
		err = raw.As(&values)
	}
	var target moved
	if err == nil {
		target.manifest, err = types.DynamicType.ValueFromTerraform(ctx, values["manifest"])
	}
	if err == nil {
		target.object, err = types.DynamicType.ValueFromTerraform(ctx, values["object"])
	}
	var source struct {
		ComputedFields []string `json:"computed_fields"`
		FieldManager   []struct {
			Name           *string `json:"name"`
			ForceConflicts *bool   `json:"force_conflicts"`
		} `json:"field_manager"`
		Wait []struct {
			Rollout   *bool             `json:"rollout"`
			Fields    map[string]string `json:"fields"`
			Condition []struct {
				Type   *string `json:"type"`
				Status *string `json:"status"`
			} `json:"condition"`
		} `json:"wait"`
		WaitFor *struct {
			Fields map[string]string `json:"fields"`
		} `json:"wait_for"`
		Timeouts []struct {
			Create *string `json:"create"`
			Update *string `json:"update"`
			Delete *string `json:"delete"`
		} `json:"timeouts"`
	}
	if err == nil {
		err = json.Unmarshal(req.SourceRawState.JSON, &source)
	}
	if err != nil {
		resp.Diagnostics.AddError(summary, "The source state could not be decoded: "+err.Error())
		return
	}
	target.computedFields = source.ComputedFields
	if len(source.FieldManager) == 1 {
		target.fieldManager = &fieldManagerModel{
			Name:           types.StringPointerValue(source.FieldManager[0].Name),
			ForceConflicts: types.BoolPointerValue(source.FieldManager[0].ForceConflicts),
		}
	}
	fields := func(m map[string]string) types.Map {
		if m == nil {
			return types.MapNull(types.StringType)
		}
		value, _ := types.MapValueFrom(ctx, types.StringType, m)
		return value
	}
	switch {
	case len(source.Wait) == 1:
		target.wait = &waitModel{
			Rollout:    types.BoolPointerValue(source.Wait[0].Rollout),
			Fields:     fields(source.Wait[0].Fields),
			Conditions: []conditionModel{},
		}
		for _, condition := range source.Wait[0].Condition {
			target.wait.Conditions = append(target.wait.Conditions, conditionModel{
				Type: types.StringPointerValue(condition.Type), Status: types.StringPointerValue(condition.Status),
			})
		}
	case source.WaitFor != nil:
		// The deprecated wait_for is written as wait { fields } here.
		target.wait = &waitModel{Rollout: types.BoolNull(), Fields: fields(source.WaitFor.Fields), Conditions: []conditionModel{}}
	}
	if len(source.Timeouts) == 1 {
		target.timeouts = &timeoutsModel{
			Create: types.StringPointerValue(source.Timeouts[0].Create),
			Update: types.StringPointerValue(source.Timeouts[0].Update),
			Delete: types.StringPointerValue(source.Timeouts[0].Delete),
		}
	}
	// kubernetes_manifest keeps no UID. Read fills it in.
	namespace, _ := member(target.object, "metadata", "namespace").(types.String)
	target.namespace = namespace.ValueString()
	finishMove(ctx, resp, summary, target)
}

func finishMove(ctx context.Context, resp *resource.MoveStateResponse, summary string, target moved) {
	manifest, _ := target.manifest.(types.Dynamic)
	id, known, err := manifestIdentity(manifest)
	if err != nil || !known {
		resp.Diagnostics.AddError(summary, "The manifest in the source state does not identify an object.")
		return
	}
	if id.Namespace == "" {
		id.Namespace = target.namespace
	}
	state := model{}
	state.setIdentity(id)
	// An unknown namespace stays null, like the UID, for Read to fill in.
	if id.Namespace == "" {
		state.Namespace = types.StringNull()
	}
	computedFields := types.ListNull(types.StringType)
	if target.computedFields != nil {
		computedFields, _ = types.ListValueFrom(ctx, types.StringType, target.computedFields)
	}
	for attribute, value := range map[string]any{
		"manifest": manifest, "object": target.object, "computed_fields": computedFields,
		"field_manager": target.fieldManager, "wait": target.wait, "timeouts": target.timeouts,
		"api_version": state.APIVersion, "kind": state.Kind, "namespace": state.Namespace, "name": state.Name,
		"uid": types.StringNull(), "drift": types.ListValueMust(types.StringType, nil),
	} {
		resp.Diagnostics.Append(resp.TargetState.SetAttribute(ctx, path.Root(attribute), value)...)
	}
	resp.Diagnostics.Append(common.SetIdentity(ctx, resp.TargetIdentity, state.resourceIdentity())...)
	// kubernetes_manifest applied the object with server-side apply too, so its
	// fields move with a later change of the manager name.
	manager := defaultFieldManager
	if target.fieldManager != nil && target.fieldManager.Name.ValueString() != "" {
		manager = target.fieldManager.Name.ValueString()
	}
	if resp.TargetPrivate != nil {
		encoded, _ := json.Marshal(manager)
		resp.Diagnostics.Append(resp.TargetPrivate.SetKey(ctx, managerKey, encoded)...)
	}
}
