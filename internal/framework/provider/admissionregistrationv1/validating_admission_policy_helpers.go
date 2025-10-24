// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package admissionregistrationv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	arv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func expandStringMap(m map[string]types.String) map[string]string {
	if m == nil {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		if !v.IsNull() && !v.IsUnknown() {
			result[k] = v.ValueString()
		}
	}
	return result
}

func flattenStringMap(m map[string]string) map[string]types.String {
	if m == nil {
		return nil
	}
	result := make(map[string]types.String, len(m))
	for k, v := range m {
		result[k] = types.StringValue(v)
	}
	return result
}

func expandStringSlice(s []types.String) []string {
	if s == nil {
		return nil
	}
	result := make([]string, 0, len(s))
	for _, v := range s {
		if !v.IsNull() && !v.IsUnknown() {
			result = append(result, v.ValueString())
		}
	}
	return result
}

func flattenStringSlice(s []string) []types.String {
	if s == nil {
		return nil
	}
	result := make([]types.String, len(s))
	for i, v := range s {
		result[i] = types.StringValue(v)
	}
	return result
}

func expandValidatingAdmissionPolicySpec(spec ValidatingAdmissionPolicySpecModel) arv1.ValidatingAdmissionPolicySpec {
	result := arv1.ValidatingAdmissionPolicySpec{}

	if len(spec.AuditAnnotations) > 0 {
		result.AuditAnnotations = make([]arv1.AuditAnnotation, len(spec.AuditAnnotations))
		for i, aa := range spec.AuditAnnotations {
			result.AuditAnnotations[i] = arv1.AuditAnnotation{
				Key:             aa.Key.ValueString(),
				ValueExpression: aa.ValueExpression.ValueString(),
			}
		}
	}

	if !spec.FailurePolicy.IsNull() && !spec.FailurePolicy.IsUnknown() {
		fp := arv1.FailurePolicyType(spec.FailurePolicy.ValueString())
		result.FailurePolicy = &fp
	}

	if len(spec.MatchConditions) > 0 {
		result.MatchConditions = make([]arv1.MatchCondition, len(spec.MatchConditions))
		for i, mc := range spec.MatchConditions {
			result.MatchConditions[i] = arv1.MatchCondition{
				Name:       mc.Name.ValueString(),
				Expression: mc.Expression.ValueString(),
			}
		}
	}

	result.MatchConstraints = expandMatchConstraints(spec.MatchConstraints)

	if spec.ParamKind != nil {
		result.ParamKind = &arv1.ParamKind{
			APIVersion: spec.ParamKind.APIVersion.ValueString(),
			Kind:       spec.ParamKind.Kind.ValueString(),
		}
	}

	if len(spec.Validations) > 0 {
		result.Validations = make([]arv1.Validation, len(spec.Validations))
		for i, v := range spec.Validations {
			validation := arv1.Validation{
				Expression: v.Expression.ValueString(),
			}
			if !v.Message.IsNull() && !v.Message.IsUnknown() {
				validation.Message = v.Message.ValueString()
			}
			if !v.MessageExpression.IsNull() && !v.MessageExpression.IsUnknown() {
				validation.MessageExpression = v.MessageExpression.ValueString()
			}
			// v1beta1 doesn't have Reason field
			result.Validations[i] = validation
		}
	}

	if len(spec.Variables) > 0 {
		result.Variables = make([]arv1.Variable, len(spec.Variables))
		for i, v := range spec.Variables {
			result.Variables[i] = arv1.Variable{
				Name:       v.Name.ValueString(),
				Expression: v.Expression.ValueString(),
			}
		}
	}

	return result
}

func expandMatchConstraints(mc MatchConstraintsModel) *arv1.MatchResources {
	result := &arv1.MatchResources{}

	if len(mc.ExcludeResourceRules) > 0 {
		result.ExcludeResourceRules = make([]arv1.NamedRuleWithOperations, len(mc.ExcludeResourceRules))
		for i, rule := range mc.ExcludeResourceRules {
			result.ExcludeResourceRules[i] = expandNamedRuleWithOperations(rule)
		}
	}

	if !mc.MatchPolicy.IsNull() && !mc.MatchPolicy.IsUnknown() {
		mp := arv1.MatchPolicyType(mc.MatchPolicy.ValueString())
		result.MatchPolicy = &mp
	}

	if mc.NamespaceSelector != nil {
		result.NamespaceSelector = expandLabelSelector(*mc.NamespaceSelector)
	}

	if mc.ObjectSelector != nil {
		result.ObjectSelector = expandLabelSelector(mc.ObjectSelector.LabelSelector)
	}

	if len(mc.ResourceRules) > 0 {
		result.ResourceRules = make([]arv1.NamedRuleWithOperations, len(mc.ResourceRules))
		for i, rule := range mc.ResourceRules {
			result.ResourceRules[i] = expandNamedRuleWithOperations(rule)
		}
	}

	return result
}

