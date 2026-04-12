// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func flattenGatewayV1Spec(in gatewayv1.GatewaySpec) []interface{} {
	att := make(map[string]interface{})

	if in.GatewayClassName != "" {
		att["gateway_class_name"] = string(in.GatewayClassName)
	}

	listeners := make([]interface{}, len(in.Listeners))
	for i, listener := range in.Listeners {
		listeners[i] = flattenGatewayV1Listeners(listener)
	}
	att["listeners"] = listeners

	if len(in.Addresses) > 0 {
		att["addresses"] = flattenGatewayV1Addresses(in.Addresses)
	}

	if in.Infrastructure != nil {
		att["infrastructure"] = flattenGatewayV1Infrastructure(in.Infrastructure)
	}

	if in.AllowedListeners != nil {
		att["allowed_listeners"] = flattenAllowedListeners(in.AllowedListeners)
	}

	if in.TLS != nil {
		att["tls"] = flattenGatewayTLSConfig(in.TLS)
	}

	if in.DefaultScope != "" {
		att["default_scope"] = string(in.DefaultScope)
	}

	return []interface{}{att}
}

func flattenGatewayV1Listeners(in gatewayv1.Listener) map[string]interface{} {
	listener := make(map[string]interface{})

	listener["name"] = string(in.Name)

	if in.Hostname != nil {
		listener["hostname"] = string(*in.Hostname)
	}

	listener["port"] = in.Port
	listener["protocol"] = string(in.Protocol)

	if in.TLS != nil {
		listener["tls"] = flattenListenerTLSConfig(in.TLS)
	}

	if in.AllowedRoutes != nil {
		listener["allowed_routes"] = flattenAllowedRoutes(in.AllowedRoutes)
	}

	return listener
}

func flattenListenerTLSConfig(in *gatewayv1.ListenerTLSConfig) []interface{} {
	if in == nil {
		return nil
	}
	tls := make(map[string]interface{})

	if in.Mode != nil {
		tls["mode"] = string(*in.Mode)
	}

	if len(in.CertificateRefs) > 0 {
		certs := make([]interface{}, len(in.CertificateRefs))
		for i, ref := range in.CertificateRefs {
			certs[i] = flattenSecretObjectReference(ref)
		}
		tls["certificate_refs"] = certs
	}

	if len(in.Options) > 0 {
		opts := make(map[string]string)
		for k, v := range in.Options {
			opts[string(k)] = string(v)
		}
		tls["options"] = opts
	}

	return []interface{}{tls}
}

func flattenAllowedRoutes(in *gatewayv1.AllowedRoutes) []interface{} {
	if in == nil {
		return nil
	}
	ar := make(map[string]interface{})

	if in.Namespaces != nil {
		ar["namespaces"] = flattenRouteNamespaces(in.Namespaces)
	}

	if len(in.Kinds) > 0 {
		ar["kinds"] = flattenRouteGroupKinds(in.Kinds)
	}

	return []interface{}{ar}
}

func flattenRouteNamespaces(in *gatewayv1.RouteNamespaces) []interface{} {
	if in == nil {
		return nil
	}
	ns := make(map[string]interface{})

	if in.From != nil {
		ns["from"] = string(*in.From)
	}

	if in.Selector != nil {
		ns["selector"] = flattenLabelSelectorGateway(in.Selector)
	}

	return []interface{}{ns}
}

func flattenRouteGroupKinds(in []gatewayv1.RouteGroupKind) []interface{} {
	result := make([]interface{}, len(in))
	for i, rgk := range in {
		m := make(map[string]interface{})
		if rgk.Group != nil {
			m["group"] = string(*rgk.Group)
		}
		m["kind"] = string(rgk.Kind)
		result[i] = m
	}
	return result
}

func flattenSecretObjectReference(in gatewayv1.SecretObjectReference) map[string]interface{} {
	att := make(map[string]interface{})

	if in.Group != nil {
		att["group"] = string(*in.Group)
	}

	if in.Kind != nil {
		att["kind"] = string(*in.Kind)
	}

	att["name"] = string(in.Name)

	if in.Namespace != nil {
		att["namespace"] = string(*in.Namespace)
	}

	return att
}

func flattenGatewayV1Addresses(in []gatewayv1.GatewaySpecAddress) []interface{} {
	result := make([]interface{}, len(in))
	for i, addr := range in {
		m := make(map[string]interface{})
		if addr.Type != nil {
			m["type"] = string(*addr.Type)
		}
		m["value"] = addr.Value
		result[i] = m
	}
	return result
}

