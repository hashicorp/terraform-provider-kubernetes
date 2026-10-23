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

var (
	_ resource.Resource                    = (*ClusterRoleBinding)(nil)
	_ resource.ResourceWithConfigure       = (*ClusterRoleBinding)(nil)
	_ resource.ResourceWithImportState     = (*ClusterRoleBinding)(nil)
	_ resource.ResourceWithIdentity        = (*ClusterRoleBinding)(nil)
	_ resource.ResourceWithMoveState       = (*ClusterRoleBinding)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*ClusterRoleBinding)(nil)
)

// ClusterRoleBinding is the Plugin Framework implementation of the
// kubernetes_cluster_role_binding_v1 resource. It is served alongside the SDKv2
// provider through the mux server. The schema intentionally uses blocks (rather
// than nested attributes) so that both the HCL configuration syntax and the
// persisted state layout remain compatible with the SDKv2 version.
type ClusterRoleBinding struct {
	// SDKv2Meta returns the provider meta struct configured by the SDKv2
	// provider (shared through the mux server).
	SDKv2Meta func() any
}

func NewClusterRoleBinding() resource.Resource {
	return &ClusterRoleBinding{}
}

func (r *ClusterRoleBinding) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_role_binding_v1"
}

func (r *ClusterRoleBinding) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", "Expected the SDKv2 provider metadata accessor.")
		return
	}
	r.SDKv2Meta = meta
}

func (r *ClusterRoleBinding) UpgradeIdentity(_ context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeIdentity(clusterRoleBindingKind, clusterRoleBindingAPIVersion)
}

const (
	deprecatedClusterRoleBindingTypeName      = "kubernetes_cluster_role_binding"
	deprecatedClusterRoleBindingSchemaVersion = 0
	moveStateCRBErrSummary                    = "Unable to move kubernetes_cluster_role_binding state"
)

// MoveState implements [resource.ResourceWithMoveState] for
//
//	moved {
//	  from = kubernetes_cluster_role_binding.example
//	  to   = kubernetes_cluster_role_binding_v1.example
//	}
//
// the supported route off the deprecated alias, which stays on SDKv2. Cross-type
// moves need Terraform 1.8 or later.
//
// The source is decoded from raw JSON into the frozen sdkv2ClusterRoleBindingStateV0
// rather than through a SourceSchema, so it does not follow this resource's
// schema as that changes.
func (r *ClusterRoleBinding) MoveState(_ context.Context) []resource.StateMover {
	return []resource.StateMover{{
		StateMover: r.moveFromDeprecatedClusterRoleBinding,
	}}
}

func (r *ClusterRoleBinding) moveFromDeprecatedClusterRoleBinding(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
	if req.SourceTypeName != deprecatedClusterRoleBindingTypeName ||
		req.SourceSchemaVersion != deprecatedClusterRoleBindingSchemaVersion ||
		!strings.HasSuffix(req.SourceProviderAddress, providerAddressSuffix) {
		// Skipping returns no state and no diagnostics, and the framework then
		// fails the move without saying why, so leave a reason in the log.
		tflog.Debug(ctx, "MoveState: not a kubernetes_cluster_role_binding v0 source, skipping", map[string]any{
			"source_type_name":        req.SourceTypeName,
			"source_schema_version":   req.SourceSchemaVersion,
			"source_provider_address": req.SourceProviderAddress,
		})
		return
	}
	if req.SourceRawState == nil || len(req.SourceRawState.JSON) == 0 {
		resp.Diagnostics.AddError(moveStateCRBErrSummary, "The source state has no JSON data. Flatmap state is not supported.")
		return
	}

	var src sdkv2ClusterRoleBindingStateV0
	if err := json.Unmarshal(req.SourceRawState.JSON, &src); err != nil {
		resp.Diagnostics.AddError(moveStateCRBErrSummary,
			"The prior state of the source resource could not be decoded: "+err.Error())
		return
	}
	if src.ID == "" {
		resp.Diagnostics.AddError(moveStateCRBErrSummary, "The source state must have a non-empty id.")
		return
	}
	if len(src.Metadata) != 1 {
		resp.Diagnostics.AddError(moveStateCRBErrSummary,
			fmt.Sprintf("Expected exactly 1 metadata element in the source state, got %d.", len(src.Metadata)))
		return
	}
	if len(src.RoleRef) != 1 {
		resp.Diagnostics.AddError(moveStateCRBErrSummary,
			fmt.Sprintf("Expected exactly 1 role_ref element in the source state, got %d.", len(src.RoleRef)))
		return
	}

	m := src.Metadata[0]
	annotations, diags := types.MapValueFrom(ctx, types.StringType, m.Annotations)
	resp.Diagnostics.Append(diags...)
	labels, diags := types.MapValueFrom(ctx, types.StringType, m.Labels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// SDKv2 stores an unset generate_name as "", where the framework holds null.
	// Left as "", a plan that skips refresh sees a change on a RequiresReplace
	// attribute and recreates the binding. Normalise as Role and Namespace do.
	generateName := types.StringNull()
	if m.GenerateName != "" {
		generateName = types.StringValue(m.GenerateName)
	}

	subjects := make([]SubjectModel, len(src.Subject))
	for i, sub := range src.Subject {
		subjects[i] = SubjectModel{
			APIGroup:  types.StringValue(sub.APIGroup),
			Kind:      types.StringValue(sub.Kind),
			Name:      types.StringValue(sub.Name),
			Namespace: types.StringValue(sub.Namespace),
		}
	}

	ref := src.RoleRef[0]
	target := ClusterRoleBindingModel{
		ID: types.StringValue(src.ID),
		Metadata: []common.MetadataModel{{
			MetadataBase: common.MetadataBase{
				Annotations:     annotations,
				Generation:      types.Int64Value(m.Generation),
				Labels:          labels,
				Name:            types.StringValue(m.Name),
				ResourceVersion: types.StringValue(m.ResourceVersion),
				UID:             types.StringValue(m.UID),
			},
			GenerateName: generateName,
		}},
		RoleRef: []RoleRefModel{{
			APIGroup: types.StringValue(ref.APIGroup),
			Kind:     types.StringValue(ref.Kind),
			Name:     types.StringValue(ref.Name),
		}},
		Subject: subjects,
	}

	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &target)...)
	if resp.Diagnostics.HasError() || resp.TargetIdentity == nil {
		return
	}

	// Build the identity rather than decoding req.SourceIdentity: api_version and
	// kind are constants and the name is in the moved state. This also covers
	// state written before resource identity existed.
	resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, r.identity(m.Name))...)
}
