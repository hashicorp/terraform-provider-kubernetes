package kubernetes

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

const defaultBackendDescription = `A default backend capable of servicing requests that don't match any rule. At least one of 'backend' or 'rules' must be specified. This field is optional to allow the loadbalancer controller or defaulting logic to specify a global default.`
const ruleBackedDescription = `Backend defines the referenced service endpoint to which the traffic will be forwarded to.`

func backendSpecFields(description string) *schema.Schema {
	s := &schema.Schema{
		Type:        schema.TypeList,
		Description: description,
		MaxItems:    1,
		Optional:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"service": {
					Type:        schema.TypeList,
					Description: "Specifies the name of the referenced service.",
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"name": {
								Type:        schema.TypeString,
								Description: "Specifies the name of the referenced service.",
								Optional:    true,
							},
							"port": {
								Type:        schema.TypeList,
								Description: "Specifies the port of the referenced service.",
								Computed:    true,
								Optional:    true,
								MaxItems:    1,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"name": {
											Type:        schema.TypeString,
											Description: "Specifies the port name",
											Optional:    true,
										},
										"port_number": {
											Type:        schema.TypeString,
											Description: "Specifies the port number",
											Optional:    true,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return s
}
