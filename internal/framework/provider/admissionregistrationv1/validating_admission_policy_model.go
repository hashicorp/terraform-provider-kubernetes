package admissionregistrationv1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ValidatingAdmissionPolicyModel struct {
	Timeouts timeouts.Value  `tfsdk:"timeouts"`
	ID       types.String    `tfsdk:"id" manifest:""`
	Metadata MetadataModel   `tfsdk:"metadata" manifest:"metadata"`
	Spec     VAPSpecModel    `tfsdk:"spec" manifest:"spec"`
	Status   *VAPStatusModel `tfsdk:"status" manifest:"status"`
}

type MetadataModel struct {
	Annotations     map[string]types.String `tfsdk:"annotations" manifest:"annotations"`
	GenerateName    types.String            `tfsdk:"generate_name" manifest:"generateName"`
	Generation      types.Int64             `tfsdk:"generation" manifest:"generation"`
	Labels          map[string]types.String `tfsdk:"labels" manifest:"labels"`
	Name            types.String            `tfsdk:"name" manifest:"name"`
	Namespace       types.String            `tfsdk:"namespace" manifest:"namespace"`
	ResourceVersion types.String            `tfsdk:"resource_version" manifest:"resourceVersion"`
	UID             types.String            `tfsdk:"uid" manifest:"uid"`
}

type VAPSpecModel struct {
	AuditAnnotations []AuditAnnotationModel `tfsdk:"audit_annotations" manifest:"auditAnnotations"`
	FailurePolicy    types.String           `tfsdk:"failure_policy" manifest:"failurePolicy"`
	MatchConditions  []MatchConditionModel  `tfsdk:"match_conditions" manifest:"matchConditions"`
	MatchConstraints MatchConstraintsModel  `tfsdk:"match_constraints" manifest:"matchConstraints"`
	ParamKind        *ParamKindModel        `tfsdk:"param_kind" manifest:"paramKind"`
	Validations      []ValidationModel      `tfsdk:"validations" manifest:"validations"`
	Variables        []VariableModel        `tfsdk:"variables" manifest:"variables"`
}

type AuditAnnotationModel struct {
	Key             types.String `tfsdk:"key" manifest:"key"`
	ValueExpression types.String `tfsdk:"value_expression" manifest:"valueExpression"`
}

type MatchConditionModel struct {
	Expression types.String `tfsdk:"expression" manifest:"expression"`
	Name       types.String `tfsdk:"name" manifest:"name"`
}

type MatchConstraintsModel struct {
	ExcludeResourceRules []RuleWithOperationsModel `tfsdk:"exclude_resource_rules" manifest:"excludeResourceRules"`
	MatchPolicy          types.String              `tfsdk:"match_policy" manifest:"matchPolicy"`
	NamespaceSelector    *LabelSelectorModel       `tfsdk:"namespace_selector" manifest:"namespaceSelector"`
	ObjectSelector       *ObjectSelectorModel      `tfsdk:"object_selector" manifest:"objectSelector"`
	ResourceRules        []RuleWithOperationsModel `tfsdk:"resource_rules" manifest:"resourceRules"`
}

type RuleWithOperationsModel struct {
	APIGroups     []types.String `tfsdk:"api_groups" manifest:"apiGroups"`
	APIVersions   []types.String `tfsdk:"api_versions" manifest:"apiVersions"`
	Operations    []types.String `tfsdk:"operations" manifest:"operations"`
	ResourceNames []types.String `tfsdk:"resource_names" manifest:"resourceNames"`
	Resources     []types.String `tfsdk:"resources" manifest:"resources"`
	Scope         types.String   `tfsdk:"scope" manifest:"scope"`
}

type LabelSelectorModel struct {
	MatchLabels      types.Map                       `tfsdk:"match_labels" manifest:"matchLabels"`
	MatchExpressions []LabelSelectorRequirementModel `tfsdk:"match_expressions" manifest:"matchExpressions"`
}

type LabelSelectorRequirementModel struct {
	Key      types.String   `tfsdk:"key" manifest:"key"`
	Operator types.String   `tfsdk:"operator" manifest:"operator"`
	Values   []types.String `tfsdk:"values" manifest:"values"`
}

type ObjectSelectorModel struct {
	LabelSelector LabelSelectorModel `tfsdk:"label_selector" manifest:"labelSelector"`
}

type ParamKindModel struct {
	APIVersion types.String `tfsdk:"api_version" manifest:"apiVersion"`
	Kind       types.String `tfsdk:"kind" manifest:"kind"`
}

type ValidationModel struct {
	Expression        types.String `tfsdk:"expression" manifest:"expression"`
	Message           types.String `tfsdk:"message" manifest:"message"`
	MessageExpression types.String `tfsdk:"message_expression" manifest:"messageExpression"`
	Reason            types.String `tfsdk:"reason" manifest:"reason"`
}

type VariableModel struct {
	Expression types.String `tfsdk:"expression" manifest:"expression"`
	Name       types.String `tfsdk:"name" manifest:"name"`
}

type VAPStatusModel struct {
	Conditions         []ConditionModel   `tfsdk:"conditions" manifest:"conditions"`
	ObservedGeneration types.Int64        `tfsdk:"observed_generation" manifest:"observedGeneration"`
	TypeChecking       *TypeCheckingModel `tfsdk:"type_checking" manifest:"typeChecking"`
}

type ConditionModel struct {
	LastTransitionTime types.String `tfsdk:"last_transition_time" manifest:"lastTransitionTime"`
	Message            types.String `tfsdk:"message" manifest:"message"`
	ObservedGeneration types.Int64  `tfsdk:"observed_generation" manifest:"observedGeneration"`
	Reason             types.String `tfsdk:"reason" manifest:"reason"`
	Status             types.String `tfsdk:"status" manifest:"status"`
	Type               types.String `tfsdk:"type" manifest:"type"`
}

type TypeCheckingModel struct {
	ExpressionWarning []ExpressionWarningModel `tfsdk:"expression_warning" manifest:"expressionWarning"`
}

type ExpressionWarningModel struct {
	FieldRef types.String `tfsdk:"field_ref" manifest:"fieldRef"`
	Warning  types.String `tfsdk:"warning" manifest:"warning"`
}
