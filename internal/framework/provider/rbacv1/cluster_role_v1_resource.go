// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

var (
	_ resource.Resource                    = (*ClusterRoleV1)(nil)
	_ resource.ResourceWithConfigure       = (*ClusterRoleV1)(nil)
	_ resource.ResourceWithImportState     = (*ClusterRoleV1)(nil)
	_ resource.ResourceWithIdentity        = (*ClusterRoleV1)(nil)
	_ resource.ResourceWithMoveState       = (*ClusterRoleV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*ClusterRoleV1)(nil)
)

// ClusterRoleV1 is the Plugin Framework implementation of kubernetes_cluster_role_v1.
// It is served alongside the SDKv2 provider through the mux server and keeps the SDKv2
// block syntax, so existing configuration and state stay valid.
type ClusterRoleV1 struct {
	// SDKv2Meta returns the provider meta struct configured by the SDKv2 provider.
	SDKv2Meta func() any
}

func NewClusterRoleV1() resource.Resource {
	return &ClusterRoleV1{}
}

func (r *ClusterRoleV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_role_v1"
}

func (r *ClusterRoleV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// UpgradeIdentity implements [resource.ResourceWithUpgradeIdentity].
//
// Identity shipped in provider 2.38.0, so a ClusterRole created by an older provider
// carries identity_schema_version 0 and no identity. Terraform then asks for an upgrade
// even when nothing is stored. SDKv2 answers that generically; the framework requires
// each resource to supply it, or the first plan fails with "Unable to Upgrade Resource
// Identity".
func (r *ClusterRoleV1) UpgradeIdentity(_ context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeIdentity(clusterRoleKind, clusterRoleAPIVersion)
}

const (
	deprecatedClusterRoleTypeName      = "kubernetes_cluster_role"
	deprecatedClusterRoleSchemaVersion = 0
	moveStateClusterRoleErrSummary     = "Unable to move kubernetes_cluster_role state"
)

// MoveState implements [resource.ResourceWithMoveState] for
//
//	moved {
//	  from = kubernetes_cluster_role.example
//	  to   = kubernetes_cluster_role_v1.example
//	}
//
// the supported route off the deprecated alias, which stays on SDKv2. Cross-type moves
// need Terraform 1.8 or later.
//
// The source is decoded from raw JSON into the frozen sdkv2ClusterRoleStateV0 rather
// than through a SourceSchema, so it does not follow this resource's schema as that
// changes.
func (r *ClusterRoleV1) MoveState(_ context.Context) []resource.StateMover {
	return []resource.StateMover{{
		StateMover: r.moveFromDeprecatedClusterRole,
	}}
}

func (r *ClusterRoleV1) moveFromDeprecatedClusterRole(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
	if req.SourceTypeName != deprecatedClusterRoleTypeName ||
		req.SourceSchemaVersion != deprecatedClusterRoleSchemaVersion ||
		!strings.HasSuffix(req.SourceProviderAddress, providerAddressSuffix) {
		// Skipping returns no state and no diagnostics, and the framework then fails the
		// move without saying why, so leave a reason in the log.
		tflog.Debug(ctx, "MoveState: not a kubernetes_cluster_role v0 source, skipping", map[string]any{
			"source_type_name":        req.SourceTypeName,
			"source_schema_version":   req.SourceSchemaVersion,
			"source_provider_address": req.SourceProviderAddress,
		})
		return
	}
	if req.SourceRawState == nil || len(req.SourceRawState.JSON) == 0 {
		resp.Diagnostics.AddError(moveStateClusterRoleErrSummary, "The source state has no JSON data. Flatmap state is not supported.")
		return
	}

	var src sdkv2ClusterRoleStateV0
	if err := json.Unmarshal(req.SourceRawState.JSON, &src); err != nil {
		resp.Diagnostics.AddError(moveStateClusterRoleErrSummary,
			"The prior state of the source resource could not be decoded: "+err.Error())
		return
	}
	if src.ID == "" {
		resp.Diagnostics.AddError(moveStateClusterRoleErrSummary, "The source state must have a non-empty id.")
		return
	}
	if len(src.Metadata) != 1 {
		resp.Diagnostics.AddError(moveStateClusterRoleErrSummary,
			fmt.Sprintf("Expected exactly 1 metadata element in the source state, got %d.", len(src.Metadata)))
		return
	}
	if len(src.AggregationRule) > 1 {
		resp.Diagnostics.AddError(moveStateClusterRoleErrSummary,
			fmt.Sprintf("Expected at most 1 aggregation_rule element in the source state, got %d.", len(src.AggregationRule)))
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

	// SDKv2 stores an unset generate_name as "", where the framework holds null. Left as
	// "", a plan that skips refresh sees a change on a RequiresReplace attribute and
	// recreates the ClusterRole. Normalise as Role and ClusterRoleBinding do.
	generateName := types.StringNull()
	if m.GenerateName != "" {
		generateName = types.StringValue(m.GenerateName)
	}

	// An omitted rule list is null (after create) or [] (after a refresh) in SDKv2 state.
	// The schema defaults it to an empty set, so both map to that: a null here would plan
	// an in-place update on the first plan after the move when refresh is skipped.
	rules := make([]ClusterRoleRuleModel, len(src.Rule))
	for i, rule := range src.Rule {
		apiGroups, d := flattenStringSet(rule.APIGroups)
		resp.Diagnostics.Append(d...)
		nonResourceURLs, d := flattenStringSet(rule.NonResourceURLs)
		resp.Diagnostics.Append(d...)
		resourceNames, d := flattenStringSet(rule.ResourceNames)
		resp.Diagnostics.Append(d...)
		resources, d := flattenStringSet(rule.Resources)
		resp.Diagnostics.Append(d...)
		verbs, d := flattenStringSet(rule.Verbs)
		resp.Diagnostics.Append(d...)
		rules[i] = ClusterRoleRuleModel{
			APIGroups:       apiGroups,
			NonResourceURLs: nonResourceURLs,
			ResourceNames:   resourceNames,
			Resources:       resources,
			Verbs:           verbs,
		}
	}

	aggregation := make([]AggregationRuleModel, len(src.AggregationRule))
	for i, agg := range src.AggregationRule {
		selectors, d := moveSelectors(ctx, agg.ClusterRoleSelectors)
		resp.Diagnostics.Append(d...)
		aggregation[i] = AggregationRuleModel{ClusterRoleSelectors: selectors}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	target := ClusterRoleModel{
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
		Rule:            rules,
		AggregationRule: aggregation,
	}

	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &target)...)
	if resp.Diagnostics.HasError() || resp.TargetIdentity == nil {
		return
	}

	// Build the identity rather than decoding req.SourceIdentity: api_version and kind
	// are constants and the name is in the moved state. This also covers state written
	// before resource identity existed.
	resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, r.identity(m.Name))...)
}

