// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import "github.com/hashicorp/terraform-plugin-framework/types"

// NamespaceV1DataSourceModel is the top-level state model for the
// kubernetes_namespace_v1 data source. Every field maps 1:1 to a schema
// attribute or block via the tfsdk struct tag.
type NamespaceV1DataSourceModel struct {
	ID       types.String         `tfsdk:"id"`
	Metadata []MetadataModel      `tfsdk:"metadata"`
	Spec     []NamespaceSpecModel `tfsdk:"spec"`
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

// NamespaceSpecModel mirrors the spec block from the SDKv2 data source schema.
// finalizers is a list of strings populated by the Kubernetes API.
type NamespaceSpecModel struct {
	Finalizers []types.String `tfsdk:"finalizers"`
}
