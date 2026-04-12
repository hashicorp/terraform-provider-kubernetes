// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func flattenListenerSetSpec(in gatewayv1.ListenerSetSpec) []interface{} {
	att := make(map[string]interface{})

	att["parent_ref"] = flattenListenerSetParentRef(in.ParentRef)

	if len(in.Listeners) > 0 {
		listeners := make([]interface{}, len(in.Listeners))
		for i, l := range in.Listeners {
			listeners[i] = flattenListenerEntry(l)
		}
		att["listeners"] = listeners
	}

	return []interface{}{att}
}

func flattenListenerSetParentRef(in gatewayv1.ParentGatewayReference) []interface{} {
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

	return []interface{}{ref}
}

func flattenListenerEntry(in gatewayv1.ListenerEntry) map[string]interface{} {
	listener := make(map[string]interface{})

	listener["name"] = string(in.Name)

	if in.Hostname != nil {
		listener["hostname"] = string(*in.Hostname)
	}

	listener["port"] = int(in.Port)

	listener["protocol"] = string(in.Protocol)

	if in.TLS != nil {
		listener["tls"] = flattenListenerTLSConfigLS(in.TLS)
	}

	if in.AllowedRoutes != nil {
		listener["allowed_routes"] = flattenListenerAllowedRoutes(in.AllowedRoutes)
	}

	return listener
}

func flattenListenerTLSConfigLS(in *gatewayv1.ListenerTLSConfig) []interface{} {
	tls := make(map[string]interface{})

	if in.Mode != nil {
		tls["mode"] = string(*in.Mode)
	}

	if len(in.CertificateRefs) > 0 {
		certs := make([]interface{}, len(in.CertificateRefs))
		for i, c := range in.CertificateRefs {
			certs[i] = flattenSecretObjectReferenceLS(c)
		}
		tls["certificate_refs"] = certs
	}

	if len(in.Options) > 0 {
		options := make(map[string]string)
		for k, v := range in.Options {
			options[string(k)] = string(v)
		}
		tls["options"] = options
	}

	return []interface{}{tls}
}

func flattenSecretObjectReferenceLS(in gatewayv1.SecretObjectReference) map[string]interface{} {
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

	return ref
}

func flattenListenerAllowedRoutes(in *gatewayv1.AllowedRoutes) []interface{} {
	routes := make(map[string]interface{})

	if in.Namespaces != nil {
		routes["namespaces"] = flattenRouteNamespacesLS(in.Namespaces)
	}

	if len(in.Kinds) > 0 {
		routes["kinds"] = flattenRouteGroupKindsLS(in.Kinds)
	}

	return []interface{}{routes}
}

func flattenRouteNamespacesLS(in *gatewayv1.RouteNamespaces) []interface{} {
	ns := make(map[string]interface{})

	if in.From != nil {
		ns["from"] = string(*in.From)
	}

	if in.Selector != nil {
		ns["selector"] = flattenLabelSelectorLS(in.Selector)
	}

	return []interface{}{ns}
}

func flattenLabelSelectorLS(in *metav1.LabelSelector) []interface{} {
	if in == nil {
		return nil
	}
	selector := make(map[string]interface{})

	if in.MatchLabels != nil {
		m := make(map[string]string)
		for k, v := range in.MatchLabels {
			m[k] = v
		}
		selector["match_labels"] = m
	}

	if len(in.MatchExpressions) > 0 {
		exprs := make([]interface{}, len(in.MatchExpressions))
		for i, e := range in.MatchExpressions {
			exp := make(map[string]interface{})
			exp["key"] = e.Key
			exp["operator"] = string(e.Operator)
			exp["values"] = e.Values
			exprs[i] = exp
		}
		selector["match_expressions"] = exprs
	}

	return []interface{}{selector}
}

func flattenRouteGroupKindsLS(in []gatewayv1.RouteGroupKind) []interface{} {
	result := make([]interface{}, len(in))
	for i, rg := range in {
		r := make(map[string]interface{})
		if rg.Group != nil {
			r["group"] = string(*rg.Group)
		}
		r["kind"] = string(rg.Kind)
		result[i] = r
	}
	return result
}

func flattenListenerSetStatus(in gatewayv1.ListenerSetStatus) []interface{} {
	status := make(map[string]interface{})

	if len(in.Conditions) > 0 {
		status["conditions"] = flattenListenerSetConditions(in.Conditions)
	}

	if len(in.Listeners) > 0 {
		status["listeners"] = flattenListenerEntryStatuses(in.Listeners)
	}

	return []interface{}{status}
}

func flattenListenerSetConditions(in []metav1.Condition) []interface{} {
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

func flattenListenerEntryStatuses(in []gatewayv1.ListenerEntryStatus) []interface{} {
	result := make([]interface{}, len(in))
	for i, l := range in {
		status := make(map[string]interface{})
		status["name"] = string(l.Name)
		if len(l.SupportedKinds) > 0 {
			status["supported_kinds"] = flattenRouteGroupKindsLS(l.SupportedKinds)
		}
		if l.AttachedRoutes > 0 {
			status["attached_routes"] = l.AttachedRoutes
		}
		if len(l.Conditions) > 0 {
			status["conditions"] = flattenListenerSetConditions(l.Conditions)
		}
		result[i] = status
	}
	return result
}

func expandListenerSetSpec(l []interface{}) gatewayv1.ListenerSetSpec {
	if len(l) == 0 || l[0] == nil {
		return gatewayv1.ListenerSetSpec{}
	}

	in := l[0].(map[string]interface{})
	obj := gatewayv1.ListenerSetSpec{}

	if v, ok := in["parent_ref"].([]interface{}); ok && len(v) > 0 {
		obj.ParentRef = expandListenerSetParentRef(v)
	}

	if v, ok := in["listeners"].([]interface{}); ok && len(v) > 0 {
		obj.Listeners = expandListenerEntries(v)
	}

	return obj
}

func expandListenerSetParentRef(l []interface{}) gatewayv1.ParentGatewayReference {
	if len(l) == 0 || l[0] == nil {
		return gatewayv1.ParentGatewayReference{}
	}

	in := l[0].(map[string]interface{})
	obj := gatewayv1.ParentGatewayReference{}

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

	return obj
}

func expandListenerEntries(l []interface{}) []gatewayv1.ListenerEntry {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.ListenerEntry, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		result[i] = expandListenerEntry(item.(map[string]interface{}))
	}
	return result
}

