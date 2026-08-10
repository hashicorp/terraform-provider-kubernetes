// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NamespaceV1Model struct {
	ID                           types.String `tfsdk:"id"`
	WaitForDefaultServiceAccount types.Bool   `tfsdk:"wait_for_default_service_account"`
	// Metadata is a slice because the schema declares metadata as a
	// ListNestedBlock. The SizeAtMost(1) validator constrains the value, not the
	// type — so this stays a list and callers index [0] after a length check.
	Metadata []MetadataModel `tfsdk:"metadata"`
	Timeouts timeouts.Value  `tfsdk:"timeouts"`
}

type MetadataModel struct {
	Annotations     types.Map    `tfsdk:"annotations"`
	GenerateName    types.String `tfsdk:"generate_name"`
	Generation      types.Int64  `tfsdk:"generation"`
	Labels          types.Map    `tfsdk:"labels"`
	Name            types.String `tfsdk:"name"`
	ResourceVersion types.String `tfsdk:"resource_version"`
	UID             types.String `tfsdk:"uid"`
}

type NamespaceResourceIdentity struct {
	APIVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`
	Name       types.String `tfsdk:"name"`
}
