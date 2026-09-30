// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *ServiceV1) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var specs types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("spec"), &specs)...)
	if resp.Diagnostics.HasError() || specs.IsNull() || specs.IsUnknown() {
		return
	}
	for i, value := range specs.Elements() {
		if value.IsNull() || value.IsUnknown() {
			continue
		}
		spec := value.(types.Object).Attributes()
		resp.Diagnostics.Append(validateServiceIPAllocations(
			spec["type"].(types.String),
			spec["cluster_ips"].(types.List),
			spec["ip_families"].(types.List),
			path.Root("spec").AtListIndex(i),
		)...)
		resp.Diagnostics.Append(validateServiceAffinityCollections(
			spec["session_affinity"].(types.String),
			spec["session_affinity_config"].(types.List),
			path.Root("spec").AtListIndex(i).AtName("session_affinity_config"),
		)...)
		ports := spec["port"].(types.List)
		if ports.IsNull() || ports.IsUnknown() {
			continue
		}
		for j, port := range ports.Elements() {
			if port.IsNull() || port.IsUnknown() {
				continue
			}
			resp.Diagnostics.Append(validateServiceNodePortAllocation(
				spec["type"].(types.String),
				spec["allocate_load_balancer_node_ports"].(types.Bool),
				port.(types.Object).Attributes()["node_port"].(types.Int64),
				path.Root("spec").AtListIndex(i).AtName("port").AtListIndex(j).AtName("node_port"),
			)...)
			resp.Diagnostics.Append(validateServiceTargetPort(
				port.(types.Object).Attributes()["target_port"].(types.String),
				path.Root("spec").AtListIndex(i).AtName("port").AtListIndex(j).AtName("target_port"),
			)...)
		}
	}
}

func validateServiceWritePlan(spec ServiceV1SpecModel) diag.Diagnostics {
	p := path.Root("spec").AtListIndex(0)
	diags := validateServiceAffinityCollections(spec.SessionAffinity, spec.SessionAffinityConfig, p.AtName("session_affinity_config"))
	diags.Append(validateServiceIPAllocations(spec.Type, spec.ClusterIPs, spec.IPFamilies, p)...)
	for i, port := range spec.Ports {
		diags.Append(validateServiceNodePortAllocation(spec.Type, spec.AllocateLoadBalancerNodePorts, port.NodePort, p.AtName("port").AtListIndex(i).AtName("node_port"))...)
		diags.Append(validateServiceTargetPort(port.TargetPort, p.AtName("port").AtListIndex(i).AtName("target_port"))...)
	}
	return diags
}

func validateServiceTargetPort(target types.String, p path.Path) diag.Diagnostics {
	var diags diag.Diagnostics
	if target.IsNull() || target.IsUnknown() {
		return diags
	}
	if target.ValueString() == "" {
		diags.AddAttributeError(p, "Invalid empty target_port",
			"Omit target_port or set it to null to use or retain the API-resolved target port. Kubernetes replaces an empty string with a concrete target port, which cannot be represented as the configured empty value. Configurations using the earlier SDK behavior must remove target_port = \"\".")
	} else if intstr.Parse(target.ValueString()) == intstr.FromInt(0) {
		diags.AddAttributeError(p, "Invalid zero target_port",
			"Omit target_port or set it to null to use or retain the API-resolved target port. Kubernetes replaces a numeric zero with a concrete target port, which cannot be represented as the configured zero value.")
	}
	return diags
}

func validateServiceIPAllocations(serviceType types.String, clusterIPs, ipFamilies types.List, p path.Path) diag.Diagnostics {
	var diags diag.Diagnostics
	if serviceType.IsUnknown() || serviceType.ValueString() == "ExternalName" {
		return diags
	}
	for _, field := range []struct {
		name  string
		value types.List
	}{
		{"cluster_ips", clusterIPs},
		{"ip_families", ipFamilies},
	} {
		if !field.value.IsNull() && !field.value.IsUnknown() && len(field.value.Elements()) == 0 {
			diags.AddAttributeError(p.AtName(field.name), "Invalid empty "+field.name,
				"For a Service other than ExternalName, omit this argument or set it to null to use or retain API-assigned values. Kubernetes populates this list, so an explicit empty list cannot represent the resulting Service.")
		}
	}
	return diags
}

func validateServiceNodePortAllocation(serviceType types.String, allocate types.Bool, nodePort types.Int64, p path.Path) diag.Diagnostics {
	var diags diag.Diagnostics
	if serviceType.IsUnknown() || nodePort.IsNull() || nodePort.IsUnknown() || nodePort.ValueInt64() != 0 {
		return diags
	}
	autoAllocate := serviceType.ValueString() == "NodePort"
	if serviceType.ValueString() == "LoadBalancer" && !allocate.IsUnknown() {
		autoAllocate = allocate.IsNull() || allocate.ValueBool()
	}
	if autoAllocate {
		diags.AddAttributeError(p, "Invalid zero node_port",
			"When the Service automatically allocates node ports (type NodePort, or LoadBalancer with allocate_load_balancer_node_ports enabled), omit node_port or set it to null instead of 0. Kubernetes replaces zero with an assigned port, which cannot be represented as the configured zero value. Remove node_port = 0 from the configuration to use automatic allocation.")
	}
	return diags
}

func validateServiceAffinityCollections(affinity types.String, config types.List, p path.Path) diag.Diagnostics {
	var diags diag.Diagnostics
	if affinity.IsUnknown() || affinity.ValueString() != "ClientIP" || config.IsNull() || config.IsUnknown() {
		return diags
	}
	// ClientIP always receives an API-populated configuration. A known empty
	// collection cannot be populated without violating the configured value.
	if len(config.Elements()) == 0 {
		diags.AddAttributeError(p, "Empty ClientIP configuration",
			"When session_affinity is ClientIP, omit session_affinity_config or set it to null to retain provider-computed values. Use [{}] to request defaults, or configure one object with explicit values. An empty list cannot represent the API-generated configuration.")
		return diags
	}
	for i, value := range config.Elements() {
		if value.IsNull() || value.IsUnknown() {
			continue
		}
		clientIP := value.(types.Object).Attributes()["client_ip"].(types.List)
		if !clientIP.IsNull() && !clientIP.IsUnknown() && len(clientIP.Elements()) == 0 {
			diags.AddAttributeError(p.AtListIndex(i).AtName("client_ip"), "Empty ClientIP configuration",
				"When session_affinity is ClientIP, omit client_ip or set it to null to retain provider-computed values. Use [{}] to request defaults, or configure one object with explicit values. An empty list cannot represent the API-generated configuration.")
		}
	}
	return diags
}
