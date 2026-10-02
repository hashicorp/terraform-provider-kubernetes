// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// These structs describe stored SDKv2 JSON, not the current Framework schema.
// Schema v0 differs by the removed top-level load_balancer_ingress field.
type sdkv2ServiceStateV0 struct {
	sdkv2ServiceStateV1
	LoadBalancerIngress []sdkv2ServiceIngressV1 `json:"load_balancer_ingress"`
}

type sdkv2ServiceStateV1 struct {
	ID                  string                   `json:"id"`
	Metadata            []sdkv2ServiceMetadataV1 `json:"metadata"`
	Spec                []sdkv2ServiceSpecV1     `json:"spec"`
	WaitForLoadBalancer *bool                    `json:"wait_for_load_balancer"`
	Status              []sdkv2ServiceStatusV1   `json:"status"`
	Timeouts            *sdkv2ServiceTimeoutsV1  `json:"timeouts"`
}

type sdkv2ServiceMetadataV1 struct {
	Annotations     map[string]string `json:"annotations"`
	Labels          map[string]string `json:"labels"`
	GenerateName    *string           `json:"generate_name"`
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	Generation      int64             `json:"generation"`
	ResourceVersion string            `json:"resource_version"`
	UID             string            `json:"uid"`
}

type sdkv2ServiceSpecV1 struct {
	AllocateLoadBalancerNodePorts *bool                    `json:"allocate_load_balancer_node_ports"`
	ClusterIP                     *string                  `json:"cluster_ip"`
	ClusterIPs                    []string                 `json:"cluster_ips"`
	ExternalIPs                   []string                 `json:"external_ips"`
	ExternalName                  *string                  `json:"external_name"`
	ExternalTrafficPolicy         *string                  `json:"external_traffic_policy"`
	IPFamilies                    []string                 `json:"ip_families"`
	IPFamilyPolicy                *string                  `json:"ip_family_policy"`
	InternalTrafficPolicy         *string                  `json:"internal_traffic_policy"`
	LoadBalancerClass             *string                  `json:"load_balancer_class"`
	LoadBalancerIP                *string                  `json:"load_balancer_ip"`
	LoadBalancerSourceRanges      []string                 `json:"load_balancer_source_ranges"`
	Ports                         []sdkv2ServicePortV1     `json:"port"`
	PublishNotReadyAddresses      *bool                    `json:"publish_not_ready_addresses"`
	Selector                      map[string]string        `json:"selector"`
	SessionAffinity               *string                  `json:"session_affinity"`
	SessionAffinityConfig         []sdkv2ServiceAffinityV1 `json:"session_affinity_config"`
	Type                          *string                  `json:"type"`
	HealthCheckNodePort           *int64                   `json:"health_check_node_port"`
}

type sdkv2ServicePortV1 struct {
	AppProtocol *string `json:"app_protocol"`
	Name        *string `json:"name"`
	NodePort    *int64  `json:"node_port"`
	Port        int64   `json:"port"`
	Protocol    *string `json:"protocol"`
	TargetPort  *string `json:"target_port"`
}

type sdkv2ServiceAffinityV1 struct {
	ClientIP []sdkv2ServiceClientIPV1 `json:"client_ip"`
}

type sdkv2ServiceClientIPV1 struct {
	TimeoutSeconds *int64 `json:"timeout_seconds"`
}

type sdkv2ServiceStatusV1 struct {
	LoadBalancer []sdkv2ServiceLoadBalancerV1 `json:"load_balancer"`
}

type sdkv2ServiceLoadBalancerV1 struct {
	Ingress []sdkv2ServiceIngressV1 `json:"ingress"`
}

type sdkv2ServiceIngressV1 struct {
	IP       string `json:"ip"`
	IPMode   string `json:"ip_mode"`
	Hostname string `json:"hostname"`
}

type sdkv2ServiceTimeoutsV1 struct {
	Create *string `json:"create"`
}

func (r *ServiceV1) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	upgraders := make(map[int64]resource.StateUpgrader, 2)
	for _, version := range []int64{0, 1} {
		upgraders[version] = resource.StateUpgrader{
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				if req.RawState == nil || len(req.RawState.JSON) == 0 {
					resp.Diagnostics.AddError("Unable to upgrade service state", "The stored state has no JSON data; flatmap state is not supported.")
					return
				}
				state, d := r.decodeServiceState(ctx, req.RawState.JSON, version)
				resp.Diagnostics.Append(d...)
				if !resp.Diagnostics.HasError() {
					resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
				}
			},
		}
	}
	return upgraders
}

