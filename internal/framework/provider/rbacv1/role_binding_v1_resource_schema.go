// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

func (r *RoleBindingV1) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = RoleBindingV1Schema()
}

// RoleBindingV1Schema returns the Plugin Framework schema for RoleBindingV1.
// Exported so that unit tests can construct tfsdk.State values for the state mover.
func RoleBindingV1Schema() schema.Schema {
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
			// role_ref is ForceNew in SDKv2 — modelled here with RequiresReplace on
			// all three fields because the Kubernetes API forbids patching roleRef.
			"role_ref": schema.ListNestedBlock{
				MarkdownDescription: "RoleRef references the Role or ClusterRole granting the permissions defined in this binding.",
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
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"kind": schema.StringAttribute{
							MarkdownDescription: "The kind of the referenced role. Must be `Role` or `ClusterRole`.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("Role", "ClusterRole"),
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the Role or ClusterRole to bind to.",
							Required:            true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
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
							MarkdownDescription: "The namespace of the subject. Required and used only for `ServiceAccount` subjects.",
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