func expandListenerEntry(in map[string]interface{}) gatewayv1.ListenerEntry {
	obj := gatewayv1.ListenerEntry{}

	if v, ok := in["name"].(string); ok && v != "" {
		obj.Name = gatewayv1.SectionName(v)
	}

	if v, ok := in["hostname"].(string); ok && v != "" {
		h := gatewayv1.Hostname(v)
		obj.Hostname = &h
	}

	if v, ok := in["port"].(int); ok && v > 0 {
		obj.Port = gatewayv1.PortNumber(v)
	}

	if v, ok := in["protocol"].(string); ok && v != "" {
		obj.Protocol = gatewayv1.ProtocolType(v)
	}

	if v, ok := in["tls"].([]interface{}); ok && len(v) > 0 {
		obj.TLS = expandListenerTLSConfigLS(v)
	}

	if v, ok := in["allowed_routes"].([]interface{}); ok && len(v) > 0 {
		obj.AllowedRoutes = expandAllowedRoutesLS(v)
	}

	return obj
}

func expandListenerTLSConfigLS(l []interface{}) *gatewayv1.ListenerTLSConfig {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.ListenerTLSConfig{}

	if v, ok := in["mode"].(string); ok && v != "" {
		m := gatewayv1.TLSModeType(v)
		obj.Mode = &m
	}

	if v, ok := in["certificate_refs"].([]interface{}); ok && len(v) > 0 {
		certs := make([]gatewayv1.SecretObjectReference, len(v))
		for i, c := range v {
			certs[i] = expandSecretObjectReferenceLS(c.(map[string]interface{}))
		}
		obj.CertificateRefs = certs
	}

	if v, ok := in["options"].(map[string]interface{}); ok && len(v) > 0 {
		options := make(map[gatewayv1.AnnotationKey]gatewayv1.AnnotationValue)
		for k, val := range v {
			options[gatewayv1.AnnotationKey(k)] = gatewayv1.AnnotationValue(val.(string))
		}
		obj.Options = options
	}

	return obj
}

func expandSecretObjectReferenceLS(in map[string]interface{}) gatewayv1.SecretObjectReference {
	obj := gatewayv1.SecretObjectReference{}

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

	return obj
}

func expandAllowedRoutesLS(l []interface{}) *gatewayv1.AllowedRoutes {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.AllowedRoutes{}

	if v, ok := in["namespaces"].([]interface{}); ok && len(v) > 0 {
		obj.Namespaces = expandRouteNamespacesLS(v)
	}

	if v, ok := in["kinds"].([]interface{}); ok && len(v) > 0 {
		obj.Kinds = expandRouteGroupKindsLS(v)
	}

	return obj
}

func expandRouteNamespacesLS(l []interface{}) *gatewayv1.RouteNamespaces {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.RouteNamespaces{}

	if v, ok := in["from"].(string); ok && v != "" {
		f := gatewayv1.FromNamespaces(v)
		obj.From = &f
	}

	if v, ok := in["selector"].([]interface{}); ok && len(v) > 0 {
		obj.Selector = expandLabelSelectorLS(v)
	}

	return obj
}

func expandLabelSelectorLS(l []interface{}) *metav1.LabelSelector {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &metav1.LabelSelector{}

	if v, ok := in["match_labels"].(map[string]interface{}); ok && len(v) > 0 {
		m := make(map[string]string)
		for k, val := range v {
			m[k] = val.(string)
		}
		obj.MatchLabels = m
	}

	if v, ok := in["match_expressions"].([]interface{}); ok && len(v) > 0 {
		exprs := make([]metav1.LabelSelectorRequirement, len(v))
		for i, e := range v {
			em := e.(map[string]interface{})
			exprs[i] = metav1.LabelSelectorRequirement{
				Key:      em["key"].(string),
				Operator: metav1.LabelSelectorOperator(em["operator"].(string)),
				Values:   sliceOfString(em["values"].([]interface{})),
			}
		}
		obj.MatchExpressions = exprs
	}

	return obj
}

func expandInterfaceSlice(in interface{}) []string {
	if in == nil {
		return nil
	}
	v := in.([]interface{})
	result := make([]string, len(v))
	for i := range v {
		result[i] = v[i].(string)
	}
	return result
}

func expandRouteGroupKindsLS(l []interface{}) []gatewayv1.RouteGroupKind {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.RouteGroupKind, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		in := item.(map[string]interface{})
		rg := gatewayv1.RouteGroupKind{}
		if v, ok := in["group"].(string); ok && v != "" {
			g := gatewayv1.Group(v)
			rg.Group = &g
		}
		if v, ok := in["kind"].(string); ok && v != "" {
			rg.Kind = gatewayv1.Kind(v)
		}
		result[i] = rg
	}
	return result
}
