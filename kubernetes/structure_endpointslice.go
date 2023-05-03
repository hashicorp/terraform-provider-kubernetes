// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	v1 "k8s.io/api/core/v1"
	api "k8s.io/api/discovery/v1"
)

func expandEndpointSliceEndpoints(in *schema.Set) []api.Endpoint {
	if in == nil || in.Len() == 0 {
		return []api.Endpoint{}
	}
	endpoints := make([]api.Endpoint, in.Len())
	for i, endpoint := range in.List() {
		r := api.Endpoint{}
		endpointConfig := endpoint.(map[string]interface{})
		if v, ok := endpointConfig["addresses"].([]string); ok {
			r.Addresses = v
		}
		if v, ok := endpointConfig["conditions"].(api.EndpointConditions); ok {
			r.Conditions = v
		}
		if v, ok := endpointConfig["hostname"].(string); ok && v != "" {
			r.Hostname = ptrToString(v)
		}
		if v, ok := endpointConfig["node_name"].(string); ok && v != "" {
			r.NodeName = ptrToString(v)
		}
		if v, ok := endpointConfig["target_ref"].(v1.ObjectReference); ok {
			r.TargetRef = &v
		}
		if v, ok := endpointConfig["zone"].(string); ok && v != "" {
			r.Zone = ptrToString(v)
		}

		endpoints[i] = r
	}
	return endpoints
}

func expandEndpointSlicePorts(in *schema.Set) []api.EndpointPort {
	if in == nil || in.Len() == 0 {
		return []api.EndpointPort{}
	}
	ports := make([]api.EndpointPort, in.Len())
	for i, port := range in.List() {
		r := api.EndpointPort{}
		portCfg := port.(map[string]interface{})
		if v, ok := portCfg["name"].(string); ok {
			r.Name = ptrToString(v)
		}
		if v, ok := portCfg["port"].(int32); ok {
			r.Port = &v
		}
		if v, ok := portCfg["protocol"].(v1.Protocol); ok {
			r.Protocol = &v
		}
		if v, ok := portCfg["app_protocol"].(string); ok {
			r.AppProtocol = ptrToString(v)
		}
		ports[i] = r
	}
	return ports
}

func flattenEndpointSliceEndpoints(in []api.Endpoint) *schema.Set {
	att := make([]interface{}, len(in), len(in))
	for i, e := range in {
		m := make(map[string]interface{})
		if *e.Hostname != "" {
			m["hostname"] = e.Hostname
		}
		if *e.NodeName != "" {
			m["node_name"] = e.NodeName
		}
		if *e.Zone != "" {
			m["zone"] = e.Zone
		}
		if len(e.Addresses) != 0 {
			m["addresses"] = e.Addresses
		}
		if e.TargetRef != nil {
			m["target_ref"] = e.TargetRef
		}
		if &e.Conditions != nil {
			m["hostname"] = e.Hostname
		}
		att[i] = m
	}
	return schema.NewSet(hashEndpointSliceEndpoints(), att)
}

func flattenEndpointSlicePorts(in []api.EndpointPort) *schema.Set {
	att := make([]interface{}, len(in), len(in))
	for i, e := range in {
		m := make(map[string]interface{})
		if *e.Name != "" {
			m["name"] = e.Name
		}
		if e.Port != nil {
			m["port"] = int(*e.Port)
		}
		if e.Protocol != nil {
			m["protocol"] = string(*e.Protocol)
		}
		if e.AppProtocol != nil {
			m["app_protocol"] = string(*e.AppProtocol)
		}
		att[i] = m
	}
	return schema.NewSet(hashEndpointSlicePorts(), att)
}
