// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func expandServiceSpec(ctx context.Context, in ServiceV1SpecModel) (corev1.ServiceSpec, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := corev1.ServiceSpec{
		ClusterIP: in.ClusterIP.ValueString(), Type: corev1.ServiceType(in.Type.ValueString()),
		ExternalName: in.ExternalName.ValueString(), ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicy(in.ExternalTrafficPolicy.ValueString()),
		LoadBalancerIP: in.LoadBalancerIP.ValueString(), SessionAffinity: corev1.ServiceAffinity(in.SessionAffinity.ValueString()),
		PublishNotReadyAddresses: in.PublishNotReadyAddresses.ValueBool(), HealthCheckNodePort: int32(in.HealthCheckNodePort.ValueInt64()),
	}
	if !in.Selector.IsNull() && !in.Selector.IsUnknown() {
		diags.Append(in.Selector.ElementsAs(ctx, &out.Selector, false)...)
	}
	if !in.ClusterIPs.IsNull() && !in.ClusterIPs.IsUnknown() {
		diags.Append(in.ClusterIPs.ElementsAs(ctx, &out.ClusterIPs, false)...)
	}
	if !in.ExternalIPs.IsNull() && !in.ExternalIPs.IsUnknown() {
		diags.Append(in.ExternalIPs.ElementsAs(ctx, &out.ExternalIPs, false)...)
	}
	if !in.LoadBalancerSourceRanges.IsNull() && !in.LoadBalancerSourceRanges.IsUnknown() {
		diags.Append(in.LoadBalancerSourceRanges.ElementsAs(ctx, &out.LoadBalancerSourceRanges, false)...)
	}
	if !in.IPFamilies.IsNull() && !in.IPFamilies.IsUnknown() {
		var families []string
		diags.Append(in.IPFamilies.ElementsAs(ctx, &families, false)...)
		for _, family := range families {
			out.IPFamilies = append(out.IPFamilies, corev1.IPFamily(family))
		}
	}
	if in.IPFamilyPolicy.ValueString() != "" {
		out.IPFamilyPolicy = ptr.To(corev1.IPFamilyPolicy(in.IPFamilyPolicy.ValueString()))
	}
	if in.InternalTrafficPolicy.ValueString() != "" {
		out.InternalTrafficPolicy = ptr.To(corev1.ServiceInternalTrafficPolicy(in.InternalTrafficPolicy.ValueString()))
	}
	if out.Type == corev1.ServiceTypeLoadBalancer {
		if !in.AllocateLoadBalancerNodePorts.IsNull() && !in.AllocateLoadBalancerNodePorts.IsUnknown() {
			out.AllocateLoadBalancerNodePorts = in.AllocateLoadBalancerNodePorts.ValueBoolPointer()
		}
		if in.LoadBalancerClass.ValueString() != "" {
			out.LoadBalancerClass = in.LoadBalancerClass.ValueStringPointer()
		}
	}
	for _, port := range in.Ports {
		out.Ports = append(out.Ports, expandServicePort(port))
	}
	out.SessionAffinityConfig, diags = expandServiceAffinity(ctx, in.SessionAffinityConfig, diags)
	return out, diags
}

func expandServicePort(in ServiceV1PortModel) corev1.ServicePort {
	out := corev1.ServicePort{
		Name: in.Name.ValueString(), Port: int32(in.Port.ValueInt64()),
		Protocol: corev1.Protocol(in.Protocol.ValueString()), NodePort: int32(in.NodePort.ValueInt64()),
		TargetPort: intstr.Parse(in.TargetPort.ValueString()),
	}
	if in.AppProtocol.ValueString() != "" {
		out.AppProtocol = in.AppProtocol.ValueStringPointer()
	}
	return out
}

