// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// MetadataModel is the model for the block built by MetadataSchema(objectName, true):
// metadata for a cluster-scoped object whose name may be server-generated. It mirrors
// metadataSchema(objectName, true) from kubernetes/schema_metadata.go.
//
// Every field must correspond to an attribute in the schema, or the framework fails to
// decode — a struct field with no matching attribute is an error, not a zero value.
// That is why the generate_name variant below is a separate type rather than this one
// reused: the attribute is absent from the schema, so the field must be absent here.
type MetadataModel struct {
	Annotations     types.Map    `tfsdk:"annotations"`
	GenerateName    types.String `tfsdk:"generate_name"`
	Generation      types.Int64  `tfsdk:"generation"`
	Labels          types.Map    `tfsdk:"labels"`
	Name            types.String `tfsdk:"name"`
	ResourceVersion types.String `tfsdk:"resource_version"`
	UID             types.String `tfsdk:"uid"`
}

// MetadataModelExcludingGenerateName is the model for the block built by
// MetadataSchema(objectName, false). It is MetadataModel without GenerateName, and is
// what a data source decodes into when its SDKv2 counterpart passed
// generatableName=false — as the namespace and config_map data sources do.
//
// Keep the shared fields identical to MetadataModel's.
type MetadataModelExcludingGenerateName struct {
	Annotations     types.Map    `tfsdk:"annotations"`
	Generation      types.Int64  `tfsdk:"generation"`
	Labels          types.Map    `tfsdk:"labels"`
	Name            types.String `tfsdk:"name"`
	ResourceVersion types.String `tfsdk:"resource_version"`
	UID             types.String `tfsdk:"uid"`
}