func flattenGatewayV1Infrastructure(in *gatewayv1.GatewayInfrastructure) []interface{} {
	if in == nil {
		return nil
	}
	infra := make(map[string]interface{})

	if len(in.Labels) > 0 {
		labels := make(map[string]string)
		for k, v := range in.Labels {
			labels[string(k)] = string(v)
		}
		infra["labels"] = labels
	}

	if len(in.Annotations) > 0 {
		annotations := make(map[string]string)
		for k, v := range in.Annotations {
			annotations[string(k)] = string(v)
		}
		infra["annotations"] = annotations
	}

	if in.ParametersRef != nil {
		infra["parameters_ref"] = flattenLocalParametersReference(in.ParametersRef)
	}

	return []interface{}{infra}
}

func flattenLocalParametersReference(in *gatewayv1.LocalParametersReference) []interface{} {
	if in == nil {
		return nil
	}
	ref := make(map[string]interface{})
	ref["group"] = string(in.Group)
	ref["kind"] = string(in.Kind)
	ref["name"] = in.Name
	return []interface{}{ref}
}

func flattenAllowedListeners(in *gatewayv1.AllowedListeners) []interface{} {
	if in == nil {
		return nil
	}
	al := make(map[string]interface{})

	if in.Namespaces != nil {
		al["namespaces"] = flattenListenerNamespaces(in.Namespaces)
	}

	return []interface{}{al}
}

func flattenListenerNamespaces(in *gatewayv1.ListenerNamespaces) []interface{} {
	if in == nil {
		return nil
	}
	ns := make(map[string]interface{})

	if in.From != nil {
		ns["from"] = string(*in.From)
	}

	if in.Selector != nil {
		ns["selector"] = flattenLabelSelectorGateway(in.Selector)
	}

	return []interface{}{ns}
}

func flattenGatewayTLSConfig(in *gatewayv1.GatewayTLSConfig) []interface{} {
	if in == nil {
		return nil
	}
	cfg := make(map[string]interface{})

	if in.Backend != nil {
		cfg["backend"] = flattenGatewayBackendTLS(in.Backend)
	}

	if in.Frontend != nil {
		cfg["frontend"] = flattenFrontendTLSConfig(in.Frontend)
	}

	return []interface{}{cfg}
}

func flattenGatewayBackendTLS(in *gatewayv1.GatewayBackendTLS) []interface{} {
	if in == nil {
		return nil
	}
	backend := make(map[string]interface{})

	if in.ClientCertificateRef != nil {
		backend["client_certificate_ref"] = flattenSecretObjectReference(*in.ClientCertificateRef)
	}

	return []interface{}{backend}
}

func flattenFrontendTLSConfig(in *gatewayv1.FrontendTLSConfig) []interface{} {
	if in == nil {
		return nil
	}
	frontend := make(map[string]interface{})

	frontend["default"] = flattenTLSConfig(&in.Default)

	if len(in.PerPort) > 0 {
		ports := make([]interface{}, len(in.PerPort))
		for i, pp := range in.PerPort {
			m := make(map[string]interface{})
			m["port"] = pp.Port
			m["tls"] = flattenTLSConfig(&pp.TLS)
			ports[i] = m
		}
		frontend["per_port"] = ports
	}

	return []interface{}{frontend}
}

func flattenTLSConfig(in *gatewayv1.TLSConfig) []interface{} {
	if in == nil {
		return nil
	}
	cfg := make(map[string]interface{})

	if in.Validation != nil {
		cfg["validation"] = flattenFrontendTLSValidation(in.Validation)
	}

	return []interface{}{cfg}
}

func flattenFrontendTLSValidation(in *gatewayv1.FrontendTLSValidation) []interface{} {
	if in == nil {
		return nil
	}
	cv := make(map[string]interface{})

	if len(in.CACertificateRefs) > 0 {
		refs := make([]interface{}, len(in.CACertificateRefs))
		for i, ref := range in.CACertificateRefs {
			m := make(map[string]interface{})
			if ref.Group != "" {
				m["group"] = string(ref.Group)
			}
			if ref.Kind != "" {
				m["kind"] = string(ref.Kind)
			}
			m["name"] = string(ref.Name)
			if ref.Namespace != nil && *ref.Namespace != "" {
				m["namespace"] = string(*ref.Namespace)
			}
			refs[i] = m
		}
		cv["ca_certificate_refs"] = refs
	}

	if in.Mode != "" {
		cv["mode"] = string(in.Mode)
	}

	return []interface{}{cv}
}

