// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// priorStateV0 represents the SDK v2 state shape (schema version 0).
// In SDK v2, metadata was stored as a TypeList — a slice with exactly one element.
// Because the framework v1 schema also uses ListNestedAttribute, the shapes are
// structurally identical; the upgrader only needs to re-emit the state so Terraform
// records the bumped schema_version.
type priorStateV0 struct {
	ID               types.String      `tfsdk:"id"`
	Metadata         []priorMetadataV0 `tfsdk:"metadata"`
	Value            types.Int64       `tfsdk:"value"`
	Description      types.String      `tfsdk:"description"`
	GlobalDefault    types.Bool        `tfsdk:"global_default"`
	PreemptionPolicy types.String      `tfsdk:"preemption_policy"`
}

// priorMetadataV0 is the single element inside the SDK v2 metadata TypeList.
type priorMetadataV0 struct {
	Name            types.String            `tfsdk:"name"`
	GenerateName    types.String            `tfsdk:"generate_name"`
	Annotations     map[string]types.String `tfsdk:"annotations"`
	Labels          map[string]types.String `tfsdk:"labels"`
	ResourceVersion types.String            `tfsdk:"resource_version"`
	UID             types.String            `tfsdk:"uid"`
	Generation      types.Int64             `tfsdk:"generation"`
}

// upgradeStateHandlers returns the map of state upgraders for PriorityClassV1.
// Handles v0 → v1: both versions use ListNestedAttribute for metadata, so the
// upgrader is structural-only — it bumps schema_version without changing any values.
func upgradeStateHandlers() map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			// PriorSchema tells the framework how to decode the v0 state into priorStateV0.
			// It must exactly mirror the SDK v2 schema shape — ListNestedAttribute for metadata.
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"value": schema.Int64Attribute{
						Required: true,
					},
					"description": schema.StringAttribute{
						Optional: true,
						Computed: true,
					},
					"global_default": schema.BoolAttribute{
						Optional: true,
						Computed: true,
					},
					"preemption_policy": schema.StringAttribute{
						Optional: true,
						Computed: true,
					},
					"metadata": schema.ListNestedAttribute{
						Required: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Optional: true,
									Computed: true,
								},
								"generate_name": schema.StringAttribute{
									Optional: true,
									Computed: true,
								},
								"annotations": schema.MapAttribute{
									Optional:    true,
									ElementType: types.StringType,
								},
								"labels": schema.MapAttribute{
									Optional:    true,
									ElementType: types.StringType,
								},
								"resource_version": schema.StringAttribute{
									Computed: true,
								},
								"uid": schema.StringAttribute{
									Computed: true,
								},
								"generation": schema.Int64Attribute{
									Computed: true,
								},
							},
						},
					},
				},
			},
			StateUpgrader: upgradeStateV0Handler,
		},
	}
}

// upgradeStateV0Handler converts SDK v2 state (v0) to framework state (v1).
// Both versions store metadata as a list, so this is a structural re-emit that
// bumps the schema_version and normalises generate_name / empty maps.
func upgradeStateV0Handler(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var prior priorStateV0
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(prior.Metadata) != 1 {
		resp.Diagnostics.AddError(
			"state upgrade failed",
			fmt.Sprintf("expected exactly 1 metadata element in prior state, got %d", len(prior.Metadata)),
		)
		return
	}
	if prior.ID.ValueString() == "" {
		resp.Diagnostics.AddError("state upgrade failed", "empty 'id' in prior state")
		return
	}

	m := prior.Metadata[0]

	meta := MetadataModel{
		Name:            m.Name,
		Generation:      m.Generation,
		ResourceVersion: m.ResourceVersion,
		UID:             m.UID,
	}

	// generate_name: empty string in SDK v2 state → null in framework to avoid plan drift
	if m.GenerateName.IsNull() || m.GenerateName.ValueString() == "" {
		meta.GenerateName = types.StringNull()
	} else {
		meta.GenerateName = m.GenerateName
	}

	// Empty/null maps → nil to avoid perpetual plan diff
	if len(m.Annotations) > 0 {
		meta.Annotations = m.Annotations
	}
	if len(m.Labels) > 0 {
		meta.Labels = m.Labels
	}

	upgraded := PriorityClassModel{
		ID:               prior.ID,
		Metadata:         []MetadataModel{meta},
		Value:            prior.Value,
		Description:      prior.Description,
		GlobalDefault:    prior.GlobalDefault,
		PreemptionPolicy: prior.PreemptionPolicy,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
}
