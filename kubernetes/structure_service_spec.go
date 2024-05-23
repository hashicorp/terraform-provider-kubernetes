// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	gversion "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

// Flatteners

func flattenServicePort(in []v1.ServicePort) []interface{} {
	att := make([]interface{}, len(in))
	for i, n := range in {
		m := make(map[string]interface{})
		m["app_protocol"] = n.AppProtocol
		m["name"] = n.Name
		m["protocol"] = string(n.Protocol)
		m["port"] = int(n.Port)
		m["target_port"] = n.TargetPort.String()
		m["node_port"] = int(n.NodePort)

		att[i] = m
	}
	return att
}

func flattenIPFamilies(in []v1.IPFamily) []interface{} {
	att := make([]interface{}, len(in))
	for i, n := range in {
		att[i] = string(n)
	}
	return att
}

func flattenSessionAffinityConfigClientIP(in v1.ClientIPConfig) []interface{} {
	return []interface{}{
		map[string]interface{}{
			"timeout_seconds": in.TimeoutSeconds,
		},
	}
}

func flattenSessionAffinityConfig(in v1.SessionAffinityConfig) []interface{} {
	return []interface{}{
		map[string]interface{}{
			"client_ip": flattenSessionAffinityConfigClientIP(*in.ClientIP),
		},
	}
}

func flattenServiceSpec(in v1.ServiceSpec) []interface{} {
	att := make(map[string]interface{})
	if len(in.Ports) > 0 {
		att["port"] = flattenServicePort(in.Ports)
	}
	if len(in.Selector) > 0 {
		att["selector"] = in.Selector
	}
	if in.ClusterIP != "" {
		att["cluster_ip"] = in.ClusterIP
	}
	if len(in.ClusterIPs) > 0 {
		att["cluster_ips"] = in.ClusterIPs
	}
	// Set 'allocate_load_balancer_node_ports' to 'true' to match with its default value
	// when it is not declared in the TF code. That helps to avoid plan diff when
	// service type is not 'LoadBalancer'.
	att["allocate_load_balancer_node_ports"] = true
	if in.Type != "" {
		att["type"] = string(in.Type)
		if in.Type == v1.ServiceTypeLoadBalancer {
			// spec.allocateLoadBalancerNodePorts may only be used when `type` is 'LoadBalancer'
			if in.AllocateLoadBalancerNodePorts != nil {
				att["allocate_load_balancer_node_ports"] = in.AllocateLoadBalancerNodePorts
			}
			// spec.loadBalancerClass may only be used when `type` is 'LoadBalancer'
			if in.LoadBalancerClass != nil {
				att["load_balancer_class"] = in.LoadBalancerClass
			}
		}
	}
	if len(in.ExternalIPs) > 0 {
		att["external_ips"] = newStringSet(schema.HashString, in.ExternalIPs)
	}
	if in.InternalTrafficPolicy != nil {
		att["internal_traffic_policy"] = in.InternalTrafficPolicy
	}
	if len(in.IPFamilies) > 0 {
		att["ip_families"] = flattenIPFamilies(in.IPFamilies)
	}
	if in.IPFamilyPolicy != nil {
		att["ip_family_policy"] = *in.IPFamilyPolicy
	}
	if in.SessionAffinity != "" {
		att["session_affinity"] = string(in.SessionAffinity)
	}
	if in.SessionAffinityConfig != nil {
		att["session_affinity_config"] = flattenSessionAffinityConfig(*in.SessionAffinityConfig)
	}
	if in.LoadBalancerIP != "" {
		att["load_balancer_ip"] = in.LoadBalancerIP
	}
	if len(in.LoadBalancerSourceRanges) > 0 {
		att["load_balancer_source_ranges"] = newStringSet(schema.HashString, in.LoadBalancerSourceRanges)
	}
	if in.ExternalName != "" {
		att["external_name"] = in.ExternalName
	}
	att["publish_not_ready_addresses"] = in.PublishNotReadyAddresses

	if in.ExternalTrafficPolicy != "" {
		att["external_traffic_policy"] = string(in.ExternalTrafficPolicy)
	}

	att["health_check_node_port"] = int(in.HealthCheckNodePort)

	return []interface{}{att}
}

func flattenLoadBalancerStatus(in v1.LoadBalancerStatus) []interface{} {
	out := make([]interface{}, len(in.Ingress))
	for i, ingress := range in.Ingress {
		att := make(map[string]interface{})

		att["ip"] = ingress.IP
		att["hostname"] = ingress.Hostname

		out[i] = att
	}

	return []interface{}{
		map[string][]interface{}{
			"ingress": out,
		},
	}
}

// Expanders

