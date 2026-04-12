// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func flattenGRPCRouteSpec(in gatewayv1.GRPCRouteSpec) []interface{} {
	att := make(map[string]interface{})

	if len(in.ParentRefs) > 0 {
		parentRefs := make([]interface{}, len(in.ParentRefs))
		for i, p := range in.ParentRefs {
			parentRefs[i] = flattenParentReferenceGRPC(p)
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
			rules[i] = flattenGRPCRouteRule(rule)
		}
		att["rules"] = rules
	}

	return []interface{}{att}
}

func flattenGRPCRouteRule(in gatewayv1.GRPCRouteRule) map[string]interface{} {
	rule := make(map[string]interface{})

	if in.Name != nil {
		rule["name"] = string(*in.Name)
	}

	if len(in.Matches) > 0 {
		matches := make([]interface{}, len(in.Matches))
		for i, m := range in.Matches {
			matches[i] = flattenGRPCRouteMatch(m)
		}
		rule["matches"] = matches
	}

	if len(in.Filters) > 0 {
		filters := make([]interface{}, len(in.Filters))
		for i, f := range in.Filters {
			filters[i] = flattenGRPCRouteFilter(f)
		}
		rule["filters"] = filters
	}

	if len(in.BackendRefs) > 0 {
		backendRefs := make([]interface{}, len(in.BackendRefs))
		for i, br := range in.BackendRefs {
			backendRefs[i] = flattenGRPCBackendRef(br)
		}
		rule["backend_refs"] = backendRefs
	}

	if in.SessionPersistence != nil {
		rule["session_persistence"] = flattenSessionPersistenceGRPC(in.SessionPersistence)
	}

	return rule
}

func flattenGRPCRouteMatch(in gatewayv1.GRPCRouteMatch) map[string]interface{} {
	match := make(map[string]interface{})

	if in.Method != nil {
		match["method"] = flattenGRPCMethodMatch(in.Method)
	}

	if len(in.Headers) > 0 {
		headers := make([]interface{}, len(in.Headers))
		for i, h := range in.Headers {
			headers[i] = flattenGRPCHeaderMatch(h)
		}
		match["headers"] = headers
	}

	return match
}

func flattenGRPCMethodMatch(in *gatewayv1.GRPCMethodMatch) []interface{} {
	m := make(map[string]interface{})
	if in.Type != nil {
		m["type"] = string(*in.Type)
	}
	if in.Service != nil {
		m["service"] = *in.Service
	}
	if in.Method != nil {
		m["method"] = *in.Method
	}
	return []interface{}{m}
}

func flattenGRPCHeaderMatch(in gatewayv1.GRPCHeaderMatch) map[string]interface{} {
	header := make(map[string]interface{})
	header["name"] = string(in.Name)
	header["value"] = in.Value
	if in.Type != nil {
		header["type"] = string(*in.Type)
	}
	return header
}

func flattenGRPCRouteFilter(in gatewayv1.GRPCRouteFilter) map[string]interface{} {
	filter := make(map[string]interface{})
	filter["type"] = string(in.Type)

	if in.RequestHeaderModifier != nil {
		filter["request_header_modifier"] = flattenHTTPHeaderFilterGRPC(in.RequestHeaderModifier)
	}

	if in.ResponseHeaderModifier != nil {
		filter["response_header_modifier"] = flattenHTTPHeaderFilterGRPC(in.ResponseHeaderModifier)
	}

	if in.RequestMirror != nil {
		filter["request_mirror"] = flattenHTTPRequestMirrorFilter(in.RequestMirror)
	}

	if in.ExtensionRef != nil {
		filter["extension_ref"] = flattenLocalObjectReferenceGRPC(*in.ExtensionRef)
	}

	return filter
}

