package appsv1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ValidatingAdmissionPolicyModel struct {
	Timeouts timeouts.Value `tfsdk:"timeouts"`

	ID types.String `tfsdk:"id" manifest:""`

	Metadata struct {
		Annotations     map[string]types.String `tfsdk:"annotations" manifest:"annotations"`
		GenerateName    types.String            `tfsdk:"generate_name" manifest:"generateName"`
		Generation      types.Int64             `tfsdk:"generation" manifest:"generation"`
		Labels          map[string]types.String `tfsdk:"labels" manifest:"labels"`
		Name            types.String            `tfsdk:"name" manifest:"name"`
		Namespace       types.String            `tfsdk:"namespace" manifest:"namespace"`
		ResourceVersion types.String            `tfsdk:"resource_version" manifest:"resourceVersion"`
		UID             types.String            `tfsdk:"uid" manifest:"uid"`
	} `tfsdk:"metadata" manifest:"metadata"`

	Spec struct {
		AuditAnnotations []struct {
			Key             types.String `tfsdk:"key" manifest:"key"`
			ValueExpression types.String `tfsdk:"value_expression" manifest:"valueExpression"`
		} `tfsdk:"audit_annotations" manifest:"auditAnnotations"`
		FailurePolicy   types.String `tfsdk:"failure_policy" manifest:"failurePolicy"`
		MatchConditions []struct {
			Expression types.String `tfsdk:"expression" manifest:"expression"`
			Name       types.String `tfsdk:"name" manifest:"name"`
		} `tfsdk:"match_conditions" manifest:"matchConditions"`
		MatchConstraints struct {
			ExcludeResourceRules []struct {
				APIGroups     []types.String `tfsdk:"api_groups" manifest:"apiGroups"`
				APIVersions   []types.String `tfsdk:"api_versions" manifest:"apiVersions"`
				Operations    []types.String `tfsdk:"operations" manifest:"operations"`
				ResourceNames []types.String `tfsdk:"resource_names" manifest:"resourceNames"`
				Resources     []types.String `tfsdk:"resources" manifest:"resources"`
				Scope         types.String   `tfsdk:"scope" manifest:"scope"`
			} `tfsdk:"exclude_resource_rules" manifest:"excludeResourceRules"`
			MatchPolicy       types.String `tfsdk:"match_policy" manifest:"matchPolicy"`
			NamespaceSelector *struct {
				MatchLabels      types.Map `tfsdk:"match_labels" manifest:"matchLabels"`
				MatchExpressions []struct {
					Key      types.String   `tfsdk:"key" manifest:"key"`
					Operator types.String   `tfsdk:"operator" manifest:"operator"`
					Values   []types.String `tfsdk:"values" manifest:"values"`
				} `tfsdk:"match_expressions" manifest:"matchExpressions"`
			} `tfsdk:"namespace_selector" manifest:"namespaceSelector"`
			ObjectSelector *struct {
				LabelSelector struct {
					MatchExpressions []struct {
						Key      types.String   `tfsdk:"key" manifest:"key"`
						Operator types.String   `tfsdk:"operator" manifest:"operator"`
						Values   []types.String `tfsdkxwww:"values" manifest:"values"`
					} `tfsdk:"match_expressions" manifest:"matchExpressions"`
					MatchLabels types.Map `tfsdk:"match_labels" manifest:"matchLabels"`
				} `tfsdk:"label_selector" manifest:"labelSelector"`
			} `tfsdk:"object_selector" manifest:"objectSelector"`
			ResourceRules []struct {
				APIGroups     []types.String `tfsdk:"api_groups" manifest:"apiGroups"`
				APIVersions   []types.String `tfsdk:"api_versions" manifest:"apiVersions"`
				Operations    []types.String `tfsdk:"operations" manifest:"operations"`
				ResourceNames []types.String `tfsdk:"resource_names" manifest:"resourceNames"`
				Resources     []types.String `tfsdk:"resources" manifest:"resources"`
				Scope         types.String   `tfsdk:"scope" manifest:"scope"`
			} `tfsdk:"resource_rules" manifest:"resourceRules"`
		} `tfsdk:"match_constraints" manifest:"matchConstraints"`
		ParamKind *struct {
			APIVersion types.String `tfsdk:"api_version" manifest:"apiVersion"`
			Kind       types.String `tfsdk:"kind" manifest:"kind"`
		} `tfsdk:"param_kind" manifest:"paramKind"`
		Validations []struct {
			Expression        types.String `tfsdk:"expression" manifest:"expression"`
			Message           types.String `tfsdk:"message" manifest:"message"`
			MessageExpression types.String `tfsdk:"message_expression" manifest:"messageExpression"`
			Reason            types.String `tfsdk:"reason" manifest:"reason"`
		} `tfsdk:"validations" manifest:"validations"`
		Variables []struct {
			Expression types.String `tfsdk:"expression" manifest:"expression"`
			Name       types.String `tfsdk:"name" manifest:"name"`
		} `tfsdk:"variables" manifest:"variables"`
	} `tfsdk:"spec" manifest:"spec"`

	Status *struct {
		Conditions []struct {
			LastTransitionTime types.String `tfsdk:"last_transition_time" manifest:"lastTransitionTime"`
			Message            types.String `tfsdk:"message" manifest:"message"`
			ObservedGeneration types.Int64  `tfsdk:"observed_generation" manifest:"observedGeneration"`
			Reason             types.String `tfsdk:"reason" manifest:"reason"`
			Status             types.String `tfsdk:"status" manifest:"status"`
			Type               types.String `tfsdk:"type" manifest:"type"`
		} `tfsdk:"conditions" manifest:"conditions"`
		ObservedGeneration types.Int64 `tfsdk:"observed_generation" manifest:"observedGeneration"`
		TypeChecking       *struct {
			ExpressionWarning []struct {
				FieldRef types.String `tfsdk:"field_ref" manifest:"fieldRef"`
				Warning  types.String `tfsdk:"warning" manifest:"warning"`
			} `tfsdk:"expression_warning" manifest:"expressionWarning"`
		} `tfsdk:"type_checking" manifest:"typeChecking"`
	} `tfsdk:"status" manifest:"status"`
}
