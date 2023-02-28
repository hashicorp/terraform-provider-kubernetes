package kubernetes

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func tokenRequestSpecFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"audiences": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Optional pod scheduling constraints.",
		},
		"boundObjectRef": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "Optional pod scheduling constraints.",
			// Elem: &schema.Resource{
			// 	Schema: (),
			// },
		},
		"expirationSeconds": {},
	}
	return s
}

func tokenRequestStatusFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"expirationTimestamp": {
			Type:        schema.TypeList,
			MaxItems:    1,
			Optional:    true,
			Description: "Optional pod scheduling constraints.",
		},
		"token": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Optional pod scheduling constraints.",
			// Elem: &schema.Resource{
			// 	Schema: (),
			// },
		},
	}
	return s
}