func flattenHTTPHeaderFilterGRPC(in *gatewayv1.HTTPHeaderFilter) []interface{} {
	filter := make(map[string]interface{})

	if len(in.Set) > 0 {
		set := make([]interface{}, len(in.Set))
		for i, h := range in.Set {
			set[i] = flattenHTTPHeaderGRPC(h)
		}
		filter["set"] = set
	}

	if len(in.Add) > 0 {
		add := make([]interface{}, len(in.Add))
		for i, h := range in.Add {
			add[i] = flattenHTTPHeaderGRPC(h)
		}
		filter["add"] = add
	}

	if len(in.Remove) > 0 {
		filter["remove"] = in.Remove
	}

	return []interface{}{filter}
}

func flattenHTTPHeaderGRPC(in gatewayv1.HTTPHeader) map[string]interface{} {
	header := make(map[string]interface{})
	header["name"] = string(in.Name)
	header["value"] = in.Value
	return header
}

func flattenGRPCBackendRef(in gatewayv1.GRPCBackendRef) map[string]interface{} {
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

	if len(in.Filters) > 0 {
		filters := make([]interface{}, len(in.Filters))
		for i, f := range in.Filters {
			filters[i] = flattenGRPCRouteFilter(f)
		}
		ref["filters"] = filters
	}

	return ref
}

func flattenLocalObjectReferenceGRPC(in gatewayv1.LocalObjectReference) []interface{} {
	ref := make(map[string]interface{})
	ref["name"] = in.Name
	return []interface{}{ref}
}

func flattenSessionPersistenceGRPC(in *gatewayv1.SessionPersistence) []interface{} {
	sp := make(map[string]interface{})

	if in.SessionName != nil {
		sp["session_name"] = *in.SessionName
	}

	if in.AbsoluteTimeout != nil {
		sp["absolute_timeout"] = string(*in.AbsoluteTimeout)
	}

	if in.IdleTimeout != nil {
		sp["idle_timeout"] = string(*in.IdleTimeout)
	}

	if in.Type != nil {
		sp["type"] = string(*in.Type)
	}

	return []interface{}{sp}
}

func flattenGRPCRouteStatus(in gatewayv1.GRPCRouteStatus) []interface{} {
	status := make(map[string]interface{})

	if len(in.Parents) > 0 {
		status["parents"] = flattenRouteParentStatusesGRPC(in.Parents)
	}

	return []interface{}{status}
}

func flattenRouteParentStatusesGRPC(in []gatewayv1.RouteParentStatus) []interface{} {
	result := make([]interface{}, len(in))
	for i, p := range in {
		result[i] = flattenRouteParentStatusGRPC(p)
	}
	return result
}

func flattenRouteParentStatusGRPC(in gatewayv1.RouteParentStatus) map[string]interface{} {
	parent := make(map[string]interface{})

	parent["parent_ref"] = []interface{}{flattenParentReferenceGRPC(in.ParentRef)}
	parent["controller_name"] = string(in.ControllerName)

	if len(in.Conditions) > 0 {
		parent["conditions"] = flattenConditionsGRPC(in.Conditions)
	}

	return parent
}

