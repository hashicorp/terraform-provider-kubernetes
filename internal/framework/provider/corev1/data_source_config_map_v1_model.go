// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ConfigMapV1Model struct {
	Metadata   []ConfigMapMetadataModel `tfsdk:"metadata"`
	Data       types.Map                `tfsdk:"data"`
	BinaryData types.Map                `tfsdk:"binary_data"`
	Immutable  types.Bool               `tfsdk:"immutable"`
}

type ConfigMapMetadataModel struct {
	Name            types.String            `tfsdk:"name"`
	Namespace       types.String            `tfsdk:"namespace"`
	Annotations     map[string]types.String `tfsdk:"annotations"`
	Labels          map[string]types.String `tfsdk:"labels"`
	ResourceVersion types.String            `tfsdk:"resource_version"`
	UID             types.String            `tfsdk:"uid"`
	Generation      types.Int64             `tfsdk:"generation"`
}
