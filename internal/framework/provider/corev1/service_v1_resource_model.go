// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

type ServiceV1Model struct {
	ID                  types.String                     `tfsdk:"id"`
	Metadata            []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec                []ServiceV1SpecModel             `tfsdk:"spec"`
	WaitForLoadBalancer types.Bool                       `tfsdk:"wait_for_load_balancer"`
	Status              types.List                       `tfsdk:"status"`
	Timeouts            timeouts.Value                   `tfsdk:"timeouts"`
}

type ServiceV1SpecModel struct {
	AllocateLoadBalancerNodePorts types.Bool           `tfsdk:"allocate_load_balancer_node_ports"`
	ClusterIP                     types.String         `tfsdk:"cluster_ip"`
	ClusterIPs                    types.List           `tfsdk:"cluster_ips"`
	ExternalIPs                   types.Set            `tfsdk:"external_ips"`
	ExternalName                  types.String         `tfsdk:"external_name"`
	ExternalTrafficPolicy         types.String         `tfsdk:"external_traffic_policy"`
	IPFamilies                    types.List           `tfsdk:"ip_families"`
	IPFamilyPolicy                types.String         `tfsdk:"ip_family_policy"`
	InternalTrafficPolicy         types.String         `tfsdk:"internal_traffic_policy"`
	LoadBalancerClass             types.String         `tfsdk:"load_balancer_class"`
	LoadBalancerIP                types.String         `tfsdk:"load_balancer_ip"`
	LoadBalancerSourceRanges      types.Set            `tfsdk:"load_balancer_source_ranges"`
	Ports                         []ServiceV1PortModel `tfsdk:"port"`
	PublishNotReadyAddresses      types.Bool           `tfsdk:"publish_not_ready_addresses"`
	Selector                      types.Map            `tfsdk:"selector"`
	SessionAffinity               types.String         `tfsdk:"session_affinity"`
	SessionAffinityConfig         types.List           `tfsdk:"session_affinity_config"`
	Type                          types.String         `tfsdk:"type"`
	HealthCheckNodePort           types.Int64          `tfsdk:"health_check_node_port"`
}

type ServiceV1PortModel struct {
	AppProtocol types.String `tfsdk:"app_protocol"`
	Name        types.String `tfsdk:"name"`
	NodePort    types.Int64  `tfsdk:"node_port"`
	Port        types.Int64  `tfsdk:"port"`
	Protocol    types.String `tfsdk:"protocol"`
	TargetPort  types.String `tfsdk:"target_port"`
}

type ServiceV1SessionAffinityConfigModel struct {
	ClientIP types.List `tfsdk:"client_ip"`
}

type ServiceV1ClientIPModel struct {
	TimeoutSeconds types.Int64 `tfsdk:"timeout_seconds"`
}

var serviceClientIPType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"timeout_seconds": types.Int64Type,
}}

var serviceSessionAffinityConfigType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"client_ip": types.ListType{ElemType: serviceClientIPType},
}}

var serviceIngressType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"ip": types.StringType, "ip_mode": types.StringType, "hostname": types.StringType,
}}

var serviceLoadBalancerType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"ingress": types.ListType{ElemType: serviceIngressType},
}}

var serviceStatusType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"load_balancer": types.ListType{ElemType: serviceLoadBalancerType},
}}
