// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

// Flatteners

func flattenNetworkPolicyV1Spec(in networkingv1.NetworkPolicySpec) []interface{} {
	att := make(map[string]interface{})
	att["ingress"] = flattenNetworkPolicyV1Ingress(in.Ingress)
	att["egress"] = flattenNetworkPolicyV1Egress(in.Egress)
	if len(in.PodSelector.MatchExpressions) > 0 || len(in.PodSelector.MatchLabels) > 0 {
		att["pod_selector"] = flattenLabelSelector(&in.PodSelector)
	} else {
		att["pod_selector"] = []interface{}{make(map[string]interface{})}
	}
	if len(in.PolicyTypes) > 0 {
		att["policy_types"] = in.PolicyTypes
	}
	return []interface{}{att}
}

func flattenNetworkPolicyV1Ingress(in []networkingv1.NetworkPolicyIngressRule) []interface{} {
	att := make([]interface{}, len(in))
	for i, ingress := range in {
		m := make(map[string]interface{})
		if len(ingress.Ports) > 0 {
			m["ports"] = flattenNetworkPolicyV1Ports(ingress.Ports)
		}
		if len(ingress.From) > 0 {
			m["from"] = flattenNetworkPolicyV1Peer(ingress.From)
		}
		att[i] = m
	}
	return att
}

func flattenNetworkPolicyV1Egress(in []networkingv1.NetworkPolicyEgressRule) []interface{} {
	att := make([]interface{}, len(in))
	for i, egress := range in {
		m := make(map[string]interface{})
		if len(egress.Ports) > 0 {
			m["ports"] = flattenNetworkPolicyV1Ports(egress.Ports)
		}
		if len(egress.To) > 0 {
			m["to"] = flattenNetworkPolicyV1Peer(egress.To)
		}
		att[i] = m
	}
	return att
}

func flattenNetworkPolicyV1Ports(in []networkingv1.NetworkPolicyPort) []interface{} {
	att := make([]interface{}, len(in))
	for i, port := range in {
		m := make(map[string]interface{})
		if port.Port != nil {
			m["port"] = port.Port.String()
		}
		if port.EndPort != nil && *port.EndPort != 0 {
			m["end_port"] = int(*port.EndPort)
		}
		if port.Protocol != nil {
			m["protocol"] = string(*port.Protocol)
		}
		att[i] = m
	}
	return att
}

func flattenNetworkPolicyV1Peer(in []networkingv1.NetworkPolicyPeer) []interface{} {
	att := make([]interface{}, len(in))
	for i, peer := range in {
		m := make(map[string]interface{})
		if peer.IPBlock != nil {
			m["ip_block"] = flattenIPBlock(peer.IPBlock)
		}
		if peer.NamespaceSelector != nil {
			m["namespace_selector"] = flattenLabelSelector(peer.NamespaceSelector)
		}
		if peer.PodSelector != nil {
			m["pod_selector"] = flattenLabelSelector(peer.PodSelector)
		}
		att[i] = m
	}
	return att
}

func flattenIPBlock(in *networkingv1.IPBlock) []interface{} {
	att := make(map[string]interface{})
	if in.CIDR != "" {
		att["cidr"] = in.CIDR
	}
	if len(in.Except) > 0 {
		att["except"] = in.Except
	}
	return []interface{}{att}
}

// Expanders

func expandNetworkPolicyV1Spec(in []interface{}) (*networkingv1.NetworkPolicySpec, error) {
	spec := networkingv1.NetworkPolicySpec{}

	if len(in) == 0 || in[0] == nil {
		return nil, fmt.Errorf("failed to expand NetworkPolicy.Spec: null or empty input")
	}

	m, ok := in[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to expand NetworkPolicy.Spec: malformed input")
	}
	spec.PodSelector = *expandLabelSelector(m["pod_selector"].([]interface{}))
	if v, ok := m["ingress"].([]interface{}); ok && len(v) > 0 {
		ingress, err := expandNetworkPolicyV1Ingress(v)
		if err != nil {
			return nil, err
		}
		spec.Ingress = *ingress
	}
	if v, ok := m["egress"].([]interface{}); ok && len(v) > 0 {
		egress, err := expandNetworkPolicyV1Egress(v)
		if err != nil {
			return nil, err
		}
		spec.Egress = *egress
	}

	spec.PolicyTypes = expandNetworkPolicyV1Types(m["policy_types"].([]interface{}))

	return &spec, nil
}

func expandNetworkPolicyV1Ingress(l []interface{}) (*[]networkingv1.NetworkPolicyIngressRule, error) {
	ingresses := make([]networkingv1.NetworkPolicyIngressRule, len(l))
	for i, ingress := range l {
		if ingress == nil {
			continue
		}
		in, ok := ingress.(map[string]interface{})
		if !ok {
			continue
		}
		ingresses[i] = networkingv1.NetworkPolicyIngressRule{}
		if v, ok := in["ports"].([]interface{}); ok && len(v) > 0 {
			ingresses[i].Ports = *expandNetworkPolicyV1Ports(v)
		}
		if v, ok := in["from"].([]interface{}); ok && len(v) > 0 {
			policyPeers, err := expandNetworkPolicyV1Peer(v)
			if err != nil {
				return nil, err
			}
			ingresses[i].From = *policyPeers
		}
	}
	return &ingresses, nil
}

