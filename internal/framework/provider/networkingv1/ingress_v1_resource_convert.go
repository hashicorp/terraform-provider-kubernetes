// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
)

func ingressExpandSpec(ctx context.Context, in []IngressV1SpecModel) (networking.IngressSpec, diag.Diagnostics) {
	var diags diag.Diagnostics
	var out networking.IngressSpec
	if len(in) != 1 {
		diags.AddError("Invalid ingress spec", "Exactly one spec block is required.")
		return out, diags
	}
	spec := in[0]
	if !spec.IngressClassName.IsUnknown() && spec.IngressClassName.ValueString() != "" {
		value := spec.IngressClassName.ValueString()
		out.IngressClassName = &value
	}
	if spec.DefaultBackend != nil {
		out.DefaultBackend = ingressExpandBackend(spec.DefaultBackend, &diags)
	}
	for _, rule := range spec.Rule {
		ingressKnown(&diags, "rule.host", rule.Host)
		r := networking.IngressRule{Host: rule.Host.ValueString()}
		if rule.HTTP != nil {
			r.HTTP = &networking.HTTPIngressRuleValue{}
			for _, p := range rule.HTTP.Path {
				ingressKnown(&diags, "path.path", p.Path)
				ingressKnown(&diags, "path.path_type", p.PathType)
				pathType := networking.PathType(p.PathType.ValueString())
				r.HTTP.Paths = append(r.HTTP.Paths, networking.HTTPIngressPath{
					Path: p.Path.ValueString(), PathType: &pathType,
					Backend: *ingressExpandBackend(p.Backend, &diags),
				})
			}
		}
		out.Rules = append(out.Rules, r)
	}
	for _, tls := range spec.TLS {
		ingressKnown(&diags, "tls.secret_name", tls.SecretName)
		ingressKnown(&diags, "tls.hosts", tls.Hosts)
		t := networking.IngressTLS{SecretName: tls.SecretName.ValueString()}
		if !tls.Hosts.IsNull() && !tls.Hosts.IsUnknown() {
			diags.Append(tls.Hosts.ElementsAs(ctx, &t.Hosts, false)...)
		}
		out.TLS = append(out.TLS, t)
	}
	return out, diags
}

func ingressExpandBackend(in *IngressV1BackendModel, diags *diag.Diagnostics) *networking.IngressBackend {
	out := &networking.IngressBackend{}
	if in == nil {
		return out
	}
	if in.Resource != nil {
		r := in.Resource
		ingressKnown(diags, "resource.api_group", r.APIGroup)
		ingressKnown(diags, "resource.kind", r.Kind)
		ingressKnown(diags, "resource.name", r.Name)
		group := r.APIGroup.ValueString()
		out.Resource = &corev1.TypedLocalObjectReference{
			APIGroup: &group, Kind: r.Kind.ValueString(), Name: r.Name.ValueString(),
		}
	}
	if in.Service != nil {
		s := in.Service
		ingressKnown(diags, "service.name", s.Name)
		out.Service = &networking.IngressServiceBackend{Name: s.Name.ValueString()}
		if s.Port == nil {
			diags.AddError("Invalid ingress backend", "A port object is required for a service backend.")
			return out
		}
		ingressKnown(diags, "port.name", s.Port.Name)
		ingressKnown(diags, "port.number", s.Port.Number)
		out.Service.Port.Name = s.Port.Name.ValueString()
		out.Service.Port.Number = int32(s.Port.Number.ValueInt64())
	}
	return out
}

func ingressKnown(diags *diag.Diagnostics, field string, value attr.Value) {
	if value.IsUnknown() {
		diags.AddError("Unknown ingress value", fmt.Sprintf("%s must be known before applying the ingress.", field))
	}
}

func ingressFlattenSpec(ctx context.Context, in networking.IngressSpec, prior []IngressV1SpecModel) ([]IngressV1SpecModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := IngressV1SpecModel{
		IngressClassName: types.StringValue(""),
		DefaultBackend:   nil,
		Rule:             []IngressV1RuleModel{},
		TLS:              []IngressV1TLSModel{},
	}
	if in.IngressClassName != nil {
		out.IngressClassName = types.StringValue(*in.IngressClassName)
	}
	if in.DefaultBackend != nil {
		out.DefaultBackend = ingressFlattenBackend(in.DefaultBackend)
	}
	for _, rule := range in.Rules {
		r := IngressV1RuleModel{Host: types.StringValue(rule.Host), HTTP: nil}
		if rule.HTTP != nil {
			h := IngressV1HTTPModel{Path: []IngressV1PathModel{}}
			for _, p := range rule.HTTP.Paths {
				pathType := ""
				if p.PathType != nil {
					pathType = string(*p.PathType)
				}
				h.Path = append(h.Path, IngressV1PathModel{
					Path: types.StringValue(p.Path), PathType: types.StringValue(pathType),
					Backend: ingressFlattenBackend(&p.Backend),
				})
			}
			r.HTTP = &h
		}
		out.Rule = append(out.Rule, r)
	}
	for i, tls := range in.TLS {
		hosts := types.ListNull(types.StringType)
		// Preserve explicit [] without turning an omitted optional attribute into [].
		if len(tls.Hosts) > 0 || (len(prior) == 1 && i < len(prior[0].TLS) && !prior[0].TLS[i].Hosts.IsNull()) {
			values := make([]attr.Value, len(tls.Hosts))
			for j, host := range tls.Hosts {
				values[j] = types.StringValue(host)
			}
			var d diag.Diagnostics
			hosts, d = types.ListValue(types.StringType, values)
			diags.Append(d...)
		}
		out.TLS = append(out.TLS, IngressV1TLSModel{Hosts: hosts, SecretName: types.StringValue(tls.SecretName)})
	}
	return []IngressV1SpecModel{out}, diags
}

func ingressFlattenBackend(in *networking.IngressBackend) *IngressV1BackendModel {
	if in == nil {
		return nil
	}
	out := IngressV1BackendModel{
		Resource: nil,
		Service:  nil,
	}
	if in.Resource != nil {
		group := ""
		if in.Resource.APIGroup != nil {
			group = *in.Resource.APIGroup
		}
		out.Resource = &IngressV1ResourceBackendModel{
			APIGroup: types.StringValue(group), Kind: types.StringValue(in.Resource.Kind),
			Name: types.StringValue(in.Resource.Name),
		}
	}
	if in.Service != nil {
		out.Service = &IngressV1ServiceBackendModel{
			Name: types.StringValue(in.Service.Name),
			Port: &IngressV1PortModel{
				Name: types.StringValue(in.Service.Port.Name), Number: types.Int64Value(int64(in.Service.Port.Number)),
			},
		}
	}
	return &out
}

func ingressFlattenStatus(ctx context.Context, in networking.IngressLoadBalancerStatus) (types.List, diag.Diagnostics) {
	type endpoint struct {
		IP       types.String `tfsdk:"ip"`
		Hostname types.String `tfsdk:"hostname"`
	}
	type loadBalancer struct {
		Ingress []endpoint `tfsdk:"ingress"`
	}
	type status struct {
		LoadBalancer []loadBalancer `tfsdk:"load_balancer"`
	}
	endpoints := make([]endpoint, len(in.Ingress))
	for i, value := range in.Ingress {
		endpoints[i] = endpoint{IP: types.StringValue(value.IP), Hostname: types.StringValue(value.Hostname)}
	}
	return types.ListValueFrom(ctx, ingressStatusType(), []status{{LoadBalancer: []loadBalancer{{Ingress: endpoints}}}})
}
