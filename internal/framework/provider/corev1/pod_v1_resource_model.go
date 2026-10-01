// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

type PodV1Model struct {
	ID          types.String                     `tfsdk:"id"`
	Metadata    []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec        types.List                       `tfsdk:"spec"`
	TargetState types.List                       `tfsdk:"target_state"`
	Timeouts    timeouts.Value                   `tfsdk:"timeouts"`
}