func moveSelectors(ctx context.Context, in []sdkv2ClusterRoleSelectorV0) ([]ClusterRoleSelectorModel, diag.Diagnostics) {
	var allDiags diag.Diagnostics
	out := make([]ClusterRoleSelectorModel, len(in))
	for i, sel := range in {
		// SDKv2 keeps null and {} apart for maps, so a nil Go map is null here too.
		matchLabels := types.MapNull(types.StringType)
		if sel.MatchLabels != nil {
			elems := make(map[string]attr.Value, len(sel.MatchLabels))
			for k, v := range sel.MatchLabels {
				elems[k] = types.StringValue(v)
			}
			m, d := types.MapValue(types.StringType, elems)
			allDiags.Append(d...)
			matchLabels = m
		}

		exprs := make([]MatchExpressionModel, len(sel.MatchExpressions))
		for j, expr := range sel.MatchExpressions {
			values := types.SetNull(types.StringType)
			if expr.Values != nil {
				v, d := flattenStringSet(expr.Values)
				allDiags.Append(d...)
				values = v
			}
			exprs[j] = MatchExpressionModel{
				Key:      types.StringValue(expr.Key),
				Operator: types.StringValue(expr.Operator),
				Values:   values,
			}
		}
		out[i] = ClusterRoleSelectorModel{MatchLabels: matchLabels, MatchExpressions: exprs}
	}
	return out, allDiags
}
