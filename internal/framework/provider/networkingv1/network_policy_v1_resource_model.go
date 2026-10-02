// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

type NetworkPolicyV1Model struct {
	ID       types.String                     `tfsdk:"id"`
	Metadata []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec     []networkPolicySpecModel         `tfsdk:"spec"`
}

type networkPolicySpecModel struct {
	PodSelector []networkPolicySelectorModel `tfsdk:"pod_selector"`
	Ingress     []networkPolicyIngressModel  `tfsdk:"ingress"`
	Egress      []networkPolicyEgressModel   `tfsdk:"egress"`
	PolicyTypes types.List                   `tfsdk:"policy_types"`
}

type networkPolicyIngressModel struct {
	Ports []networkPolicyPortModel `tfsdk:"ports"`
	From  []networkPolicyPeerModel `tfsdk:"from"`
}

type networkPolicyEgressModel struct {
	Ports []networkPolicyPortModel `tfsdk:"ports"`
	To    []networkPolicyPeerModel `tfsdk:"to"`
}

type networkPolicyPortModel struct {
	Port     types.String `tfsdk:"port"`
	EndPort  types.Int64  `tfsdk:"end_port"`
	Protocol types.String `tfsdk:"protocol"`
}

type networkPolicyPeerModel struct {
	IPBlock           []networkPolicyIPBlockModel  `tfsdk:"ip_block"`
	NamespaceSelector []networkPolicySelectorModel `tfsdk:"namespace_selector"`
	PodSelector       []networkPolicySelectorModel `tfsdk:"pod_selector"`
}

type networkPolicyIPBlockModel struct {
	CIDR   types.String `tfsdk:"cidr"`
	Except types.List   `tfsdk:"except"`
}

type networkPolicySelectorModel struct {
	MatchLabels      types.Map                      `tfsdk:"match_labels"`
	MatchExpressions []networkPolicyExpressionModel `tfsdk:"match_expressions"`
}

type networkPolicyExpressionModel struct {
	Key      types.String `tfsdk:"key"`
	Operator types.String `tfsdk:"operator"`
	Values   types.Set    `tfsdk:"values"`
}
