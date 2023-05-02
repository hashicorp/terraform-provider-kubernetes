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
			r.Name = v
		}
		if v, ok := portCfg["port"].(int); ok {
			r.Port = int32(v)
		}
		if v, ok := portCfg["protocol"].(string); ok {
			r.Protocol = api.Protocol(v)
		}
		ports[i] = r
	}
	return ports
}

func flattenEndpointsAddresses(in []api.EndpointAddress) *schema.Set {
	att := make([]interface{}, len(in), len(in))
	for i, n := range in {
		m := make(map[string]interface{})
		if n.Hostname != "" {
			m["hostname"] = n.Hostname
		}
		m["ip"] = n.IP
		if n.NodeName != nil {
			m["node_name"] = *n.NodeName
		}
		att[i] = m
	}
	return schema.NewSet(hashEndpointsSubsetAddress(), att)
}

func flattenEndpointsPorts(in []api.EndpointPort) *schema.Set {
	att := make([]interface{}, len(in), len(in))
	for i, n := range in {
		m := make(map[string]interface{})
		if n.Name != "" {
			m["name"] = n.Name
		}
		m["port"] = int(n.Port)
		m["protocol"] = string(n.Protocol)
		att[i] = m
	}
	return schema.NewSet(hashEndpointsSubsetPort(), att)
}
