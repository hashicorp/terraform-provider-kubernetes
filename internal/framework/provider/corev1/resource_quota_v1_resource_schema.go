// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// resourceQuotaScopes lists the scope names SDKv2 accepted for both spec.scopes and
// match_expression.scope_name.
var resourceQuotaScopes = []string{"Terminating", "NotTerminating", "BestEffort", "NotBestEffort", "PriorityClass"}

// Schema implements [resource.Resource].
func (r *ResourceQuotaV1) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A resource quota provides constraints that limit aggregate resource consumption per namespace. It can limit the quantity of objects that can be created in a namespace by type, as well as the total amount of compute resources that may be consumed by resources in that project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
		Blocks: map[string]schema.Block{
			// namespacedMetadataSchema("resource quota", true) in SDKv2.
			"metadata": common.NamespacedMetadataSchema("resource quota", true),
			"spec": schema.ListNestedBlock{
				Description: "Spec defines the desired quota. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
				// Optional in SDKv2: no IsRequired. MaxItems: 1.
				Validators: []validator.List{listvalidator.SizeAtMost(1)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"hard": schema.MapAttribute{
							Description: "The set of desired hard limits for each named resource. More info: http://releases.k8s.io/HEAD/docs/design/admission_control_resource_quota.md#admissioncontrol-plugin-resourcequota",
							// The element type carries the semantic equality that replaces SDKv2's
							// suppressEquivalentResourceQuantity; the validator ports validateResourceList.
							ElementType: common.QuantityType{},
							Optional:    true,
							Validators:  []validator.Map{common.QuantityMapValidator()},
						},
						"scopes": schema.SetAttribute{
							Description: "A collection of filters that must match each object tracked by a quota. If not specified, the quota matches all objects.",
							ElementType: types.StringType,
							Optional:    true,
							Validators: []validator.Set{
								setvalidator.ValueStringsAre(stringvalidator.OneOf(resourceQuotaScopes...)),
							},
							// SDKv2 ForceNew. SDKv2 state holds either null or [] for an unset
							// set, so a plain RequiresReplace would replace the quota when
							// those two are compared.
							PlanModifiers: []planmodifier.Set{
								setplanmodifier.RequiresReplaceIf(
									func(_ context.Context, req planmodifier.SetRequest, resp *setplanmodifier.RequiresReplaceIfFuncResponse) {
										stateEmpty := req.StateValue.IsNull() || len(req.StateValue.Elements()) == 0
										planEmpty := req.PlanValue.IsNull() || len(req.PlanValue.Elements()) == 0
										if stateEmpty && planEmpty {
											return
										}
										resp.RequiresReplace = !req.PlanValue.Equal(req.StateValue)
									},
									"Replaces the quota when scopes change; null and an empty set are the same.",
									"Replaces the quota when scopes change; null and an empty set are the same.",
								),
							},
						},
					},
					Blocks: map[string]schema.Block{
						"scope_selector": schema.ListNestedBlock{
							Description: "A collection of filters like scopes that must match each object tracked by a quota but expressed using ScopeSelectorOperator in combination with possible values. For a resource to match, both scopes AND scopeSelector (if specified in spec), must be matched.",
							Validators:  []validator.List{listvalidator.SizeAtMost(1)},
							NestedObject: schema.NestedBlockObject{
								Blocks: map[string]schema.Block{
									"match_expression": schema.ListNestedBlock{
										Description: "A list of scope selector requirements by scope of the resources.",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"scope_name": schema.StringAttribute{
													Description: "The name of the scope that the selector applies to.",
													Required:    true,
													Validators:  []validator.String{stringvalidator.OneOf(resourceQuotaScopes...)},
												},
												"operator": schema.StringAttribute{
													Description: "Represents a scope's relationship to a set of values.",
													Required:    true,
													Validators:  []validator.String{stringvalidator.OneOf("In", "NotIn", "Exists", "DoesNotExist")},
												},
												"values": schema.SetAttribute{
													Description: "A list of scope selector requirements by scope of the resources.",
													ElementType: types.StringType,
													Optional:    true,
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
			// SDKv2 declares Create and Update, 1 minute each.
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Update: true}),
		},
	}
}

// IdentitySchema implements [resource.ResourceWithIdentity].
//
// SDKv2 resource_quota_v1 declared no identity (Decision 1, option B: this migration adds
// one). The version matches common.IdentitySchemaVersion so it lines up with every other
// namespaced resource, and namespace is OptionalForImport like resourceIdentitySchemaNamespaced.
func (r *ResourceQuotaV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}
