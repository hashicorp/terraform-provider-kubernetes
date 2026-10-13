// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

type ConfigMapV1Model struct {
	ID         types.String                    `tfsdk:"id"`
	Metadata   []common.NamespacedMetadataBase `tfsdk:"metadata"`
	Data       types.Map                       `tfsdk:"data"`
	BinaryData types.Map                       `tfsdk:"binary_data"`
	Immutable  types.Bool                      `tfsdk:"immutable"`
}
