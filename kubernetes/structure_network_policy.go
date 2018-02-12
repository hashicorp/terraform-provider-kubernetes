package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	api "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/networking/v1"
	"strconv"
)

// Flatteners

func flattenNetworkPolicySpec(in v1.NetworkPolicySpec) []interface{} {
	att := make(map[string]interface{})
	att["ingress"] = flattenNetworkPolicyIngress(in.Ingress)
	if len(in.PodSelector.MatchExpressions) > 0 || len(in.PodSelector.MatchLabels) > 0 {
		att["pod_selector"] = flattenLabelSelector(&in.PodSelector)
	} else {
		att["pod_selector"] = []interface{}{make(map[string]interface{})}
	}
	return []interface{}{att}
}

func flattenNetworkPolicyIngress(in []v1.NetworkPolicyIngressRule) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for i, ingress := range in {
		m := make(map[string]interface{})
		if ingress.Ports != nil && len(ingress.Ports) > 0 {
			m["ports"] = flattenNetworkPolicyIngressPorts(ingress.Ports)
		}
		if ingress.From != nil && len(ingress.From) > 0 {
			m["from"] = flattenNetworkPolicyIngressFrom(ingress.From)
		}
		att[i] = m
	}
	return att
}

func flattenNetworkPolicyIngressPorts(in []v1.NetworkPolicyPort) []interface{} {
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

func flattenNetworkPolicyIngressFrom(in []v1.NetworkPolicyPeer) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for i, from := range in {
		m := make(map[string]interface{})
		if from.NamespaceSelector != nil {
			m["namespace_selector"] = flattenLabelSelector(from.NamespaceSelector)
		}
		if from.PodSelector != nil {
			m["pod_selector"] = flattenLabelSelector(from.PodSelector)
		}
		att[i] = m
	}
	return att
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
				obj[i].Ports = expandNetworkPolicyIngressPorts(v)
			}
			if v, ok := in["from"].([]interface{}); ok && len(v) > 0 {
				obj[i].From = expandNetworkPolicyIngressFrom(v)
			}
		}
	}
	return obj
}

func expandNetworkPolicyIngressPorts(l []interface{}) []v1.NetworkPolicyPort {
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

func expandNetworkPolicyIngressFrom(l []interface{}) []v1.NetworkPolicyPeer {
	obj := make([]v1.NetworkPolicyPeer, len(l), len(l))
	for i, from := range l {
		in := from.(map[string]interface{})
		if v, ok := in["namespace_selector"].([]interface{}); ok && len(v) > 0 {
			obj[i].NamespaceSelector = expandLabelSelector(v)
		}
		if v, ok := in["pod_selector"].([]interface{}); ok && len(v) > 0 {
			obj[i].NamespaceSelector = expandLabelSelector(v)
		}
	}
	return obj
}

// Patchers

func patchNetworkPolicySpec(keyPrefix, pathPrefix string, d *schema.ResourceData) PatchOperations {
	ops := make([]PatchOperation, 0, 0)
	if d.HasChange(keyPrefix + "ingress") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/ingress",
			Value: expandNetworkPolicyIngress(d.Get(keyPrefix + "ingress").([]interface{})),
		})
	}
	if d.HasChange(keyPrefix + "pod_selector") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/podSelector",
			Value: expandLabelSelector(d.Get(keyPrefix + "pod_selector").([]interface{})),
		})
	}
	return ops
}
