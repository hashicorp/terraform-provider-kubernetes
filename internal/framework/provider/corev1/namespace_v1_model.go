// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import "github.com/hashicorp/terraform-plugin-framework/types"

// NamespaceV1DataSourceModel is the top-level state model for the
// kubernetes_namespace_v1 data source. Every field maps 1:1 to a schema
// attribute or block via the tfsdk struct tag.
//
// Spec is types.List (not []NamespaceSpecModel) because spec is a
// ListNestedAttribute in the schema. Framework decodes ListNestedAttribute
// into types.List; slices are used for ListNestedBlock only.
type NamespaceV1DataSourceModel struct {
	ID       types.String    `tfsdk:"id"`
	Metadata []MetadataModel `tfsdk:"metadata"`
	Spec     types.List      `tfsdk:"spec"`
}

// MetadataModel mirrors the fields produced by metadataSchema("namespace", false)
// in the SDKv2 codebase (schema_metadata.go). All fields that the API fills in
// are Computed in the schema and therefore typed as types.String / types.Int64
// here — the framework will always populate them after a Read.
type MetadataModel struct {
	Annotations     map[string]types.String `tfsdk:"annotations"`
	Generation      types.Int64             `tfsdk:"generation"`
	Labels          map[string]types.String `tfsdk:"labels"`
	Name            types.String            `tfsdk:"name"`
	ResourceVersion types.String            `tfsdk:"resource_version"`
	UID             types.String            `tfsdk:"uid"`
}

// NamespaceSpecModel is the element type for the spec ListNestedAttribute.
// Framework uses it as the element object type when encoding/decoding types.List.
type NamespaceSpecModel struct {
	Finalizers []types.String `tfsdk:"finalizers"`
}
