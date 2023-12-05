// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	corev1 "k8s.io/api/core/v1"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
)

func schemaEndpointSliceSubsetEndpoints() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"addresses": {
				Type:        schema.TypeList,
				Description: "addresses of this endpoint. The contents of this field are interpreted according to the corresponding EndpointSlice addressType field.",
				Required:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"condition": {
				Type:        schema.TypeList,
				Description: "condition contains information about the current status of the endpoint.",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ready": {
							Type:        schema.TypeBool,
							Description: "ready indicates that this endpoint is prepared to receive traffic, according to whatever system is managing the endpoint.",
							Optional:    true,
						},
						"serving": {
							Type:        schema.TypeBool,
							Description: "serving is identical to ready except that it is set regardless of the terminating state of endpoints.",
							Optional:    true,
						},
						"terminating": {
							Type:        schema.TypeBool,
							Description: "terminating indicates that this endpoint is terminating.",
							Optional:    true,
						},
					},
				},
			},
			"hostname": {
				Type:        schema.TypeString,
				Description: "hostname of this endpoint. This field may be used by consumers of endpoints to distinguish endpoints from each other.",
				Optional:    true,
				ValidateFunc: func(v interface{}, k string) ([]string, []error) {
					hostname := v.(string)
					errs := []error{}
					errLabels := k8svalidation.IsDNS1123Label(hostname)
					for _, e := range errLabels {
						errs = append(errs, errors.New(e))
					}

					return nil, errs
				},
			},
			"node_name": {
				Type:        schema.TypeString,
				Description: "nodeName represents the name of the Node hosting this endpoint. This can be used to determine endpoints local to a Node.",
				Optional:    true,
			},
			"target_ref": {
				Type:        schema.TypeList,
				Description: "targetRef is a reference to a Kubernetes object that represents this endpoint.",
				MaxItems:    1,
				Optional:    true,
				Elem:        schemaObjectReference(),
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
				Type:        schema.TypeString,
				Description: "port represents the port number of the endpoint.",
				Required:    true,
				ValidateFunc: func(value interface{}, key string) ([]string, []error) {
					v, err := strconv.Atoi(value.(string))
					if err != nil {
						return []string{}, []error{fmt.Errorf("%s is not a valid integer", key)}
					}
					return validateNonNegativeInteger(v, key)
				},
			},
			"protocol": {
				Type:        schema.TypeString,
				Description: "protocol represents the IP protocol for this port. Must be UDP, TCP, or SCTP. Default is TCP.",
				Optional:    true,
				Default:     string(corev1.ProtocolTCP),
				ValidateFunc: validation.StringInSlice([]string{
					string(corev1.ProtocolTCP),
					string(corev1.ProtocolUDP),
					string(corev1.ProtocolSCTP),
				}, false),
			},
			"name": {
				Type:        schema.TypeString,
				Description: "name represents the name of this port. All ports in an EndpointSlice must have a unique name.",
				Optional:    true,
			},
			"app_protocol": {
				Type:        schema.TypeString,
				Description: "The application protocol for this port. This is used as a hint for implementations to offer richer behavior for protocols that they understand.",
				Required:    true,
			},
		},
	}
}

func schemaObjectReference() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "Name of the referent.",
				Required:    true,
			},
			"namespace": {
				Type:        schema.TypeString,
				Description: "Namespace of the referent.",
				Optional:    true,
				Default:     "default",
			},
			"resource_version": {
				Type:        schema.TypeString,
				Description: "Specific resourceVersion to which this reference is made, if any.",
				Optional:    true,
			},
			"uid": {
				Type:        schema.TypeString,
				Description: "If referring to a piece of an object instead of an entire object, this string should contain a valid JSON/Go field access statement, such as desiredState.manifest.containers[2].",
				Optional:    true,
			},
			"field_path": {
				Type:        schema.TypeString,
				Description: "If referring to a piece of an object instead of an entire object, this string should contain a valid JSON/Go field access statement, such as desiredState.manifest.containers[2].",
				Optional:    true,
			},
		},
	}
}
