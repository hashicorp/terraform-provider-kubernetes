// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func expandGatewayClassV1Spec(l []interface{}) gatewayv1.GatewayClassSpec {
	if len(l) == 0 || l[0] == nil {
		return gatewayv1.GatewayClassSpec{}
	}
	in := l[0].(map[string]interface{})
	obj := gatewayv1.GatewayClassSpec{}

	if v, ok := in["controller_name"].(string); ok && v != "" {
		obj.ControllerName = gatewayv1.GatewayController(v)
	}

	if v, ok := in["description"].(string); ok && v != "" {
		obj.Description = &v
	}

	if v, ok := in["parameters_ref"].([]interface{}); ok && len(v) > 0 {
		obj.ParametersRef = expandGatewayClassV1ParametersRef(v)
	}

	return obj
}

func expandGatewayClassV1ParametersRef(l []interface{}) *gatewayv1.ParametersReference {
	if len(l) == 0 || l[0] == nil {
		return nil
	}
	in := l[0].(map[string]interface{})
	obj := &gatewayv1.ParametersReference{}

	if v, ok := in["group"].(string); ok && v != "" {
		obj.Group = gatewayv1.Group(v)
	}

	if v, ok := in["kind"].(string); ok && v != "" {
		obj.Kind = gatewayv1.Kind(v)
	}

	if v, ok := in["name"].(string); ok && v != "" {
		obj.Name = v
	}

	if v, ok := in["namespace"].(string); ok && v != "" {
		ns := gatewayv1.Namespace(v)
		obj.Namespace = &ns
	}

	return obj
}

func flattenGatewayClassV1Spec(in gatewayv1.GatewayClassSpec) []interface{} {
	att := make(map[string]interface{})

	if in.ControllerName != "" {
		att["controller_name"] = string(in.ControllerName)
	}

	if in.Description != nil {
		att["description"] = *in.Description
	}

	if in.ParametersRef != nil {
		att["parameters_ref"] = flattenGatewayClassV1ParametersRef(in.ParametersRef)
	}

	return []interface{}{att}
}

func flattenGatewayClassV1ParametersRef(in *gatewayv1.ParametersReference) []interface{} {
	if in == nil {
		return nil
	}
	att := make(map[string]interface{})

	att["group"] = string(in.Group)
	att["kind"] = string(in.Kind)
	att["name"] = in.Name

	if in.Namespace != nil {
		att["namespace"] = string(*in.Namespace)
	}

	return []interface{}{att}
}

func flattenGatewayClassV1Status(in gatewayv1.GatewayClassStatus) []interface{} {
	att := make(map[string]interface{})

	att["conditions"] = flattenGatewayClassV1Conditions(in.Conditions)

	if len(in.SupportedFeatures) > 0 {
		att["supported_features"] = flattenGatewayClassV1SupportedFeatures(in.SupportedFeatures)
	}

	return []interface{}{att}
}

func flattenGatewayClassV1Conditions(in []metav1.Condition) []interface{} {
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

func flattenGatewayClassV1SupportedFeatures(in []gatewayv1.SupportedFeature) []interface{} {
	att := make([]interface{}, len(in))
	for i, f := range in {
		att[i] = string(f.Name)
	}
	return att
}
