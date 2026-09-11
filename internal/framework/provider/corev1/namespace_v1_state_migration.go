// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// priorStateV0 represents the decoded SDKv2 state (schema version 0) when PriorSchema is set.
// Used by upgradeStateV0Handler via req.State.Get().
type priorStateV0 struct {
	ID                           types.String      `tfsdk:"id"`
	WaitForDefaultServiceAccount types.Bool        `tfsdk:"wait_for_default_service_account"`
	Metadata                     []priorMetadataV0 `tfsdk:"metadata"`
	Timeouts                     types.Object      `tfsdk:"timeouts"`
}

type priorMetadataV0 struct {
	Name            types.String            `tfsdk:"name"`
	GenerateName    types.String            `tfsdk:"generate_name"`
	Annotations     map[string]types.String `tfsdk:"annotations"`
	Labels          map[string]types.String `tfsdk:"labels"`
	ResourceVersion types.String            `tfsdk:"resource_version"`
	UID             types.String            `tfsdk:"uid"`
	Generation      types.Int64             `tfsdk:"generation"`
}

// upgradeStateHandlers returns the map of UpgradeState handlers for NamespaceV1.
// Currently handles only v0 → v1 (SDKv2 TypeList metadata → SingleNestedAttribute).
func upgradeStateHandlers() map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id":                               schema.StringAttribute{Computed: true},
					"wait_for_default_service_account": schema.BoolAttribute{Optional: true, Computed: true},
					"metadata": schema.ListNestedAttribute{
						Required: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name":             schema.StringAttribute{Optional: true, Computed: true},
								"generate_name":    schema.StringAttribute{Optional: true, Computed: true},
								"annotations":      schema.MapAttribute{Optional: true, ElementType: types.StringType},
								"labels":           schema.MapAttribute{Optional: true, ElementType: types.StringType},
								"resource_version": schema.StringAttribute{Computed: true},
								"uid":              schema.StringAttribute{Computed: true},
								"generation":       schema.Int64Attribute{Computed: true},
							},
						},
					},
					"timeouts": schema.SingleNestedAttribute{
						Optional: true,
						Computed: true,
						Attributes: map[string]schema.Attribute{
							"create": schema.StringAttribute{Optional: true},
							"read":   schema.StringAttribute{Optional: true},
							"update": schema.StringAttribute{Optional: true},
							"delete": schema.StringAttribute{Optional: true},
						},
					},
				},
			},
			StateUpgrader: upgradeStateV0Handler,
		},
	}
}

// moveStateHandlers returns the list of StateMover handlers for NamespaceV1.
// Handles: kubernetes_namespace (deprecated alias).
func moveStateHandlers() []resource.StateMover {
	return []resource.StateMover{
		{
			StateMover: moveStateFromKubernetesNamespaceHandler,
		},
	}
}

// sdkv2MetadataElement represents a single element of the SDKv2 TypeList metadata.
type sdkv2MetadataElement struct {
	Name            string            `json:"name"`
	GenerateName    string            `json:"generate_name"`
	Annotations     map[string]string `json:"annotations"`
	Labels          map[string]string `json:"labels"`
	ResourceVersion string            `json:"resource_version"`
	UID             string            `json:"uid"`
	Generation      int64             `json:"generation"`
}

// sdkv2NamespaceStateV0 is the raw JSON shape of an SDKv2 namespace state (version 0).
type sdkv2NamespaceStateV0 struct {
	ID                           string                 `json:"id"`
	WaitForDefaultServiceAccount bool                   `json:"wait_for_default_service_account"`
	Metadata                     []sdkv2MetadataElement `json:"metadata"`
}

// parseSDKv2NamespaceStateV0 unmarshals raw JSON state bytes from SDKv2.
// Used by both upgradeStateV0Handler and moveStateFromKubernetesNamespaceHandler.
func parseSDKv2NamespaceStateV0(rawJSON []byte) (*sdkv2NamespaceStateV0, error) {
	var prior sdkv2NamespaceStateV0
	if err := json.Unmarshal(rawJSON, &prior); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SDKv2 namespace state: %w", err)
	}
	return &prior, nil
}

// upgradeStateV0Handler upgrades state from schema version 0 (SDKv2) to version 1 (Framework).
// Since PriorSchema is set, the framework decodes the prior state into req.State automatically.
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

	upgraded := buildFrameworkStateFromPriorV0(prior)
	resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
}

// moveStateFromKubernetesNamespaceHandler handles MoveState from kubernetes_namespace (deprecated type).
// Uses req.SourceRawState.JSON directly since no SourceSchema is declared.
func moveStateFromKubernetesNamespaceHandler(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
	if req.SourceTypeName != "kubernetes_namespace" {
		return
	}

	prior, err := parseSDKv2NamespaceStateV0(req.SourceRawState.JSON)
	if err != nil {
		resp.Diagnostics.AddError("state move failed", err.Error())
		return
	}

	if len(prior.Metadata) != 1 {
		resp.Diagnostics.AddError(
			"state move failed",
			fmt.Sprintf("expected exactly 1 metadata element in source state, got %d", len(prior.Metadata)),
		)
		return
	}
	if prior.ID == "" {
		resp.Diagnostics.AddError("state move failed", "empty 'id' in source state")
		return
	}

	upgraded := buildFrameworkStateFromRawSDKv2(prior)
	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &upgraded)...)
}

// buildFrameworkStateFromPriorV0 converts a decoded priorStateV0 (from req.State.Get) to Framework model.
// Pure in-memory transformation — no API calls.
func buildFrameworkStateFromPriorV0(prior priorStateV0) NamespaceV1Model {
	m := prior.Metadata[0]

	meta := NamespaceMetadataModel{
		Name:            m.Name,
		Generation:      m.Generation,
		ResourceVersion: m.ResourceVersion,
		UID:             m.UID,
	}

	// Optional-only field: empty/null string → null to avoid perpetual plan diff
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

	timeoutsAttrTypes := nullTimeoutsAttrTypes()
	return NamespaceV1Model{
		ID:                           prior.ID,
		Metadata:                     meta,
		WaitForDefaultServiceAccount: prior.WaitForDefaultServiceAccount,
		Timeouts: timeouts.Value{
			Object: types.ObjectNull(timeoutsAttrTypes),
		},
	}
}

// buildFrameworkStateFromRawSDKv2 converts a parsed raw JSON SDKv2 state to a Framework model.
// Used by MoveState (no SourceSchema, so raw JSON is parsed manually).
// Pure in-memory transformation — no API calls.
func buildFrameworkStateFromRawSDKv2(prior *sdkv2NamespaceStateV0) NamespaceV1Model {
	m := prior.Metadata[0]

	meta := NamespaceMetadataModel{
		Name:            types.StringValue(m.Name),
		Generation:      types.Int64Value(m.Generation),
		ResourceVersion: types.StringValue(m.ResourceVersion),
		UID:             types.StringValue(m.UID),
	}

	// Optional-only field: empty string → null to avoid perpetual plan diff
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

	timeoutsAttrTypes := nullTimeoutsAttrTypes()
	return NamespaceV1Model{
		ID:                           types.StringValue(prior.ID),
		Metadata:                     meta,
		WaitForDefaultServiceAccount: types.BoolValue(prior.WaitForDefaultServiceAccount),
		Timeouts: timeouts.Value{
			Object: types.ObjectNull(timeoutsAttrTypes),
		},
	}
}

// nullTimeoutsAttrTypes returns the attr.Type map for the timeouts block null object.
func nullTimeoutsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"create": types.StringType,
		"delete": types.StringType,
	}
}
