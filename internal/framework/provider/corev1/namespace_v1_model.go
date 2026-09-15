// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NamespaceV1Model mirrors the SDKv2 schema of kubernetes_namespace_v1 exactly.
//
// `id` is modelled explicitly because SDKv2 injected a top-level id into every
// resource (helper/schema/core_schema.go). It is a published attribute of this
// resource and in-repo configurations consume it, so the Framework schema has
// to declare it or those configurations stop resolving.
//
// Metadata is a Go slice because `metadata` stays a ListNestedBlock: SDKv2's
// TypeList{MaxItems: 1} renders as NestingList, and any shape that renders as a
// protocol Object cannot decode existing state.
type NamespaceV1Model struct {
	Timeouts                     timeouts.Value           `tfsdk:"timeouts"`
	ID                           types.String             `tfsdk:"id"`
	Metadata                     []NamespaceMetadataModel `tfsdk:"metadata"`
	WaitForDefaultServiceAccount types.Bool               `tfsdk:"wait_for_default_service_account"`
}

// NamespaceMetadataModel is metadataSchema("namespace", true) — the
// cluster-scoped metadata shape, which has no `namespace` attribute.
//
// Annotations and Labels are types.Map rather than map[string]types.String
// because this migration turns on being able to tell a null map from a known
// empty one; a Go map collapses that distinction.
type NamespaceMetadataModel struct {
	Annotations     types.Map    `tfsdk:"annotations"`
	GenerateName    types.String `tfsdk:"generate_name"`
	Generation      types.Int64  `tfsdk:"generation"`
	Labels          types.Map    `tfsdk:"labels"`
	Name            types.String `tfsdk:"name"`
	ResourceVersion types.String `tfsdk:"resource_version"`
	UID             types.String `tfsdk:"uid"`
}

// NamespaceV1IdentityModel reproduces resourceIdentitySchemaNonNamespaced().
type NamespaceV1IdentityModel struct {
	APIVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`
	Name       types.String `tfsdk:"name"`
}
