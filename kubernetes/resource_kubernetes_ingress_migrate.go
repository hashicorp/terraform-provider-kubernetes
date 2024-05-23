// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	networking "k8s.io/api/networking/v1beta1"
)

// resourceKubernetesIngressV0 is a copy of the Kubernetes Ingress schema (before migration).
func resourceKubernetesIngressV0() *schema.Resource {
	docHTTPIngressPath := networking.HTTPIngressPath{}.SwaggerDoc()
	docHTTPIngressRuleValue := networking.HTTPIngressPath{}.SwaggerDoc()
	docIngress := networking.Ingress{}.SwaggerDoc()
	docIngressTLS := networking.IngressTLS{}.SwaggerDoc()
	docIngressRule := networking.IngressRule{}.SwaggerDoc()
	docIngressSpec := networking.IngressSpec{}.SwaggerDoc()
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("ingress", true),
			"spec": {
				Type:        schema.TypeList,
				Description: docIngress["spec"],
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ingress_class_name": {
							Type:        schema.TypeString,
							Description: docIngressSpec["ingressClassName"],
							Optional:    true,
						},
						"backend": backendSpecFields(defaultBackendDescription),
						"rule": {
							Type:        schema.TypeList,
							Description: docIngress["rules"],
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"host": {
										Type:        schema.TypeString,
										Description: docIngressRule["host"],
										Optional:    true,
									},
									"http": {
										Type:        schema.TypeList,
										Required:    true,
										MaxItems:    1,
										Description: "http is a list of http selectors pointing to backends. In the example: http:///? -> backend where where parts of the url correspond to RFC 3986, this resource will be used to match against everything after the last '/' and before the first '?' or '#'.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"path": {
													Type:        schema.TypeList,
													Required:    true,
													Description: docHTTPIngressRuleValue["paths"],
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"path": {
																Type:        schema.TypeString,
																Description: docHTTPIngressPath["path"],
																Optional:    true,
															},
															"backend": backendSpecFields(ruleBackedDescription),
														},
													},
												},
											},
										},
									},
								},
							},
						},
						"tls": {
							Type:        schema.TypeList,
							Description: docIngressSpec["tls"],
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"hosts": {
										Type:        schema.TypeList,
										Description: docIngressTLS["hosts"],
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"secret_name": {
										Type:        schema.TypeString,
										Description: docIngressTLS["secretName"],
										Optional:    true,
									},
								},
							},
						},
					},
				},
			},
			"status": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"load_balancer": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ingress": {
										Type:     schema.TypeList,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"ip": {
													Type:     schema.TypeString,
													Computed: true,
												},
												"hostname": {
													Type:     schema.TypeString,
													Computed: true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"wait_for_load_balancer": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Terraform will wait for the load balancer to have at least 1 endpoint before considering the resource created.",
			},
		},
	}
}

func resourceKubernetesIngressStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	log.Println("[INFO] Found Kubernetes Service state v0; upgrading state to v1")
	delete(rawState, "load_balancer_ingress")
	// Return a nil error here to satisfy StateUpgradeFunc signature
	return rawState, nil
}
