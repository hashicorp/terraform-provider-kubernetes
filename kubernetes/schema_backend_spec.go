package kubernetes

import "github.com/hashicorp/terraform/helper/schema"

func backendSpecFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"service_name": {
			Type:        schema.TypeString,
			Description: "Specifies the name of the referenced service.",
			Optional:    true,
		},
		"service_port": {
			Type:        schema.TypeInt,
			Description: "Specifies the port of the referenced service.",
			Computed:    true,
			Optional:    true,
		},
	}

	return s
}