func flattenSecretObjectReferences(in []gatewayv1.SecretObjectReference) []interface{} {
	result := make([]interface{}, len(in))
	for i, ref := range in {
		result[i] = flattenSecretObjectReference(ref)
	}
	return result
}

func flattenAnnotationValueMap(in map[gatewayv1.AnnotationKey]gatewayv1.AnnotationValue) map[string]string {
	if in == nil {
		return nil
	}
	result := make(map[string]string)
	for k, v := range in {
		result[string(k)] = string(v)
	}
	return result
}

func flattenGatewayV1Status(in gatewayv1.GatewayStatus) []interface{} {
	status := make(map[string]interface{})

	if len(in.Addresses) > 0 {
		status["addresses"] = flattenGatewayV1StatusAddresses(in.Addresses)
	}

	if len(in.Conditions) > 0 {
		status["conditions"] = flattenGatewayV1Conditions(in.Conditions)
	}

	if len(in.Listeners) > 0 {
		status["listeners"] = flattenGatewayV1ListenersStatus(in.Listeners)
	}

	if in.AttachedListenerSets != nil {
		status["attached_listener_sets"] = *in.AttachedListenerSets
	}

	return []interface{}{status}
}

func flattenGatewayV1StatusAddresses(in []gatewayv1.GatewayStatusAddress) []interface{} {
	result := make([]interface{}, len(in))
	for i, addr := range in {
		m := make(map[string]interface{})
		if addr.Type != nil {
			m["type"] = string(*addr.Type)
		}
		m["value"] = addr.Value
		result[i] = m
	}
	return result
}

func flattenGatewayV1ListenersStatus(in []gatewayv1.ListenerStatus) []interface{} {
	result := make([]interface{}, len(in))
	for i, ls := range in {
		m := make(map[string]interface{})
		m["name"] = string(ls.Name)
		if len(ls.SupportedKinds) > 0 {
			m["supported_kinds"] = flattenRouteGroupKind(ls.SupportedKinds)
		}
		m["attached_routes"] = ls.AttachedRoutes
		m["conditions"] = flattenGatewayV1Conditions(ls.Conditions)
		result[i] = m
	}
	return result
}

func flattenRouteGroupKind(in []gatewayv1.RouteGroupKind) []interface{} {
	result := make([]interface{}, len(in))
	for i, rg := range in {
		m := make(map[string]interface{})
		if rg.Group != nil {
			m["group"] = string(*rg.Group)
		}
		m["kind"] = string(rg.Kind)
		result[i] = m
	}
	return result
}

func flattenGatewayV1Conditions(in []metav1.Condition) []interface{} {
	att := make([]interface{}, len(in))
	for i, c := range in {
		m := make(map[string]interface{})
		m["type"] = c.Type
		m["status"] = string(c.Status)
		m["message"] = c.Message
		m["reason"] = c.Reason
		if !c.LastTransitionTime.IsZero() {
			m["last_transition_time"] = c.LastTransitionTime.String()
		}
		m["observed_generation"] = c.ObservedGeneration
		att[i] = m
	}
	return att
}

func flattenLabelSelectorGateway(in *metav1.LabelSelector) []interface{} {
	att := make(map[string]interface{})
	if len(in.MatchLabels) > 0 {
		att["match_labels"] = in.MatchLabels
	}
	if len(in.MatchExpressions) > 0 {
		att["match_expressions"] = flattenLabelSelectorRequirementGateway(in.MatchExpressions)
	}
	return []interface{}{att}
}

func flattenLabelSelectorRequirementGateway(in []metav1.LabelSelectorRequirement) []interface{} {
	att := make([]interface{}, len(in))
	for i, n := range in {
		m := make(map[string]interface{})
		m["key"] = n.Key
		m["operator"] = string(n.Operator)
		m["values"] = n.Values
		att[i] = m
	}
	return att
}

// Expanders

