// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	v1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"k8s.io/utils/ptr"
)

// Flatteners

func flattenAPIServiceV1Spec(in v1.APIServiceSpec) []interface{} {
	att := make(map[string]interface{})

	att["ca_bundle"] = string(in.CABundle)
	att["group"] = in.Group
	att["group_priority_minimum"] = in.GroupPriorityMinimum
	att["insecure_skip_tls_verify"] = in.InsecureSkipTLSVerify

	if in.Service != nil {
		m := make(map[string]interface{})
		m["name"] = in.Service.Name
		m["namespace"] = in.Service.Namespace
		if in.Service.Port != nil {
			m["port"] = *in.Service.Port
		}
		att["service"] = []interface{}{m}
	}

	att["version"] = in.Version
	att["version_priority"] = in.VersionPriority

	return []interface{}{att}
}

// Expanders

func expandAPIServiceV1Spec(l []interface{}) v1.APIServiceSpec {
	if len(l) == 0 || l[0] == nil {
		return v1.APIServiceSpec{}
	}
	in := l[0].(map[string]interface{})
	obj := v1.APIServiceSpec{}

	if v, ok := in["ca_bundle"].(string); ok {
		obj.CABundle = []byte(v)
	}
	if v, ok := in["group"].(string); ok {
		obj.Group = v
	}
	if v, ok := in["group_priority_minimum"].(int); ok {
		obj.GroupPriorityMinimum = int32(v)
	}
	if v, ok := in["insecure_skip_tls_verify"].(bool); ok {
		obj.InsecureSkipTLSVerify = v
	}
	if v, ok := in["service"].([]interface{}); ok && len(v) > 0 {
		m := v[0].(map[string]interface{})
		obj.Service = &v1.ServiceReference{
			Name:      m["name"].(string),
			Namespace: m["namespace"].(string),
		}

		if v, ok := m["port"].(int); ok && v > 0 {
			obj.Service.Port = ptr.To(int32(v))
		}
	}
	if v, ok := in["version"].(string); ok {
		obj.Version = v
	}
	if v, ok := in["version_priority"].(int); ok {
		obj.VersionPriority = int32(v)
	}

	return obj
}
