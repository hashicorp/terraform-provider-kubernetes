package kubernetes

import (
	"errors"
	"net/url"

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
			Optional:    true,
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
			ValidateFunc: func(v interface{}, k string) ([]string, []error) {
				u, err := url.Parse(v.(string))
				if err != nil {
					return nil, []error{err}
				}

				errs := []error{}

				if u.Scheme != "https" {
					errs = append(errs, errors.New("url: scheme must be https"))
				}

				if u.Host == "" {
					errs = append(errs, errors.New("url: host must be provided"))
				}

				if u.User != nil {
					errs = append(errs, errors.New("url: user info is not permitted"))
				}

				if u.Fragment != "" || u.RawQuery != "" {
					errs = append(errs, errors.New("url: fragments and query parameters are not permitted"))
				}

				return nil, errs
			},
		},
	}
}
