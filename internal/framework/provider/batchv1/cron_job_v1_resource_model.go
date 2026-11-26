// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

type CronJobV1Model struct {
	Metadata []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec     types.List                       `tfsdk:"spec"`
	Timeouts timeouts.Value                   `tfsdk:"timeouts"`
	ID       types.String                     `tfsdk:"id"`
}