func expandServiceAffinity(ctx context.Context, in types.Object, diags diag.Diagnostics) (*corev1.SessionAffinityConfig, diag.Diagnostics) {
	if in.IsNull() || in.IsUnknown() {
		return nil, diags
	}
	var model ServiceV1SessionAffinityConfigModel
	diags.Append(in.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}
	out := &corev1.SessionAffinityConfig{}
	if !model.ClientIP.IsNull() && !model.ClientIP.IsUnknown() {
		var client ServiceV1ClientIPModel
		diags.Append(model.ClientIP.As(ctx, &client, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}
		out.ClientIP = &corev1.ClientIPConfig{}
		if !client.TimeoutSeconds.IsNull() && !client.TimeoutSeconds.IsUnknown() {
			out.ClientIP.TimeoutSeconds = ptr.To(int32(client.TimeoutSeconds.ValueInt64()))
		}
	}
	return out, diags
}

func flattenService(ctx context.Context, in *corev1.Service, model *ServiceV1Model, filters interface {
	GetIgnoreAnnotations() []string
	GetIgnoreLabels() []string
}, applying bool) diag.Diagnostics {
	var diags diag.Diagnostics
	model.ID = types.StringValue(in.Namespace + "/" + in.Name)
	if applying && len(model.Metadata) == 1 {
		meta := &model.Metadata[0]
		meta.Name = types.StringValue(in.Name)
		meta.Namespace = types.StringValue(in.Namespace)
		meta.UID = types.StringValue(string(in.UID))
		meta.Generation = types.Int64Value(in.Generation)
		meta.ResourceVersion = types.StringValue(in.ResourceVersion)
		meta.GenerateName = serviceStringValue(in.GenerateName, meta.GenerateName, true, true)
	} else {
		model.Metadata, diags = common.FlattenNamespacedMetadata(ctx, in.ObjectMeta, model.Metadata, filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
		if len(model.Metadata) == 1 {
			model.Metadata[0].GenerateName = types.StringValue(in.GenerateName)
		}
	}
	var prior ServiceV1SpecModel
	if len(model.Spec) == 1 {
		prior = model.Spec[0]
	}
	spec, d := flattenServiceSpec(ctx, in.Spec, prior, applying)
	diags.Append(d...)
	model.Spec = []ServiceV1SpecModel{spec}
	model.Status, d = flattenServiceStatus(ctx, in.Status)
	diags.Append(d...)
	if model.WaitForLoadBalancer.IsNull() || model.WaitForLoadBalancer.IsUnknown() {
		model.WaitForLoadBalancer = types.BoolValue(true)
	}
	return diags
}

func flattenServiceSpec(ctx context.Context, in corev1.ServiceSpec, prior ServiceV1SpecModel, applying bool) (ServiceV1SpecModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := ServiceV1SpecModel{
		ClusterIP:                     serviceStringValue(in.ClusterIP, prior.ClusterIP, applying, true),
		ExternalName:                  serviceStringValue(in.ExternalName, prior.ExternalName, applying, true),
		ExternalTrafficPolicy:         serviceStringValue(string(in.ExternalTrafficPolicy), prior.ExternalTrafficPolicy, applying, true),
		IPFamilyPolicy:                serviceStringValue(string(ptr.Deref(in.IPFamilyPolicy, "")), prior.IPFamilyPolicy, applying, true),
		InternalTrafficPolicy:         serviceStringValue(string(ptr.Deref(in.InternalTrafficPolicy, "")), prior.InternalTrafficPolicy, applying, true),
		LoadBalancerClass:             serviceStringValue(ptr.Deref(in.LoadBalancerClass, ""), prior.LoadBalancerClass, applying, true),
		LoadBalancerIP:                serviceStringValue(in.LoadBalancerIP, prior.LoadBalancerIP, applying, true),
		Type:                          serviceStringValue(string(in.Type), prior.Type, applying, true),
		SessionAffinity:               serviceStringValue(string(in.SessionAffinity), prior.SessionAffinity, applying, true),
		AllocateLoadBalancerNodePorts: types.BoolValue(ptr.Deref(in.AllocateLoadBalancerNodePorts, true)),
		PublishNotReadyAddresses:      types.BoolValue(in.PublishNotReadyAddresses),
		HealthCheckNodePort:           types.Int64Value(int64(in.HealthCheckNodePort)),
	}
	// Non-load-balancer Services do not store this field. Match the shipped
	// true default without replacing a caller's explicit false value.
	if in.AllocateLoadBalancerNodePorts == nil && !prior.AllocateLoadBalancerNodePorts.IsNull() && !prior.AllocateLoadBalancerNodePorts.IsUnknown() {
		out.AllocateLoadBalancerNodePorts = prior.AllocateLoadBalancerNodePorts
	}
	if applying {
		if !prior.AllocateLoadBalancerNodePorts.IsUnknown() && !prior.AllocateLoadBalancerNodePorts.IsNull() {
			out.AllocateLoadBalancerNodePorts = prior.AllocateLoadBalancerNodePorts
		}
		if !prior.PublishNotReadyAddresses.IsUnknown() && !prior.PublishNotReadyAddresses.IsNull() {
			out.PublishNotReadyAddresses = prior.PublishNotReadyAddresses
		}
		if !prior.HealthCheckNodePort.IsUnknown() && !prior.HealthCheckNodePort.IsNull() {
			out.HealthCheckNodePort = prior.HealthCheckNodePort
		}
	}
	out.ClusterIPs = serviceStringList(ctx, in.ClusterIPs, prior.ClusterIPs, applying, &diags)
	families := make([]string, len(in.IPFamilies))
	for i, family := range in.IPFamilies {
		families[i] = string(family)
	}
	out.IPFamilies = serviceStringList(ctx, families, prior.IPFamilies, applying, &diags)
	out.ExternalIPs = serviceStringSet(ctx, in.ExternalIPs, prior.ExternalIPs, applying, &diags)
	out.LoadBalancerSourceRanges = serviceStringSet(ctx, in.LoadBalancerSourceRanges, prior.LoadBalancerSourceRanges, applying, &diags)
	if applying {
		out.Selector = prior.Selector
	} else if len(in.Selector) == 0 && prior.Selector.IsNull() {
		out.Selector = types.MapNull(types.StringType)
	} else {
		selector := in.Selector
		if selector == nil && !prior.Selector.IsUnknown() && len(prior.Selector.Elements()) == 0 {
			// The API omits empty maps. Preserve an explicitly empty selector,
			// but still report removal of previously nonempty selectors.
			selector = map[string]string{}
		}
		var d diag.Diagnostics
		out.Selector, d = types.MapValueFrom(ctx, types.StringType, selector)
		diags.Append(d...)
	}
	ports := in.Ports
	if applying {
		ports = make([]corev1.ServicePort, len(prior.Ports))
		for i, planned := range prior.Ports {
			port := expandServicePort(planned)
			if j := serviceAPIPortIndex(in.Ports, port, 0); j >= 0 {
				port = in.Ports[j]
			}
			ports[i] = port
		}
	}
	out.Ports = make([]ServiceV1PortModel, len(ports))
	for i, port := range ports {
		var old ServiceV1PortModel
		if applying && i < len(prior.Ports) {
			old = prior.Ports[i]
		} else if j := serviceModelPortIndex(prior.Ports, port); j >= 0 {
			old = prior.Ports[j]
		}
		out.Ports[i] = ServiceV1PortModel{
			Name:        serviceStringValue(port.Name, old.Name, applying, true),
			AppProtocol: serviceStringValue(ptr.Deref(port.AppProtocol, ""), old.AppProtocol, applying, true),
			Protocol:    serviceStringValue(string(port.Protocol), old.Protocol, applying, true),
			Port:        types.Int64Value(int64(port.Port)), NodePort: types.Int64Value(int64(port.NodePort)),
			TargetPort: serviceStringValue(port.TargetPort.String(), old.TargetPort, applying && old.TargetPort.ValueString() != "", true),
		}
		// An allowed zero must really remain unallocated; never substitute
		// a configured zero for a port that the API assigned.
		if applying && !old.NodePort.IsUnknown() && !old.NodePort.IsNull() && old.NodePort.ValueInt64() != 0 {
			out.Ports[i].NodePort = old.NodePort
		}
	}
	affinity, d := flattenServiceAffinity(ctx, in.SessionAffinityConfig)
	diags.Append(d...)
	out.SessionAffinityConfig = affinity
	if applying {
		resolved, d := serviceResolveAffinityValue(ctx, prior.SessionAffinityConfig, affinity)
		diags.Append(d...)
		out.SessionAffinityConfig = resolved.(types.Object)
	}
	return out, diags
}

func serviceStringValue(value string, prior types.String, applying, computed bool) types.String {
	if applying && !prior.IsUnknown() && (!prior.IsNull() || !computed) {
		return prior
	}
	if value != "" || computed || (!prior.IsNull() && !prior.IsUnknown() && prior.ValueString() == "") {
		return types.StringValue(value)
	}
	return types.StringNull()
}

func serviceStringList(ctx context.Context, values []string, prior types.List, applying bool, diags *diag.Diagnostics) types.List {
	if applying && !prior.IsUnknown() && !prior.IsNull() {
		return prior
	}
	if values == nil {
		values = []string{}
	}
	out, d := types.ListValueFrom(ctx, types.StringType, values)
	diags.Append(d...)
	return out
}

func serviceStringSet(ctx context.Context, values []string, prior types.Set, applying bool, diags *diag.Diagnostics) types.Set {
	if applying {
		return prior
	}
	if len(values) == 0 && prior.IsNull() {
		return types.SetNull(types.StringType)
	}
	if values == nil {
		values = []string{}
	}
	out, d := types.SetValueFrom(ctx, types.StringType, values)
	diags.Append(d...)
	return out
}

func flattenServiceAffinity(ctx context.Context, in *corev1.SessionAffinityConfig) (types.Object, diag.Diagnostics) {
	if in == nil {
		return types.ObjectNull(serviceSessionAffinityConfigType.AttrTypes), nil
	}
	client := types.ObjectNull(serviceClientIPType.AttrTypes)
	var diags diag.Diagnostics
	if in.ClientIP != nil {
		timeout := types.Int64Null()
		if in.ClientIP.TimeoutSeconds != nil {
			timeout = types.Int64Value(int64(*in.ClientIP.TimeoutSeconds))
		}
		var d diag.Diagnostics
		client, d = types.ObjectValueFrom(ctx, serviceClientIPType.AttrTypes, ServiceV1ClientIPModel{TimeoutSeconds: timeout})
		diags.Append(d...)
	}
	out, d := types.ObjectValueFrom(ctx, serviceSessionAffinityConfigType.AttrTypes, ServiceV1SessionAffinityConfigModel{ClientIP: client})
	diags.Append(d...)
	return out, diags
}

// Resolve computed descendants without replacing configured affinity blocks.
func serviceResolveAffinityValue(ctx context.Context, planned, actual attr.Value) (attr.Value, diag.Diagnostics) {
	if planned.IsNull() || planned.IsUnknown() {
		return actual, nil
	}
	switch planned := planned.(type) {
	case types.Object:
		actual, ok := actual.(types.Object)
		if !ok {
			return planned, nil
		}
		values := planned.Attributes()
		api := actual.Attributes()
		var diags diag.Diagnostics
		for name, value := range values {
			actualValue, ok := api[name]
			if !ok {
				t := planned.AttributeTypes(ctx)[name]
				var d diag.Diagnostics
				actualValue, d = serviceNullValue(ctx, t)
				diags.Append(d...)
			}
			resolved, d := serviceResolveAffinityValue(ctx, value, actualValue)
			diags.Append(d...)
			values[name] = resolved
		}
		out, d := types.ObjectValue(planned.AttributeTypes(ctx), values)
		diags.Append(d...)
		return out, diags
	default:
		return planned, nil
	}
}

func serviceNullValue(ctx context.Context, t attr.Type) (attr.Value, diag.Diagnostics) {
	if object, ok := t.(basetypes.ObjectType); ok {
		return types.ObjectNull(object.AttrTypes), nil
	}
	if t.Equal(types.Int64Type) {
		return types.Int64Null(), nil
	}
	var diags diag.Diagnostics
	diags.AddError("Invalid affinity value type", "The Service affinity schema contains an unsupported type.")
	return types.Int64Null(), diags
}

func flattenServiceStatus(ctx context.Context, status corev1.ServiceStatus) (types.List, diag.Diagnostics) {
	ingress := make([]attr.Value, 0, len(status.LoadBalancer.Ingress))
	var diags diag.Diagnostics
	for _, item := range status.LoadBalancer.Ingress {
		object, d := types.ObjectValue(serviceIngressType.AttrTypes, map[string]attr.Value{
			"ip": types.StringValue(item.IP), "ip_mode": types.StringValue(string(ptr.Deref(item.IPMode, ""))), "hostname": types.StringValue(item.Hostname),
		})
		diags.Append(d...)
		ingress = append(ingress, object)
	}
	items, d := types.ListValue(serviceIngressType, ingress)
	diags.Append(d...)
	lb, d := types.ObjectValue(serviceLoadBalancerType.AttrTypes, map[string]attr.Value{"ingress": items})
	diags.Append(d...)
	lbs, d := types.ListValue(serviceLoadBalancerType, []attr.Value{lb})
	diags.Append(d...)
	out, d := types.ObjectValue(serviceStatusType.AttrTypes, map[string]attr.Value{"load_balancer": lbs})
	diags.Append(d...)
	result, d := types.ListValue(serviceStatusType, []attr.Value{out})
	diags.Append(d...)
	return result, diags
}

func serviceModelPortIndex(ports []ServiceV1PortModel, port corev1.ServicePort) int {
	for i, candidate := range ports {
		if port.Name != "" && candidate.Name.ValueString() == port.Name {
			return i
		}
	}
	for i, candidate := range ports {
		if candidate.Port.ValueInt64() == int64(port.Port) && candidate.Protocol.ValueString() == string(port.Protocol) {
			return i
		}
	}
	if len(ports) == 1 && port.Name == "" && ports[0].Name.ValueString() == "" {
		return 0
	}
	return -1
}