func expandGatewayV1Spec(l []interface{}) (*gatewayv1.GatewaySpec, error) {
	if len(l) == 0 || l[0] == nil {
		return &gatewayv1.GatewaySpec{}, nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.GatewaySpec{}

	if v, ok := in["gateway_class_name"].(string); ok && v != "" {
		obj.GatewayClassName = gatewayv1.ObjectName(v)
	}

	if v, ok := in["listeners"].([]interface{}); ok && len(v) > 0 {
		listeners, err := expandGatewayV1Listeners(v)
		if err != nil {
			return nil, err
		}
		obj.Listeners = listeners
	}

	if v, ok := in["addresses"].([]interface{}); ok && len(v) > 0 {
		obj.Addresses = expandGatewayV1Addresses(v)
	}

	if v, ok := in["infrastructure"].([]interface{}); ok && len(v) > 0 {
		obj.Infrastructure = expandGatewayV1Infrastructure(v)
	}

	if v, ok := in["allowed_listeners"].([]interface{}); ok && len(v) > 0 {
		obj.AllowedListeners = expandAllowedListeners(v)
	}

	if v, ok := in["tls"].([]interface{}); ok && len(v) > 0 {
		obj.TLS = expandGatewayTLSConfig(v)
	}

	if v, ok := in["default_scope"].(string); ok && v != "" {
		obj.DefaultScope = gatewayv1.GatewayDefaultScope(v)
	}

	return obj, nil
}

func expandGatewayV1Listeners(l []interface{}) ([]gatewayv1.Listener, error) {
	if len(l) == 0 {
		return []gatewayv1.Listener{}, nil
	}

	result := make([]gatewayv1.Listener, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		in := item.(map[string]interface{})
		listener, err := expandGatewayV1Listener(in)
		if err != nil {
			return nil, err
		}
		result[i] = *listener
	}
	return result, nil
}

func expandGatewayV1Listener(in map[string]interface{}) (*gatewayv1.Listener, error) {
	obj := &gatewayv1.Listener{}

	if v, ok := in["name"].(string); ok && v != "" {
		obj.Name = gatewayv1.SectionName(v)
	}

	if v, ok := in["hostname"].(string); ok && v != "" {
		hostname := gatewayv1.Hostname(v)
		obj.Hostname = &hostname
	}

	if v, ok := in["port"].(int); ok && v > 0 {
		obj.Port = gatewayv1.PortNumber(v)
	}

	if v, ok := in["protocol"].(string); ok && v != "" {
		obj.Protocol = gatewayv1.ProtocolType(v)
	}

	if v, ok := in["tls"].([]interface{}); ok && len(v) > 0 {
		obj.TLS = expandListenerTLSConfig(v)
	}

	if v, ok := in["allowed_routes"].([]interface{}); ok && len(v) > 0 {
		obj.AllowedRoutes = expandAllowedRoutes(v)
	}

	return obj, nil
}

func expandListenerTLSConfig(l []interface{}) *gatewayv1.ListenerTLSConfig {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.ListenerTLSConfig{}

	if v, ok := in["mode"].(string); ok && v != "" {
		mode := gatewayv1.TLSModeType(v)
		obj.Mode = &mode
	}

	if v, ok := in["certificate_refs"].([]interface{}); ok && len(v) > 0 {
		obj.CertificateRefs = expandSecretObjectReferences(v)
	}

	if v, ok := in["options"].(map[string]interface{}); ok && len(v) > 0 {
		obj.Options = expandAnnotationValueMap(v)
	}

	return obj
}

func expandAllowedRoutes(l []interface{}) *gatewayv1.AllowedRoutes {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.AllowedRoutes{}

	if v, ok := in["namespaces"].([]interface{}); ok && len(v) > 0 {
		obj.Namespaces = expandRouteNamespaces(v)
	}

	if v, ok := in["kinds"].([]interface{}); ok && len(v) > 0 {
		obj.Kinds = expandRouteGroupKinds(v)
	}

	return obj
}

func expandRouteNamespaces(l []interface{}) *gatewayv1.RouteNamespaces {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.RouteNamespaces{}

	if v, ok := in["from"].(string); ok && v != "" {
		from := gatewayv1.FromNamespaces(v)
		obj.From = &from
	}

	if v, ok := in["selector"].([]interface{}); ok && len(v) > 0 {
		obj.Selector = expandLabelSelectorGateway(v)
	}

	return obj
}

func expandRouteGroupKinds(l []interface{}) []gatewayv1.RouteGroupKind {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.RouteGroupKind, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		in := item.(map[string]interface{})
		rgk := gatewayv1.RouteGroupKind{}

		if v, ok := in["group"].(string); ok && v != "" {
			group := gatewayv1.Group(v)
			rgk.Group = &group
		}

		if v, ok := in["kind"].(string); ok && v != "" {
			rgk.Kind = gatewayv1.Kind(v)
		}

		result[i] = rgk
	}
	return result
}

