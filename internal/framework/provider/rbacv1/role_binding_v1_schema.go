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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *RoleBindingV1) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = RoleBindingV1Schema()
}

// namespacedMetadataBlockAttrs returns the attribute map used inside the
// metadata block for a namespaced resource. Includes all standard fields plus
// namespace. Shared between the live schema and the v0 upgrade schema so both
// describe the same fields.
func namespacedMetadataBlockAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"annotations": schema.MapAttribute{
			MarkdownDescription: "An unstructured key value map stored with the role binding that may be used to store arbitrary metadata. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations",
			ElementType:         types.StringType,
			Optional:            true,
		},
		"generate_name": schema.StringAttribute{
			MarkdownDescription: "Prefix, used by the server, to generate a unique name ONLY IF the name field has not been provided. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#idempotency",
			Optional:            true,
			Validators: []validator.String{
				stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("name")),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"generation": schema.Int64Attribute{
			MarkdownDescription: "A sequence number representing a specific generation of the desired state.",
			Computed:            true,
		},
		"labels": schema.MapAttribute{
			MarkdownDescription: "Map of string keys and values that can be used to organize and categorize (scope and select) the role binding. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels",
			ElementType:         types.StringType,
			Optional:            true,
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "Name of the role binding, must be unique within the namespace. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names",
			Optional:            true,
			Computed:            true,
			Validators: []validator.String{
				stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("generate_name")),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"namespace": schema.StringAttribute{
			MarkdownDescription: "Namespace defines the space within which the name of the role binding must be unique. Defaults to `default`. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("default"),
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"resource_version": schema.StringAttribute{
			MarkdownDescription: "An opaque value that represents the internal version of this object that can be used by clients to determine when objects have changed. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency",
			Computed:            true,
		},
		"uid": schema.StringAttribute{
			MarkdownDescription: "The unique in time and space value for this role binding. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#uids",
			Computed:            true,
		},
	}
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
			// metadata uses ListNestedBlock — the framework equivalent of SDK v2's
			// TypeList{MaxItems:1}. Produces state paths metadata.0.name etc.,
			// preserving full SDK v2 compatibility.
			"metadata": schema.ListNestedBlock{
				MarkdownDescription: "Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata",
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: namespacedMetadataBlockAttrs(),
				},
			},
			// role_ref is ForceNew in SDKv2 — modelled here with RequiresReplace on
			// all three fields because the Kubernetes API forbids patching roleRef.
			"role_ref": schema.ListNestedBlock{
				MarkdownDescription: "RoleRef references the Role or ClusterRole granting the permissions defined in this binding.",
				Validators: []validator.List{
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
