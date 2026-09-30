// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func networkPolicyExpandSpec(ctx context.Context, in []networkPolicySpecModel) (networking.NetworkPolicySpec, diag.Diagnostics) {
	var diags diag.Diagnostics
	var out networking.NetworkPolicySpec
	if len(in) != 1 || len(in[0].PodSelector) != 1 {
		diags.AddError("Invalid network policy specification", "Exactly one spec and pod_selector block is required.")
		return out, diags
	}
	model := in[0]
	selector, d := networkPolicyExpandSelector(ctx, model.PodSelector)
	diags.Append(d...)
	out.PodSelector = *selector
	if len(model.Ingress) > 0 {
		out.Ingress = make([]networking.NetworkPolicyIngressRule, len(model.Ingress))
	}
	for i, rule := range model.Ingress {
		peers, d := networkPolicyExpandPeers(ctx, rule.From)
		diags.Append(d...)
		out.Ingress[i] = networking.NetworkPolicyIngressRule{Ports: networkPolicyExpandPorts(rule.Ports), From: peers}
	}
	if len(model.Egress) > 0 {
		out.Egress = make([]networking.NetworkPolicyEgressRule, len(model.Egress))
	}
	for i, rule := range model.Egress {
		peers, d := networkPolicyExpandPeers(ctx, rule.To)
		diags.Append(d...)
		out.Egress[i] = networking.NetworkPolicyEgressRule{Ports: networkPolicyExpandPorts(rule.Ports), To: peers}
	}
	diags.Append(model.PolicyTypes.ElementsAs(ctx, &out.PolicyTypes, false)...)
	return out, diags
}

func networkPolicyExpandPorts(in []networkPolicyPortModel) []networking.NetworkPolicyPort {
	if len(in) == 0 {
		return nil
	}
	out := make([]networking.NetworkPolicyPort, len(in))
	for i, port := range in {
		if port.Port.ValueString() != "" {
			out[i].Port = ptr.To(intstr.Parse(port.Port.ValueString()))
		}
		if port.EndPort.ValueInt64() != 0 {
			out[i].EndPort = ptr.To(int32(port.EndPort.ValueInt64()))
		}
		if port.Protocol.ValueString() != "" {
			out[i].Protocol = ptr.To(corev1.Protocol(port.Protocol.ValueString()))
		}
	}
	return out
}

func networkPolicyExpandPeers(ctx context.Context, in []networkPolicyPeerModel) ([]networking.NetworkPolicyPeer, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(in) == 0 {
		return nil, diags
	}
	out := make([]networking.NetworkPolicyPeer, len(in))
	for i, peer := range in {
		if len(peer.IPBlock) > 0 {
			block := peer.IPBlock[0]
			out[i].IPBlock = &networking.IPBlock{CIDR: block.CIDR.ValueString()}
			diags.Append(block.Except.ElementsAs(ctx, &out[i].IPBlock.Except, false)...)
		}
		selector, d := networkPolicyExpandSelector(ctx, peer.NamespaceSelector)
		diags.Append(d...)
		out[i].NamespaceSelector = selector
		selector, d = networkPolicyExpandSelector(ctx, peer.PodSelector)
		diags.Append(d...)
		out[i].PodSelector = selector
	}
	return out, diags
}

func networkPolicyExpandSelector(ctx context.Context, in []networkPolicySelectorModel) (*metav1.LabelSelector, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(in) == 0 {
		return nil, diags
	}
	// A present empty selector must remain non-nil: it selects everything, rather
	// than omitting this peer's selector.
	out := &metav1.LabelSelector{}
	diags.Append(in[0].MatchLabels.ElementsAs(ctx, &out.MatchLabels, false)...)
	if len(in[0].MatchExpressions) > 0 {
		out.MatchExpressions = make([]metav1.LabelSelectorRequirement, len(in[0].MatchExpressions))
	}
	for i, expression := range in[0].MatchExpressions {
		out.MatchExpressions[i].Key = expression.Key.ValueString()
		out.MatchExpressions[i].Operator = metav1.LabelSelectorOperator(expression.Operator.ValueString())
		diags.Append(expression.Values.ElementsAs(ctx, &out.MatchExpressions[i].Values, false)...)
	}
	return out, diags
}

func networkPolicyPrior[T any](in []T, index int) T {
	if index < len(in) {
		return in[index]
	}
	var zero T
	return zero
}

func networkPolicyFlattenSpec(ctx context.Context, in networking.NetworkPolicySpec, prior []networkPolicySpecModel) ([]networkPolicySpecModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	old := networkPolicyPrior(prior, 0)
	out := networkPolicySpecModel{
		Ingress: make([]networkPolicyIngressModel, len(in.Ingress)),
		Egress:  make([]networkPolicyEgressModel, len(in.Egress)),
	}
	selector, d := networkPolicyFlattenSelector(ctx, &in.PodSelector, old.PodSelector)
	diags.Append(d...)
	out.PodSelector = selector
	for i, rule := range in.Ingress {
		oldRule := networkPolicyPrior(old.Ingress, i)
		peers, d := networkPolicyFlattenPeers(ctx, rule.From, oldRule.From)
		diags.Append(d...)
		out.Ingress[i] = networkPolicyIngressModel{
			Ports: networkPolicyFlattenPorts(rule.Ports, oldRule.Ports),
			From:  peers,
		}
	}
	for i, rule := range in.Egress {
		oldRule := networkPolicyPrior(old.Egress, i)
		peers, d := networkPolicyFlattenPeers(ctx, rule.To, oldRule.To)
		diags.Append(d...)
		out.Egress[i] = networkPolicyEgressModel{
			Ports: networkPolicyFlattenPorts(rule.Ports, oldRule.Ports),
			To:    peers,
		}
	}
	out.PolicyTypes, d = types.ListValueFrom(ctx, types.StringType, in.PolicyTypes)
	diags.Append(d...)
	return []networkPolicySpecModel{out}, diags
}