func flattenParentReferenceGRPC(in gatewayv1.ParentReference) map[string]interface{} {
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

func flattenConditionsGRPC(in []metav1.Condition) []interface{} {
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

func expandGRPCRouteSpec(l []interface{}) gatewayv1.GRPCRouteSpec {
	if len(l) == 0 || l[0] == nil {
		return gatewayv1.GRPCRouteSpec{}
	}

	in := l[0].(map[string]interface{})
	obj := gatewayv1.GRPCRouteSpec{}

	if v, ok := in["parent_refs"].([]interface{}); ok && len(v) > 0 {
		obj.ParentRefs = expandParentReferencesGRPC(v)
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
		obj.Rules = expandGRPCRouteRules(v)
	}

	return obj
}

func expandGRPCRouteRules(l []interface{}) []gatewayv1.GRPCRouteRule {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.GRPCRouteRule, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		result[i] = expandGRPCRouteRule(item.(map[string]interface{}))
	}
	return result
}

func expandGRPCRouteRule(in map[string]interface{}) gatewayv1.GRPCRouteRule {
	obj := gatewayv1.GRPCRouteRule{}

	if v, ok := in["name"].(string); ok && v != "" {
		name := gatewayv1.SectionName(v)
		obj.Name = &name
	}

	if v, ok := in["matches"].([]interface{}); ok && len(v) > 0 {
		obj.Matches = expandGRPCRouteMatches(v)
	}

	if v, ok := in["filters"].([]interface{}); ok && len(v) > 0 {
		obj.Filters = expandGRPCRouteFilters(v)
	}

	if v, ok := in["backend_refs"].([]interface{}); ok && len(v) > 0 {
		obj.BackendRefs = expandGRPCBackendRefs(v)
	}

	if v, ok := in["session_persistence"].([]interface{}); ok && len(v) > 0 {
		obj.SessionPersistence = expandSessionPersistenceGRPC(v)
	}

	return obj
}

func expandGRPCRouteMatches(l []interface{}) []gatewayv1.GRPCRouteMatch {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.GRPCRouteMatch, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		result[i] = expandGRPCRouteMatch(item.(map[string]interface{}))
	}
	return result
}

func expandGRPCRouteMatch(in map[string]interface{}) gatewayv1.GRPCRouteMatch {
	obj := gatewayv1.GRPCRouteMatch{}

	if v, ok := in["method"].([]interface{}); ok && len(v) > 0 {
		obj.Method = expandGRPCMethodMatch(v)
	}

	if v, ok := in["headers"].([]interface{}); ok && len(v) > 0 {
		obj.Headers = expandGRPCHeaderMatches(v)
	}

	return obj
}

func expandGRPCMethodMatch(l []interface{}) *gatewayv1.GRPCMethodMatch {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.GRPCMethodMatch{}

	if v, ok := in["type"].(string); ok && v != "" {
		t := gatewayv1.GRPCMethodMatchType(v)
		obj.Type = &t
	}

	if v, ok := in["service"].(string); ok && v != "" {
		obj.Service = &v
	}

	if v, ok := in["method"].(string); ok && v != "" {
		obj.Method = &v
	}

	return obj
}

func expandGRPCHeaderMatches(l []interface{}) []gatewayv1.GRPCHeaderMatch {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.GRPCHeaderMatch, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		result[i] = expandGRPCHeaderMatch(item.(map[string]interface{}))
	}
	return result
}

func expandGRPCHeaderMatch(in map[string]interface{}) gatewayv1.GRPCHeaderMatch {
	obj := gatewayv1.GRPCHeaderMatch{}

	if v, ok := in["name"].(string); ok && v != "" {
		obj.Name = gatewayv1.GRPCHeaderName(v)
	}

	if v, ok := in["value"].(string); ok && v != "" {
		obj.Value = v
	}

	if v, ok := in["type"].(string); ok && v != "" {
		t := gatewayv1.GRPCHeaderMatchType(v)
		obj.Type = &t
	}

	return obj
}

func expandGRPCRouteFilters(l []interface{}) []gatewayv1.GRPCRouteFilter {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.GRPCRouteFilter, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		result[i] = expandGRPCRouteFilter(item.(map[string]interface{}))
	}
	return result
}

func expandGRPCRouteFilter(in map[string]interface{}) gatewayv1.GRPCRouteFilter {
	obj := gatewayv1.GRPCRouteFilter{}

	if v, ok := in["type"].(string); ok && v != "" {
		obj.Type = gatewayv1.GRPCRouteFilterType(v)
	}

	if v, ok := in["request_header_modifier"].([]interface{}); ok && len(v) > 0 {
		obj.RequestHeaderModifier = expandHTTPHeaderFilterGRPC(v)
	}

	if v, ok := in["response_header_modifier"].([]interface{}); ok && len(v) > 0 {
		obj.ResponseHeaderModifier = expandHTTPHeaderFilterGRPC(v)
	}

	if v, ok := in["request_mirror"].([]interface{}); ok && len(v) > 0 {
		obj.RequestMirror = expandHTTPRequestMirrorFilter(v)
	}

	if v, ok := in["extension_ref"].([]interface{}); ok && len(v) > 0 {
		ref := expandLocalObjectReferenceGRPC(v)
		obj.ExtensionRef = &ref
	}

	return obj
}

