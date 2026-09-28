// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// MetadataBase matches MetadataSchema(objectName, false).
// Its fields are promoted into variants without changing the Terraform object shape.
type MetadataBase struct {
	Annotations     types.Map    `tfsdk:"annotations"`
	Generation      types.Int64  `tfsdk:"generation"`
	Labels          types.Map    `tfsdk:"labels"`
	Name            types.String `tfsdk:"name"`
	ResourceVersion types.String `tfsdk:"resource_version"`
	UID             types.String `tfsdk:"uid"`
}

// MetadataModel matches MetadataSchema(objectName, true).
// Embedded fields must have no tfsdk tag; keyed literals must name MetadataBase.
type MetadataModel struct {
	MetadataBase
	GenerateName types.String `tfsdk:"generate_name"`
}

// NamespacedMetadataModel matches NamespacedMetadataSchema(objectName, true).
type NamespacedMetadataModel struct {
	MetadataModel
	Namespace types.String `tfsdk:"namespace"`
}
