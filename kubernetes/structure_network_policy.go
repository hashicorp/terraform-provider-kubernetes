package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
	api "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strconv"
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

func expandNetworkPolicySpec(l []interface{}) v1.NetworkPolicySpec {
	// There must be exactly one spec element
	obj := v1.NetworkPolicySpec{}
	for _, spec := range l {
		in := spec.(map[string]interface{})
		obj.PodSelector = *expandLabelSelector(in["pod_selector"].([]interface{}))
		if v, ok := in["ingress"].([]interface{}); ok && len(v) > 0 {
			obj.Ingress = expandNetworkPolicyIngress(v)
		}
		if v, ok := in["egress"].([]interface{}); ok && len(v) > 0 {
			obj.Egress = expandNetworkPolicyEgress(v)
		}
	}
	return obj
}

func expandNetworkPolicyIngress(l []interface{}) []v1.NetworkPolicyIngressRule {
	obj := make([]v1.NetworkPolicyIngressRule, len(l), len(l))
	for i, ingress := range l {
		if ingress != nil {
			in := ingress.(map[string]interface{})
			obj[i] = v1.NetworkPolicyIngressRule{}
			if v, ok := in["ports"].([]interface{}); ok && len(v) > 0 {
				obj[i].Ports = expandNetworkPolicyPorts(v)
			}
			if v, ok := in["from"].([]interface{}); ok && len(v) > 0 {
				obj[i].From = expandNetworkPolicyPeer(v)
			}
		}
	}
	return obj
}

func expandNetworkPolicyEgress(l []interface{}) []v1.NetworkPolicyEgressRule {
	obj := make([]v1.NetworkPolicyEgressRule, len(l), len(l))
	for i, ingress := range l {
		if ingress != nil {
			in := ingress.(map[string]interface{})
			obj[i] = v1.NetworkPolicyEgressRule{}
			if v, ok := in["ports"].([]interface{}); ok && len(v) > 0 {
				obj[i].Ports = expandNetworkPolicyPorts(v)
			}
			if v, ok := in["to"].([]interface{}); ok && len(v) > 0 {
				obj[i].To = expandNetworkPolicyPeer(v)
			}
		}
	}
	return obj
}

func expandNetworkPolicyPorts(l []interface{}) []v1.NetworkPolicyPort {
	obj := make([]v1.NetworkPolicyPort, len(l), len(l))
	for i, port := range l {
		in := port.(map[string]interface{})
		if in["port"] != nil && in["port"] != "" {
			portStr := in["port"].(string)
			if portInt, err := strconv.Atoi(portStr); err == nil && strconv.Itoa(portInt) == portStr {
				v := intstr.FromInt(portInt)
				obj[i].Port = &v
			} else {
				v := intstr.FromString(portStr)
				obj[i].Port = &v
			}
		}
		if in["protocol"] != nil && in["protocol"] != "" {
			v := api.Protocol(in["protocol"].(string))
			obj[i].Protocol = &v

		}
	}
	return obj
}

func expandNetworkPolicyPeer(l []interface{}) []v1.NetworkPolicyPeer {
	obj := make([]v1.NetworkPolicyPeer, len(l), len(l))
	for i, peer := range l {
		in := peer.(map[string]interface{})
		if v, ok := in["ip_block"].([]interface{}); ok && len(v) > 0 {
			obj[i].IPBlock = expandIPBlock(v)
		}
		if v, ok := in["namespace_selector"].([]interface{}); ok && len(v) > 0 {
			obj[i].NamespaceSelector = expandLabelSelector(v)
		}
		if v, ok := in["pod_selector"].([]interface{}); ok && len(v) > 0 {
			obj[i].PodSelector = expandLabelSelector(v)
		}
	}
	return obj
}

func expandIPBlock(l []interface{}) *v1.IPBlock {
	obj := &v1.IPBlock{}
	if len(l) == 0 || l[0] == nil {
		return &v1.IPBlock{}
	}
	in := l[0].(map[string]interface{})
	if v, ok := in["cidr"].(string); ok && v != "" {
		obj.CIDR = v
	}
	if v, ok := in["except"].([]interface{}); ok && len(v) > 0 {
		obj.Except = expandStringSlice(v)
	}
	return obj
}

// Patchers

func patchNetworkPolicySpec(keyPrefix, pathPrefix string, d *schema.ResourceData) PatchOperations {
	ops := make([]PatchOperation, 0, 0)
	if d.HasChange(keyPrefix + "ingress") {
		oldV, _ := d.GetChange(keyPrefix + "ingress")
		ingress := expandNetworkPolicyIngress(d.Get(keyPrefix + "ingress").([]interface{}))
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
		egress := expandNetworkPolicyEgress(d.Get(keyPrefix + "egress").([]interface{}))
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
			Value: expandStringSlice(d.Get(keyPrefix + "policy_types").([]interface{})),
		})
	}
	return ops
}
