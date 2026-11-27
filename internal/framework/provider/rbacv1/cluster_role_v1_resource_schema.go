// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// emptyStringSetDefault is the plan value for an omitted rule list. SDKv2 persists an
// omitted list as null after create and as [] after the next refresh, so a known empty
// set is the one value both states converge on without a perpetual diff.
func emptyStringSetDefault() defaults.Set {
	return setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{}))
}

func (r *ClusterRoleV1) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A ClusterRole creates a role at the cluster level and in all namespaces.",
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
			"metadata": common.MetadataSchemaRBAC("clusterRole", true, false),
			"rule": schema.ListNestedBlock{
				MarkdownDescription: "List of PolicyRules for this ClusterRole. Rules are only tracked when they are configured: " +
					"for a ClusterRole with an `aggregation_rule` the Kubernetes controller manager owns the rules, so they are not read into state.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"api_groups": schema.SetAttribute{
							MarkdownDescription: "APIGroups is the name of the APIGroup that contains the resources. If multiple API groups are specified, any action requested against one of the enumerated resources in any API group will be allowed.",
							ElementType:         types.StringType,
							Optional:            true,
							Computed:            true,
							Default:             emptyStringSetDefault(),
							Validators: []validator.Set{
								setvalidator.SizeAtLeast(1),
								setvalidator.NoNullValues(),
							},
						},
						"non_resource_urls": schema.SetAttribute{
							MarkdownDescription: "NonResourceURLs is a set of partial urls that a user should have access to. *s are allowed, but only as the full, final step in the path Since non-resource URLs are not namespaced, this field is only applicable for ClusterRoles referenced from a ClusterRoleBinding. Rules can either apply to API resources (such as \"pods\" or \"secrets\") or non-resource URL paths (such as \"/api\"), but not both.",
							ElementType:         types.StringType,
							Optional:            true,
							Computed:            true,
							Default:             emptyStringSetDefault(),
							Validators:          []validator.Set{setvalidator.NoNullValues()},
						},
						"resource_names": schema.SetAttribute{
							MarkdownDescription: "ResourceNames is an optional white list of names that the rule applies to. An empty set means that everything is allowed.",
							ElementType:         types.StringType,
							Optional:            true,
							Computed:            true,
							Default:             emptyStringSetDefault(),
							Validators:          []validator.Set{setvalidator.NoNullValues()},
						},
						"resources": schema.SetAttribute{
							MarkdownDescription: "Resources is a list of resources this rule applies to. ResourceAll represents all resources.",
							ElementType:         types.StringType,
							Optional:            true,
							Computed:            true,
							Default:             emptyStringSetDefault(),
							Validators:          []validator.Set{setvalidator.NoNullValues()},
						},
						"verbs": schema.SetAttribute{
							MarkdownDescription: "Verbs is a list of Verbs that apply to ALL the ResourceKinds and AttributeRestrictions contained in this rule. VerbAll represents all kinds.",
							ElementType:         types.StringType,
							Required:            true,
							Validators:          []validator.Set{setvalidator.NoNullValues()},
						},
					},
				},
			},
			"aggregation_rule": schema.ListNestedBlock{
				MarkdownDescription: "Describes how to build the Rules for this ClusterRole.",
				Validators:          []validator.List{listvalidator.SizeAtMost(1)},
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"cluster_role_selectors": schema.ListNestedBlock{
							MarkdownDescription: "A list of selectors which will be used to find ClusterRoles and create the rules.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"match_labels": schema.MapAttribute{
										MarkdownDescription: "A map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of `match_expressions`, whose key field is \"key\", the operator is \"In\", and the values array contains only \"value\". The requirements are ANDed.",
										ElementType:         types.StringType,
										Optional:            true,
										Validators:          []validator.Map{mapvalidator.NoNullValues()},
									},
								},
								Blocks: map[string]schema.Block{
									"match_expressions": schema.ListNestedBlock{
										MarkdownDescription: "A list of label selector requirements. The requirements are ANDed.",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"key": schema.StringAttribute{
													MarkdownDescription: "The label key that the selector applies to.",
													Optional:            true,
												},
												"operator": schema.StringAttribute{
													MarkdownDescription: "A key's relationship to a set of values. Valid operators ard `In`, `NotIn`, `Exists` and `DoesNotExist`.",
													Optional:            true,
												},
												"values": schema.SetAttribute{
													MarkdownDescription: "An array of string values. If the operator is `In` or `NotIn`, the values array must be non-empty. If the operator is `Exists` or `DoesNotExist`, the values array must be empty. This array is replaced during a strategic merge patch.",
													ElementType:         types.StringType,
													Optional:            true,
													Validators:          []validator.Set{setvalidator.NoNullValues()},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *ClusterRoleV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	// Reproduces resourceIdentitySchemaNonNamespaced() from kubernetes/resourceidentity.go,
	// including identity schema version 1.
	resp.IdentitySchema = common.IdentitySchema()
}
