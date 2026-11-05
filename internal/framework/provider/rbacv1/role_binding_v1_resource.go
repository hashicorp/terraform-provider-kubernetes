// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// Compile-time interface assertions — ensure RoleBindingV1 implements all
// required Plugin Framework interfaces.
var (
	_ resource.Resource                    = (*RoleBindingV1)(nil)
	_ resource.ResourceWithConfigure       = (*RoleBindingV1)(nil)
	_ resource.ResourceWithImportState     = (*RoleBindingV1)(nil)
	_ resource.ResourceWithIdentity        = (*RoleBindingV1)(nil)
	_ resource.ResourceWithMoveState       = (*RoleBindingV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*RoleBindingV1)(nil)
)

// RoleBindingV1 is the Plugin Framework resource for kubernetes_role_binding_v1.
type RoleBindingV1 struct {
	// SDKv2Meta is a function that returns the provider metadata (KubeClientsets).
	// It bridges the mux setup where the framework provider delegates client
	// creation to the SDKv2 provider configuration.
	SDKv2Meta func() any
}

// NewRoleBindingV1 returns a new instance of RoleBindingV1 as a resource.Resource.
// Registered in the framework provider's Resources() method.
func NewRoleBindingV1() resource.Resource {
	return &RoleBindingV1{}
}

// Metadata sets the Terraform resource type name to "kubernetes_role_binding_v1".
func (r *RoleBindingV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_binding_v1"
}

// Configure stores the SDKv2Meta function passed through from the mux provider.
func (r *RoleBindingV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	sdkv2Meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected func() any, got %T. This is a bug in the provider.", req.ProviderData))
		return
	}
	r.SDKv2Meta = sdkv2Meta
}

// MoveState returns the StateMover handlers that enable `moved {}` block support
// for migrating from the deprecated kubernetes_role_binding (SDKv2) resource.
func (r *RoleBindingV1) MoveState(_ context.Context) []resource.StateMover {
	return []resource.StateMover{{StateMover: moveStateFromKubernetesRoleBindingHandler}}
}

func moveStateFromKubernetesRoleBindingHandler(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
	if !strings.HasSuffix(req.SourceProviderAddress, providerAddressSuffix) ||
		req.SourceTypeName != "kubernetes_role_binding" || req.SourceSchemaVersion != 0 {
		tflog.Debug(ctx, "MoveState: not a kubernetes_role_binding v0 source, skipping", map[string]any{
			"source_type_name":        req.SourceTypeName,
			"source_schema_version":   req.SourceSchemaVersion,
			"source_provider_address": req.SourceProviderAddress,
		})
		return
	}
	if req.SourceRawState == nil || len(req.SourceRawState.JSON) == 0 {
		resp.Diagnostics.AddError("state move failed", "The source state has no JSON data. Flatmap state is not supported.")
		return
	}
	var raw sdkv2RoleBindingStateV0
	if err := json.Unmarshal(req.SourceRawState.JSON, &raw); err != nil {
		resp.Diagnostics.AddError("state move failed", fmt.Sprintf("failed to unmarshal kubernetes_role_binding state: %s", err))
		return
	}
	if len(raw.Metadata) != 1 {
		resp.Diagnostics.AddError("state move failed", fmt.Sprintf("expected exactly 1 metadata element in source state, got %d", len(raw.Metadata)))
		return
	}
	if len(raw.RoleRef) != 1 {
		resp.Diagnostics.AddError("state move failed", fmt.Sprintf("expected exactly 1 role_ref element in source state, got %d", len(raw.RoleRef)))
		return
	}
	if raw.ID == "" {
		resp.Diagnostics.AddError("state move failed", "empty 'id' in source state")
		return
	}

	m := raw.Metadata[0]
	annotations, annotationDiags := common.SDKv2MapToFramework(ctx, m.Annotations)
	resp.Diagnostics.Append(annotationDiags...)
	labels, labelDiags := common.SDKv2MapToFramework(ctx, m.Labels)
	resp.Diagnostics.Append(labelDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	meta := common.NamespacedMetadataModel{
		MetadataModel: common.MetadataModel{
			MetadataBase: common.MetadataBase{
				Name:            types.StringValue(m.Name),
				Generation:      types.Int64Value(m.Generation),
				ResourceVersion: types.StringValue(m.ResourceVersion),
				UID:             types.StringValue(m.UID),
				Annotations:     annotations,
				Labels:          labels,
			},
			GenerateName: types.StringNull(),
		},
		Namespace: types.StringValue(m.Namespace),
	}
	// Normalize SDKv2's unset prefix even when the first plan skips refresh.
	if m.GenerateName != "" {
		meta.GenerateName = types.StringValue(m.GenerateName)
	}
	subjects := make([]SubjectModel, 0, len(raw.Subjects))
	for _, s := range raw.Subjects {
		subjects = append(subjects, SubjectModel{
			APIGroup: types.StringValue(s.APIGroup), Kind: types.StringValue(s.Kind),
			Name: types.StringValue(s.Name), Namespace: types.StringValue(s.Namespace),
		})
	}
	rr := raw.RoleRef[0]
	moved := RoleBindingModel{
		ID:       types.StringValue(raw.ID),
		Metadata: []common.NamespacedMetadataModel{meta},
		RoleRef: []RoleRefModel{{
			APIGroup: types.StringValue(rr.APIGroup), Kind: types.StringValue(rr.Kind), Name: types.StringValue(rr.Name),
		}},
		Subject: subjects,
	}
	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &moved)...)
	if resp.Diagnostics.HasError() || resp.TargetIdentity == nil {
		return
	}
	resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, roleBindingIdentity(m.Namespace, m.Name))...)
}