func expandServicePort(l []interface{}, removeNodePort bool) []v1.ServicePort {
	if len(l) == 0 || l[0] == nil {
		return []v1.ServicePort{}
	}
	obj := make([]v1.ServicePort, len(l))
	for i, n := range l {
		cfg := n.(map[string]interface{})
		obj[i] = v1.ServicePort{
			Port:       int32(cfg["port"].(int)),
			TargetPort: intstr.Parse(cfg["target_port"].(string)),
		}
		if v, ok := cfg["app_protocol"].(string); ok && v != "" {
			obj[i].AppProtocol = &v
		}
		if v, ok := cfg["name"].(string); ok {
			obj[i].Name = v
		}
		if v, ok := cfg["protocol"].(string); ok {
			obj[i].Protocol = v1.Protocol(v)
		}
		if v, ok := cfg["node_port"].(int); ok && !removeNodePort {
			obj[i].NodePort = int32(v)
		}
	}
	return obj
}

func expandIPFamilies(l []interface{}) []v1.IPFamily {
	if l[0] == nil {
		return []v1.IPFamily{}
	}
	obj := make([]v1.IPFamily, len(l))
	for i, n := range l {
		obj[i] = v1.IPFamily(n.(string))
	}
	return obj
}

func expandSessionAffinityConfigClientIP(l []interface{}) *v1.ClientIPConfig {
	obj := &v1.ClientIPConfig{}

	if len(l) == 0 || l[0] == nil {
		return obj
	}

	in := l[0].(map[string]interface{})
	if v, ok := in["timeout_seconds"].(int); ok {
		obj.TimeoutSeconds = ptr.To(int32(v))
	}

	return obj
}

func expandSessionAffinityConfig(l []interface{}) *v1.SessionAffinityConfig {
	if len(l) == 0 || l[0] == nil {
		return &v1.SessionAffinityConfig{}
	}

	in := l[0].(map[string]interface{})
	obj := &v1.SessionAffinityConfig{}

	if v, ok := in["client_ip"].([]interface{}); ok {
		obj.ClientIP = expandSessionAffinityConfigClientIP(v)
	}

	return obj
}

func expandServiceSpec(l []interface{}) v1.ServiceSpec {
	if len(l) == 0 || l[0] == nil {
		return v1.ServiceSpec{}
	}
	in := l[0].(map[string]interface{})
	obj := v1.ServiceSpec{}

	if v, ok := in["port"].([]interface{}); ok && len(v) > 0 {
		obj.Ports = expandServicePort(v, false)
	}
	if v, ok := in["selector"].(map[string]interface{}); ok && len(v) > 0 {
		obj.Selector = expandStringMap(v)
	}
	if v, ok := in["cluster_ip"].(string); ok {
		obj.ClusterIP = v
	}
	if v, ok := in["cluster_ips"].([]interface{}); ok && len(v) > 0 {
		obj.ClusterIPs = expandStringSlice(v)
	}
	if v, ok := in["type"].(string); ok {
		obj.Type = v1.ServiceType(v)

		if v == string(v1.ServiceTypeLoadBalancer) {
			// spec.allocateLoadBalancerNodePorts may only be used when `type` is 'LoadBalancer'
			if v, ok := in["allocate_load_balancer_node_ports"].(bool); ok {
				obj.AllocateLoadBalancerNodePorts = &v
			}
			// spec.loadBalancerClass may only be used when `type` is 'LoadBalancer'
			if v, ok := in["load_balancer_class"].(string); ok && v != "" {
				obj.LoadBalancerClass = &v
			}
		}
	}
	if v, ok := in["external_ips"].(*schema.Set); ok && v.Len() > 0 {
		obj.ExternalIPs = sliceOfString(v.List())
	}
	if v, ok := in["internal_traffic_policy"].(string); ok && v != "" {
		p := v1.ServiceInternalTrafficPolicyType(v)
		obj.InternalTrafficPolicy = &p
	}
	if v, ok := in["ip_families"].([]interface{}); ok && len(v) > 0 {
		obj.IPFamilies = expandIPFamilies(v)
	}
	if v, ok := in["ip_family_policy"].(string); ok && len(v) > 0 {
		p := v1.IPFamilyPolicyType(v)
		obj.IPFamilyPolicy = &p
	}
	if v, ok := in["session_affinity"].(string); ok {
		obj.SessionAffinity = v1.ServiceAffinity(v)
	}
	if v, ok := in["session_affinity_config"].([]interface{}); ok && len(v) > 0 {
		obj.SessionAffinityConfig = expandSessionAffinityConfig(v)
	}
	if v, ok := in["load_balancer_ip"].(string); ok {
		obj.LoadBalancerIP = v
	}
	if v, ok := in["load_balancer_source_ranges"].(*schema.Set); ok && v.Len() > 0 {
		obj.LoadBalancerSourceRanges = sliceOfString(v.List())
	}
	if v, ok := in["external_name"].(string); ok {
		obj.ExternalName = v
	}
	if v, ok := in["publish_not_ready_addresses"].(bool); ok {
		obj.PublishNotReadyAddresses = v
	}
	if v, ok := in["external_traffic_policy"].(string); ok {
		obj.ExternalTrafficPolicy = v1.ServiceExternalTrafficPolicyType(v)
	}
	if v, ok := in["health_check_node_port"].(int); ok {
		obj.HealthCheckNodePort = int32(v)
	}

	return obj
}

