// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

func (r *RoleBindingV1) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = roleBindingV1Schema()
}

func roleBindingV1Schema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "A RoleBinding may be used to grant permissions defined in a Role to a set of subjects within a namespace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique ID for this terraform resource in the form `namespace/name`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			// MetadataSchemaRBAC mirrors metadataSchemaRBAC("roleBinding", true, true) from SDKv2.
			// It uses the RBAC path-segment name validator rather than DNS-subdomain, so names like
			// "system:controller:foo" are accepted. generatableName=true, namespaced=true.
			"metadata": common.MetadataSchemaRBAC("role binding", true, true),
			"role_ref": schema.ListNestedBlock{
				MarkdownDescription: "RoleRef references the Role or ClusterRole granting the permissions defined in this binding.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				Validators: []validator.List{
					listvalidator.IsRequired(),
					listvalidator.SizeBetween(1, 1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"api_group": schema.StringAttribute{
							MarkdownDescription: "The API group of the referenced Role. The only valid value is `rbac.authorization.k8s.io`.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("rbac.authorization.k8s.io"),
							},
						},
						"kind": schema.StringAttribute{
							MarkdownDescription: "The kind of the referenced role. Must be `Role` or `ClusterRole`.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("Role", "ClusterRole"),
							},
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the Role or ClusterRole to bind to.",
							Required:            true,
						},
					},
				},
			},
			// subject is a list of one or more entities that receive the permissions.
			"subject": schema.ListNestedBlock{
				MarkdownDescription: "Subjects defines the entities (users, service accounts, or groups) to bind the role to.",
				Validators: []validator.List{
					listvalidator.IsRequired(),
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"api_group": schema.StringAttribute{
							MarkdownDescription: "The API group of the subject. For User and Group subjects use `rbac.authorization.k8s.io`. For ServiceAccount subjects use `\"\"`.",
							Optional:            true,
							Computed:            true,
							PlanModifiers: []planmodifier.String{
								subjectAPIGroupUseState{},
							},
						},
						"kind": schema.StringAttribute{
							MarkdownDescription: "The kind of the subject. One of `User`, `ServiceAccount`, or `Group`.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("User", "ServiceAccount", "Group"),
							},
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the subject.",
							Required:            true,
						},
						"namespace": schema.StringAttribute{
							MarkdownDescription: "The namespace of the referenced ServiceAccount. Defaults to `default`. Used only for `ServiceAccount` subjects.",
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString("default"),
						},
					},
				},
			},
		},
	}
}

func (r *RoleBindingV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}

func (r *RoleBindingV1) UpgradeIdentity(_ context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity(roleBindingKind, rbacAPIVersion)
}

type subjectAPIGroupUseState struct{}

func (subjectAPIGroupUseState) Description(context.Context) string {
	return "Retains an omitted API group when the subject at the same list index is unchanged."
}

func (m subjectAPIGroupUseState) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (subjectAPIGroupUseState) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() || !req.PlanValue.IsUnknown() ||
		req.StateValue.IsNull() || req.StateValue.IsUnknown() ||
		req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	// Lists are positional: a different subject must not inherit this one's API group.
	for _, field := range []string{"kind", "name"} {
		var prior, planned types.String
		fieldPath := req.Path.ParentPath().AtName(field)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, fieldPath, &prior)...)
		resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, fieldPath, &planned)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if prior.IsNull() || prior.IsUnknown() || planned.IsNull() || planned.IsUnknown() || !prior.Equal(planned) {
			return
		}
	}
	resp.PlanValue = req.StateValue
}
