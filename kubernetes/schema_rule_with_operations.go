package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
)

func ruleWithOperationsFields() map[string]*schema.Schema {
	apiDoc := admissionregistrationv1.RuleWithOperations{}.SwaggerDoc()
	return map[string]*schema.Schema{
		"api_groups": {
			Type:        schema.TypeList,
			Description: apiDoc["apiGroups"],
			Required:    true,
			MinItems:    1,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"api_versions": {
			Type:        schema.TypeList,
			Description: apiDoc["apiVersions"],
			Required:    true,
			MinItems:    1,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"operations": {
			Type:        schema.TypeList,
			Description: apiDoc["operations"],
			Required:    true,
			MinItems:    1,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"resources": {
			Type:        schema.TypeList,
			Description: apiDoc["resources"],
			Required:    true,
			MinItems:    1,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"scope": {
			Type:        schema.TypeString,
			Description: apiDoc["scope"],
			Optional:    true,
			Default:     "*",
		},
	}
}