func expandGatewayV1Addresses(l []interface{}) []gatewayv1.GatewaySpecAddress {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.GatewaySpecAddress, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		in := item.(map[string]interface{})
		addr := gatewayv1.GatewaySpecAddress{}

		if v, ok := in["type"].(string); ok && v != "" {
			t := gatewayv1.AddressType(v)
			addr.Type = &t
		}

		if v, ok := in["value"].(string); ok {
			addr.Value = v
		}

		result[i] = addr
	}
	return result
}

func expandGatewayV1Infrastructure(l []interface{}) *gatewayv1.GatewayInfrastructure {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.GatewayInfrastructure{}

	if v, ok := in["labels"].(map[string]interface{}); ok && len(v) > 0 {
		obj.Labels = expandLabelMap(v)
	}

	if v, ok := in["annotations"].(map[string]interface{}); ok && len(v) > 0 {
		obj.Annotations = expandAnnotationMap(v)
	}

	if v, ok := in["parameters_ref"].([]interface{}); ok && len(v) > 0 {
		obj.ParametersRef = expandLocalParametersReference(v)
	}

	return obj
}

func expandLocalParametersReference(l []interface{}) *gatewayv1.LocalParametersReference {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.LocalParametersReference{}

	if v, ok := in["group"].(string); ok && v != "" {
		obj.Group = gatewayv1.Group(v)
	}

	if v, ok := in["kind"].(string); ok && v != "" {
		obj.Kind = gatewayv1.Kind(v)
	}

	if v, ok := in["name"].(string); ok && v != "" {
		obj.Name = v
	}

	return obj
}

func expandLabelMap(m map[string]interface{}) map[gatewayv1.LabelKey]gatewayv1.LabelValue {
	if m == nil {
		return nil
	}
	result := make(map[gatewayv1.LabelKey]gatewayv1.LabelValue)
	for k, v := range m {
		result[gatewayv1.LabelKey(k)] = gatewayv1.LabelValue(v.(string))
	}
	return result
}

func expandAnnotationMap(m map[string]interface{}) map[gatewayv1.AnnotationKey]gatewayv1.AnnotationValue {
	if m == nil {
		return nil
	}
	result := make(map[gatewayv1.AnnotationKey]gatewayv1.AnnotationValue)
	for k, v := range m {
		result[gatewayv1.AnnotationKey(k)] = gatewayv1.AnnotationValue(v.(string))
	}
	return result
}

func expandAllowedListeners(l []interface{}) *gatewayv1.AllowedListeners {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.AllowedListeners{}

	if v, ok := in["namespaces"].([]interface{}); ok && len(v) > 0 {
		obj.Namespaces = expandListenerNamespaces(v)
	}

	return obj
}

func expandListenerNamespaces(l []interface{}) *gatewayv1.ListenerNamespaces {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.ListenerNamespaces{}

	if v, ok := in["from"].(string); ok && v != "" {
		from := gatewayv1.FromNamespaces(v)
		obj.From = &from
	}

	if v, ok := in["selector"].([]interface{}); ok && len(v) > 0 {
		obj.Selector = expandLabelSelectorGateway(v)
	}

	return obj
}

func expandGatewayTLSConfig(l []interface{}) *gatewayv1.GatewayTLSConfig {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.GatewayTLSConfig{}

	if v, ok := in["backend"].([]interface{}); ok && len(v) > 0 {
		obj.Backend = expandGatewayBackendTLS(v)
	}

	if v, ok := in["frontend"].([]interface{}); ok && len(v) > 0 {
		obj.Frontend = expandFrontendTLSConfig(v)
	}

	return obj
}

func expandGatewayBackendTLS(l []interface{}) *gatewayv1.GatewayBackendTLS {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.GatewayBackendTLS{}

	if v, ok := in["client_certificate_ref"].([]interface{}); ok && len(v) > 0 {
		refs := expandSecretObjectReferences(v)
		if len(refs) > 0 {
			obj.ClientCertificateRef = &refs[0]
		}
	}

	return obj
}

func expandFrontendTLSConfig(l []interface{}) *gatewayv1.FrontendTLSConfig {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.FrontendTLSConfig{}

	if v, ok := in["default"].([]interface{}); ok && len(v) > 0 {
		obj.Default = *expandTLSConfig(v)
	}

	if v, ok := in["per_port"].([]interface{}); ok && len(v) > 0 {
		obj.PerPort = expandTLSPortConfigs(v)
	}

	return obj
}

func expandTLSConfig(l []interface{}) *gatewayv1.TLSConfig {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.TLSConfig{}

	if v, ok := in["validation"].([]interface{}); ok && len(v) > 0 {
		obj.Validation = expandFrontendTLSValidation(v)
	}

	return obj
}

