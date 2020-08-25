package kubernetes

import (
	"context"
	"fmt"
	networking "k8s.io/api/networking/v1beta1"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func resourceKubernetesIngress() *schema.Resource {
	docHTTPIngressPath := networking.HTTPIngressPath{}.SwaggerDoc()
	docHTTPIngressRuleValue := networking.HTTPIngressPath{}.SwaggerDoc()
	docIngress := networking.Ingress{}.SwaggerDoc()
	docIngressTLS := networking.IngressTLS{}.SwaggerDoc()
	docIngressRule := networking.IngressRule{}.SwaggerDoc()
	docIngressSpec := networking.IngressSpec{}.SwaggerDoc()
	return &schema.Resource{
		Create: resourceKubernetesIngressCreate,
		Read:   resourceKubernetesIngressRead,
		Exists: resourceKubernetesIngressExists,
		Update: resourceKubernetesIngressUpdate,
		Delete: resourceKubernetesIngressDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("ingress", true),
			"spec": {
				Type:        schema.TypeList,
				Description: docIngress["spec"],
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"backend": backendSpecFields(defaultBackendDescription),
						// FIXME: this field is inconsistent with the k8s API 'rules'
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
												// FIXME: this field is inconsistent with the k8s API 'paths'
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
			"wait_for_load_balancer": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Terraform will wait for the load balancer to have at least 1 endpoint before considering the resource created.",
			},
		},
	}
}

func resourceKubernetesIngressCreate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	ing := &v1beta1.Ingress{
		Spec: expandIngressSpec(d.Get("spec").([]interface{})),
	}
	ing.ObjectMeta = metadata
	log.Printf("[INFO] Creating new ingress: %#v", ing)
	out, err := conn.ExtensionsV1beta1().Ingresses(metadata.Namespace).Create(ctx, ing, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("Failed to create Ingress '%s' because: %s", buildId(ing.ObjectMeta), err)
	}
	log.Printf("[INFO] Submitted new ingress: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	if !d.Get("wait_for_load_balancer").(bool) {
		return resourceKubernetesIngressRead(d, meta)
	}

	log.Printf("[INFO] Waiting for load balancer to become ready: %#v", out)
	return resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		res, err := conn.ExtensionsV1beta1().Ingresses(metadata.Namespace).Get(ctx, metadata.Name, metav1.GetOptions{})
		if err != nil {
			// NOTE it is possible in some HA apiserver setups that are eventually consistent
			// that we could get a 404 when doing a Get immediately after a Create
			if errors.IsNotFound(err) {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}

		if len(res.Status.LoadBalancer.Ingress) > 0 {
			return resource.NonRetryableError(resourceKubernetesIngressRead(d, meta))
		}

		log.Printf("[INFO] Load Balancer not ready yet...")
		return resource.RetryableError(fmt.Errorf("Load Balancer is not ready yet"))
	})
}

func resourceKubernetesIngressRead(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading ingress %s", name)
	ing, err := conn.ExtensionsV1beta1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return fmt.Errorf("Failed to read Ingress '%s' because: %s", buildId(ing.ObjectMeta), err)
	}
	log.Printf("[INFO] Received ingress: %#v", ing)
	err = d.Set("metadata", flattenMetadata(ing.ObjectMeta, d))
	if err != nil {
		return err
	}

	flattened := flattenIngressSpec(ing.Spec)
	log.Printf("[DEBUG] Flattened ingress spec: %#v", flattened)
	err = d.Set("spec", flattened)
	if err != nil {
		return err
	}

	err = d.Set("load_balancer_ingress", flattenLoadBalancerIngress(ing.Status.LoadBalancer.Ingress))
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesIngressUpdate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	namespace, _, err := idParts(d.Id())
	if err != nil {
		return err
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec := expandIngressSpec(d.Get("spec").([]interface{}))

	if metadata.Namespace == "" {
		metadata.Namespace = "default"
	}

	ingress := &v1beta1.Ingress{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	out, err := conn.ExtensionsV1beta1().Ingresses(namespace).Update(ctx, ingress, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("Failed to update Ingress %s because: %s", buildId(ingress.ObjectMeta), err)
	}
	log.Printf("[INFO] Submitted updated ingress: %#v", out)

	return resourceKubernetesIngressRead(d, meta)
}

func resourceKubernetesIngressDelete(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting ingress: %#v", name)
	err = conn.ExtensionsV1beta1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("Failed to delete Ingress %s because: %s", d.Id(), err)
	}

	log.Printf("[INFO] Ingress %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesIngressExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking ingress %s", name)
	_, err = conn.ExtensionsV1beta1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
