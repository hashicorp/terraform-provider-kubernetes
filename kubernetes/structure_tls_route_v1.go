// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func flattenTLSRouteSpec(in gatewayv1.TLSRouteSpec) []interface{} {
	att := make(map[string]interface{})

	if len(in.ParentRefs) > 0 {
		parentRefs := make([]interface{}, len(in.ParentRefs))
		for i, p := range in.ParentRefs {
			parentRefs[i] = flattenTLSParentReference(p)
		}
		att["parent_refs"] = parentRefs
	}

	if len(in.Hostnames) > 0 {
		hostnames := make([]string, len(in.Hostnames))
		for i, h := range in.Hostnames {
			hostnames[i] = string(h)
		}
		att["hostnames"] = hostnames
	}

	if in.UseDefaultGateways != "" {
		att["use_default_gateways"] = string(in.UseDefaultGateways)
	}

	if len(in.Rules) > 0 {
		rules := make([]interface{}, len(in.Rules))
		for i, rule := range in.Rules {
			rules[i] = flattenTLSRouteRule(rule)
		}
		att["rules"] = rules
	}

	return []interface{}{att}
}

func flattenTLSRouteRule(in gatewayv1.TLSRouteRule) map[string]interface{} {
	rule := make(map[string]interface{})

	if in.Name != nil {
		rule["name"] = string(*in.Name)
	}

	if len(in.BackendRefs) > 0 {
		backendRefs := make([]interface{}, len(in.BackendRefs))
		for i, br := range in.BackendRefs {
			backendRefs[i] = flattenTLSBackendRef(br)
		}
		rule["backend_refs"] = backendRefs
	}

	return rule
}

func flattenTLSBackendRef(in gatewayv1.BackendRef) map[string]interface{} {
	ref := make(map[string]interface{})

	if in.Group != nil {
		ref["group"] = string(*in.Group)
	}

	if in.Kind != nil {
		ref["kind"] = string(*in.Kind)
	}

	ref["name"] = string(in.Name)

	if in.Namespace != nil {
		ref["namespace"] = string(*in.Namespace)
	}

	if in.Port != nil {
		ref["port"] = int(*in.Port)
	}

	if in.Weight != nil {
		ref["weight"] = int(*in.Weight)
	}

	return ref
}

func flattenTLSRouteStatus(in gatewayv1.TLSRouteStatus) []interface{} {
	status := make(map[string]interface{})

	if len(in.Parents) > 0 {
		status["parents"] = flattenTLSRouteParentStatuses(in.Parents)
	}

	return []interface{}{status}
}

func flattenTLSRouteParentStatuses(in []gatewayv1.RouteParentStatus) []interface{} {
	result := make([]interface{}, len(in))
	for i, p := range in {
		result[i] = flattenTLSRouteParentStatus(p)
	}
	return result
}

func flattenTLSRouteParentStatus(in gatewayv1.RouteParentStatus) map[string]interface{} {
	parent := make(map[string]interface{})

	parent["parent_ref"] = []interface{}{flattenTLSParentReference(in.ParentRef)}
	parent["controller_name"] = string(in.ControllerName)

	if len(in.Conditions) > 0 {
		parent["conditions"] = flattenTLSRouteConditions(in.Conditions)
	}

	return parent
}

func flattenTLSParentReference(in gatewayv1.ParentReference) map[string]interface{} {
	ref := make(map[string]interface{})

	if in.Group != nil {
		ref["group"] = string(*in.Group)
	}

	if in.Kind != nil {
		ref["kind"] = string(*in.Kind)
	}

	if in.Namespace != nil {
		ref["namespace"] = string(*in.Namespace)
	}

	ref["name"] = string(in.Name)

	if in.SectionName != nil {
		ref["section_name"] = string(*in.SectionName)
	}

	if in.Port != nil {
		ref["port"] = *in.Port
	}

	return ref
}

func flattenTLSRouteConditions(in []metav1.Condition) []interface{} {
	result := make([]interface{}, len(in))
	for i, c := range in {
		condition := make(map[string]interface{})
		condition["type"] = c.Type
		condition["status"] = string(c.Status)
		condition["message"] = c.Message
		condition["reason"] = c.Reason
		if c.LastTransitionTime.IsZero() == false {
			condition["last_transition_time"] = c.LastTransitionTime.Format("2006-01-02T15:04:05Z")
		}
		if c.ObservedGeneration != 0 {
			condition["observed_generation"] = c.ObservedGeneration
		}
		result[i] = condition
	}
	return result
}

