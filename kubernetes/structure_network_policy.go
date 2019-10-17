package kubernetes

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	api "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Flatteners

func flattenNetworkPolicySpec(in v1.NetworkPolicySpec) []interface{} {
	att := make(map[string]interface{})
	att["ingress"] = flattenNetworkPolicyIngress(in.Ingress)
	att["egress"] = flattenNetworkPolicyEgress(in.Egress)
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

func flattenNetworkPolicyIngress(in []v1.NetworkPolicyIngressRule) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for i, ingress := range in {
		m := make(map[string]interface{})
		if ingress.Ports != nil && len(ingress.Ports) > 0 {
			m["ports"] = flattenNetworkPolicyPorts(ingress.Ports)
		}
		if ingress.From != nil && len(ingress.From) > 0 {
			m["from"] = flattenNetworkPolicyPeer(ingress.From)
		}
		att[i] = m
	}
	return att
}

func flattenNetworkPolicyEgress(in []v1.NetworkPolicyEgressRule) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for i, egress := range in {
		m := make(map[string]interface{})
		if egress.Ports != nil && len(egress.Ports) > 0 {
			m["ports"] = flattenNetworkPolicyPorts(egress.Ports)
		}
		if egress.To != nil && len(egress.To) > 0 {
			m["to"] = flattenNetworkPolicyPeer(egress.To)
		}
		att[i] = m
	}
	return att
}

func flattenNetworkPolicyPorts(in []v1.NetworkPolicyPort) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for i, port := range in {
		m := make(map[string]interface{})
		if port.Port != nil {
			if (*port.Port).Type == intstr.Int {
				m["port"] = strconv.Itoa(int((*port.Port).IntVal))
			} else {
				m["port"] = (*port.Port).StrVal
			}
		}
		if port.Protocol != nil {
			m["protocol"] = string(*port.Protocol)
		}
		att[i] = m
	}
	return att
}

func flattenNetworkPolicyPeer(in []v1.NetworkPolicyPeer) []interface{} {
	att := make([]interface{}, len(in), len(in))
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

func flattenIPBlock(in *v1.IPBlock) []interface{} {
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

func expandNetworkPolicySpec(in []interface{}) (*v1.NetworkPolicySpec, error) {
	spec := v1.NetworkPolicySpec{}

	if len(in) == 0 || in[0] == nil {
		return nil, fmt.Errorf("failed to expand NetworkPolicy.Spec: null or empty input")
	}

	m, ok := in[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to expand NetworkPolicy.Spec: malformed input")
	}
	spec.PodSelector = *expandLabelSelector(m["pod_selector"].([]interface{}))
	if v, ok := m["ingress"].([]interface{}); ok && len(v) > 0 {
		ingress, err := expandNetworkPolicyIngress(v)
		if err != nil {
			return nil, err
		}
		spec.Ingress = *ingress
	}
	if v, ok := m["egress"].([]interface{}); ok && len(v) > 0 {
		egress, err := expandNetworkPolicyEgress(v)
		if err != nil {
			return nil, err
		}
		spec.Egress = *egress
	}
	policyTypes, err := expandNetworkPolicyTypes(m["policy_types"].([]interface{}))
	if err != nil {
		return nil, err
	}
	spec.PolicyTypes = *policyTypes

	return &spec, nil
}

func expandNetworkPolicyIngress(l []interface{}) (*[]v1.NetworkPolicyIngressRule, error) {
	ingresses := make([]v1.NetworkPolicyIngressRule, len(l), len(l))
	for i, ingress := range l {
		if ingress == nil {
			continue
		}
		in, ok := ingress.(map[string]interface{})
		if !ok {
			continue
		}
		ingresses[i] = v1.NetworkPolicyIngressRule{}
		if v, ok := in["ports"].([]interface{}); ok && len(v) > 0 {
			policyPorts, err := expandNetworkPolicyPorts(v)
			if err != nil {
				return nil, err
			}
			ingresses[i].Ports = *policyPorts
		}
		if v, ok := in["from"].([]interface{}); ok && len(v) > 0 {
			policyPeers, err := expandNetworkPolicyPeer(v)
			if err != nil {
				return nil, err
			}
			ingresses[i].From = *policyPeers
		}
	}
	return &ingresses, nil
}

func expandNetworkPolicyEgress(l []interface{}) (*[]v1.NetworkPolicyEgressRule, error) {
	egresses := make([]v1.NetworkPolicyEgressRule, len(l), len(l))
	for i, egress := range l {
		if egress == nil {
			continue
		}
		in, ok := egress.(map[string]interface{})
		if !ok {
			continue
		}
		egresses[i] = v1.NetworkPolicyEgressRule{}
		if v, ok := in["ports"].([]interface{}); ok && len(v) > 0 {
			policyPorts, err := expandNetworkPolicyPorts(v)
			if err != nil {
				return nil, err
			}
			egresses[i].Ports = *policyPorts
		}
		if v, ok := in["to"].([]interface{}); ok && len(v) > 0 {
			policyPeers, err := expandNetworkPolicyPeer(v)
			if err != nil {
				return nil, err
			}
			egresses[i].To = *policyPeers
		}
	}
	return &egresses, nil
}

func expandNetworkPolicyPorts(l []interface{}) (*[]v1.NetworkPolicyPort, error) {
	policyPorts := make([]v1.NetworkPolicyPort, len(l), len(l))
	for i, port := range l {
		in, ok := port.(map[string]interface{})
		if !ok {
			continue
		}
		if in["port"] != nil && in["port"] != "" {
			portStr := in["port"].(string)
			if portInt, err := strconv.Atoi(portStr); err == nil && strconv.Itoa(portInt) == portStr {
				v := intstr.FromInt(portInt)
				policyPorts[i].Port = &v
			} else {
				v := intstr.FromString(portStr)
				policyPorts[i].Port = &v
			}
		}
		if in["protocol"] != nil && in["protocol"] != "" {
			v := api.Protocol(in["protocol"].(string))
			policyPorts[i].Protocol = &v

		}
	}
	return &policyPorts, nil
}

func expandNetworkPolicyPeer(l []interface{}) (*[]v1.NetworkPolicyPeer, error) {
	policyPeers := make([]v1.NetworkPolicyPeer, len(l), len(l))
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

func expandIPBlock(l []interface{}) (*v1.IPBlock, error) {
	ipBlock := v1.IPBlock{}
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

func expandNetworkPolicyTypes(l []interface{}) (*[]v1.PolicyType, error) {
	policyTypes := make([]v1.PolicyType, 0, 0)
	for _, policyType := range l {
		policyTypes = append(policyTypes, v1.PolicyType(policyType.(string)))
	}
	return &policyTypes, nil
}

// Patchers

func patchNetworkPolicySpec(keyPrefix, pathPrefix string, d *schema.ResourceData) (*PatchOperations, error) {
	ops := make(PatchOperations, 0, 0)
	if d.HasChange(keyPrefix + "ingress") {
		oldV, _ := d.GetChange(keyPrefix + "ingress")
		ingress, err := expandNetworkPolicyIngress(d.Get(keyPrefix + "ingress").([]interface{}))
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
		egress, err := expandNetworkPolicyEgress(d.Get(keyPrefix + "egress").([]interface{}))
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
		policyTypes, err := expandNetworkPolicyTypes(d.Get(keyPrefix + "policy_types").([]interface{}))
		if err != nil {
			return nil, err
		}
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/policyTypes",
			Value: *policyTypes,
		})
	}
	return &ops, nil
}
