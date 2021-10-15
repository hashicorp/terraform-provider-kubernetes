package kubernetes

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

const defaultBackendDescriptionV1 = `A default backend capable of servicing requests that don't match any rule. At least one of 'backend' or 'rules' must be specified. This field is optional to allow the loadbalancer controller or defaulting logic to specify a global default.`
const ruleBackedDescriptionV1 = `Backend defines the referenced service endpoint to which the traffic will be forwarded to.`

func backendSpecFieldsV1(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Description: description,
		MaxItems:    1,
		Optional:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"service": {
					Type:     schema.TypeList,
					MaxItems: 1,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"name": {
								Type:        schema.TypeString,
								Description: "Specifies the name of the referenced service.",
								Required:    true,
							},
							"port": {
								Type:        schema.TypeList,
								Description: "Specifies the port of the referenced service.",
								MaxItems:    1,
								Required:    true,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"number": {
											Type:        schema.TypeInt,
											Description: "Specifies the numerical port of the referenced service.",
											Optional:    true,
										},
										"name": {
											Type:        schema.TypeInt,
											Description: "Specifies the name of the port of the referenced service.",
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
}