func expandTLSRouteSpec(l []interface{}) gatewayv1.TLSRouteSpec {
	if len(l) == 0 || l[0] == nil {
		return gatewayv1.TLSRouteSpec{}
	}

	in := l[0].(map[string]interface{})
	obj := gatewayv1.TLSRouteSpec{}

	if v, ok := in["parent_refs"].([]interface{}); ok && len(v) > 0 {
		obj.ParentRefs = expandTLSParentReferences(v)
	}

	if v, ok := in["hostnames"].([]interface{}); ok && len(v) > 0 {
		hostnames := make([]gatewayv1.Hostname, len(v))
		for i, h := range v {
			hostnames[i] = gatewayv1.Hostname(h.(string))
		}
		obj.Hostnames = hostnames
	}

	if v, ok := in["use_default_gateways"].(string); ok && v != "" {
		obj.UseDefaultGateways = gatewayv1.GatewayDefaultScope(v)
	}

	if v, ok := in["rules"].([]interface{}); ok && len(v) > 0 {
		obj.Rules = expandTLSRouteRules(v)
	}

	return obj
}

func expandTLSParentReferences(l []interface{}) []gatewayv1.ParentReference {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.ParentReference, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		in := item.(map[string]interface{})
		result[i] = expandTLSParentReference(in)
	}
	return result
}

func expandTLSParentReference(in map[string]interface{}) gatewayv1.ParentReference {
	obj := gatewayv1.ParentReference{}

	if v, ok := in["group"].(string); ok && v != "" {
		g := gatewayv1.Group(v)
		obj.Group = &g
	}

	if v, ok := in["kind"].(string); ok && v != "" {
		k := gatewayv1.Kind(v)
		obj.Kind = &k
	}

	if v, ok := in["namespace"].(string); ok && v != "" {
		ns := gatewayv1.Namespace(v)
		obj.Namespace = &ns
	}

	if v, ok := in["name"].(string); ok && v != "" {
		obj.Name = gatewayv1.ObjectName(v)
	}

	if v, ok := in["section_name"].(string); ok && v != "" {
		sn := gatewayv1.SectionName(v)
		obj.SectionName = &sn
	}

	if v, ok := in["port"].(int); ok && v > 0 {
		p := gatewayv1.PortNumber(v)
		obj.Port = &p
	}

	return obj
}

func expandTLSRouteRules(l []interface{}) []gatewayv1.TLSRouteRule {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.TLSRouteRule, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		result[i] = expandTLSRouteRule(item.(map[string]interface{}))
	}
	return result
}

func expandTLSRouteRule(in map[string]interface{}) gatewayv1.TLSRouteRule {
	obj := gatewayv1.TLSRouteRule{}

	if v, ok := in["name"].(string); ok && v != "" {
		name := gatewayv1.SectionName(v)
		obj.Name = &name
	}

	if v, ok := in["backend_refs"].([]interface{}); ok && len(v) > 0 {
		obj.BackendRefs = expandTLSBackendRefs(v)
	}

	return obj
}

func expandTLSBackendRefs(l []interface{}) []gatewayv1.BackendRef {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.BackendRef, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		result[i] = expandTLSBackendRef(item.(map[string]interface{}))
	}
	return result
}

func expandTLSBackendRef(in map[string]interface{}) gatewayv1.BackendRef {
	obj := gatewayv1.BackendRef{}

	if v, ok := in["group"].(string); ok && v != "" {
		g := gatewayv1.Group(v)
		obj.Group = &g
	}

	if v, ok := in["kind"].(string); ok && v != "" {
		k := gatewayv1.Kind(v)
		obj.Kind = &k
	}

	if v, ok := in["name"].(string); ok && v != "" {
		obj.Name = gatewayv1.ObjectName(v)
	}

	if v, ok := in["namespace"].(string); ok && v != "" {
		ns := gatewayv1.Namespace(v)
		obj.Namespace = &ns
	}

	if v, ok := in["port"].(int); ok && v > 0 {
		p := gatewayv1.PortNumber(v)
		obj.Port = &p
	}

	if v, ok := in["weight"].(int); ok && v > 0 {
		w := int32(v)
		obj.Weight = &w
	}

	return obj
}
