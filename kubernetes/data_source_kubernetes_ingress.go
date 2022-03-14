package kubernetes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	networking "k8s.io/api/networking/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesIngress() *schema.Resource {
	docHTTPIngressPath := networking.HTTPIngressPath{}.SwaggerDoc()
	docHTTPIngressRuleValue := networking.HTTPIngressPath{}.SwaggerDoc()
	docIngress := networking.Ingress{}.SwaggerDoc()
	docIngressTLS := networking.IngressTLS{}.SwaggerDoc()
	docIngressRule := networking.IngressRule{}.SwaggerDoc()
	docIngressSpec := networking.IngressSpec{}.SwaggerDoc()

	return &schema.Resource{
		ReadContext: dataSourceKubernetesIngressRead,
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("ingress", false),
			"spec": {
				Type:        schema.TypeList,
				Description: docIngress["spec"],
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ingress_class_name": {
							Type:        schema.TypeString,
							Description: docIngressSpec["ingressClassName"],
							Computed:    true,
						},
						"backend": backendSpecFields(defaultBackendDescription),
						"rule": {
							Type:        schema.TypeList,
							Description: docIngressSpec["rules"],
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"host": {
										Type:        schema.TypeString,
										Description: docIngressRule["host"],
										Computed:    true,
									},
									"http": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: docIngressRule[""],
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"path": {
													Type:        schema.TypeList,
													Computed:    true,
													Description: docHTTPIngressRuleValue["paths"],
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"path": {
																Type:        schema.TypeString,
																Description: docHTTPIngressPath["path"],
																Computed:    true,
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
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"hosts": {
										Type:        schema.TypeList,
										Description: docIngressTLS["hosts"],
										Computed:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"secret_name": {
										Type:        schema.TypeString,
										Description: docIngressTLS["secretName"],
										Computed:    true,
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
		},
	}
}

func dataSourceKubernetesIngressRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	om := meta_v1.ObjectMeta{
		Namespace: metadata.Namespace,
		Name:      metadata.Name,
	}
	d.SetId(buildId(om))

	return resourceKubernetesIngressRead(ctx, d, meta)
}
