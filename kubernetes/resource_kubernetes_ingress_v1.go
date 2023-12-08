// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	networking "k8s.io/api/networking/v1"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func resourceKubernetesIngressV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesIngressV1Create,
		ReadContext:   resourceKubernetesIngressV1Read,
		UpdateContext: resourceKubernetesIngressV1Update,
		DeleteContext: resourceKubernetesIngressV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: resourceKubernetesIngressV1Schema(),
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},
	}
}

func resourceKubernetesIngressV1Schema() map[string]*schema.Schema {
	docHTTPIngressPath := networking.HTTPIngressPath{}.SwaggerDoc()
	docHTTPIngressRuleValue := networking.HTTPIngressPath{}.SwaggerDoc()
	docIngress := networking.Ingress{}.SwaggerDoc()
	docIngressTLS := networking.IngressTLS{}.SwaggerDoc()
	docIngressRule := networking.IngressRule{}.SwaggerDoc()
	docIngressSpec := networking.IngressSpec{}.SwaggerDoc()

	return map[string]*schema.Schema{
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
						Computed:    true,
					},
					"default_backend": backendSpecFieldsV1(defaultBackendDescriptionV1),
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
									Optional:    true,
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
	}
}

func resourceKubernetesIngressV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	ing := &networking.Ingress{
		Spec: expandIngressV1Spec(d.Get("spec").([]interface{})),
	}
	ing.ObjectMeta = metadata
	log.Printf("[INFO] Creating new ingress: %#v", ing)
	out, err := conn.NetworkingV1().Ingresses(metadata.Namespace).Create(ctx, ing, metav1.CreateOptions{})
	if err != nil {
		return diag.Errorf("Failed to create Ingress '%s' because: %s", buildId(ing.ObjectMeta), err)
	}
	log.Printf("[INFO] Submitted new ingress: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	if !d.Get("wait_for_load_balancer").(bool) {
		return resourceKubernetesIngressV1Read(ctx, d, meta)
	}

	log.Printf("[INFO] Waiting for load balancer to become ready: %#v", out)
	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *retry.RetryError {
		res, err := conn.NetworkingV1().Ingresses(metadata.Namespace).Get(ctx, metadata.Name, metav1.GetOptions{})
		if err != nil {
			// NOTE it is possible in some HA apiserver setups that are eventually consistent
			// that we could get a 404 when doing a Get immediately after a Create
			if errors.IsNotFound(err) {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}

		if len(res.Status.LoadBalancer.Ingress) > 0 {
			diagnostics := resourceKubernetesIngressV1Read(ctx, d, meta)
			if diagnostics.HasError() {
				errmsg := diagnostics[0].Summary
				return retry.NonRetryableError(fmt.Errorf("Error reading ingress: %v", errmsg))
			}
			return nil
		}

		log.Printf("[INFO] Load Balancer not ready yet...")
		return retry.RetryableError(fmt.Errorf("Load Balancer is not ready yet"))
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.Diagnostics{}
}

func resourceKubernetesIngressV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesIngressV1Exists(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return diag.Diagnostics{}
	}
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading ingress %s", name)
	ing, err := conn.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.Errorf("Failed to read Ingress '%s' because: %s", buildId(ing.ObjectMeta), err)
	}
	log.Printf("[INFO] Received ingress: %#v", ing)
	err = d.Set("metadata", flattenMetadata(ing.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	flattened := flattenIngressV1Spec(ing.Spec)
	log.Printf("[DEBUG] Flattened ingress spec: %#v", flattened)
	err = d.Set("spec", flattened)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("status", []interface{}{
		map[string][]interface{}{
			"load_balancer": flattenIngressV1Status(ing.Status.LoadBalancer),
		},
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesIngressV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, _, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec := expandIngressV1Spec(d.Get("spec").([]interface{}))

	if metadata.Namespace == "" {
		metadata.Namespace = "default"
	}

	ingress := &networking.Ingress{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	out, err := conn.NetworkingV1().Ingresses(namespace).Update(ctx, ingress, metav1.UpdateOptions{})
	if err != nil {
		return diag.Errorf("Failed to update Ingress %s because: %s", buildId(ingress.ObjectMeta), err)
	}
	log.Printf("[INFO] Submitted updated ingress: %#v", out)

	return resourceKubernetesIngressV1Read(ctx, d, meta)
}

func resourceKubernetesIngressV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting ingress: %#v", name)
	err = conn.NetworkingV1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return diag.Errorf("Failed to delete Ingress %s because: %s", d.Id(), err)
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		_, err := conn.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		e := fmt.Errorf("Ingress (%s) still exists", d.Id())
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Ingress %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesIngressV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking ingress %s", name)
	_, err = conn.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
