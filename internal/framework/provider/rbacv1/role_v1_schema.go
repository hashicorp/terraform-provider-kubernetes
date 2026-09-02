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

func (r *Role) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A role contains rules that represent a set of permissions. Permissions are purely additive (there are no \"deny\" rules).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique ID for this terraform resource",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": schema.ListNestedBlock{
				MarkdownDescription: "(Required) Standard role's metadata. Exactly one `metadata` block must be present. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata",
				NestedObject: schema.NestedBlockObject{
					Attributes: metadataSchemaAttributes(),
				},
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 1),
				},
			},
			"rule": schema.ListNestedBlock{
				MarkdownDescription: "(Required) Rule defining a set of permissions for the role. At least one `rule` block must be present.",
				NestedObject: schema.NestedBlockObject{
					Attributes: policyRuleSchemaAttributes(),
				},
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
		},
	}
}

func metadataSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"annotations": schema.MapAttribute{
			MarkdownDescription: "An unstructured key value map stored with the role that may be used to store arbitrary metadata. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations",
			ElementType:         types.StringType,
			Optional:            true,
		},
		"generate_name": schema.StringAttribute{
			MarkdownDescription: "Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency",
			Optional:            true,
			Validators: []validator.String{
				rbacNameValidator{},
				stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("name")),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"generation": schema.Int64Attribute{
			MarkdownDescription: "A sequence number representing a specific generation of the desired state. Populated by the system. Read-only.",
			Computed:            true,
		},
		"labels": schema.MapAttribute{
			MarkdownDescription: "Map of string keys and values that can be used to organize and categorize (scope and select) the role. May match selectors of replication controllers and services. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels",
			ElementType:         types.StringType,
			Optional:            true,
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "Name of the role, must be unique. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names",
			Optional:            true,
			Computed:            true,
			Validators: []validator.String{
				rbacNameValidator{},
				stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("generate_name")),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"namespace": schema.StringAttribute{
			MarkdownDescription: "Namespace defines the space within which name of the role must be unique.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("default"),
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"resource_version": schema.StringAttribute{
			MarkdownDescription: "An opaque value that represents the internal version of this role that can be used by clients to determine when the role has changed. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency",
			Computed:            true,
		},
		"uid": schema.StringAttribute{
			MarkdownDescription: "The unique in time and space value for this role. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#uids",
			Computed:            true,
		},
	}
}

func policyRuleSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"api_groups": schema.SetAttribute{
			MarkdownDescription: "Name of the APIGroup that contains the resources",
			ElementType:         types.StringType,
			Required:            true,
		},
		"resources": schema.SetAttribute{
			MarkdownDescription: "List of resources that the rule applies to",
			ElementType:         types.StringType,
			Required:            true,
		},
		"resource_names": schema.SetAttribute{
			MarkdownDescription: "White list of names that the rule applies to",
			ElementType:         types.StringType,
			Optional:            true,
		},
		"verbs": schema.SetAttribute{
			MarkdownDescription: "List of Verbs that apply to ALL the ResourceKinds and AttributeRestrictions contained in this rule",
			ElementType:         types.StringType,
			Required:            true,
		},
	}
}
