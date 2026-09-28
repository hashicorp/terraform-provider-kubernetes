// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

var (
	_ resource.Resource                    = (*Role)(nil)
	_ resource.ResourceWithConfigure       = (*Role)(nil)
	_ resource.ResourceWithImportState     = (*Role)(nil)
	_ resource.ResourceWithIdentity        = (*Role)(nil)
	_ resource.ResourceWithMoveState       = (*Role)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*Role)(nil)
)

type Role struct {
	SDKv2Meta func() any
}

func NewRole() resource.Resource {
	return &Role{}
}

func (r *Role) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_v1"
}

func (r *Role) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.SDKv2Meta = req.ProviderData.(func() any)
}

func (r *Role) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	// common.NamespacedIdentitySchema reproduces resourceIdentitySchemaNamespaced() from
	// kubernetes/resourceidentity.go, including Version 1 and namespace being
	// OptionalForImport rather than required.
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}

// UpgradeIdentity implements [resource.ResourceWithUpgradeIdentity].
//
// Without this, any Role created by provider 2.37.x or older fails its first plan with
// "Unable to Upgrade Resource Identity": identity shipped in 2.38.0, so older state carries
// identity_schema_version 0 and no identity, and Terraform asks for an upgrade whenever the
// stored version differs from the declared one — even when nothing is stored. SDKv2 answers
// that generically in its gRPC server; the framework requires each resource to supply it.
func (r *Role) UpgradeIdentity(ctx context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity(rbacAPIVersion, roleKind)
}

const (
	deprecatedRoleTypeName      = "kubernetes_role"
	deprecatedRoleSchemaVersion = 0
	providerAddressSuffix       = "hashicorp/kubernetes"
)

func (r *Role) MoveState(_ context.Context) []resource.StateMover {
	return []resource.StateMover{{
		StateMover: r.moveFromDeprecatedRole,
	}}
}

func (r *Role) moveFromDeprecatedRole(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
	if !strings.HasSuffix(req.SourceProviderAddress, providerAddressSuffix) {
		return
	}
	if req.SourceTypeName != deprecatedRoleTypeName {
		return
	}
	if req.SourceSchemaVersion != deprecatedRoleSchemaVersion {
		return
	}
	if req.SourceRawState == nil {
		return
	}

	var src RoleSourceState
	if err := json.Unmarshal(req.SourceRawState.JSON, &src); err != nil {
		resp.Diagnostics.AddError(
			"Unable to move kubernetes_role state",
			"The prior state of the source resource could not be decoded: "+err.Error(),
		)
		return
	}

	var metaModel common.NamespacedMetadataModel
	if len(src.Metadata) > 0 {
		m := src.Metadata[0]

		annotations, annDiags := types.MapValueFrom(ctx, types.StringType, m.Annotations)
		resp.Diagnostics.Append(annDiags...)
		labels, labelDiags := types.MapValueFrom(ctx, types.StringType, m.Labels)
		resp.Diagnostics.Append(labelDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// SDKv2 has no null for primitives, so it stores an unset generate_name as "".
		// Carrying that across verbatim leaves state holding "" where the framework would
		// hold null, and a plan that skips refresh then sees a change on a ForceNew
		// attribute. Normalise here, as the namespace mover does.
		generateName := types.StringNull()
		if m.GenerateName != "" {
			generateName = types.StringValue(m.GenerateName)
		}

		metaModel = common.NamespacedMetadataModel{
			MetadataModel: common.MetadataModel{
				MetadataBase: common.MetadataBase{
					Generation:      types.Int64Value(m.Generation),
					Name:            types.StringValue(m.Name),
					ResourceVersion: types.StringValue(m.ResourceVersion),
					UID:             types.StringValue(m.UID),
					Annotations:     annotations,
					Labels:          labels,
				},
				GenerateName: generateName,
			},
			Namespace: types.StringValue(m.Namespace),
		}
	}

	rules := make([]RuleModel, len(src.Rule))
	for i, r := range src.Rule {
		apiGroups, diags := flattenStringSet(r.APIGroups)
		resp.Diagnostics.Append(diags...)
		resources, diags := flattenStringSet(r.Resources)
		resp.Diagnostics.Append(diags...)
		resourceNames, diags := flattenStringSet(r.ResourceNames)
		resp.Diagnostics.Append(diags...)
		verbs, diags := flattenStringSet(r.Verbs)
		resp.Diagnostics.Append(diags...)

		rules[i] = RuleModel{
			APIGroups:     apiGroups,
			Resources:     resources,
			ResourceNames: resourceNames,
			Verbs:         verbs,
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	target := RoleModel{
		ID:       types.StringValue(src.ID),
		Metadata: []common.NamespacedMetadataModel{metaModel},
		Rule:     rules,
	}

	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &target)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if resp.TargetIdentity != nil {
		var ns, name string
		if len(src.Metadata) > 0 {
			ns = src.Metadata[0].Namespace
			name = src.Metadata[0].Name
		}
		identity := common.NamespacedResourceIdentity{
			ResourceIdentity: common.ResourceIdentity{
				APIVersion: types.StringValue(rbacAPIVersion),
				Kind:       types.StringValue(roleKind),
				Name:       types.StringValue(name),
			},
			Namespace: types.StringValue(ns),
		}
		resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, identity)...)
	}
}