func networkPolicyFlattenPorts(in []networking.NetworkPolicyPort, prior []networkPolicyPortModel) []networkPolicyPortModel {
	out := make([]networkPolicyPortModel, len(in))
	for i, port := range in {
		old := networkPolicyPrior(prior, i)
		out[i] = networkPolicyPortModel{
			Port:     types.StringValue(""),
			EndPort:  types.Int64Value(0),
			Protocol: types.StringValue(""),
		}
		if port.Port != nil {
			out[i].Port = types.StringValue(port.Port.String())
			// intstr.Parse uses base ten: "0080" and "80" have the same API
			// representation. Never rewrite a known configured spelling.
			if !old.Port.IsNull() && !old.Port.IsUnknown() && old.Port.ValueString() != "" &&
				intstr.Parse(old.Port.ValueString()) == *port.Port {
				out[i].Port = old.Port
			}
		}
		if port.EndPort != nil {
			out[i].EndPort = types.Int64Value(int64(*port.EndPort))
			if !old.EndPort.IsNull() && !old.EndPort.IsUnknown() && int32(old.EndPort.ValueInt64()) == *port.EndPort {
				out[i].EndPort = old.EndPort
			}
		}
		if port.Protocol != nil {
			out[i].Protocol = types.StringValue(string(*port.Protocol))
		}
		if !old.Protocol.IsNull() && !old.Protocol.IsUnknown() && old.Protocol.ValueString() == "" &&
			(port.Protocol == nil || *port.Protocol == corev1.ProtocolTCP) {
			out[i].Protocol = old.Protocol
		}
	}
	return out
}

func networkPolicyFlattenPeers(ctx context.Context, in []networking.NetworkPolicyPeer, prior []networkPolicyPeerModel) ([]networkPolicyPeerModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := make([]networkPolicyPeerModel, len(in))
	for i, peer := range in {
		old := networkPolicyPrior(prior, i)
		out[i].IPBlock = []networkPolicyIPBlockModel{}
		if peer.IPBlock != nil {
			oldBlock := networkPolicyPrior(old.IPBlock, 0)
			except, d := networkPolicyFlattenStrings(ctx, peer.IPBlock.Except, oldBlock.Except)
			diags.Append(d...)
			out[i].IPBlock = []networkPolicyIPBlockModel{{
				CIDR:   types.StringValue(peer.IPBlock.CIDR),
				Except: except,
			}}
		}
		selector, d := networkPolicyFlattenSelector(ctx, peer.NamespaceSelector, old.NamespaceSelector)
		diags.Append(d...)
		out[i].NamespaceSelector = selector
		selector, d = networkPolicyFlattenSelector(ctx, peer.PodSelector, old.PodSelector)
		diags.Append(d...)
		out[i].PodSelector = selector
	}
	return out, diags
}

func networkPolicyFlattenSelector(ctx context.Context, in *metav1.LabelSelector, prior []networkPolicySelectorModel) ([]networkPolicySelectorModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if in == nil {
		return []networkPolicySelectorModel{}, diags
	}
	old := networkPolicyPrior(prior, 0)
	out := networkPolicySelectorModel{
		MatchLabels:      types.MapNull(types.StringType),
		MatchExpressions: make([]networkPolicyExpressionModel, len(in.MatchExpressions)),
	}
	if len(in.MatchLabels) > 0 || (!old.MatchLabels.IsNull() && !old.MatchLabels.IsUnknown()) {
		labels := in.MatchLabels
		if labels == nil {
			labels = map[string]string{}
		}
		var d diag.Diagnostics
		out.MatchLabels, d = types.MapValueFrom(ctx, types.StringType, labels)
		diags.Append(d...)
	}
	for i, expression := range in.MatchExpressions {
		oldExpression := networkPolicyPrior(old.MatchExpressions, i)
		values := types.SetNull(types.StringType)
		if len(expression.Values) > 0 || (!oldExpression.Values.IsNull() && !oldExpression.Values.IsUnknown()) {
			strings := expression.Values
			if strings == nil {
				strings = []string{}
			}
			var d diag.Diagnostics
			values, d = types.SetValueFrom(ctx, types.StringType, strings)
			diags.Append(d...)
		}
		out.MatchExpressions[i] = networkPolicyExpressionModel{
			Key:      types.StringValue(expression.Key),
			Operator: types.StringValue(string(expression.Operator)),
			Values:   values,
		}
	}
	return []networkPolicySelectorModel{out}, diags
}

func networkPolicyFlattenStrings(ctx context.Context, in []string, prior types.List) (types.List, diag.Diagnostics) {
	if len(in) == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.ListNull(types.StringType), nil
	}
	if in == nil {
		in = []string{}
	}
	return types.ListValueFrom(ctx, types.StringType, in)
}
