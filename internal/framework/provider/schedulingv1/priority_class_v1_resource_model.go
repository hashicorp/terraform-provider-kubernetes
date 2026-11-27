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

// sdkv2PriorityClassStateV0 is kubernetes_priority_class state as SDKv2 stores it at schema
// version 0. It is frozen: it describes stored JSON, not this resource's schema.
type sdkv2PriorityClassStateV0 struct {
	ID               string                   `json:"id"`
	Metadata         []common.SDKv2MetadataV0 `json:"metadata"`
	Value            int64                    `json:"value"`
	Description      string                   `json:"description"`
	GlobalDefault    bool                     `json:"global_default"`
	PreemptionPolicy string                   `json:"preemption_policy"`
}
