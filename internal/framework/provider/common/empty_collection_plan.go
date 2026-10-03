// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// WithEmptyMetadataCompatibility makes omitted annotations and labels keep a
// prior empty map, as SDKv2 state did; nonempty values are still removed.
// Resources using it also run NoOpPlan.
func WithEmptyMetadataCompatibility(block schema.ListNestedBlock) schema.ListNestedBlock {
	for _, name := range []string{"annotations", "labels"} {
		attribute := block.NestedObject.Attributes[name].(schema.MapAttribute)
		attribute.Computed = true
		attribute.Default = mapdefault.StaticValue(types.MapNull(types.StringType))
		attribute.PlanModifiers = append(attribute.PlanModifiers, EmptyMapCompatibility{})
		block.NestedObject.Attributes[name] = attribute
	}
	return block
}

// EmptyMapCompatibility keeps a prior empty map when the value is omitted. The
// attribute must be Optional+Computed, so Core accepts the retained map, with a
// null default, so computed metadata is still marked unknown on a removal.
type EmptyMapCompatibility struct{}

func (EmptyMapCompatibility) Description(context.Context) string {
	return "Preserve a prior empty map when omitted; remove prior nonempty values when omitted."
}
func (m EmptyMapCompatibility) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (EmptyMapCompatibility) PlanModifyMap(_ context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if req.Plan.Raw.IsNull() || !req.ConfigValue.IsNull() {
		return
	}
	resp.PlanValue = req.ConfigValue
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() && len(req.StateValue.Elements()) == 0 {
		resp.PlanValue = req.StateValue
	}
}

// EmptyListCompatibility is the primitive-list counterpart of EmptyMapCompatibility.
type EmptyListCompatibility struct{}

func (EmptyListCompatibility) Description(context.Context) string {
	return "Preserve a prior empty list when omitted; remove prior nonempty values when omitted."
}
func (m EmptyListCompatibility) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (EmptyListCompatibility) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.Plan.Raw.IsNull() || !req.ConfigValue.IsNull() {
		return
	}
	resp.PlanValue = req.ConfigValue
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() && len(req.StateValue.Elements()) == 0 {
		resp.PlanValue = req.StateValue
	}
}
