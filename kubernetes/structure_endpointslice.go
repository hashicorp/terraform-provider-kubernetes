// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"strconv"

	v1 "k8s.io/api/core/v1"
	api "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

func expandEndpointSliceEndpoints(in []interface{}) []api.Endpoint {
	if len(in) == 0 {
		return []api.Endpoint{}
	}
	endpoints := make([]api.Endpoint, len(in))
	for i, endpoint := range in {
		r := api.Endpoint{}
		endpointConfig := endpoint.(map[string]interface{})
		if v := endpointConfig["addresses"].([]interface{}); len(v) != 0 {
			r.Addresses = expandStringSlice(v)
		}
		if v, ok := endpointConfig["condition"].([]interface{}); ok {
			r.Conditions = expandEndpointSliceConditions(v)
		}
		if v, ok := endpointConfig["hostname"].(string); ok && v != "" {
			r.Hostname = ptr.To(v)
		}
		if v, ok := endpointConfig["node_name"].(string); ok && v != "" {
			r.NodeName = ptr.To(v)
		}
		if v, ok := endpointConfig["target_ref"].([]interface{}); ok && len(v) != 0 {
			r.TargetRef = expandObjectReference(v)
		}
		if v, ok := endpointConfig["zone"].(string); ok && v != "" {
			r.Zone = ptr.To(v)
		}

		endpoints[i] = r
	}
	return endpoints
}

func expandObjectReference(l []interface{}) *v1.ObjectReference {
	if len(l) == 0 || l == nil {
		return &v1.ObjectReference{}
	}
	in := l[0].(map[string]interface{})
	obj := &v1.ObjectReference{}

	if v, ok := in["name"].(string); ok {
		obj.Name = v
	}
	if v, ok := in["namespace"].(string); ok {
		obj.Namespace = v
	}
	if v, ok := in["resource_version"].(string); ok {
		obj.ResourceVersion = v
	}
	if v, ok := in["uid"]; ok {
		obj.UID = types.UID(v.(string))
	}
	if v, ok := in["field_path"].(string); ok {
		obj.FieldPath = v
	}

	return obj
}

func expandEndpointSlicePorts(in []interface{}) []api.EndpointPort {
	if len(in) == 0 {
		return []api.EndpointPort{}
	}
	ports := make([]api.EndpointPort, len(in))
	for i, port := range in {
		r := api.EndpointPort{}
		portCfg := port.(map[string]interface{})
		if v, ok := portCfg["name"].(string); ok {
			r.Name = ptr.To(v)
		}
		if v, ok := portCfg["port"].(string); ok {
			if v == "" {
				continue
			}
			v, _ := strconv.ParseInt(v, 10, 32)
			r.Port = ptr.To(int32(v))
		}
		if v, ok := portCfg["protocol"].(v1.Protocol); ok {
			r.Protocol = &v
		}
		if v, ok := portCfg["app_protocol"].(string); ok {
			r.AppProtocol = ptr.To(v)
		}
		ports[i] = r
	}
	return ports
}

func expandEndpointSliceConditions(in []interface{}) api.EndpointConditions {
	obj := api.EndpointConditions{}

	if len(in) == 0 || in[0] == nil {
		return obj
	}
	cond := in[0].(map[string]interface{})

	if v, ok := cond["ready"].(bool); ok {
		obj.Ready = ptr.To(v)
	}
	if v, ok := cond["serving"].(bool); ok {
		obj.Serving = ptr.To(v)
	}
	if v, ok := cond["terminating"].(bool); ok {
		obj.Terminating = ptr.To(v)
	}

	return obj
}

func flattenEndpointSliceEndpoints(in []api.Endpoint) []interface{} {
	att := make([]interface{}, len(in))
	for i, e := range in {
		m := make(map[string]interface{})
		if len(e.Addresses) != 0 {
			m["addresses"] = e.Addresses
		}
		m["condition"] = flattenEndpointSliceConditions(e.Conditions)
		if e.Hostname != nil {
			m["hostname"] = e.Hostname
		}
		if e.NodeName != nil {
			m["node_name"] = e.NodeName
		}
		if e.TargetRef != nil {
			m["target_ref"] = flattenObjectReference(e.TargetRef)
		}
		if e.Zone != nil {
			m["zone"] = e.Zone
		}
		att[i] = m
	}

	return att
}

func flattenEndpointSliceConditions(in api.EndpointConditions) []interface{} {
	att := make(map[string]interface{})
	if in.Ready != nil {
		att["ready"] = in.Ready
	}
	if in.Serving != nil {
		att["serving"] = in.Serving
	}
	if in.Terminating != nil {
		att["terminating"] = in.Terminating
	}

	if len(att) > 0 {
		return []interface{}{att}
	}

	return []interface{}{}
}

func flattenEndpointSlicePorts(in []api.EndpointPort) []interface{} {
	att := make([]interface{}, len(in))
	for i, e := range in {
		m := make(map[string]interface{})
		if *e.Name != "" {
			m["name"] = e.Name
		}
		if e.Port != nil {
			m["port"] = strconv.Itoa(int(*e.Port))
		}
		if e.Protocol != nil {
			m["protocol"] = string(*e.Protocol)
		}
		if e.AppProtocol != nil {
			m["app_protocol"] = string(*e.AppProtocol)
		}
		att[i] = m
	}
	return att
}

func flattenObjectReference(in *v1.ObjectReference) []interface{} {
	att := make(map[string]interface{})
	if in.Name != "" {
		att["name"] = in.Name
	}
	if in.Name != "" {
		att["namespace"] = in.Namespace
	}
	if in.FieldPath != "" {
		att["field_path"] = in.FieldPath
	}
	if in.ResourceVersion != "" {
		att["resource_version"] = in.ResourceVersion
	}
	if in.UID != "" {
		att["uid"] = in.UID
	}

	return []interface{}{att}
}
