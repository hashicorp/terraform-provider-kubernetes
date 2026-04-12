// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func resourceKubernetesReferenceGrantV1() *schema.Resource {
	return &schema.Resource{
		Description:   "ReferenceGrant allows cross-namespace references in Gateway API resources.",
		CreateContext: resourceKubernetesReferenceGrantV1Create,
		ReadContext:   resourceKubernetesReferenceGrantV1Read,
		UpdateContext: resourceKubernetesReferenceGrantV1Update,
		DeleteContext: resourceKubernetesReferenceGrantV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceIdentityImportNamespaced,
		},
		Identity: resourceIdentitySchemaNamespaced(),
		Schema:   resourceKubernetesReferenceGrantV1Schema(),
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
	}
}

func resourceKubernetesReferenceGrantV1Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"metadata": namespacedMetadataSchema("reference_grant_v1", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Spec defines the desired state of ReferenceGrant.",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"from": {
						Type:        schema.TypeList,
						Description: "From describes the trusted namespaces and kinds that can reference the resources described in To.",
						Required:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"group": {
									Type:        schema.TypeString,
									Description: "Group is the group of the referent.",
									Required:    true,
								},
								"kind": {
									Type:        schema.TypeString,
									Description: "Kind is the kind of the referent.",
									Required:    true,
								},
								"namespace": {
									Type:        schema.TypeString,
									Description: "Namespace is the namespace of the referent.",
									Required:    true,
								},
							},
						},
					},
					"to": {
						Type:        schema.TypeList,
						Description: "To describes the resources that may be referenced by the resources described in From.",
						Required:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"group": {
									Type:        schema.TypeString,
									Description: "Group is the group of the referent.",
									Required:    true,
								},
								"kind": {
									Type:        schema.TypeString,
									Description: "Kind is the kind of the referent.",
									Required:    true,
								},
								"name": {
									Type:        schema.TypeString,
									Description: "Name is the name of the referent.",
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

func resourceKubernetesReferenceGrantV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	obj := gatewayv1.ReferenceGrant{
		ObjectMeta: metadata,
		Spec:       expandReferenceGrantSpec(d.Get("spec").([]interface{})),
	}

	log.Printf("[INFO] Creating ReferenceGrant: %#v", obj)
	out, err := conn.ReferenceGrants(metadata.Namespace).Create(ctx, &obj, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Created ReferenceGrant: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesReferenceGrantV1Read(ctx, d, meta)
}

func resourceKubernetesReferenceGrantV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading ReferenceGrant %s", name)
	obj, err := conn.ReferenceGrants(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("[DEBUG] ReferenceGrant %s not found, removing from state", name)
			d.SetId("")
			return diag.Diagnostics{}
		}
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received ReferenceGrant: %#v", obj)

	err = d.Set("metadata", flattenMetadata(obj.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedSpec := flattenReferenceGrantSpec(obj.Spec)
	log.Printf("[DEBUG] Flattened ReferenceGrant spec: %#v", flattenedSpec)
	err = d.Set("spec", flattenedSpec)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildId(obj.ObjectMeta))

	err = setResourceIdentityNamespaced(d, "gateway.networking.k8s.io/v1", "ReferenceGrant", obj.Namespace, obj.Name)
	if err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func resourceKubernetesReferenceGrantV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	name := metadata.Name
	namespace := metadata.Namespace

	log.Printf("[INFO] Updating ReferenceGrant: %s", name)

	obj, err := conn.ReferenceGrants(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	obj.Labels = metadata.Labels
	obj.Annotations = metadata.Annotations
	obj.Spec = expandReferenceGrantSpec(d.Get("spec").([]interface{}))

	out, err := conn.ReferenceGrants(namespace).Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Updated ReferenceGrant: %#v", out)

	return resourceKubernetesReferenceGrantV1Read(ctx, d, meta)
}

func resourceKubernetesReferenceGrantV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting ReferenceGrant: %s", name)
	err = conn.ReferenceGrants(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			log.Printf("[DEBUG] ReferenceGrant %s not found, removing from state", name)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		_, err := conn.ReferenceGrants(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}
		e := fmt.Errorf("ReferenceGrant (%s) still exists", d.Id())
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] ReferenceGrant %s deleted", name)
	d.SetId("")
	return diag.Diagnostics{}
}
