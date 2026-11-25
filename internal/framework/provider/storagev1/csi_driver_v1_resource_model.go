// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

type CSIDriverV1Model struct {
	ID       types.String           `tfsdk:"id"`
	Metadata []common.MetadataModel `tfsdk:"metadata"`
	Spec     []CSIDriverV1SpecModel `tfsdk:"spec"`
}
type CSIDriverV1SpecModel struct {
	AttachRequired       types.Bool `tfsdk:"attach_required"`
	VolumeLifecycleModes types.List `tfsdk:"volume_lifecycle_modes"`
	PodInfoOnMount       types.Bool `tfsdk:"pod_info_on_mount"`
}
