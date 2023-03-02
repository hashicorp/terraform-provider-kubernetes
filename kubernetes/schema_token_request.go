package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	apiv1 "k8s.io/api/authentication/v1"
	api "k8s.io/api/core/v1"
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
				ValidateFunc: validation.Any(
					validation.StringInSlice([]string{api.ClusterIPNone}, false),
					validation.IsIPAddress,
				),
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
			Description: "test", //apiv1.TokenRequest{}.Spec.SwaggerDoc()["expirationSeconds"],
		},
	}
	return s
}
