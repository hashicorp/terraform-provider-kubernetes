// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// Schema implements [resource.Resource].
func (r *LimitRangeV1) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Limit Range sets resource usage limits (e.g. memory, cpu, storage) for supported kinds of resources in a namespace. Read more in [the official docs](https://kubernetes.io/docs/concepts/policy/limit-range/).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": common.NamespacedMetadataSchema("limit range", true),
			"spec":     limitRangeSpecBlock(),
		},
	}
}

// limitRangeSpecBlock mirrors the SDKv2 spec schema:
//
//	Optional TypeList MaxItems:1 with an Elem: *schema.Resource.
//
// The framework equivalent is a ListNestedBlock with SizeAtMost(1). The block is
// Optional (no IsRequired()), matching SDKv2 — users may omit spec entirely.
func limitRangeSpecBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Spec defines the limits enforced. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
		},
		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"limit": limitRangeItemBlock(),
			},
		},
	}
}

// limitRangeItemBlock mirrors the SDKv2 spec.limit schema:
//
//	Optional TypeList with resource-quantity TypeMap children.
func limitRangeItemBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Limits is the list of objects that are enforced.",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"default": schema.MapAttribute{
					Description:   "Default resource requirement limit value by resource name if resource limit is omitted.",
					ElementType:   types.StringType,
					Optional:      true,
					PlanModifiers: []planmodifier.Map{suppressEmptyMapDiffs()},
				},
				// Optional+Computed: the API server back-fills default_request from default
				// when the user omits it for Container limits. UseStateForUnknown keeps the
				// prior value when it is not configured, as SDKv2's Optional+Computed did, so
				// an update to another attribute neither re-plans it as unknown nor lets the
				// server back-fill it again from a changed default.
				"default_request": schema.MapAttribute{
					Description:   "The default resource requirement request value by resource name if resource request is omitted.",
					ElementType:   types.StringType,
					Optional:      true,
					Computed:      true,
					PlanModifiers: []planmodifier.Map{mapplanmodifier.UseStateForUnknown(), suppressEmptyMapDiffs()},
				},
				"max": schema.MapAttribute{
					Description:   "Max usage constraints on this kind by resource name.",
					ElementType:   types.StringType,
					Optional:      true,
					PlanModifiers: []planmodifier.Map{suppressEmptyMapDiffs()},
				},
				"max_limit_request_ratio": schema.MapAttribute{
					Description:   "The named resource must have a request and limit that are both non-zero where limit divided by request is less than or equal to the enumerated value; this represents the max burst for the named resource.",
					ElementType:   types.StringType,
					Optional:      true,
					PlanModifiers: []planmodifier.Map{suppressEmptyMapDiffs()},
				},
				"min": schema.MapAttribute{
					Description:   "Min usage constraints on this kind by resource name.",
					ElementType:   types.StringType,
					Optional:      true,
					PlanModifiers: []planmodifier.Map{suppressEmptyMapDiffs()},
				},
				"type": schema.StringAttribute{
					Description: "Type of resource that this limit applies to.",
					Optional:    true,
				},
			},
		},
	}
}

// suppressEmptyMapDiffs treats null and empty maps as semantically equal to prevent
// perpetual diffs when a user explicitly configures `{}` and the API returns null (or vice versa).
func suppressEmptyMapDiffs() planmodifier.Map {
	return emptyMapSuppressModifier{}
}

type emptyMapSuppressModifier struct{}

func (emptyMapSuppressModifier) Description(context.Context) string {
	return "Treats null and empty maps as semantically equal when both state and config represent an empty collection."
}

func (m emptyMapSuppressModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (emptyMapSuppressModifier) PlanModifyMap(ctx context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	if req.ConfigValue.IsUnknown() {
		return
	}

	configEmpty := req.ConfigValue.IsNull() || len(req.ConfigValue.Elements()) == 0
	stateEmpty := req.StateValue.IsNull() || (!req.StateValue.IsUnknown() && len(req.StateValue.Elements()) == 0)

	if configEmpty && stateEmpty {
		resp.PlanValue = req.StateValue
	}
}

// IdentitySchema implements [resource.ResourceWithIdentity].
//
// The SDKv2 resource never declared an Identity field, so all existing state
// carries identity_schema_version 0 with no identity. We declare version 1
// (the standard for this provider) and implement UpgradeIdentity to handle
// the upgrade from the absent-identity case.
func (r *LimitRangeV1) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}
