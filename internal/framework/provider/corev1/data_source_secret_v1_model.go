// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SecretV1Model is the top-level tfsdk model for data.kubernetes_secret_v1.
type SecretV1Model struct {
	Metadata   []SecretV1MetadataModel `tfsdk:"metadata"`
	Data       types.Map               `tfsdk:"data"`
	BinaryData types.Map               `tfsdk:"binary_data"`
	Type       types.String            `tfsdk:"type"`
	Immutable  types.Bool              `tfsdk:"immutable"`
}

// SecretV1MetadataModel maps to the single element of the metadata list block.
// All fields mirror namespacedMetadataSchema so that metadata.0.* paths are
// identical to the SDKv2 data source.
type SecretV1MetadataModel struct {
	Name            types.String            `tfsdk:"name"`
	Namespace       types.String            `tfsdk:"namespace"`
	GenerateName    types.String            `tfsdk:"generate_name"`
	Annotations     map[string]types.String `tfsdk:"annotations"`
	Labels          map[string]types.String `tfsdk:"labels"`
	Generation      types.Int64             `tfsdk:"generation"`
	ResourceVersion types.String            `tfsdk:"resource_version"`
	UID             types.String            `tfsdk:"uid"`
}
