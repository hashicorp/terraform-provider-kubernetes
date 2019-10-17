package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func nodeSelectorRequirementFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"key": {
			Type:        schema.TypeString,
			Description: "The label key that the selector applies to.",
			Required:    true,
			ForceNew:    true,
		},
		"operator": {
			Type:        schema.TypeString,
			Description: "A key's relationship to a set of values. Valid operators ard `In`, `NotIn`, `Exists`, `DoesNotExist`, `Gt`, and `Lt`.",
			Required:    true,
			ForceNew:    true,
		},
		"values": {
			Type:        schema.TypeSet,
			Description: "An array of string values. If the operator is `In` or `NotIn`, the values array must be non-empty. If the operator is `Exists` or `DoesNotExist`, the values array must be empty. This array is replaced during a strategic merge patch.",
			Optional:    true,
			ForceNew:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Set:         schema.HashString,
		},
	}
}

func nodeSelectorTermFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"match_expressions": {
			Type:        schema.TypeList,
			Description: "A list of node selector requirements by node's labels. The requirements are ANDed.",
			Optional:    true,
			ForceNew:    true,
			Elem: &schema.Resource{
				Schema: nodeSelectorRequirementFields(),
			},
		},
		"match_fields": {
			Type:        schema.TypeList,
			Description: "A list of node selector requirements by node's fields. The requirements are ANDed.",
			Optional:    true,
			ForceNew:    true,
			Elem: &schema.Resource{
				Schema: nodeSelectorRequirementFields(),
			},
		},
	}
}
