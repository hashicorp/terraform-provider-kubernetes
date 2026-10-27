// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// LimitRangeV1Model is the top-level Terraform state model for kubernetes_limit_range_v1.
type LimitRangeV1Model struct {
	ID       types.String                     `tfsdk:"id"`
	Metadata []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec     []LimitRangeSpecModel            `tfsdk:"spec"`
}

// LimitRangeSpecModel matches the single-element list block for spec.
type LimitRangeSpecModel struct {
	Limit []LimitRangeItemModel `tfsdk:"limit"`
}

// LimitRangeItemModel matches one element of the spec.limit list block.
// All resource-quantity maps use types.Map so the Framework can carry unknown values
// on Optional+Computed attributes (default_request).
type LimitRangeItemModel struct {
	Default              types.Map    `tfsdk:"default"`
	DefaultRequest       types.Map    `tfsdk:"default_request"`
	Max                  types.Map    `tfsdk:"max"`
	MaxLimitRequestRatio types.Map    `tfsdk:"max_limit_request_ratio"`
	Min                  types.Map    `tfsdk:"min"`
	Type                 types.String `tfsdk:"type"`
}
