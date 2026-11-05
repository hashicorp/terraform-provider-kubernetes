// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// ResourceQuotaV1Model matches the schema in resource_quota_v1_resource_schema.go. A field
// with no matching attribute, or the reverse, is a decode error at plan time.
type ResourceQuotaV1Model struct {
	ID types.String `tfsdk:"id"`
	// Metadata and Spec are slices because the schema declares them as ListNestedBlocks:
	// SizeAtMost(1) constrains the value, not the type, so callers check the length before
	// indexing [0].
	Metadata []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec     []ResourceQuotaSpecModel         `tfsdk:"spec"`
	Timeouts timeouts.Value                   `tfsdk:"timeouts"`
}

type ResourceQuotaSpecModel struct {
	// Hard is a types.Map of common.QuantityType, so a quantity the API returns in canonical
	// form ("2Gi" for "2048Mi") keeps the user's spelling through semantic equality.
	Hard          types.Map                         `tfsdk:"hard"`
	Scopes        types.Set                         `tfsdk:"scopes"`
	ScopeSelector []ResourceQuotaScopeSelectorModel `tfsdk:"scope_selector"`
}

type ResourceQuotaScopeSelectorModel struct {
	MatchExpression []ResourceQuotaMatchExpressionModel `tfsdk:"match_expression"`
}

type ResourceQuotaMatchExpressionModel struct {
	ScopeName types.String `tfsdk:"scope_name"`
	Operator  types.String `tfsdk:"operator"`
	Values    types.Set    `tfsdk:"values"`
}