func (r *ServiceV1) MoveState(context.Context) []resource.StateMover {
	return []resource.StateMover{{
		StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
			if req.SourceTypeName != "kubernetes_service" || !strings.HasSuffix(req.SourceProviderAddress, sdkv2ProviderAddressSuffix) || (req.SourceSchemaVersion != 0 && req.SourceSchemaVersion != 1) {
				return
			}
			if req.SourceRawState == nil || len(req.SourceRawState.JSON) == 0 {
				resp.Diagnostics.AddError("Unable to move service state", "The source state has no JSON data; flatmap state is not supported.")
				return
			}
			state, d := r.decodeServiceState(ctx, req.SourceRawState.JSON, req.SourceSchemaVersion)
			resp.Diagnostics.Append(d...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.Diagnostics.Append(resp.TargetState.Set(ctx, &state)...)
			if !resp.Diagnostics.HasError() && resp.TargetIdentity != nil {
				resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, serviceIdentity(state.Metadata[0].Namespace.ValueString(), state.Metadata[0].Name.ValueString()))...)
			}
		},
	}}
}

func (r *ServiceV1) decodeServiceState(ctx context.Context, raw []byte, version int64) (ServiceV1Model, diag.Diagnostics) {
	var state ServiceV1Model
	var prior sdkv2ServiceStateV1
	var diags diag.Diagnostics
	var err error
	if version == 0 {
		var old sdkv2ServiceStateV0
		err = json.Unmarshal(raw, &old)
		prior = old.sdkv2ServiceStateV1
		// Match the historical upgrader: remove load_balancer_ingress. Read
		// supplies the current status, rather than inventing historical fields.
		prior.Status = nil
	} else {
		err = json.Unmarshal(raw, &prior)
	}
	if err != nil {
		diags.AddError("Unable to decode service state", err.Error())
		return state, diags
	}
	namespace, name, err := kubernetes.IdParts(prior.ID)
	if err != nil || namespace == "" || name == "" {
		diags.AddError("Invalid service state", "The stored ID must have the form namespace/name.")
		return state, diags
	}
	if len(prior.Metadata) != 1 || len(prior.Spec) != 1 {
		diags.AddError("Invalid service state", "The stored state must have exactly one metadata and spec element.")
		return state, diags
	}
	meta := prior.Metadata[0]
	if meta.Name != name || meta.Namespace != namespace {
		diags.AddError("Invalid service state", "The stored metadata name and namespace do not match the resource ID.")
		return state, diags
	}
	metadata := common.NamespacedMetadataModel{
		MetadataModel: common.MetadataModel{
			MetadataBase: common.MetadataBase{
				Name: types.StringValue(meta.Name), Generation: types.Int64Value(meta.Generation),
				ResourceVersion: types.StringValue(meta.ResourceVersion), UID: types.StringValue(meta.UID),
			},
			GenerateName: serviceStoredString(meta.GenerateName, ""),
		},
		Namespace: types.StringValue(meta.Namespace),
	}
	metadata.Annotations, diags = sdkv2MapToFramework(ctx, meta.Annotations)
	labels, d := sdkv2MapToFramework(ctx, meta.Labels)
	diags.Append(d...)
	metadata.Labels = labels
	spec, d := serviceStateSpec(ctx, prior.Spec[0])
	diags.Append(d...)
	state.ID = types.StringValue(prior.ID)
	state.Metadata = []common.NamespacedMetadataModel{metadata}
	state.Spec = []ServiceV1SpecModel{spec}
	state.WaitForLoadBalancer = serviceStoredBool(prior.WaitForLoadBalancer, true)
	state.Status, d = serviceStoredStatus(prior.Status)
	diags.Append(d...)
	var schemaResponse resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	diags.Append(schemaResponse.Diagnostics...)
	timeoutTypes := timeoutsAttributeTypes(schemaResponse.Schema)
	state.Timeouts = timeouts.Value{Object: types.ObjectNull(timeoutTypes)}
	if prior.Timeouts != nil {
		object, d := types.ObjectValue(timeoutTypes, map[string]attr.Value{"create": types.StringPointerValue(prior.Timeouts.Create)})
		diags.Append(d...)
		state.Timeouts = timeouts.Value{Object: object}
	}
	return state, diags
}

