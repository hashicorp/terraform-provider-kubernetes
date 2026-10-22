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

// NamespacedMetadataBase is the model for the block built by
// NamespacedDataSourceMetadataSchema: MetadataBase plus namespace.
//
// It exists for namespaced data sources. The resource side has no equivalent because a
// namespaced resource always exposes generate_name too, which makes its model
// NamespacedMetadataModel.
type NamespacedMetadataBase struct {
	MetadataBase
	Namespace types.String `tfsdk:"namespace"`
}

type SDKv2NamespacedMetadataV0 struct {
	Annotations     map[string]string `json:"annotations"`
	GenerateName    string            `json:"generate_name"`
	Generation      int64             `json:"generation"`
	Labels          map[string]string `json:"labels"`
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	ResourceVersion string            `json:"resource_version"`
	UID             string            `json:"uid"`
}
