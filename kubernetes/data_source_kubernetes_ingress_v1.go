package kubernetes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	networking "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesIngressV1() *schema.Resource {
	docHTTPIngressPath := networking.HTTPIngressPath{}.SwaggerDoc()
	docHTTPIngressRuleValue := networking.HTTPIngressPath{}.SwaggerDoc()
	docIngress := networking.Ingress{}.SwaggerDoc()
	docIngressTLS := networking.IngressTLS{}.SwaggerDoc()
	docIngressRule := networking.IngressRule{}.SwaggerDoc()
	docIngressSpec := networking.IngressSpec{}.SwaggerDoc()

	return &schema.Resource{
		ReadContext: dataSourceKubernetesIngressV1Read,
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
						"default_backend": backendSpecFieldsV1(defaultBackendDescriptionV1),
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
															"path_type": {
																Type:        schema.TypeString,
																Description: docHTTPIngressPath["pathType"],
																Optional:    true,
																Default:     string(networking.PathTypeImplementationSpecific),
																ValidateFunc: validation.StringInSlice([]string{
																	string(networking.PathTypeImplementationSpecific),
																	string(networking.PathTypePrefix),
																	string(networking.PathTypeExact),
																}, false),
															},
															"backend": backendSpecFieldsV1(ruleBackedDescriptionV1),
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

func dataSourceKubernetesIngressV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	om := meta_v1.ObjectMeta{
		Namespace: metadata.Namespace,
		Name:      metadata.Name,
	}
	d.SetId(buildId(om))

	return resourceKubernetesIngressV1Read(ctx, d, meta)
}