func expandFrontendTLSValidation(l []interface{}) *gatewayv1.FrontendTLSValidation {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.FrontendTLSValidation{}

	if v, ok := in["ca_certificate_refs"].([]interface{}); ok && len(v) > 0 {
		refs := make([]gatewayv1.ObjectReference, len(v))
		for i, item := range v {
			if item == nil {
				continue
			}
			itemMap := item.(map[string]interface{})
			ref := gatewayv1.ObjectReference{}
			if g, ok := itemMap["group"].(string); ok && g != "" {
				ref.Group = gatewayv1.Group(g)
			}
			if k, ok := itemMap["kind"].(string); ok && k != "" {
				ref.Kind = gatewayv1.Kind(k)
			}
			if n, ok := itemMap["name"].(string); ok && n != "" {
				ref.Name = gatewayv1.ObjectName(n)
			}
			if ns, ok := itemMap["namespace"].(string); ok && ns != "" {
				namespace := gatewayv1.Namespace(ns)
				ref.Namespace = &namespace
			}
			refs[i] = ref
		}
		obj.CACertificateRefs = refs
	}

	if v, ok := in["mode"].(string); ok && v != "" {
		obj.Mode = gatewayv1.FrontendValidationModeType(v)
	}

	return obj
}

func expandSecretObjectReferences(l []interface{}) []gatewayv1.SecretObjectReference {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.SecretObjectReference, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		in := item.(map[string]interface{})
		ref := gatewayv1.SecretObjectReference{}

		if v, ok := in["group"].(string); ok && v != "" {
			group := gatewayv1.Group(v)
			ref.Group = &group
		}

		if v, ok := in["kind"].(string); ok && v != "" {
			kind := gatewayv1.Kind(v)
			ref.Kind = &kind
		}

		if v, ok := in["name"].(string); ok && v != "" {
			ref.Name = gatewayv1.ObjectName(v)
		}

		if v, ok := in["namespace"].(string); ok && v != "" {
			ns := gatewayv1.Namespace(v)
			ref.Namespace = &ns
		}

		result[i] = ref
	}
	return result
}

func expandAnnotationValueMap(m map[string]interface{}) map[gatewayv1.AnnotationKey]gatewayv1.AnnotationValue {
	if m == nil {
		return nil
	}
	result := make(map[gatewayv1.AnnotationKey]gatewayv1.AnnotationValue)
	for k, v := range m {
		result[gatewayv1.AnnotationKey(k)] = gatewayv1.AnnotationValue(v.(string))
	}
	return result
}

func expandTLSPortConfigs(l []interface{}) []gatewayv1.TLSPortConfig {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.TLSPortConfig, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		in := item.(map[string]interface{})
		cfg := gatewayv1.TLSPortConfig{}

		if v, ok := in["port"].(int); ok && v > 0 {
			cfg.Port = gatewayv1.PortNumber(v)
		}

		if v, ok := in["tls"].([]interface{}); ok && len(v) > 0 {
			tls := expandTLSConfig(v)
			if tls != nil {
				cfg.TLS = *tls
			}
		}

		result[i] = cfg
	}
	return result
}

func expandLabelSelectorGateway(l []interface{}) *metav1.LabelSelector {
	if len(l) == 0 || l[0] == nil {
		return &metav1.LabelSelector{}
	}
	in := l[0].(map[string]interface{})
	obj := &metav1.LabelSelector{}
	if v, ok := in["match_labels"].(map[string]interface{}); ok && len(v) > 0 {
		obj.MatchLabels = expandStringMap(v)
	}
	if v, ok := in["match_expressions"].([]interface{}); ok && len(v) > 0 {
		obj.MatchExpressions = expandLabelSelectorRequirementGateway(v)
	}
	return obj
}

func expandLabelSelectorRequirementGateway(l []interface{}) []metav1.LabelSelectorRequirement {
	if len(l) == 0 || l[0] == nil {
		return []metav1.LabelSelectorRequirement{}
	}
	obj := make([]metav1.LabelSelectorRequirement, len(l))
	for i, n := range l {
		in := n.(map[string]interface{})
		obj[i] = metav1.LabelSelectorRequirement{
			Key:      in["key"].(string),
			Operator: metav1.LabelSelectorOperator(in["operator"].(string)),
			Values:   sliceOfString(in["values"].([]interface{})),
		}
	}
	return obj
}
