// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package admissionregistrationv1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	arv1 "k8s.io/api/admissionregistration/v1"
)

func expandValidatingAdmissionPolicyBindingSpec(spec ValidatingAdmissionPolicyBindingSpecModel) arv1.ValidatingAdmissionPolicyBindingSpec {
	result := arv1.ValidatingAdmissionPolicyBindingSpec{}

	if !spec.PolicyName.IsNull() && !spec.PolicyName.IsUnknown() {
		result.PolicyName = spec.PolicyName.ValueString()
	}

	if spec.MatchResources != nil {
		result.MatchResources = expandMatchConstraints(*spec.MatchResources)
	}

	if spec.ParamRef != nil {
		result.ParamRef = &arv1.ParamRef{
			Name:                    spec.PolicyName.ValueString(),
			Namespace:               spec.ParamRef.Namespace.ValueString(),
			ParameterNotFoundAction: (*arv1.ParameterNotFoundActionType)(spec.ParamRef.ParameterNotFoundAction.ValueStringPointer()),
		}

		if spec.ParamRef.Selector != nil {
			selector := expandLabelSelector(*spec.ParamRef.Selector)
			result.ParamRef.Selector = selector
		}
	}

	if len(spec.ValidationActions) > 0 {
		result.ValidationActions = make([]arv1.ValidationAction, len(spec.ValidationActions))
		for i, v := range spec.ValidationActions {
			result.ValidationActions[i] = arv1.ValidationAction(v.ValueString())
		}
	}

	return result
}

func flattenValidatingAdmissionPolicyBinding(obj *arv1.ValidatingAdmissionPolicyBinding, model *ValidatingAdmissionPolicyBindingModel) {
	model.Metadata.Name = types.StringValue(obj.Name)

	if obj.GenerateName != "" {
		model.Metadata.GenerateName = types.StringValue(obj.GenerateName)
	}
	if obj.Namespace != "" {
		model.Metadata.Namespace = types.StringValue(obj.Namespace)
	}

	model.Metadata.UID = types.StringValue(string(obj.UID))
	model.Metadata.ResourceVersion = types.StringValue(obj.ResourceVersion)
	model.Metadata.Generation = types.Int64Value(obj.Generation)

	if len(obj.Labels) > 0 {
		model.Metadata.Labels = flattenStringMap(obj.Labels)
	}
	if len(obj.Annotations) > 0 {
		model.Metadata.Annotations = flattenStringMap(obj.Annotations)
	}

	flattenValidatingAdmissionPolicyBindingSpec(&obj.Spec, &model.Spec)
}

func flattenValidatingAdmissionPolicyBindingSpec(spec *arv1.ValidatingAdmissionPolicyBindingSpec, model *ValidatingAdmissionPolicyBindingSpecModel) {
	model.PolicyName = types.StringValue(spec.PolicyName)

	if spec.MatchResources != nil {
		flattenMatchConstraints(spec.MatchResources, model.MatchResources)
	}

	if spec.ParamRef != nil {
		model.ParamRef = &ParamRefModel{
			Name:                    types.StringValue(spec.ParamRef.Name),
			Namespace:               types.StringValue(spec.ParamRef.Namespace),
			ParameterNotFoundAction: types.StringValue(string(*spec.ParamRef.ParameterNotFoundAction)),
		}

		if spec.ParamRef.Selector != nil {
			selector := flattenLabelSelector(spec.ParamRef.Selector)
			model.ParamRef.Selector = &selector
		}
	}

	if len(spec.ValidationActions) > 0 {
		model.ValidationActions = make([]types.String, len(spec.ValidationActions))
		for i, v := range spec.ValidationActions {
			model.ValidationActions[i] = types.StringValue(string(v))
		}
	}
}
