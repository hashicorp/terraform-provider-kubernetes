package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func serviceReferenceFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": {
			Type:        schema.TypeString,
			Description: "The name of the service.",
			Required:    true,
		},
		"namespace": {
			Type:        schema.TypeString,
			Description: "The namespace of the service.",
			Required:    true,
		},
		"path": {
			Type:        schema.TypeString,
			Description: "An optional URL path which will be sent in any request to this service.",
			Optional:    true,
		},
		"port": {
			Type:        schema.TypeInt,
			Description: "The port on the service that hosting webhook. Default to 443 for backward compatibility. `port` should be a valid port number (1-65535, inclusive).",
			Optional:    true,
			Default:     443,
		},
	}
}

func webhookClientConfigFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"ca_bundle": {
			Type:        schema.TypeString,
			Description: "A PEM encoded CA bundle which will be used to validate the webhook's server certificate. If unspecified, system trust roots on the apiserver are used.",
			Optional:    true,
		},
		"service": {
			Type:        schema.TypeList,
			Required:    true,
			Description: "A reference to the service for this webhook. Either `service` or `url` must be specified. If the webhook is running within the cluster, then you should use `service`.",
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: serviceReferenceFields(),
			},
		},
		"url": {
			Type:        schema.TypeString,
			Description: "Gives the location of the webhook, in standard URL form (`scheme://host:port/path`). Exactly one of `url` or `service` must be specified. The `host` should not refer to a service running in the cluster; use the `service` field instead. The host might be resolved via external DNS in some apiservers (e.g., `kube-apiserver` cannot resolve in-cluster DNS as that would be a layering violation). `host` may also be an IP address. Please note that using `localhost` or `127.0.0.1` as a `host` is risky unless you take great care to run this webhook on all hosts which run an apiserver which might need to make calls to this webhook. Such installs are likely to be non-portable, i.e., not easy to turn up in a new cluster. The scheme must be \"https\"; the URL must begin with \"https://\". A path is optional, and if present may be any string permissible in a URL. You may use the path to pass an arbitrary string to the webhook, for example, a cluster identifier. Attempting to use a user or basic auth e.g. \"user:password@\" is not allowed. Fragments (\"#...\") and query parameters (\"?...\") are not allowed, either.",
			Optional:    true,
		},
	}
}
