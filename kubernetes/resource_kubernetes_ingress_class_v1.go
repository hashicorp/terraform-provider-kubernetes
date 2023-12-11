// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func resourceKubernetesIngressClassV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesIngressClassV1Create,
		ReadContext:   resourceKubernetesIngressClassV1Read,
		UpdateContext: resourceKubernetesIngressClassV1Update,
		DeleteContext: resourceKubernetesIngressClassV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: resourceKubernetesIngressClassV1Schema(),
	}
}

func resourceKubernetesIngressClassV1Schema() map[string]*schema.Schema {
	docIngressClass := networking.IngressClass{}.SwaggerDoc()
	docIngressClassSpec := networking.IngressClassSpec{}.SwaggerDoc()
	docIngressClassSpecParametes := corev1.TypedLocalObjectReference{}.SwaggerDoc()

	return map[string]*schema.Schema{
		"metadata": metadataSchema("ingress_class_v1", true),
		"spec": {
			Type:        schema.TypeList,
			Description: docIngressClass["spec"],
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"controller": {
						Type:        schema.TypeString,
						Description: docIngressClassSpec["controller"],
						Optional:    true,
					},
					"parameters": {
						Type:        schema.TypeList,
						Description: docIngressClass["parameters"],
						Optional:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"api_group": {
									Type:        schema.TypeString,
									Description: docIngressClassSpecParametes["apiGroup"],
									Optional:    true,
								},
								"kind": {
									Type:        schema.TypeString,
									Description: docIngressClassSpecParametes["kind"],
									Required:    true,
								},
								"name": {
									Type:        schema.TypeString,
									Description: docIngressClassSpecParametes["name"],
									Required:    true,
								},
								"scope": {
									Type:         schema.TypeString,
									Description:  docIngressClassSpecParametes["scope"],
									Optional:     true,
									Computed:     true,
									ValidateFunc: validation.StringInSlice([]string{"Cluster", "Namespace"}, false),
								},
								"namespace": {
									Type:        schema.TypeString,
									Description: docIngressClassSpecParametes["namespace"],
									Optional:    true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesIngressClassV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	ing := &networking.IngressClass{
		Spec: expandIngressClassV1Spec(d.Get("spec").([]interface{})),
	}
	ing.ObjectMeta = metadata
	log.Printf("[INFO] Creating new Ingress Class: %#v", ing)
	out, err := conn.NetworkingV1().IngressClasses().Create(ctx, ing, metav1.CreateOptions{})
	if err != nil {
		return diag.Errorf("Failed to create Ingress Class '%s' because: %s", buildId(ing.ObjectMeta), err)
	}
	log.Printf("[INFO] Submitted new IngressClass: %#v", out)
	d.SetId(out.ObjectMeta.GetName())

	return diag.Diagnostics{}
}

func resourceKubernetesIngressClassV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesIngressClassV1Exists(ctx, d, meta)
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

	name := d.Id()

	log.Printf("[INFO] Reading Ingress Class %s", name)
	ing, err := conn.NetworkingV1().IngressClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.Errorf("Failed to read Ingress Class '%s' because: %s", buildId(ing.ObjectMeta), err)
	}
	log.Printf("[INFO] Received Ingress Class: %#v", ing)
	err = d.Set("metadata", flattenMetadata(ing.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	flattened := flattenIngressClassV1Spec(ing.Spec)
	log.Printf("[DEBUG] Flattened Ingress Class spec: %#v", flattened)
	err = d.Set("spec", flattened)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesIngressClassV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec := expandIngressClassV1Spec(d.Get("spec").([]interface{}))

	if metadata.Namespace == "" {
		metadata.Namespace = "default"
	}

	ingressClass := &networking.IngressClass{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	out, err := conn.NetworkingV1().IngressClasses().Update(ctx, ingressClass, metav1.UpdateOptions{})
	if err != nil {
		return diag.Errorf("Failed to update Ingress Class %s because: %s", buildId(ingressClass.ObjectMeta), err)
	}
	log.Printf("[INFO] Submitted updated Ingress Class: %#v", out)

	return resourceKubernetesIngressClassV1Read(ctx, d, meta)
}

func resourceKubernetesIngressClassV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()

	log.Printf("[INFO] Deleting Ingress Class: %#v", name)
	err = conn.NetworkingV1().IngressClasses().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return diag.Errorf("Failed to delete Ingress Class %s because: %s", d.Id(), err)
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		_, err := conn.NetworkingV1().IngressClasses().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		e := fmt.Errorf("Ingress Class (%s) still exists", d.Id())
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Ingress Class %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesIngressClassV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	name := d.Id()

	log.Printf("[INFO] Checking Ingress Class %s", name)
	_, err = conn.NetworkingV1().IngressClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func expandIngressClassV1Spec(l []interface{}) networking.IngressClassSpec {
	if len(l) == 0 || l[0] == nil {
		return networking.IngressClassSpec{}
	}
	in := l[0].(map[string]interface{})
	obj := networking.IngressClassSpec{}

	if v, ok := in["controller"].(string); ok && len(v) > 0 {
		obj.Controller = v
	}

	if v, ok := in["parameters"].([]interface{}); ok && len(v) > 0 {
		obj.Parameters = expandIngressClassV1Parameters(v)
	}

	return obj
}

func expandIngressClassV1Parameters(l []interface{}) *networking.IngressClassParametersReference {
	if len(l) == 0 || l[0] == nil {
		return &networking.IngressClassParametersReference{}
	}
	in := l[0].(map[string]interface{})
	obj := &networking.IngressClassParametersReference{}

	if v, ok := in["api_group"].(string); ok && v != "" {
		obj.APIGroup = &v
	}

	if v, ok := in["kind"].(string); ok {
		obj.Kind = v
	}

	if v, ok := in["name"].(string); ok {
		obj.Name = v
	}

	if v, ok := in["scope"].(string); ok && v != "" {
		obj.Scope = &v
	}

	if v, ok := in["namespace"].(string); ok && v != "" {
		obj.Namespace = &v
	}

	return obj
}

func flattenIngressClassV1Spec(in networking.IngressClassSpec) []interface{} {
	att := make(map[string]interface{})

	if in.Controller != "" {
		att["controller"] = in.Controller
	}

	if in.Parameters != nil {
		att["parameters"] = flattenIngressClassV1Parameters(in.Parameters)
	}

	return []interface{}{att}
}

func flattenIngressClassV1Parameters(in *networking.IngressClassParametersReference) []interface{} {
	att := make([]interface{}, 1)

	m := make(map[string]interface{})
	m["kind"] = in.Kind
	m["name"] = in.Name

	if in.APIGroup != nil {
		m["api_group"] = *in.APIGroup
	}

	if in.Scope != nil {
		m["scope"] = *in.Scope
	}

	if in.Namespace != nil {
		m["namespace"] = *in.Namespace
	}

	att[0] = m

	return att
}
