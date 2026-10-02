// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package nodev1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

type RuntimeClassV1Model struct {
	ID types.String `tfsdk:"id"`
	// Metadata is a slice because the schema declares metadata as a ListNestedBlock,
	// matching SDKv2's TypeList+MaxItems:1 wire shape.
	Metadata []common.MetadataModel `tfsdk:"metadata"`
	Handler  types.String           `tfsdk:"handler"`
}
