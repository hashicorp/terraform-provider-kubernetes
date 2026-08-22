// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package nodev1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RuntimeClassV1Model maps 1-to-1 to the Terraform schema via tfsdk tags.
// The Framework serialises/deserialises state into this struct automatically.
// Metadata is a []MetadataModel (slice of one) because the schema uses
// ListNestedBlock — matching the SDKv2 TypeList(MaxItems:1) shape so that
// HCL syntax (metadata { }) and state paths (metadata.0.name) are unchanged.
type RuntimeClassV1Model struct {
	ID       types.String    `tfsdk:"id"`
	Metadata []MetadataModel `tfsdk:"metadata"`
	Handler  types.String    `tfsdk:"handler"`
}

// MetadataModel mirrors the Kubernetes ObjectMeta fields exposed to Terraform users.
type MetadataModel struct {
	Annotations     map[string]types.String `tfsdk:"annotations"`
	GenerateName    types.String            `tfsdk:"generate_name"`
	Generation      types.Int64             `tfsdk:"generation"`
	Labels          map[string]types.String `tfsdk:"labels"`
	Name            types.String            `tfsdk:"name"`
	ResourceVersion types.String            `tfsdk:"resource_version"`
	UID             types.String            `tfsdk:"uid"`
}

// RuntimeClassV1IdentityModel is used by ImportState for structured import.
// Terraform 1.12+ can import using {api_version, kind, name}.
type RuntimeClassV1IdentityModel struct {
	APIVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`
	Name       types.String `tfsdk:"name"`
}