func expandNamedRuleWithOperations(rule RuleWithOperationsModel) arv1.NamedRuleWithOperations {
	result := arv1.NamedRuleWithOperations{
		ResourceNames: expandStringSlice(rule.ResourceNames),
		RuleWithOperations: arv1.RuleWithOperations{
			Operations: expandOperations(rule.Operations),
			Rule: arv1.Rule{
				APIGroups:   expandStringSlice(rule.APIGroups),
				APIVersions: expandStringSlice(rule.APIVersions),
				Resources:   expandStringSlice(rule.Resources),
			},
		},
	}

	if !rule.Scope.IsNull() && !rule.Scope.IsUnknown() {
		scope := arv1.ScopeType(rule.Scope.ValueString())
		result.RuleWithOperations.Rule.Scope = &scope
	}

	return result
}

func expandOperations(ops []types.String) []arv1.OperationType {
	if ops == nil {
		return nil
	}
	result := make([]arv1.OperationType, 0, len(ops))
	for _, op := range ops {
		if !op.IsNull() && !op.IsUnknown() {
			result = append(result, arv1.OperationType(op.ValueString()))
		}
	}
	return result
}

func expandLabelSelector(ls LabelSelectorModel) *metav1.LabelSelector {
	result := &metav1.LabelSelector{}

	if !ls.MatchLabels.IsNull() && !ls.MatchLabels.IsUnknown() {
		matchLabels := make(map[string]string)
		ls.MatchLabels.ElementsAs(context.Background(), &matchLabels, false)
		result.MatchLabels = matchLabels
	}

	if len(ls.MatchExpressions) > 0 {
		result.MatchExpressions = make([]metav1.LabelSelectorRequirement, len(ls.MatchExpressions))
		for i, expr := range ls.MatchExpressions {
			result.MatchExpressions[i] = metav1.LabelSelectorRequirement{
				Key:      expr.Key.ValueString(),
				Operator: metav1.LabelSelectorOperator(expr.Operator.ValueString()),
				Values:   expandStringSlice(expr.Values),
			}
		}
	}

	return result
}

func flattenValidatingAdmissionPolicy(obj *arv1.ValidatingAdmissionPolicy, model *ValidatingAdmissionPolicyModel) {
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

	flattenValidatingAdmissionPolicySpec(&obj.Spec, &model.Spec)
}

func flattenValidatingAdmissionPolicySpec(spec *arv1.ValidatingAdmissionPolicySpec, model *ValidatingAdmissionPolicySpecModel) {
	if len(spec.AuditAnnotations) > 0 {
		model.AuditAnnotations = make([]AuditAnnotationModel, len(spec.AuditAnnotations))
		for i, aa := range spec.AuditAnnotations {
			model.AuditAnnotations[i] = AuditAnnotationModel{
				Key:             types.StringValue(aa.Key),
				ValueExpression: types.StringValue(aa.ValueExpression),
			}
		}
	}

	if spec.FailurePolicy != nil {
		model.FailurePolicy = types.StringValue(string(*spec.FailurePolicy))
	}

	if len(spec.MatchConditions) > 0 {
		model.MatchConditions = make([]MatchConditionModel, len(spec.MatchConditions))
		for i, mc := range spec.MatchConditions {
			model.MatchConditions[i] = MatchConditionModel{
				Name:       types.StringValue(mc.Name),
				Expression: types.StringValue(mc.Expression),
			}
		}
	}

	if spec.MatchConstraints != nil {
		flattenMatchConstraints(spec.MatchConstraints, &model.MatchConstraints)
	}

	if spec.ParamKind != nil {
		model.ParamKind = &ParamKindModel{
			APIVersion: types.StringValue(spec.ParamKind.APIVersion),
			Kind:       types.StringValue(spec.ParamKind.Kind),
		}
	}

	if len(spec.Validations) > 0 {
		model.Validations = make([]ValidationModel, len(spec.Validations))
		for i, v := range spec.Validations {
			validation := ValidationModel{
				Expression: types.StringValue(v.Expression),
				Message:    types.StringValue(v.Message),
			}
			if v.MessageExpression != "" {
				validation.MessageExpression = types.StringValue(v.MessageExpression)
			}
			// v1beta1 doesn't have Reason field
			model.Validations[i] = validation
		}
	}

	if len(spec.Variables) > 0 {
		model.Variables = make([]VariableModel, len(spec.Variables))
		for i, v := range spec.Variables {
			model.Variables[i] = VariableModel{
				Name:       types.StringValue(v.Name),
				Expression: types.StringValue(v.Expression),
			}
		}
	}
}