func serviceStateSpec(ctx context.Context, in sdkv2ServiceSpecV1) (ServiceV1SpecModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := ServiceV1SpecModel{
		AllocateLoadBalancerNodePorts: serviceStoredBool(in.AllocateLoadBalancerNodePorts, true),
		ClusterIP:                     types.StringPointerValue(in.ClusterIP), ExternalName: serviceStoredString(in.ExternalName, ""),
		ExternalTrafficPolicy: types.StringPointerValue(in.ExternalTrafficPolicy),
		IPFamilyPolicy:        types.StringPointerValue(in.IPFamilyPolicy), InternalTrafficPolicy: types.StringPointerValue(in.InternalTrafficPolicy),
		LoadBalancerClass: serviceStoredString(in.LoadBalancerClass, ""), LoadBalancerIP: serviceStoredString(in.LoadBalancerIP, ""),
		PublishNotReadyAddresses: serviceStoredBool(in.PublishNotReadyAddresses, false),
		SessionAffinity:          serviceStoredString(in.SessionAffinity, "None"), Type: serviceStoredString(in.Type, "ClusterIP"),
		HealthCheckNodePort: types.Int64PointerValue(in.HealthCheckNodePort),
	}
	var d diag.Diagnostics
	out.ClusterIPs, d = types.ListValueFrom(ctx, types.StringType, in.ClusterIPs)
	diags.Append(d...)
	out.IPFamilies, d = types.ListValueFrom(ctx, types.StringType, in.IPFamilies)
	diags.Append(d...)
	out.ExternalIPs, d = types.SetValueFrom(ctx, types.StringType, in.ExternalIPs)
	diags.Append(d...)
	out.LoadBalancerSourceRanges, d = types.SetValueFrom(ctx, types.StringType, in.LoadBalancerSourceRanges)
	diags.Append(d...)
	out.Selector, d = sdkv2MapToFramework(ctx, in.Selector)
	diags.Append(d...)
	out.Ports = make([]ServiceV1PortModel, len(in.Ports))
	for i, port := range in.Ports {
		out.Ports[i] = ServiceV1PortModel{
			AppProtocol: serviceStoredString(port.AppProtocol, ""), Name: serviceStoredString(port.Name, ""),
			Protocol: serviceStoredString(port.Protocol, "TCP"), Port: types.Int64Value(port.Port),
			NodePort: types.Int64PointerValue(port.NodePort), TargetPort: types.StringPointerValue(port.TargetPort),
		}
	}
	if len(in.SessionAffinityConfig) > 1 {
		diags.AddError("Invalid service state", "session_affinity_config must contain at most one element.")
		return out, diags
	}
	affinity := make([]ServiceV1SessionAffinityConfigModel, len(in.SessionAffinityConfig))
	for i, entry := range in.SessionAffinityConfig {
		if len(entry.ClientIP) > 1 {
			diags.AddError("Invalid service state", "client_ip must contain at most one element.")
			return out, diags
		}
		clientIPs := make([]ServiceV1ClientIPModel, len(entry.ClientIP))
		for j, client := range entry.ClientIP {
			clientIPs[j] = ServiceV1ClientIPModel{TimeoutSeconds: types.Int64PointerValue(client.TimeoutSeconds)}
		}
		affinity[i].ClientIP, d = types.ListValueFrom(ctx, serviceClientIPType, clientIPs)
		diags.Append(d...)
	}
	out.SessionAffinityConfig, d = types.ListValueFrom(ctx, serviceSessionAffinityConfigType, affinity)
	diags.Append(d...)
	return out, diags
}

func serviceStoredBool(value *bool, fallback bool) types.Bool {
	if value == nil {
		return types.BoolValue(fallback)
	}
	return types.BoolValue(*value)
}

func serviceStoredString(value *string, fallback string) types.String {
	if value == nil {
		return types.StringValue(fallback)
	}
	return types.StringValue(*value)
}

func serviceStoredStatus(in []sdkv2ServiceStatusV1) (types.List, diag.Diagnostics) {
	if in == nil {
		return types.ListNull(serviceStatusType), nil
	}
	var diags diag.Diagnostics
	statuses := make([]attr.Value, len(in))
	for i, status := range in {
		lbs := make([]attr.Value, len(status.LoadBalancer))
		for j, lb := range status.LoadBalancer {
			items := make([]attr.Value, len(lb.Ingress))
			for k, item := range lb.Ingress {
				object, d := types.ObjectValue(serviceIngressType.AttrTypes, map[string]attr.Value{
					"ip": types.StringValue(item.IP), "ip_mode": types.StringValue(item.IPMode), "hostname": types.StringValue(item.Hostname),
				})
				diags.Append(d...)
				items[k] = object
			}
			list, d := types.ListValue(serviceIngressType, items)
			diags.Append(d...)
			object, d := types.ObjectValue(serviceLoadBalancerType.AttrTypes, map[string]attr.Value{"ingress": list})
			diags.Append(d...)
			lbs[j] = object
		}
		list, d := types.ListValue(serviceLoadBalancerType, lbs)
		diags.Append(d...)
		object, d := types.ObjectValue(serviceStatusType.AttrTypes, map[string]attr.Value{"load_balancer": list})
		diags.Append(d...)
		statuses[i] = object
	}
	out, d := types.ListValue(serviceStatusType, statuses)
	diags.Append(d...)
	return out, diags
}
