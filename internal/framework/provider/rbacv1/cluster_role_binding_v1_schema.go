// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
			"metadata": schema.ListNestedBlock{
				Description: "Standard cluster role binding's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata",
				Validators: []validator.List{
					// Required, MaxItems: 1 in SDKv2 -> exactly one block.
					listvalidator.SizeBetween(1, 1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"annotations": schema.MapAttribute{
							Description: "An unstructured key value map stored with the clusterRoleBinding that may be used to store arbitrary metadata. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/",
							ElementType: types.StringType,
							Optional:    true,
						},
						"generate_name": schema.StringAttribute{
							Description: "Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency",
							Optional:    true,
							Validators: []validator.String{
								clusterRoleBindingNameValidator{},
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("name")),
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"generation": schema.Int64Attribute{
							Description: "A sequence number representing a specific generation of the desired state.",
							Computed:    true,
						},
						"labels": schema.MapAttribute{
							Description: "Map of string keys and values that can be used to organize and categorize (scope and select) the clusterRoleBinding. May match selectors of replication controllers and services. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/",
							ElementType: types.StringType,
							Optional:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the clusterRoleBinding, must be unique. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
							Optional:    true,
							Computed:    true,
							Validators: []validator.String{
								clusterRoleBindingNameValidator{},
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("generate_name")),
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"resource_version": schema.StringAttribute{
							Description: "An opaque value that represents the internal version of this clusterRoleBinding that can be used by clients to determine when clusterRoleBinding has changed. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency",
							Computed:    true,
						},
						"uid": schema.StringAttribute{
							Description: "The unique in time and space value for this clusterRoleBinding. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids",
							Computed:    true,
						},
					},
				},
			},
			"role_ref": schema.ListNestedBlock{
				Description: "RoleRef references the Cluster Role for this binding",
				Validators: []validator.List{
					// Required, MaxItems: 1 in SDKv2 -> exactly one block.
					listvalidator.SizeBetween(1, 1),
				},
				PlanModifiers: []planmodifier.List{
					// The entire role_ref (and all of its attributes) is ForceNew.
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
					// Required, MinItems: 1 in SDKv2.
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
