package kubernetes

import "github.com/hashicorp/terraform/helper/schema"

func backendSpecFields() *schema.Schema {
	s := &schema.Schema{
		Type:        schema.TypeList,
		Description: "A default backend capable of servicing requests that don't match any rule. At least one of 'backend' or 'rules' must be specified. This field is optional to allow the loadbalancer controller or defaulting logic to specify a global default.",
		MaxItems:    1,
		Optional:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
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
			},
		},
	}

	return s
}
