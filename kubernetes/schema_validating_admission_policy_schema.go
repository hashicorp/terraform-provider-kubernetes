package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func auditAnnotationsFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"key": {
			Type:        schema.TypeString,
			Description: "key specifies the audit annotation key.",
			Required:    true,
		},
		"value_expression": {
			Type:        schema.TypeString,
			Description: "valueExpression represents the expression which is evaluated by CEL to produce an audit annotation value.",
			Required:    true,
		},
	}
}

func matchConditionsFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"expression": {
			Type:        schema.TypeString,
			Description: "Expression represents the expression which will be evaluated by CEL.",
			Required:    true,
		},
		"name": {
			Type:        schema.TypeString,
			Description: "Name is an identifier for this match condition, used for strategic merging of MatchConditions, as well as providing an identifier for logging purposes.",
			Required:    true,
		},
	}
}

func matchConstraintsFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"exclude_resource_rules": {
			Type:        schema.TypeList,
			Description: "ExcludeResourceRules describes what operations on what resources/subresources the ValidatingAdmissionPolicy should not care about.",
			Optional:    true,
			Elem: &schema.Resource{
				Schema: namedRuleWithOperationsFields(),
			},
		},
		"matchPolicy": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "matchPolicy defines how the MatchResources list is used to match incoming requests. Allowed values are Exact or Equivalent.",
			ValidateFunc: validation.StringInSlice([]string{
				"Exact",
				"Equivalent",
			}, false),
		},
		"namespace_selector": {
			Type:        schema.TypeList,
			Description: "NamespaceSelector decides whether to run the admission control policy on an object based on whether the namespace for that object matches the selector.",
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: labelSelectorFields(true),
			},
		},
		"object_selector": {
			Type:        schema.TypeList,
			Description: "ObjectSelector decides whether to run the validation based on if the object has matching labels.",
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: labelSelectorFields(true),
			},
		},
		"resource_rules": {
			Type:        schema.TypeList,
			Description: "ResourceRules describes what operations on what resources/subresources the ValidatingAdmissionPolicy matches.",
			Optional:    true,
			Elem: &schema.Resource{
				Schema: namedRuleWithOperationsFields(),
			},
		},
	}
}

func paramKindFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"api_version": {
			Type:        schema.TypeString,
			Description: "APIVersion is the API group version the resources belong to. In format of \"group/version\"",
			Required:    true,
		},
		"kind": {
			Type:        schema.TypeString,
			Description: "Kind is the API kind the resources belong to.",
			Required:    true,
		},
	}
}

func validationFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"expression": {
			Type:        schema.TypeString,
			Description: "Expression represents the expression which will be evaluated by CEL.",
			Required:    true,
		},
		"message": {
			Type:        schema.TypeString,
			Description: "Message represents the message displayed when validation fails.",
			Required:    true,
		},
		"message_expression": {
			Type:        schema.TypeString,
			Description: "Message Expression declares a CEL expression that evaluates to the validation failure message that is returned when this rule fails.",
			Optional:    true,
		},
		"reason": {
			Type:        schema.TypeString,
			Description: "Reason represents a machine-readable description of why this validation failed.",
			Optional:    true,
		},
	}
}

func variableFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"expression": {
			Type:        schema.TypeString,
			Description: "Expression is the expression that will be evaluated as the value of the variable.",
			Optional:    true,
		},
		"name": {
			Type:        schema.TypeString,
			Description: "Name is the name of the variable.",
			Optional:    true,
		},
	}
}

func namedRuleWithOperationsFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"api_groups": {
			Type:        schema.TypeList,
			Description: "APIGroups is the API groups the resources belong to. '\\*' is all groups. If '\\*' is present, the length of the slice must be one.",
			Required:    true,
			Elem:        schema.TypeString,
		},
		"api_versions": {
			Type:        schema.TypeList,
			Description: "APIVersions is the API versions the resources belong to. '\\*' is all versions. If '\\*' is present, the length of the slice must be one. Required.",
			Required:    true,
			Elem:        schema.TypeString,
		},
		"operations": {
			Type:        schema.TypeList,
			Description: "Operations is the operations the admission hook cares about - CREATE, UPDATE, DELETE, CONNECT or * for all of those operations and any future admission operations that are added.",
			Required:    true,
			Elem:        schema.TypeString,
		},
		"resource_names": {
			Type:        schema.TypeList,
			Description: "ResourceNames is an optional white list of names that the rule applies to. An empty set means that everything is allowed.",
			Optional:    true,
			Elem:        schema.TypeString,
		},
		"resources": {
			Type:        schema.TypeList,
			Description: "Resources is a list of resources this rule applies to.",
			Required:    true,
			Elem:        schema.TypeString,
		},
		"scope": {
			Type:        schema.TypeString,
			Description: "scope specifies the scope of this rule.",
			Optional:    true,
			Default:     "*",
		},
	}
}
