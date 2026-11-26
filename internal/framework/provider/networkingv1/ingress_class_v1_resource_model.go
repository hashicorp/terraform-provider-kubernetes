// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	networking "k8s.io/api/networking/v1"
)

type IngressClassV1Model struct {
	ID       types.String              `tfsdk:"id"`
	Metadata []common.MetadataModel    `tfsdk:"metadata"`
	Spec     []IngressClassV1SpecModel `tfsdk:"spec"`
}

type IngressClassV1SpecModel struct {
	Controller types.String                   `tfsdk:"controller"`
	Parameters *IngressClassV1ParametersModel `tfsdk:"parameters"`
}

type IngressClassV1ParametersModel struct {
	APIGroup  types.String `tfsdk:"api_group"`
	Kind      types.String `tfsdk:"kind"`
	Name      types.String `tfsdk:"name"`
	Scope     types.String `tfsdk:"scope"`
	Namespace types.String `tfsdk:"namespace"`
}

func expandIngressClassSpec(model IngressClassV1SpecModel) networking.IngressClassSpec {
	spec := networking.IngressClassSpec{Controller: model.Controller.ValueString()}
	if model.Parameters == nil {
		return spec
	}
	p := model.Parameters
	spec.Parameters = &networking.IngressClassParametersReference{
		Kind: p.Kind.ValueString(),
		Name: p.Name.ValueString(),
	}
	if !p.APIGroup.IsNull() && !p.APIGroup.IsUnknown() && p.APIGroup.ValueString() != "" {
		value := p.APIGroup.ValueString()
		spec.Parameters.APIGroup = &value
	}
	if !p.Scope.IsNull() && !p.Scope.IsUnknown() && p.Scope.ValueString() != "" {
		value := p.Scope.ValueString()
		spec.Parameters.Scope = &value
	}
	if !p.Namespace.IsNull() && !p.Namespace.IsUnknown() && p.Namespace.ValueString() != "" {
		value := p.Namespace.ValueString()
		spec.Parameters.Namespace = &value
	}
	return spec
}

func flattenIngressClassSpec(spec networking.IngressClassSpec) []IngressClassV1SpecModel {
	model := IngressClassV1SpecModel{
		Controller: types.StringValue(spec.Controller),
		Parameters: nil,
	}
	if spec.Parameters != nil {
		p := spec.Parameters
		parameters := IngressClassV1ParametersModel{
			APIGroup:  types.StringValue(""),
			Kind:      types.StringValue(p.Kind),
			Name:      types.StringValue(p.Name),
			Scope:     types.StringValue(""),
			Namespace: types.StringValue(""),
		}
		if p.APIGroup != nil {
			parameters.APIGroup = types.StringValue(*p.APIGroup)
		}
		if p.Scope != nil {
			parameters.Scope = types.StringValue(*p.Scope)
		}
		if p.Namespace != nil {
			parameters.Namespace = types.StringValue(*p.Namespace)
		}
		model.Parameters = &parameters
	}
	return []IngressClassV1SpecModel{model}
}
