// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
