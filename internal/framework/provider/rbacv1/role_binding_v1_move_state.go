// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ── MoveState support — moved block from kubernetes_role_binding ──────────────

// moveStateHandlers returns the StateMover list for RoleBindingV1.
// Handles: kubernetes_role_binding (deprecated alias without _v1 suffix).
func moveStateHandlers() []resource.StateMover {
	return []resource.StateMover{
		{
			// No SourceSchema: raw JSON is parsed manually so we stay decoupled
			// from the SDKv2 schema definition.
			StateMover: moveStateFromKubernetesRoleBindingHandler,
		},
	}
}

// ── SDKv2 raw JSON shapes ─────────────────────────────────────────────────────

// sdkv2RBMetadataElement is the JSON shape of one element in the SDKv2
// TypeList metadata for kubernetes_role_binding (schema version 0).
type sdkv2RBMetadataElement struct {
	Name            string            `json:"name"`
	GenerateName    string            `json:"generate_name"`
	Namespace       string            `json:"namespace"`
	Annotations     map[string]string `json:"annotations"`
	Labels          map[string]string `json:"labels"`
	ResourceVersion string            `json:"resource_version"`
	UID             string            `json:"uid"`
	Generation      int64             `json:"generation"`
}

// sdkv2RBRoleRefElement is the JSON shape of one role_ref element.
type sdkv2RBRoleRefElement struct {
	APIGroup string `json:"api_group"`
	Kind     string `json:"kind"`
	Name     string `json:"name"`
}

// sdkv2RBSubjectElement is the JSON shape of one subject element.
type sdkv2RBSubjectElement struct {
	APIGroup  string `json:"api_group"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// sdkv2RoleBindingStateV0 is the raw JSON shape of an SDKv2
// kubernetes_role_binding state (schema version 0).
type sdkv2RoleBindingStateV0 struct {
	ID       string                   `json:"id"`
	Metadata []sdkv2RBMetadataElement `json:"metadata"`
	RoleRef  []sdkv2RBRoleRefElement  `json:"role_ref"`
	Subjects []sdkv2RBSubjectElement  `json:"subject"`
}

// ── MoveState handler ─────────────────────────────────────────────────────────

// moveStateFromKubernetesRoleBindingHandler converts kubernetes_role_binding
// SDKv2 state into kubernetes_role_binding_v1 Plugin Framework state.
//
// Users add a moved block in their configuration:
//
//	moved {
//	  from = kubernetes_role_binding.example
//	  to   = kubernetes_role_binding_v1.example
//	}
func moveStateFromKubernetesRoleBindingHandler(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
	if req.SourceTypeName != "kubernetes_role_binding" {
		return
	}

	var raw sdkv2RoleBindingStateV0
	if err := json.Unmarshal(req.SourceRawState.JSON, &raw); err != nil {
		resp.Diagnostics.AddError(
			"state move failed",
			fmt.Sprintf("failed to unmarshal kubernetes_role_binding state: %s", err),
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
	if len(raw.RoleRef) != 1 {
		resp.Diagnostics.AddError(
			"state move failed",
			fmt.Sprintf("expected exactly 1 role_ref element in source state, got %d", len(raw.RoleRef)),
		)
		return
	}
	if raw.ID == "" {
		resp.Diagnostics.AddError("state move failed", "empty 'id' in source state")
		return
	}

	m := raw.Metadata[0]

	meta := NamespacedMetadataModel{
		Name:            types.StringValue(m.Name),
		Namespace:       types.StringValue(m.Namespace),
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

	// Empty maps → nil to avoid perpetual plan diff against configs that
	// omit annotations/labels entirely.
	if len(m.Annotations) > 0 {
		meta.Annotations = flattenStringMap(m.Annotations)
	}
	if len(m.Labels) > 0 {
		meta.Labels = flattenStringMap(m.Labels)
	}

	// role_ref
	rr := raw.RoleRef[0]
	roleRef := RoleRefModel{
		APIGroup: types.StringValue(rr.APIGroup),
		Kind:     types.StringValue(rr.Kind),
		Name:     types.StringValue(rr.Name),
	}

	// subjects
	subjects := make([]SubjectModel, 0, len(raw.Subjects))
	for _, s := range raw.Subjects {
		subjects = append(subjects, SubjectModel{
			APIGroup:  types.StringValue(s.APIGroup),
			Kind:      types.StringValue(s.Kind),
			Name:      types.StringValue(s.Name),
			Namespace: types.StringValue(s.Namespace),
		})
	}

	moved := RoleBindingModel{
		ID:       types.StringValue(raw.ID),
		Metadata: []NamespacedMetadataModel{meta},
		RoleRef:  []RoleRefModel{roleRef},
		Subject:  subjects,
	}

	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &moved)...)
}
