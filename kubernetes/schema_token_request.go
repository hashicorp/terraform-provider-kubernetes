package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	apiv1 "k8s.io/api/authentication/v1"
)

func tokenRequestSpecFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"audiences": {
			Type:        schema.TypeList,
			Optional:    true,
			ForceNew:    true,
			Description: "Optional pod scheduling constraints.",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"boundobjectref": {
			Type:        schema.TypeList,
			Optional:    true,
			ForceNew:    true,
			MaxItems:    1,
			Description: apiv1.TokenRequest{}.Spec.SwaggerDoc()["boundObjectRef"],
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"apiversion": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "test", //apiv1.TokenRequest{}.Spec.BoundObjectRef.SwaggerDoc()["apiVersion"],
					},
					"kind": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "test", //apiv1.TokenRequest{}.Spec.BoundObjectRef.SwaggerDoc()["kind"],
					},
					"name": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "test", //apiv1.TokenRequest{}.Spec.BoundObjectRef.SwaggerDoc()["name"],
					},
					"uid": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "test", //apiv1.TokenRequest{}.Spec.BoundObjectRef.SwaggerDoc()["uid"],
					},
				},
			},
		},
		"expirationseconds": {
			Type:        schema.TypeInt,
			Optional:    true,
			Default:     600,    // must be minimum of 10 minutes for expiration
			Description: "test", //apiv1.TokenRequest{}.Spec.SwaggerDoc()["expirationSeconds"],
		},
	}
	return s
}
