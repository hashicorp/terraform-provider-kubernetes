// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// NamespaceV1DataSourceModel is the top-level state model for the
// kubernetes_namespace_v1 data source. Every field maps 1:1 to a schema
// attribute or block via the tfsdk struct tag.
type NamespaceV1DataSourceModel struct {
	ID       types.String          `tfsdk:"id"`
	Metadata []common.MetadataBase `tfsdk:"metadata"`
	Spec     types.List            `tfsdk:"spec"`
}

// NamespaceSpecModel is the element type for the spec ListNestedAttribute.
// Framework uses it as the element object type when encoding/decoding types.List.
type NamespaceSpecModel struct {
	Finalizers []types.String `tfsdk:"finalizers"`
}