func flattenMatchConstraints(mc *arv1.MatchResources, model *MatchConstraintsModel) {
	if len(mc.ExcludeResourceRules) > 0 {
		model.ExcludeResourceRules = make([]RuleWithOperationsModel, len(mc.ExcludeResourceRules))
		for i, rule := range mc.ExcludeResourceRules {
			model.ExcludeResourceRules[i] = flattenNamedRuleWithOperations(rule)
		}
	}

	if mc.MatchPolicy != nil {
		model.MatchPolicy = types.StringValue(string(*mc.MatchPolicy))
	}

	if mc.NamespaceSelector != nil && (len(mc.NamespaceSelector.MatchLabels) > 0 || len(mc.NamespaceSelector.MatchExpressions) > 0) {
		selector := flattenLabelSelector(mc.NamespaceSelector)
		model.NamespaceSelector = &selector
	}

	if mc.ObjectSelector != nil && (len(mc.ObjectSelector.MatchLabels) > 0 || len(mc.ObjectSelector.MatchExpressions) > 0) {
		selector := flattenLabelSelector(mc.ObjectSelector)
		model.ObjectSelector = &ObjectSelectorModel{
			LabelSelector: selector,
		}
	}

	if len(mc.ResourceRules) > 0 {
		model.ResourceRules = make([]RuleWithOperationsModel, len(mc.ResourceRules))
		for i, rule := range mc.ResourceRules {
			model.ResourceRules[i] = flattenNamedRuleWithOperations(rule)
		}
	}
}

func flattenNamedRuleWithOperations(rule arv1.NamedRuleWithOperations) RuleWithOperationsModel {
	result := RuleWithOperationsModel{
		APIGroups:     flattenStringSlice(rule.RuleWithOperations.Rule.APIGroups),
		APIVersions:   flattenStringSlice(rule.RuleWithOperations.Rule.APIVersions),
		Resources:     flattenStringSlice(rule.RuleWithOperations.Rule.Resources),
		ResourceNames: flattenStringSlice(rule.ResourceNames),
		Operations:    flattenOperations(rule.RuleWithOperations.Operations),
	}

	if rule.RuleWithOperations.Rule.Scope != nil && string(*rule.RuleWithOperations.Rule.Scope) != "*" {
		result.Scope = types.StringValue(string(*rule.RuleWithOperations.Rule.Scope))
	}

	return result
}

func flattenOperations(ops []arv1.OperationType) []types.String {
	if ops == nil {
		return nil
	}
	result := make([]types.String, len(ops))
	for i, op := range ops {
		result[i] = types.StringValue(string(op))
	}
	return result
}

func flattenLabelSelector(ls *metav1.LabelSelector) LabelSelectorModel {
	result := LabelSelectorModel{}

	if len(ls.MatchLabels) > 0 {
		matchLabels := make(map[string]attr.Value, len(ls.MatchLabels))
		for k, v := range ls.MatchLabels {
			matchLabels[k] = types.StringValue(v)
		}
		result.MatchLabels, _ = types.MapValue(types.StringType, matchLabels)
	} else {
		result.MatchLabels = types.MapNull(types.StringType)
	}

	if len(ls.MatchExpressions) > 0 {
		result.MatchExpressions = make([]LabelSelectorRequirementModel, len(ls.MatchExpressions))
		for i, expr := range ls.MatchExpressions {
			result.MatchExpressions[i] = LabelSelectorRequirementModel{
				Key:      types.StringValue(expr.Key),
				Operator: types.StringValue(string(expr.Operator)),
				Values:   flattenStringSlice(expr.Values),
			}
		}
	}

	return result
}