func expandNetworkPolicyV1Egress(l []interface{}) (*[]networkingv1.NetworkPolicyEgressRule, error) {
	egresses := make([]networkingv1.NetworkPolicyEgressRule, len(l))
	for i, egress := range l {
		if egress == nil {
			continue
		}
		in, ok := egress.(map[string]interface{})
		if !ok {
			continue
		}
		egresses[i] = networkingv1.NetworkPolicyEgressRule{}
		if v, ok := in["ports"].([]interface{}); ok && len(v) > 0 {
			egresses[i].Ports = *expandNetworkPolicyV1Ports(v)
		}
		if v, ok := in["to"].([]interface{}); ok && len(v) > 0 {
			policyPeers, err := expandNetworkPolicyV1Peer(v)
			if err != nil {
				return nil, err
			}
			egresses[i].To = *policyPeers
		}
	}
	return &egresses, nil
}

func expandNetworkPolicyV1Ports(l []interface{}) *[]networkingv1.NetworkPolicyPort {
	policyPorts := make([]networkingv1.NetworkPolicyPort, len(l))
	for i, port := range l {
		in, ok := port.(map[string]interface{})
		if !ok {
			continue
		}
		if v, ok := in["port"].(string); ok && len(v) > 0 {
			val := intstr.Parse(v)
			policyPorts[i].Port = &val
		}
		if v, ok := in["end_port"].(int); ok && v != 0 {
			policyPorts[i].EndPort = ptr.To(int32(v))
		}
		if in["protocol"] != nil && in["protocol"] != "" {
			v := corev1.Protocol(in["protocol"].(string))
			policyPorts[i].Protocol = &v
		}
	}
	return &policyPorts
}

func expandNetworkPolicyV1Peer(l []interface{}) (*[]networkingv1.NetworkPolicyPeer, error) {
	policyPeers := make([]networkingv1.NetworkPolicyPeer, len(l))
	for i, peer := range l {
		if peer == nil {
			continue
		}
		in, ok := peer.(map[string]interface{})
		if !ok {
			continue
		}
		if v, ok := in["ip_block"].([]interface{}); ok && len(v) > 0 {
			ipBlock, err := expandIPBlock(v)
			if err != nil {
				return nil, err
			}
			policyPeers[i].IPBlock = ipBlock
		}
		if v, ok := in["namespace_selector"].([]interface{}); ok && len(v) > 0 {
			policyPeers[i].NamespaceSelector = expandLabelSelector(v)
		}
		if v, ok := in["pod_selector"].([]interface{}); ok && len(v) > 0 {
			policyPeers[i].PodSelector = expandLabelSelector(v)
		}
	}
	return &policyPeers, nil
}

func expandIPBlock(l []interface{}) (*networkingv1.IPBlock, error) {
	ipBlock := networkingv1.IPBlock{}
	if len(l) == 0 || l[0] == nil {
		return nil, fmt.Errorf("failed to expand IPBlock: null or empty input")
	}
	in, ok := l[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to expand IPBlock: malformed input")
	}
	if v, ok := in["cidr"].(string); ok && v != "" {
		ipBlock.CIDR = v
	}
	if v, ok := in["except"].([]interface{}); ok && len(v) > 0 {
		ipBlock.Except = expandStringSlice(v)
	}
	return &ipBlock, nil
}

func expandNetworkPolicyV1Types(l []interface{}) []networkingv1.PolicyType {
	policyTypes := make([]networkingv1.PolicyType, 0)
	for _, policyType := range l {
		policyTypes = append(policyTypes, networkingv1.PolicyType(policyType.(string)))
	}
	return policyTypes
}

// Patchers

func patchNetworkPolicyV1Spec(keyPrefix, pathPrefix string, d *schema.ResourceData) (*PatchOperations, error) {
	ops := make(PatchOperations, 0)
	if d.HasChange(keyPrefix + "ingress") {
		oldV, _ := d.GetChange(keyPrefix + "ingress")
		ingress, err := expandNetworkPolicyV1Ingress(d.Get(keyPrefix + "ingress").([]interface{}))
		if err != nil {
			return nil, err
		}
		if len(oldV.([]interface{})) == 0 {
			ops = append(ops, &AddOperation{
				Path:  pathPrefix + "/ingress",
				Value: ingress,
			})
		} else {
			ops = append(ops, &ReplaceOperation{
				Path:  pathPrefix + "/ingress",
				Value: ingress,
			})
		}
	}
	if d.HasChange(keyPrefix + "egress") {
		oldV, _ := d.GetChange(keyPrefix + "egress")
		egress, err := expandNetworkPolicyV1Egress(d.Get(keyPrefix + "egress").([]interface{}))
		if err != nil {
			return nil, err
		}
		if len(oldV.([]interface{})) == 0 {
			ops = append(ops, &AddOperation{
				Path:  pathPrefix + "/egress",
				Value: egress,
			})
		} else {
			ops = append(ops, &ReplaceOperation{
				Path:  pathPrefix + "/egress",
				Value: egress,
			})
		}
	}
	if d.HasChange(keyPrefix + "pod_selector") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/podSelector",
			Value: expandLabelSelector(d.Get(keyPrefix + "pod_selector").([]interface{})),
		})
	}
	if d.HasChange(keyPrefix + "policy_types") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/policyTypes",
			Value: expandNetworkPolicyV1Types(d.Get(keyPrefix + "policy_types").([]interface{})),
		})
	}
	return &ops, nil
}