// Patch Ops

func patchServiceSpec(keyPrefix, pathPrefix string, d *schema.ResourceData, kv *gversion.Version) PatchOperations {
	ops := make([]PatchOperation, 0)

	if d.HasChange(keyPrefix + "allocate_load_balancer_node_ports") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "allocateLoadBalancerNodePorts",
			Value: d.Get(keyPrefix + "allocate_load_balancer_node_ports").(bool),
		})
	}

	if d.HasChange(keyPrefix + "selector") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "selector",
			Value: d.Get(keyPrefix + "selector").(map[string]interface{}),
		})
	}

	if d.HasChange(keyPrefix + "session_affinity") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "sessionAffinity",
			Value: d.Get(keyPrefix + "session_affinity").(string),
		})
	}
	if d.HasChange(keyPrefix + "session_affinity_config") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "sessionAffinityConfig",
			Value: expandSessionAffinityConfig(d.Get(keyPrefix + "session_affinity_config").([]interface{})),
		})
	}
	if d.HasChange(keyPrefix + "load_balancer_ip") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "loadBalancerIP",
			Value: d.Get(keyPrefix + "load_balancer_ip").(string),
		})
	}
	if d.HasChange(keyPrefix + "load_balancer_source_ranges") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "loadBalancerSourceRanges",
			Value: d.Get(keyPrefix + "load_balancer_source_ranges").(*schema.Set).List(),
		})
	}
	if d.HasChange(keyPrefix + "port") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "ports",
			Value: expandServicePort(d.Get(keyPrefix+"port").([]interface{}), false),
		})
	}
	if d.HasChange(keyPrefix + "type") {
		_, n := d.GetChange(keyPrefix + "type")

		if n.(string) == "ExternalName" {
			ops = append(ops, &RemoveOperation{
				Path: pathPrefix + "clusterIP",
			})
		}

		if n.(string) == "ClusterIP" {
			ops = append(ops, &ReplaceOperation{
				Path:  pathPrefix + "ports",
				Value: expandServicePort(d.Get(keyPrefix+"port").([]interface{}), true),
			})
		}

		if n.(string) == "LoadBalancer" {
			ops = append(ops, &ReplaceOperation{
				Path:  pathPrefix + "allocateLoadBalancerNodePorts",
				Value: d.Get(keyPrefix + "allocate_load_balancer_node_ports").(bool),
			})
		}

		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "type",
			Value: d.Get(keyPrefix + "type").(string),
		})
	}
	if d.HasChange(keyPrefix + "external_ips") {
		version, _ := gversion.NewVersion("1.8.0")
		if kv.LessThan(version) {
			// If we haven't done this the deprecated field would have priority
			ops = append(ops, &ReplaceOperation{
				Path:  pathPrefix + "deprecatedPublicIPs",
				Value: nil,
			})
		}

		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "externalIPs",
			Value: d.Get(keyPrefix + "external_ips").(*schema.Set).List(),
		})
	}
	if d.HasChange(keyPrefix + "external_name") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "externalName",
			Value: d.Get(keyPrefix + "external_name").(string),
		})
	}
	if d.HasChange(keyPrefix + "external_traffic_policy") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "externalTrafficPolicy",
			Value: d.Get(keyPrefix + "external_traffic_policy").(string),
		})
	}
	if d.HasChange(keyPrefix + "internal_traffic_policy") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "internalTrafficPolicy",
			Value: d.Get(keyPrefix + "internal_traffic_policy").(string),
		})
	}
	if d.HasChange(keyPrefix + "ip_families") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "ipFamilies",
			Value: expandIPFamilies(d.Get(keyPrefix + "ip_families").([]interface{})),
		})
	}
	if d.HasChange(keyPrefix + "ip_family_policy") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "ipFamilyPolicy",
			Value: d.Get(keyPrefix + "ip_family_policy").(string),
		})
	}
	if d.HasChange(keyPrefix + "publish_not_ready_addresses") {
		p := pathPrefix + "publishNotReadyAddresses"
		v := d.Get(keyPrefix + "publish_not_ready_addresses").(bool)
		if v {
			ops = append(ops, &AddOperation{
				Path:  p,
				Value: v,
			})
		} else {
			ops = append(ops, &RemoveOperation{
				Path: p,
			})
		}
	}
	if d.HasChange(keyPrefix + "health_check_node_port") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "healthCheckNodePort",
			Value: int32(d.Get(keyPrefix + "health_check_node_port").(int)),
		})
	}
	return ops
}
