// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1

import (
	"context"
	"encoding/json"
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

// ---------------------------------------------------------------------------
// MoveState support — moved block from kubernetes_priority_class
// ---------------------------------------------------------------------------

// moveStateHandlers returns the StateMover list for PriorityClassV1.
// Handles: kubernetes_priority_class (deprecated alias without version suffix).
func moveStateHandlers() []resource.StateMover {
	return []resource.StateMover{
		{
			// No SourceSchema: raw JSON is parsed manually so we stay decoupled
			// from the SDK v2 schema definition.
			StateMover: moveStateFromKubernetesPriorityClassHandler,
		},
	}
}

// sdkv2PCMetadataElement is the JSON shape of one element in the SDKv2
// TypeList metadata for kubernetes_priority_class (schema version 0).
type sdkv2PCMetadataElement struct {
	Name            string            `json:"name"`
	GenerateName    string            `json:"generate_name"`
	Annotations     map[string]string `json:"annotations"`
	Labels          map[string]string `json:"labels"`
	ResourceVersion string            `json:"resource_version"`
	UID             string            `json:"uid"`
	Generation      int64             `json:"generation"`
}

// sdkv2PriorityClassStateV0 is the raw JSON shape of an SDKv2
// kubernetes_priority_class state (schema version 0).
type sdkv2PriorityClassStateV0 struct {
	ID               string                   `json:"id"`
	Metadata         []sdkv2PCMetadataElement `json:"metadata"`
	Value            int64                    `json:"value"`
	Description      string                   `json:"description"`
	GlobalDefault    bool                     `json:"global_default"`
	PreemptionPolicy string                   `json:"preemption_policy"`
}

// moveStateFromKubernetesPriorityClassHandler is the MoveState handler that
// converts kubernetes_priority_class state into kubernetes_priority_class_v1 state.
// The user adds a moved block in their configuration:
//
//	moved {
//	  from = kubernetes_priority_class.example
//	  to   = kubernetes_priority_class_v1.example
//	}
func moveStateFromKubernetesPriorityClassHandler(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
	if req.SourceTypeName != "kubernetes_priority_class" {
		return
	}

	var raw sdkv2PriorityClassStateV0
	if err := json.Unmarshal(req.SourceRawState.JSON, &raw); err != nil {
		resp.Diagnostics.AddError(
			"state move failed",
			fmt.Sprintf("failed to unmarshal kubernetes_priority_class state: %s", err),
		)
		return
	}

	if len(raw.Metadata) != 1 {
		resp.Diagnostics.AddError(
			"state move failed",
			fmt.Sprintf("expected exactly 1 metadata element in source state, got %d", len(raw.Metadata)),
		)
		return
	}
	if raw.ID == "" {
		resp.Diagnostics.AddError("state move failed", "empty 'id' in source state")
		return
	}

	m := raw.Metadata[0]

	meta := MetadataModel{
		Name:            types.StringValue(m.Name),
		Generation:      types.Int64Value(m.Generation),
		ResourceVersion: types.StringValue(m.ResourceVersion),
		UID:             types.StringValue(m.UID),
	}

	// generate_name: empty string → null to avoid perpetual plan diff
	if m.GenerateName != "" {
		meta.GenerateName = types.StringValue(m.GenerateName)
	} else {
		meta.GenerateName = types.StringNull()
	}

	// Empty maps → nil to avoid perpetual plan diff
	if len(m.Annotations) > 0 {
		meta.Annotations = flattenStringMap(m.Annotations)
	}
	if len(m.Labels) > 0 {
		meta.Labels = flattenStringMap(m.Labels)
	}

	// preemption_policy: default to PreemptLowerPriority if empty (SDKv2 stores the default)
	preemptionPolicy := raw.PreemptionPolicy
	if preemptionPolicy == "" {
		preemptionPolicy = "PreemptLowerPriority"
	}

	moved := PriorityClassModel{
		ID:               types.StringValue(raw.ID),
		Metadata:         []MetadataModel{meta},
		Value:            types.Int64Value(raw.Value),
		Description:      types.StringValue(raw.Description),
		GlobalDefault:    types.BoolValue(raw.GlobalDefault),
		PreemptionPolicy: types.StringValue(preemptionPolicy),
	}

	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &moved)...)
}