func expandHTTPHeaderFilterGRPC(l []interface{}) *gatewayv1.HTTPHeaderFilter {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.HTTPHeaderFilter{}

	if v, ok := in["set"].([]interface{}); ok && len(v) > 0 {
		set := make([]gatewayv1.HTTPHeader, len(v))
		for i, h := range v {
			set[i] = expandHTTPHeaderGRPC(h.(map[string]interface{}))
		}
		obj.Set = set
	}

	if v, ok := in["add"].([]interface{}); ok && len(v) > 0 {
		add := make([]gatewayv1.HTTPHeader, len(v))
		for i, h := range v {
			add[i] = expandHTTPHeaderGRPC(h.(map[string]interface{}))
		}
		obj.Add = add
	}

	if v, ok := in["remove"].([]interface{}); ok && len(v) > 0 {
		remove := make([]string, len(v))
		for i, r := range v {
			remove[i] = r.(string)
		}
		obj.Remove = remove
	}

	return obj
}

func expandHTTPHeaderGRPC(in map[string]interface{}) gatewayv1.HTTPHeader {
	obj := gatewayv1.HTTPHeader{}

	if v, ok := in["name"].(string); ok && v != "" {
		obj.Name = gatewayv1.HTTPHeaderName(v)
	}

	if v, ok := in["value"].(string); ok {
		obj.Value = v
	}

	return obj
}

func expandGRPCBackendRefs(l []interface{}) []gatewayv1.GRPCBackendRef {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.GRPCBackendRef, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		result[i] = expandGRPCBackendRef(item.(map[string]interface{}))
	}
	return result
}

func expandGRPCBackendRef(in map[string]interface{}) gatewayv1.GRPCBackendRef {
	obj := gatewayv1.GRPCBackendRef{}

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

	if v, ok := in["filters"].([]interface{}); ok && len(v) > 0 {
		obj.Filters = expandGRPCRouteFilters(v)
	}

	return obj
}

func expandLocalObjectReferenceGRPC(l []interface{}) gatewayv1.LocalObjectReference {
	if len(l) == 0 || l[0] == nil {
		return gatewayv1.LocalObjectReference{}
	}

	in := l[0].(map[string]interface{})
	obj := gatewayv1.LocalObjectReference{}

	if v, ok := in["name"].(string); ok && v != "" {
		obj.Name = gatewayv1.ObjectName(v)
	}

	return obj
}

func expandSessionPersistenceGRPC(l []interface{}) *gatewayv1.SessionPersistence {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.SessionPersistence{}

	if v, ok := in["session_name"].(string); ok && v != "" {
		obj.SessionName = &v
	}

	if v, ok := in["absolute_timeout"].(string); ok && v != "" {
		d := gatewayv1.Duration(v)
		obj.AbsoluteTimeout = &d
	}

	if v, ok := in["idle_timeout"].(string); ok && v != "" {
		d := gatewayv1.Duration(v)
		obj.IdleTimeout = &d
	}

	if v, ok := in["type"].(string); ok && v != "" {
		t := gatewayv1.SessionPersistenceType(v)
		obj.Type = &t
	}

	return obj
}

func expandParentReferencesGRPC(l []interface{}) []gatewayv1.ParentReference {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.ParentReference, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		result[i] = expandParentReferenceGRPC(item.(map[string]interface{}))
	}
	return result
}

func expandParentReferenceGRPC(in map[string]interface{}) gatewayv1.ParentReference {
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
