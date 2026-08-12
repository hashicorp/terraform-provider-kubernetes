// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package nodev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// UpgradeState converts existing terraform.tfstate written by the SDKv2 provider
// (schema version 0) into the Framework schema (version 1).
//
// SDKv2 stored metadata as a TypeList with MaxItems:1 — so state looks like:
//
//	"metadata": [{ "name": "x", "uid": "abc", ... }]
//
// The Framework stores metadata as a SingleNestedAttribute — so state looks like:
//
//	"metadata": { "name": "x", "uid": "abc", ... }
//
// This upgrader unwraps the list into a plain object. Terraform calls this
// automatically when it finds schema version 0 in terraform.tfstate.
func (r *RuntimeClassV1) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		// 0 = SDKv2 schema version (TypeList metadata shape).
		// PriorSchema is required — without it the Framework cannot deserialise
		// the old raw JSON state and passes a nil req.State to the upgrader,
		// causing a nil pointer dereference.
		0: {
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"handler": schema.StringAttribute{
						Required: true,
					},
				},
				Blocks: map[string]schema.Block{
					// SDKv2 stored metadata as TypeList (MaxItems:1).
					// In raw state this serialises as a list of objects: metadata[0].name etc.
					// The Framework PriorSchema represents a TypeList as ListNestedBlock.
					"metadata": schema.ListNestedBlock{
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"annotations": schema.MapAttribute{
									ElementType: types.StringType,
									Optional:    true,
								},
								"generate_name": schema.StringAttribute{
									Optional: true,
								},
								"generation": schema.Int64Attribute{
									Computed: true,
								},
								"labels": schema.MapAttribute{
									ElementType: types.StringType,
									Optional:    true,
								},
								"name": schema.StringAttribute{
									Optional: true,
									Computed: true,
								},
								"resource_version": schema.StringAttribute{
									Computed: true,
								},
								"uid": schema.StringAttribute{
									Computed: true,
								},
							},
						},
					},
				},
			},
			StateUpgrader: upgradeStateV0toV1,
		},
	}
}

// priorStateV0 represents the old SDKv2 state shape.
// metadata is a []metadataV0 (TypeList) — the [0] element is the actual data.
// Fields use Framework types (types.String etc.) to match how PriorSchema
// deserialises them — plain Go types would cause a deserialisation mismatch.
type priorStateV0 struct {
	ID       types.String `tfsdk:"id"`
	Handler  types.String `tfsdk:"handler"`
	Metadata []metadataV0 `tfsdk:"metadata"`
}

type metadataV0 struct {
	Annotations     map[string]types.String `tfsdk:"annotations"`
	GenerateName    types.String            `tfsdk:"generate_name"`
	Generation      types.Int64             `tfsdk:"generation"`
	Labels          map[string]types.String `tfsdk:"labels"`
	Name            types.String            `tfsdk:"name"`
	ResourceVersion types.String            `tfsdk:"resource_version"`
	UID             types.String            `tfsdk:"uid"`
}

func upgradeStateV0toV1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	// 1. Read old state (TypeList format).
	//    req.State is populated because PriorSchema was provided above.
	var old priorStateV0
	resp.Diagnostics.Append(req.State.Get(ctx, &old)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(old.Metadata) == 0 {
		resp.Diagnostics.AddError(
			"state upgrade failed",
			"expected metadata list to have at least one element but it was empty",
		)
		return
	}

	// 2. Unwrap metadata[0] into the new flat MetadataModel.
	//    Field types already match MetadataModel — no conversion needed.
	m := old.Metadata[0]

	newMeta := MetadataModel{
		Name:            m.Name,
		UID:             m.UID,
		ResourceVersion: m.ResourceVersion,
		Generation:      m.Generation,
	}

	if !m.GenerateName.IsNull() && !m.GenerateName.IsUnknown() && m.GenerateName.ValueString() != "" {
		newMeta.GenerateName = m.GenerateName
	}
	if len(m.Labels) > 0 {
		newMeta.Labels = m.Labels
	}
	if len(m.Annotations) > 0 {
		newMeta.Annotations = m.Annotations
	}

	// 3. Build new state — timeouts are null (not stored in cluster).
	timeoutsObj := types.ObjectNull(map[string]attr.Type{
		"create": types.StringType,
		"delete": types.StringType,
		"read":   types.StringType,
		"update": types.StringType,
	})

	newState := RuntimeClassV1Model{
		ID:       old.ID,
		Handler:  old.Handler,
		Metadata: newMeta,
		Timeouts: timeouts.Value{Object: timeoutsObj},
	}

	// 4. Write the new state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}
