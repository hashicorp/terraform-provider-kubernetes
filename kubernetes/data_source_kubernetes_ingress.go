package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
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
		Read: dataSourceKubernetesIngressRead,

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("ingress", false),
			"spec": {
				Type:        schema.TypeList,
				Description: docIngress["spec"],
				Computed:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"backend": backendSpecFields(defaultBackendDescription),
						// FIXME: this field is inconsistent with the k8s API 'rules'
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
										MaxItems:    1,
										Description: docIngressRule[""],
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												// FIXME: this field is inconsistent with the k8s API 'paths'
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
			"load_balancer_ingress": {
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
	}
}

func dataSourceKubernetesIngressRead(d *schema.ResourceData, meta interface{}) error {
	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	om := meta_v1.ObjectMeta{
		Namespace: metadata.Namespace,
		Name:      metadata.Name,
	}
	d.SetId(buildId(om))

	return resourceKubernetesIngressRead(d, meta)
}
