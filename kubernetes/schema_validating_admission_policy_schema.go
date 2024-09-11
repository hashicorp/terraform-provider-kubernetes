package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func auditAnnotationsFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"key": schema.StringAttribute{
			Description: "key specifies the audit annotation key.",
			Required:    true,
		},
		"value_expressions": schema.StringAttribute{
			Description: "valueExpression represents the expression which is evaluated by CEL to produce an audit annotation value.",
			Required:    true,
		},
	}
}

func matchConditionsFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"expression": schema.StringAttribute{
			Description: "Expression represents the expression which will be evaluated by CEL.",
			Required:    true,
		},
		"name": schema.StringAttribute{
			Description: "Name is an identifier for this match condition, used for strategic merging of MatchConditions, as well as providing an identifier for logging purposes.",
			Required:    true,
		},
	}
}

func matchConstraintsFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"exclude_resource_rules": schema.ListNestedAttribute{
			Description: "ExcludeResourceRules describes what operations on what resources/subresources the ValidatingAdmissionPolicy should not care about.",
			Optional:    true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: namedRuleWithOperationsFields(),
			},
		},
		"matchPolicy": schema.StringAttribute{
			Optional:    true,
			Description: "matchPolicy defines how the MatchResources list is used to match incoming requests. Allowed values are Exact or Equivalent.",
			Validators: []validator.String{
				stringvalidator.OneOf("Exact", "Equivalent"),
			},
		},
		"namespace_selector": schema.SingleNestedAttribute{
			Description: "NamespaceSelector decides whether to run the admission control policy on an object based on whether the namespace for that object matches the selector.",
			Optional:    true,
			Attributes:  namedRuleWithOperationsFields(),
		},
		"object_selector": schema.SingleNestedAttribute{
			Description: "ObjectSelector decides whether to run the validation based on if the object has matching labels.",
			Optional:    true,
			Attributes:  labelSelectorFields(true),
		},
		"resource_rules": schema.ListNestedAttribute{
			Description: "ResourceRules describes what operations on what resources/subresources the ValidatingAdmissionPolicy matches.",
			Optional:    true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: namedRuleWithOperationsFields(),
			},
		},
	}
}

func paramKindFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"api_version": schema.StringAttribute{
			Description: "APIVersion is the API group version the resources belong to. In format of \"group/version\"",
			Required:    true,
		},
		"kind": schema.StringAttribute{
			Description: "Kind is the API kind the resources belong to.",
			Required:    true,
		},
	}
}

func validationFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"expression": schema.StringAttribute{
			Description: "Expression represents the expression which will be evaluated by CEL.",
			Required:    true,
		},
		"message": schema.StringAttribute{
			Description: "Message represents the message displayed when validation fails.",
			Required:    true,
		},
		"message_expression": schema.StringAttribute{
			Description: "Message Expression declares a CEL expression that evaluates to the validation failure message that is returned when this rule fails.",
			Optional:    true,
		},
		"reason": schema.StringAttribute{
			Description: "Reason represents a machine-readable description of why this validation failed.",
			Optional:    true,
		},
	}
}

func variableFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"expression": schema.StringAttribute{
			Description: "Expression is the expression that will be evaluated as the value of the variable.",
			Optional:    true,
		},
		"name": schema.StringAttribute{
			Description: "Name is the name of the variable.",
			Optional:    true,
		},
	}
}

func namedRuleWithOperationsFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"api_groups": schema.ListAttribute{
			Description: "APIGroups is the API groups the resources belong to. '\\*' is all groups. If '\\*' is present, the length of the slice must be one.",
			Required:    true,
			ElementType: types.StringType,
		},
		"api_versions": schema.ListAttribute{
			Description: "APIVersions is the API versions the resources belong to. '\\*' is all versions. If '\\*' is present, the length of the slice must be one. Required.",
			Required:    true,
			ElementType: types.StringType,
		},
		"operations": schema.ListAttribute{
			Description: "Operations is the operations the admission hook cares about - CREATE, UPDATE, DELETE, CONNECT or * for all of those operations and any future admission operations that are added.",
			Required:    true,
			ElementType: types.StringType,
		},
		"resource_names": schema.ListAttribute{
			Description: "ResourceNames is an optional white list of names that the rule applies to. An empty set means that everything is allowed.",
			Optional:    true,
			ElementType: types.StringType,
		},
		"resources": schema.ListAttribute{
			Description: "Resources is a list of resources this rule applies to.",
			Required:    true,
			ElementType: types.StringType,
		},
		"scope": schema.StringAttribute{
			Description: "scope specifies the scope of this rule.",
			Optional:    true,
		},
	}
}
