package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func schemaEndpointsSubset() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"address": {
				Type:        schema.TypeSet,
				Description: "IP address which offers the related ports that are marked as ready. These endpoints should be considered safe for load balancers and clients to utilize.",
				Optional:    true,
				MinItems:    1,
				Elem:        schemaEndpointsSubsetAddress(),
				Set:         hashEndpointsSubsetAddress(),
			},
			"not_ready_address": {
				Type:        schema.TypeSet,
				Description: "IP address which offers the related ports but is not currently marked as ready because it have not yet finished starting, have recently failed a readiness check, or have recently failed a liveness check.",
				Optional:    true,
				MinItems:    1,
				Elem:        schemaEndpointsSubsetAddress(),
				Set:         hashEndpointsSubsetAddress(),
			},
			"port": {
				Type:        schema.TypeSet,
				Description: "Port number available on the related IP addresses.",
				Optional:    true,
				MinItems:    1,
				Elem:        schemaEndpointsSubsetPort(),
				Set:         hashEndpointsSubsetPort(),
			},
		},
	}
}

func hashEndpointsSubset() schema.SchemaSetFunc {
	return schema.HashResource(schemaEndpointsSubset())
}

func schemaEndpointsSubsetAddress() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"ip": {
				Type:        schema.TypeString,
				Description: "The IP of this endpoint. May not be loopback (127.0.0.0/8), link-local (169.254.0.0/16), or link-local multicast ((224.0.0.0/24).",
				Required:    true,
			},
			"hostname": {
				Type:        schema.TypeString,
				Description: "The Hostname of this endpoint.",
				Optional:    true,
			},
			"node_name": {
				Type:        schema.TypeString,
				Description: "Node hosting this endpoint. This can be used to determine endpoints local to a node.",
				Optional:    true,
			},
		},
	}
}

func hashEndpointsSubsetAddress() schema.SchemaSetFunc {
	return schema.HashResource(schemaEndpointsSubsetAddress())
}

func schemaEndpointsSubsetPort() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of this port within the endpoint. Must be a DNS_LABEL. Optional if only one Port is defined on this endpoint.",
				Optional:    true,
			},
			"port": {
				Type:        schema.TypeInt,
				Description: "The port that will be exposed by this endpoint.",
				Required:    true,
			},
			"protocol": {
				Type:        schema.TypeString,
				Description: "The IP protocol for this port. Supports `TCP` and `UDP`. Default is `TCP`.",
				Optional:    true,
				Default:     "TCP",
			},
		},
	}
}

func hashEndpointsSubsetPort() schema.SchemaSetFunc {
	return schema.HashResource(schemaEndpointsSubsetPort())
}
