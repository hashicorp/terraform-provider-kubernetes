// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// moveStateHandlers returns the StateMover list for StorageClassV1.
// Handles: kubernetes_storage_class (deprecated alias without the _v1 suffix).
func moveStateHandlers() []resource.StateMover {
	return []resource.StateMover{
		{
			// No SourceSchema: raw JSON is parsed manually so we stay decoupled
			// from the SDKv2 schema definition.
			StateMover: moveStateFromKubernetesStorageClassHandler,
		},
	}
}

// ── SDKv2 raw JSON shapes ─────────────────────────────────────────────────────

// sdkv2SCMetadataElement is the JSON shape of one element in the SDKv2
// TypeList metadata for kubernetes_storage_class (schema version 0).
type sdkv2SCMetadataElement struct {
	Name            string            `json:"name"`
	GenerateName    string            `json:"generate_name"`
	Annotations     map[string]string `json:"annotations"`
	Labels          map[string]string `json:"labels"`
	ResourceVersion string            `json:"resource_version"`
	UID             string            `json:"uid"`
	Generation      int64             `json:"generation"`
}

// sdkv2SCMatchLabelExpression is the JSON shape of one match_label_expressions
// entry. SDKv2 serialises TypeSet as a plain JSON array — order is not preserved.
type sdkv2SCMatchLabelExpression struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

// sdkv2SCAllowedTopology is the JSON shape of one allowed_topologies block.
type sdkv2SCAllowedTopology struct {
	MatchLabelExpressions []sdkv2SCMatchLabelExpression `json:"match_label_expressions"`
}

// sdkv2StorageClassStateV0 is the raw JSON shape written by the SDKv2 provider
// for kubernetes_storage_class (the deprecated alias).
type sdkv2StorageClassStateV0 struct {
	ID                   string                   `json:"id"`
	Metadata             []sdkv2SCMetadataElement `json:"metadata"`
	StorageProvisioner   string                   `json:"storage_provisioner"`
	Parameters           map[string]string        `json:"parameters"`
	ReclaimPolicy        string                   `json:"reclaim_policy"`
	VolumeBindingMode    string                   `json:"volume_binding_mode"`
	AllowVolumeExpansion bool                     `json:"allow_volume_expansion"`
	// SDKv2 serialises TypeSet as a flat JSON array.
	MountOptions      []string                 `json:"mount_options"`
	AllowedTopologies []sdkv2SCAllowedTopology `json:"allowed_topologies"`
}

// ── MoveState handler ─────────────────────────────────────────────────────────

// moveStateFromKubernetesStorageClassHandler converts kubernetes_storage_class
// state (SDKv2, deprecated alias) into kubernetes_storage_class_v1 state
// (Plugin Framework). Users add a moved block:
//
//	moved {
//	  from = kubernetes_storage_class.example
//	  to   = kubernetes_storage_class_v1.example
//	}
func moveStateFromKubernetesStorageClassHandler(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
	if req.SourceProviderAddress != "" && !strings.HasSuffix(req.SourceProviderAddress, "hashicorp/kubernetes") {
		return
	}
	if req.SourceTypeName != "kubernetes_storage_class" {
		return
	}
	if req.SourceSchemaVersion != 0 {
		return
	}

	var raw sdkv2StorageClassStateV0
	if err := json.Unmarshal(req.SourceRawState.JSON, &raw); err != nil {
		resp.Diagnostics.AddError(
			"state move failed",
			fmt.Sprintf("failed to unmarshal kubernetes_storage_class state: %s", err),
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

	// ── metadata ─────────────────────────────────────────────────────────────
	m := raw.Metadata[0]
	meta := MetadataModel{
		Name:            types.StringValue(m.Name),
		Generation:      types.Int64Value(m.Generation),
		ResourceVersion: types.StringValue(m.ResourceVersion),
		UID:             types.StringValue(m.UID),
	}
	// empty string → null to avoid perpetual plan diff
	if m.GenerateName != "" {
		meta.GenerateName = types.StringValue(m.GenerateName)
	} else {
		meta.GenerateName = types.StringNull()
	}
	// empty maps → nil to avoid perpetual plan diff
	if len(m.Annotations) > 0 {
		meta.Annotations = flattenStringMap(m.Annotations)
	}
	if len(m.Labels) > 0 {
		meta.Labels = flattenStringMap(m.Labels)
	}

	// ── parameters ───────────────────────────────────────────────────────────
	var parameters map[string]types.String
	if len(raw.Parameters) > 0 {
		parameters = flattenStringMap(raw.Parameters)
	}

	// ── reclaim_policy default ────────────────────────────────────────────────
	reclaimPolicy := raw.ReclaimPolicy
	if reclaimPolicy == "" {
		reclaimPolicy = "Delete"
	}

	// ── volume_binding_mode default ───────────────────────────────────────────
	volumeBindingMode := raw.VolumeBindingMode
	if volumeBindingMode == "" {
		volumeBindingMode = "Immediate"
	}

	// ── mount_options (SDKv2 TypeSet → Framework types.Set) ──────────────────
	mountOptions := stringsToSet(raw.MountOptions)

	// ── allowed_topologies ────────────────────────────────────────────────────
	allowedTopologies := moveAllowedTopologies(raw.AllowedTopologies)

	moved := StorageClassModel{
		ID:                   types.StringValue(raw.ID),
		Metadata:             []MetadataModel{meta},
		StorageProvisioner:   types.StringValue(raw.StorageProvisioner),
		Parameters:           parameters,
		ReclaimPolicy:        types.StringValue(reclaimPolicy),
		VolumeBindingMode:    types.StringValue(volumeBindingMode),
		AllowVolumeExpansion: types.BoolValue(raw.AllowVolumeExpansion),
		MountOptions:         mountOptions,
		AllowedTopologies:    allowedTopologies,
	}

	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &moved)...)
}

// ── Move helpers ──────────────────────────────────────────────────────────────

// stringsToSet converts []string → types.Set (ElementType: StringType).
// An empty or nil slice becomes a null set so that the moved state matches
// a Framework config that omits mount_options entirely — preventing the
// "was null, but now empty set" inconsistency error after the move.
func stringsToSet(ss []string) types.Set {
	if len(ss) == 0 {
		return types.SetNull(types.StringType)
	}
	elems := make([]attr.Value, len(ss))
	for i, s := range ss {
		elems[i] = types.StringValue(s)
	}
	result, _ := types.SetValue(types.StringType, elems)
	return result
}

// moveAllowedTopologies converts the SDKv2 JSON topology shape into
// []AllowedTopologyModel for the Framework state.
func moveAllowedTopologies(in []sdkv2SCAllowedTopology) []AllowedTopologyModel {
	if len(in) == 0 {
		return nil
	}
	result := make([]AllowedTopologyModel, 0, len(in))
	for _, t := range in {
		exprs := make([]MatchLabelExpressionModel, 0, len(t.MatchLabelExpressions))
		for _, e := range t.MatchLabelExpressions {
			exprs = append(exprs, MatchLabelExpressionModel{
				Key:    types.StringValue(e.Key),
				Values: stringsToSet(e.Values),
			})
		}
		result = append(result, AllowedTopologyModel{
			MatchLabelExpressions: exprs,
		})
	}
	return result
}
