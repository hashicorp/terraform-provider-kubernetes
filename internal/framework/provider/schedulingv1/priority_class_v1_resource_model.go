// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

type PriorityClassModel struct {
	ID               types.String           `tfsdk:"id"`
	Metadata         []common.MetadataModel `tfsdk:"metadata"`
	Value            types.Int64            `tfsdk:"value"`
	Description      types.String           `tfsdk:"description"`
	GlobalDefault    types.Bool             `tfsdk:"global_default"`
	PreemptionPolicy types.String           `tfsdk:"preemption_policy"`
}
