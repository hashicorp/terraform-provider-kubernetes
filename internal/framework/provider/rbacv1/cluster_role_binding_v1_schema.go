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
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

func (r *ClusterRoleBinding) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A ClusterRoleBinding may be used to grant permission at the cluster level and in all namespaces",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The name of the ClusterRoleBinding.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": common.MetadataSchemaRBAC("clusterRoleBinding", true, false),
			"role_ref": schema.ListNestedBlock{
				Description: "RoleRef references the Cluster Role for this binding",
				Validators: []validator.List{
					listvalidator.IsRequired(),
					listvalidator.SizeBetween(1, 1),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"api_group": schema.StringAttribute{
							Description: "The API group of the user. The only value possible at the moment is `rbac.authorization.k8s.io`.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("rbac.authorization.k8s.io"),
							},
						},
						"kind": schema.StringAttribute{
							Description: "The kind of resource.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("Role", "ClusterRole"),
							},
						},
						"name": schema.StringAttribute{
							Description: "The name of the User to bind to.",
							Required:    true,
						},
					},
				},
			},
			"subject": schema.ListNestedBlock{
				Description: "Subjects defines the entities to bind a ClusterRole to.",
				Validators: []validator.List{
					listvalidator.IsRequired(),
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"api_group": schema.StringAttribute{
							Description: "The API group of the subject resource.",
							Optional:    true,
							Computed:    true,
						},
						"kind": schema.StringAttribute{
							Description: "The kind of resource.",
							Required:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the resource to bind to.",
							Required:    true,
						},
						"namespace": schema.StringAttribute{
							Description: "The Namespace of the subject resource.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("default"),
						},
					},
				},
			},
		},
	}
}
