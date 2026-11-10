// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// ConfigMapV1Model is the state of kubernetes_config_map_v1.
type ConfigMapV1Model struct {
	ID types.String `tfsdk:"id"`
	// Metadata is a slice because the schema declares metadata as a ListNestedBlock.
	// The validators constrain it to exactly one element; callers index [0].
	Metadata   []common.NamespacedMetadataModel `tfsdk:"metadata"`
	BinaryData types.Map                        `tfsdk:"binary_data"`
	Data       types.Map                        `tfsdk:"data"`
	Immutable  types.Bool                       `tfsdk:"immutable"`
}

// sdkv2ConfigMapStateV0 is kubernetes_config_map state as SDKv2 stores it at schema
// version 0. It is frozen: it describes stored JSON, not this resource's schema.
//
// SDKv2 has no null for primitives, so an unset bool is false. Maps stay Go maps, where a
// nil map (JSON null) and an empty one are distinct.
type sdkv2ConfigMapStateV0 struct {
	ID         string                             `json:"id"`
	Metadata   []common.SDKv2NamespacedMetadataV0 `json:"metadata"`
	BinaryData map[string]string                  `json:"binary_data"`
	Data       map[string]string                  `json:"data"`
	Immutable  bool                               `json:"immutable"`
}
