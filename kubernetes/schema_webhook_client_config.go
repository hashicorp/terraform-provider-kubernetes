package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
)

func serviceReferenceFields() map[string]*schema.Schema {
	apiDoc := admissionregistrationv1.ServiceReference{}.SwaggerDoc()
	return map[string]*schema.Schema{
		"name": {
			Type:        schema.TypeString,
			Description: apiDoc["name"],
			Required:    true,
		},
		"namespace": {
			Type:        schema.TypeString,
			Description: apiDoc["namespace"],
			Required:    true,
		},
		"path": {
			Type:        schema.TypeString,
			Description: apiDoc["path"],
			Optional:    true,
		},
		"port": {
			Type:        schema.TypeInt,
			Description: apiDoc["port"],
			Optional:    true,
			Default:     443,
		},
	}
}

func webhookClientConfigFields() map[string]*schema.Schema {
	apiDoc := admissionregistrationv1.WebhookClientConfig{}.SwaggerDoc()
	return map[string]*schema.Schema{
		"ca_bundle": {
			Type:        schema.TypeString,
			Description: apiDoc["caBundle"],
			Optional:    true,
		},
		"service": {
			Type:        schema.TypeList,
			Required:    true,
			Description: apiDoc["service"],
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: serviceReferenceFields(),
			},
		},
		"url": {
			Type:        schema.TypeString,
			Description: apiDoc["url"],
			Optional:    true,
		},
	}
}
