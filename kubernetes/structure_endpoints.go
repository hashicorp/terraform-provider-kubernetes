package kubernetes

import (
	api "k8s.io/api/core/v1"
)

func expandEndpointsAddresses(in []interface{}) []api.EndpointAddress {
	if len(in) == 0 {
		return []api.EndpointAddress{}
	}
	addresses := make([]api.EndpointAddress, len(in))
	for i, addr := range in {
		r := api.EndpointAddress{}
		addrCfg := addr.(map[string]interface{})
		if v, ok := addrCfg["hostname"].(string); ok {
			r.Hostname = v
		}
		if v, ok := addrCfg["ip"].(string); ok {
			r.IP = v
		}
		if v, ok := addrCfg["node_name"].(string); ok && v != "" {
			r.NodeName = ptrToString(v)
		}
		addresses[i] = r
	}
	return addresses
}

func expandEndpointsPorts(in []interface{}) []api.EndpointPort {
	if len(in) == 0 {
		return []api.EndpointPort{}
	}
	ports := make([]api.EndpointPort, len(in))
	for i, port := range in {
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

func expandEndpointsSubsets(in []interface{}) []api.EndpointSubset {
	if len(in) == 0 {
		return []api.EndpointSubset{}
	}
	subsets := make([]api.EndpointSubset, len(in))
	for i, subset := range in {
		r := api.EndpointSubset{}
		subsetCfg := subset.(map[string]interface{})
		if v, ok := subsetCfg["address"].([]interface{}); ok {
			r.Addresses = expandEndpointsAddresses(v)
		}
		if v, ok := subsetCfg["not_ready_address"].([]interface{}); ok {
			r.NotReadyAddresses = expandEndpointsAddresses(v)
		}
		if v, ok := subsetCfg["port"]; ok {
			r.Ports = expandEndpointsPorts(v.([]interface{}))
		}
		subsets[i] = r
	}
	return subsets
}

func flattenEndpointsAddresses(in []api.EndpointAddress) []interface{} {
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
	return att
}

func flattenEndpointsPorts(in []api.EndpointPort) []interface{} {
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
	return att
}

func flattenEndpointsSubsets(in []api.EndpointSubset) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for i, n := range in {
		m := make(map[string]interface{})
		if len(n.Addresses) > 0 {
			m["address"] = flattenEndpointsAddresses(n.Addresses)
		}
		if len(n.NotReadyAddresses) > 0 {
			m["not_ready_address"] = flattenEndpointsAddresses(n.NotReadyAddresses)
		}
		if len(n.Ports) > 0 {
			m["port"] = flattenEndpointsPorts(n.Ports)
		}
		att[i] = m
	}
	return att
}
