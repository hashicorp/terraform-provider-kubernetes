// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// StorageClassModel is the top-level Terraform state model for
// kubernetes_storage_class_v1.
type StorageClassModel struct {
	ID                   types.String           `tfsdk:"id"`
	Metadata             []common.MetadataModel `tfsdk:"metadata"`
	StorageProvisioner   types.String           `tfsdk:"storage_provisioner"`
	Parameters           types.Map              `tfsdk:"parameters"`
	ReclaimPolicy        types.String           `tfsdk:"reclaim_policy"`
	VolumeBindingMode    types.String           `tfsdk:"volume_binding_mode"`
	AllowVolumeExpansion types.Bool             `tfsdk:"allow_volume_expansion"`
	// MountOptions is a Set of strings — order is not significant.
	MountOptions types.Set `tfsdk:"mount_options"`
	// AllowedTopologies is MaxItems:1 — modelled as a slice to match the
	// ListNestedBlock HCL path metadata.0.* convention.
	AllowedTopologies []AllowedTopologyModel `tfsdk:"allowed_topologies"`
}

// AllowedTopologyModel represents one allowed_topologies block.
// The SDKv2 schema allows MaxItems:1 so the slice will always have 0 or 1 elements.
type AllowedTopologyModel struct {
	MatchLabelExpressions []MatchLabelExpressionModel `tfsdk:"match_label_expressions"`
}

// MatchLabelExpressionModel represents one match_label_expressions entry.
// values is a Set of strings — order is not significant.
type MatchLabelExpressionModel struct {
	Key    types.String `tfsdk:"key"`
	Values types.Set    `tfsdk:"values"`
}
