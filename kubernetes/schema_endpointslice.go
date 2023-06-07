// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func schemaEndpointSliceSubsetEndpoints() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"addresses": {
				Type:        schema.TypeList,
				Description: "Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
				Required:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"condition": {
				Type:        schema.TypeList,
				Description: "A list of references to secrets in the same namespace to use for pulling any images in pods that reference this Service Account. More info: http://kubernetes.io/docs/user-guide/secrets#manually-specifying-an-imagepullsecret",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ready": {
							Type:        schema.TypeBool,
							Description: "Specification of the desired behavior of the job",
							Optional:    true,
						},
						"serving": {
							Type:        schema.TypeBool,
							Description: "Specification of the desired behavior of the job",
							Optional:    true,
						},
						"terminating": {
							Type:        schema.TypeBool,
							Description: "Specification of the desired behavior of the job",
							Optional:    true,
						},
					},
				},
			},
			"hostname": {
				Type:        schema.TypeString,
				Description: "Host name of this endpoint.",
				Optional:    true,
			},
			"node_name": {
				Type:        schema.TypeString,
				Description: "Node name of this endpoint",
				Optional:    true,
			},
			"target_ref": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem:     schemaObjectReference(),
			},
			"zone": {
				Type:        schema.TypeString,
				Description: "zone is the name of the Zone this endpoint exists in.",
				Optional:    true,
			},
		},
	}
}

func schemaEndpointSliceSubsetPorts() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"port": {
				Type:        schema.TypeInt,
				Description: "port represents the port number of the endpoint.",
				Required:    true,
			},
			"protocol": {
				Type:        schema.TypeString,
				Description: "protocol represents the IP protocol for this port. Must be UDP, TCP, or SCTP. Default is TCP.",
				Optional:    true,
				Default:     "TCP",
			},
			"name": {
				Type:        schema.TypeString,
				Description: "name represents the name of this port. All ports in an EndpointSlice must have a unique name.",
				Optional:    true,
			},
			"app_protocol": {
				Type:        schema.TypeString,
				Description: "The application protocol for this port. This is used as a hint for implementations to offer richer behavior for protocols that they understand. This field follows standard Kubernetes label syntax.",
				Optional:    true,
			},
		},
	}
}

func schemaObjectReference() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "port represents the port number of the endpoint.",
				Required:    true,
			},
			"namespace": {
				Type:        schema.TypeString,
				Description: "protocol represents the IP protocol for this port. Must be UDP, TCP, or SCTP. Default is TCP.",
				Optional:    true,
				Default:     "default",
			},
			"resource_version": {
				Type:        schema.TypeString,
				Description: "name represents the name of this port. All ports in an EndpointSlice must have a unique name.",
				Optional:    true,
			},
			"uid": {
				Type:        schema.TypeString,
				Description: "The application protocol for this port. This is used as a hint for implementations to offer richer behavior for protocols that they understand. This field follows standard Kubernetes label syntax.",
				Optional:    true,
			},
			"field_path": {
				Type:        schema.TypeString,
				Description: "The application protocol for this port. This is used as a hint for implementations to offer richer behavior for protocols that they understand. This field follows standard Kubernetes label syntax.",
				Optional:    true,
			},
		},
	}
}

func hashEndpointSlicePorts() schema.SchemaSetFunc {
	return schema.HashResource(schemaEndpointSliceSubsetPorts())
}

func hashEndpointSliceEndpoints() schema.SchemaSetFunc {
	return schema.HashResource(schemaEndpointSliceSubsetEndpoints())
}
