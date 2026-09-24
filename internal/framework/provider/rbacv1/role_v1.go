// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*Role)(nil)
	_ resource.ResourceWithConfigure   = (*Role)(nil)
	_ resource.ResourceWithImportState = (*Role)(nil)
	_ resource.ResourceWithIdentity    = (*Role)(nil)
	_ resource.ResourceWithMoveState   = (*Role)(nil)
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
	resp.IdentitySchema = identityschema.Schema{
		// Version must match resourceIdentitySchemaNamespaced() in the SDKv2
		// implementation so that identity state written before this resource
		// migrated to Framework is still considered current.
		Version: 1,
		Attributes: map[string]identityschema.Attribute{
			"api_version": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"kind": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"namespace": identityschema.StringAttribute{
				OptionalForImport: true,
			},
			"name": identityschema.StringAttribute{
				RequiredForImport: true,
			},
		},
	}
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

	var metaModel MetadataModel
	if len(src.Metadata) > 0 {
		m := src.Metadata[0]
		metaModel = MetadataModel{
			Generation:      types.Int64Value(m.Generation),
			Name:            types.StringValue(m.Name),
			Namespace:       types.StringValue(m.Namespace),
			ResourceVersion: types.StringValue(m.ResourceVersion),
			UID:             types.StringValue(m.UID),
			Annotations:     flattenStringMap(m.Annotations),
			Labels:          flattenStringMap(m.Labels),
		}
		metaModel.GenerateName = types.StringValue(m.GenerateName)
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
		Metadata: []MetadataModel{metaModel},
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
		identity := RoleIdentityModel{
			APIVersion: types.StringValue(rbacAPIVersion),
			Kind:       types.StringValue(roleKind),
			Namespace:  types.StringValue(ns),
			Name:       types.StringValue(name),
		}
		resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, identity)...)
	}
}
