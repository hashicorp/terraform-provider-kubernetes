// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
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
			// Role is namespaced, so this is the namespaced variant. common reproduces
			// SDKv2's namespacedMetadataSchema("role", true) attribute for attribute,
			// including the descriptions, so the generated docs do not churn.
			"metadata": common.MetadataSchemaRBAC("role", true, true),
			"rule": schema.ListNestedBlock{
				MarkdownDescription: "(Required) Rule defining a set of permissions for the role. At least one `rule` block must be present.",
				NestedObject: schema.NestedBlockObject{
					Attributes: policyRuleSchemaAttributes(),
				},
				Validators: []validator.List{
					listvalidator.IsRequired(),
					listvalidator.SizeAtLeast(1),
				},
			},
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
			Computed:            true,
			Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
		},
		"verbs": schema.SetAttribute{
			MarkdownDescription: "List of Verbs that apply to ALL the ResourceKinds and AttributeRestrictions contained in this rule",
			ElementType:         types.StringType,
			Required:            true,
		},
	}
}
