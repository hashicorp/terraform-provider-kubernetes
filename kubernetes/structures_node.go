// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	v1 "k8s.io/api/core/v1"
)

func flattenNodeSpec(in v1.NodeSpec) []interface{} {
	att := make(map[string]interface{})
	if in.PodCIDR != "" {
		att["pod_cidr"] = in.PodCIDR
	}
	if len(in.PodCIDRs) > 0 {
		att["pod_cidrs"] = in.PodCIDRs
	}
	if in.ProviderID != "" {
		att["provider_id"] = in.ProviderID
	}
	att["unschedulable"] = in.Unschedulable
	if len(in.Taints) > 0 {
		att["taints"] = flattenNodeTaints(in.Taints...)
	}
	return []interface{}{att}
}

func flattenAddresses(in ...v1.NodeAddress) []interface{} {
	out := make([]interface{}, len(in))
	for i, address := range in {
		m := make(map[string]interface{})
		m["address"] = address.Address
		m["type"] = address.Type
		out[i] = m
	}
	return out
}

func flattenNodeInfo(in v1.NodeSystemInfo) []interface{} {
	att := make(map[string]interface{})
	if in.MachineID != "" {
		att["machine_id"] = in.MachineID
	}
	if in.SystemUUID != "" {
		att["system_uuid"] = in.SystemUUID
	}
	if in.BootID != "" {
		att["boot_id"] = in.BootID
	}
	if in.KernelVersion != "" {
		att["kernel_version"] = in.KernelVersion
	}
	if in.OSImage != "" {
		att["os_image"] = in.OSImage
	}
	if in.ContainerRuntimeVersion != "" {
		att["container_runtime_version"] = in.ContainerRuntimeVersion
	}
	if in.KubeletVersion != "" {
		att["kubelet_version"] = in.KubeletVersion
	}
	if in.KubeProxyVersion != "" {
		att["kube_proxy_version"] = in.KubeProxyVersion
	}
	if in.OperatingSystem != "" {
		att["operating_system"] = in.OperatingSystem
	}
	if in.Architecture != "" {
		att["architecture"] = in.Architecture
	}
	return []interface{}{att}
}

func flattenNodeStatus(in v1.NodeStatus) []interface{} {
	att := make(map[string]interface{})
	att["addresses"] = flattenAddresses(in.Addresses...)
	att["allocatable"] = flattenResourceList(in.Allocatable)
	att["capacity"] = flattenResourceList(in.Capacity)
	att["node_info"] = flattenNodeInfo(in.NodeInfo)
	return []interface{}{att}
}

func flattenNodeTaints(in ...v1.Taint) []interface{} {
	out := make([]interface{}, len(in))
	for i, taint := range in {
		m := make(map[string]interface{})
		m["key"] = taint.Key
		m["value"] = taint.Value
		m["effect"] = taint.Effect
		out[i] = m
	}
	return out
}
