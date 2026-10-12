// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// SecretV1Model is the top-level tfsdk model for data.kubernetes_secret_v1.
//
// Metadata uses common.NamespacedMetadataModel — the shared namespaced,
// generatable metadata model that matches namespacedMetadataSchema("secret", true).
type SecretV1Model struct {
	ID         types.String                     `tfsdk:"id"`
	Metadata   []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Data       types.Map                        `tfsdk:"data"`
	BinaryData types.Map                        `tfsdk:"binary_data"`
	Type       types.String                     `tfsdk:"type"`
	Immutable  types.Bool                       `tfsdk:"immutable"`
}
