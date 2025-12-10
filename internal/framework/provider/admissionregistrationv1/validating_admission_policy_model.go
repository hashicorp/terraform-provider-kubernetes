// Copyright IBM Corp. 2017, 2025
// SPDX-License-Identifier: MPL-2.0

package admissionregistrationv1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ValidatingAdmissionPolicyModel struct {
	Timeouts timeouts.Value                     `tfsdk:"timeouts"`
	ID       types.String                       `tfsdk:"id"`
	Metadata MetadataModel                      `tfsdk:"metadata"`
	Spec     ValidatingAdmissionPolicySpecModel `tfsdk:"spec"`
}

type MetadataModel struct {
	Annotations     map[string]types.String `tfsdk:"annotations"`
	GenerateName    types.String            `tfsdk:"generate_name"`
	Generation      types.Int64             `tfsdk:"generation"`
	Labels          map[string]types.String `tfsdk:"labels"`
	Name            types.String            `tfsdk:"name"`
	Namespace       types.String            `tfsdk:"namespace"`
	ResourceVersion types.String            `tfsdk:"resource_version"`
	UID             types.String            `tfsdk:"uid"`
}

type ValidatingAdmissionPolicySpecModel struct {
	AuditAnnotations []AuditAnnotationModel `tfsdk:"audit_annotations"`
	FailurePolicy    types.String           `tfsdk:"failure_policy"`
	MatchConditions  []MatchConditionModel  `tfsdk:"match_conditions"`
	MatchConstraints MatchConstraintsModel  `tfsdk:"match_constraints"`
	ParamKind        *ParamKindModel        `tfsdk:"param_kind"`
	Validations      []ValidationModel      `tfsdk:"validations"`
	Variables        []VariableModel        `tfsdk:"variables"`
}

type AuditAnnotationModel struct {
	Key             types.String `tfsdk:"key"`
	ValueExpression types.String `tfsdk:"value_expression"`
}

type MatchConditionModel struct {
	Expression types.String `tfsdk:"expression"`
	Name       types.String `tfsdk:"name"`
}

type MatchConstraintsModel struct {
	ExcludeResourceRules []RuleWithOperationsModel `tfsdk:"exclude_resource_rules"`
	MatchPolicy          types.String              `tfsdk:"match_policy"`
	NamespaceSelector    *LabelSelectorModel       `tfsdk:"namespace_selector"`
	ObjectSelector       *ObjectSelectorModel      `tfsdk:"object_selector"`
	ResourceRules        []RuleWithOperationsModel `tfsdk:"resource_rules"`
}

type RuleWithOperationsModel struct {
	APIGroups     []types.String `tfsdk:"api_groups"`
	APIVersions   []types.String `tfsdk:"api_versions"`
	Operations    []types.String `tfsdk:"operations"`
	ResourceNames []types.String `tfsdk:"resource_names"`
	Resources     []types.String `tfsdk:"resources"`
	Scope         types.String   `tfsdk:"scope"`
}

type LabelSelectorModel struct {
	MatchLabels      types.Map                       `tfsdk:"match_labels"`
	MatchExpressions []LabelSelectorRequirementModel `tfsdk:"match_expressions"`
}

type LabelSelectorRequirementModel struct {
	Key      types.String   `tfsdk:"key"`
	Operator types.String   `tfsdk:"operator"`
	Values   []types.String `tfsdk:"values"`
}

type ObjectSelectorModel struct {
	LabelSelector LabelSelectorModel `tfsdk:"label_selector"`
}

type ParamKindModel struct {
	APIVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`
}

type ValidationModel struct {
	Expression        types.String `tfsdk:"expression"`
	Message           types.String `tfsdk:"message"`
	MessageExpression types.String `tfsdk:"message_expression"`
	Reason            types.String `tfsdk:"reason"`
}

type VariableModel struct {
	Expression types.String `tfsdk:"expression"`
	Name       types.String `tfsdk:"name"`
}

type ValidatingAdmissionPolicyIdentityModel struct {
	APIVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`
	Name       types.String `tfsdk:"name"`
}
