package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	apiv1 "k8s.io/api/authentication/v1"
)

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
			Description: apiv1.TokenRequest{}.Spec.SwaggerDoc()["boundObjectRef"],
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"apiVersion": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: apiv1.TokenRequest{}.Spec.BoundObjectRef.SwaggerDoc()["apiVersion"],
					},
					"kind": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: apiv1.TokenRequest{}.Spec.BoundObjectRef.SwaggerDoc()["kind"],
					},
					"name": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: apiv1.TokenRequest{}.Spec.BoundObjectRef.SwaggerDoc()["name"],
					},
					"uid": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: apiv1.TokenRequest{}.Spec.BoundObjectRef.SwaggerDoc()["uid"],
					},
				},
			},
		},
		"expirationSeconds": {
			Type:        schema.TypeInt,
			Optional:    true,
			Description: apiv1.TokenRequest{}.Spec.SwaggerDoc()["expirationSeconds"],
		},
	}
	return s
}
