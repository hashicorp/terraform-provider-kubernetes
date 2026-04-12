// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func flattenReferenceGrantSpec(in gatewayv1.ReferenceGrantSpec) []interface{} {
	att := make(map[string]interface{})

	if len(in.From) > 0 {
		from := make([]interface{}, len(in.From))
		for i, f := range in.From {
			from[i] = flattenReferenceGrantFrom(f)
		}
		att["from"] = from
	}

	if len(in.To) > 0 {
		to := make([]interface{}, len(in.To))
		for i, t := range in.To {
			to[i] = flattenReferenceGrantTo(t)
		}
		att["to"] = to
	}

	return []interface{}{att}
}

func flattenReferenceGrantFrom(in gatewayv1.ReferenceGrantFrom) map[string]interface{} {
	ref := make(map[string]interface{})

	ref["group"] = string(in.Group)
	ref["kind"] = string(in.Kind)
	ref["namespace"] = string(in.Namespace)

	return ref
}

func flattenReferenceGrantTo(in gatewayv1.ReferenceGrantTo) map[string]interface{} {
	ref := make(map[string]interface{})

	ref["group"] = string(in.Group)
	ref["kind"] = string(in.Kind)
	if in.Name != nil {
		ref["name"] = string(*in.Name)
	}

	return ref
}

func expandReferenceGrantSpec(l []interface{}) gatewayv1.ReferenceGrantSpec {
	if len(l) == 0 || l[0] == nil {
		return gatewayv1.ReferenceGrantSpec{}
	}

	in := l[0].(map[string]interface{})
	obj := gatewayv1.ReferenceGrantSpec{}

	if v, ok := in["from"].([]interface{}); ok && len(v) > 0 {
		obj.From = expandReferenceGrantFrom(v)
	}

	if v, ok := in["to"].([]interface{}); ok && len(v) > 0 {
		obj.To = expandReferenceGrantTo(v)
	}

	return obj
}

func expandReferenceGrantFrom(l []interface{}) []gatewayv1.ReferenceGrantFrom {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.ReferenceGrantFrom, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		in := item.(map[string]interface{})
		group, _ := in["group"].(string)
		kind, _ := in["kind"].(string)
		namespace, _ := in["namespace"].(string)
		result[i] = gatewayv1.ReferenceGrantFrom{
			Group:     gatewayv1.Group(group),
			Kind:      gatewayv1.Kind(kind),
			Namespace: gatewayv1.Namespace(namespace),
		}
	}
	return result
}

func expandReferenceGrantTo(l []interface{}) []gatewayv1.ReferenceGrantTo {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.ReferenceGrantTo, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		in := item.(map[string]interface{})
		group, _ := in["group"].(string)
		kind, _ := in["kind"].(string)
		to := gatewayv1.ReferenceGrantTo{
			Group: gatewayv1.Group(group),
			Kind:  gatewayv1.Kind(kind),
		}

		if v, ok := in["name"].(string); ok && v != "" {
			name := gatewayv1.ObjectName(v)
			to.Name = &name
		}

		result[i] = to
	}
	return result
}
