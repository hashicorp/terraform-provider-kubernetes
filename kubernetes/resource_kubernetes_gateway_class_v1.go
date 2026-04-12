// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"log"
	"time"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func resourceKubernetesGatewayClassV1(deprecationMessage string) *schema.Resource {
	return &schema.Resource{
		Description:        "GatewayClass represents a class of Gateways available to the user for creating Gateway resources.",
		CreateContext:      resourceKubernetesGatewayClassV1Create,
		ReadContext:        resourceKubernetesGatewayClassV1Read,
		UpdateContext:      resourceKubernetesGatewayClassV1Update,
		DeleteContext:      resourceKubernetesGatewayClassV1Delete,
		DeprecationMessage: deprecationMessage,
		Schema:             resourceKubernetesGatewayClassV1Schema(),
		Importer: &schema.ResourceImporter{
			StateContext: resourceIdentityImportNonNamespaced,
		},
		Identity: resourceIdentitySchemaNonNamespaced(),
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
	}
}

func resourceKubernetesGatewayClassV1Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"metadata": metadataSchema("gateway_class_v1", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Spec defines the desired state of GatewayClass.",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"controller_name": {
						Type:        schema.TypeString,
						Description: "ControllerName is the name of the controller that is managing Gateways of this class. The value of this field MUST be a domain prefixed path.",
						Required:    true,
					},
					"description": {
						Type:         schema.TypeString,
						Description:  "Description helps describe a GatewayClass with more details. Max length is 64 characters.",
						Optional:     true,
						ValidateFunc: validation.StringLenBetween(0, 64),
					},
					"parameters_ref": {
						Type:        schema.TypeList,
						Description: "ParametersRef is a reference to a resource that contains the configuration parameters corresponding to the GatewayClass.",
						Optional:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"group": {
									Type:        schema.TypeString,
									Description: "Group is the group of the referent.",
									Required:    true,
								},
								"kind": {
									Type:        schema.TypeString,
									Description: "Kind is kind of the referent.",
									Required:    true,
								},
								"name": {
									Type:        schema.TypeString,
									Description: "Name is the name of the referent.",
									Required:    true,
								},
								"namespace": {
									Type:        schema.TypeString,
									Description: "Namespace is the namespace of the referent. This field is required when referring to a Namespace-scoped resource and MUST be unset when referring to a Cluster-scoped resource.",
									Optional:    true,
								},
							},
						},
					},
				},
			},
		},
		"status": {
			Type:        schema.TypeList,
			Description: "Status defines the current state of GatewayClass.",
			Computed:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"conditions": {
						Type:        schema.TypeList,
						Description: "Conditions is the current status from the controller for this GatewayClass.",
						Computed:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"type": {
									Type:        schema.TypeString,
									Description: "Type of condition.",
									Computed:    true,
								},
								"status": {
									Type:        schema.TypeString,
									Description: "Status of the condition.",
									Computed:    true,
								},
								"message": {
									Type:        schema.TypeString,
									Description: "Message is a human readable message indicating details about the condition.",
									Computed:    true,
								},
								"reason": {
									Type:        schema.TypeString,
									Description: "Reason is a unique reason for the condition's last transition.",
									Computed:    true,
								},
								"last_transition_time": {
									Type:        schema.TypeString,
									Description: "LastTransitionTime is the last time the condition transitioned from one status to another.",
									Computed:    true,
								},
								"last_update_time": {
									Type:        schema.TypeString,
									Description: "LastUpdateTime is the last time this condition was updated.",
									Computed:    true,
								},
								"observed_generation": {
									Type:        schema.TypeInt,
									Description: "ObservedGeneration represents the generation of the resource that was observed by the controller.",
									Computed:    true,
								},
							},
						},
					},
					"supported_features": {
						Type:        schema.TypeList,
						Description: "SupportedFeatures is the set of features the GatewayClass support.",
						Computed:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
					},
				},
			},
		},
	}
}

func resourceKubernetesGatewayClassV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	gc := &gatewayv1.GatewayClass{
		ObjectMeta: metadata,
		Spec:       expandGatewayClassV1Spec(d.Get("spec").([]interface{})),
	}

	log.Printf("[INFO] Creating new GatewayClass: %#v", gc)
	out, err := conn.GatewayClasses().Create(ctx, gc, metav1.CreateOptions{})
	if err != nil {
		return diag.Errorf("Failed to create GatewayClass '%s' because: %s", buildId(gc.ObjectMeta), err)
	}
	log.Printf("[INFO] Submitted new GatewayClass: %#v", out)
	d.SetId(out.ObjectMeta.GetName())

	err = setResourceIdentityNonNamespaced(d, "gateway.networking.k8s.io/v1", "GatewayClass", out.GetName())
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceKubernetesGatewayClassV1Read(ctx, d, meta)
}

func resourceKubernetesGatewayClassV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()

	log.Printf("[INFO] Reading GatewayClass %s", name)
	gc, err := conn.GatewayClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			log.Printf("[DEBUG] GatewayClass %s not found, removing from state", name)
			d.SetId("")
			return diag.Diagnostics{}
		}
		return diag.Errorf("Failed to read GatewayClass '%s' because: %s", name, err)
	}
	log.Printf("[INFO] Received GatewayClass: %#v", gc)

	err = d.Set("metadata", flattenMetadata(gc.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedSpec := flattenGatewayClassV1Spec(gc.Spec)
	log.Printf("[DEBUG] Flattened GatewayClass spec: %#v", flattenedSpec)
	err = d.Set("spec", flattenedSpec)
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedStatus := flattenGatewayClassV1Status(gc.Status)
	log.Printf("[DEBUG] Flattened GatewayClass status: %#v", flattenedStatus)
	err = d.Set("status", flattenedStatus)
	if err != nil {
		return diag.FromErr(err)
	}

	err = setResourceIdentityNonNamespaced(d, "gateway.networking.k8s.io/v1", "GatewayClass", gc.GetName())
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesGatewayClassV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()
	existing, err := conn.GatewayClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	existing.Labels = metadata.Labels
	existing.Annotations = metadata.Annotations
	existing.Spec = expandGatewayClassV1Spec(d.Get("spec").([]interface{}))

	log.Printf("[INFO] Updating GatewayClass: %#v", existing)
	out, err := conn.GatewayClasses().Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return diag.Errorf("Failed to update GatewayClass %s because: %s", d.Id(), err)
	}
	log.Printf("[INFO] Submitted updated GatewayClass: %#v", out)

	return resourceKubernetesGatewayClassV1Read(ctx, d, meta)
}

func resourceKubernetesGatewayClassV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()

	log.Printf("[INFO] Deleting GatewayClass: %#v", name)
	err = conn.GatewayClasses().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return diag.Errorf("Failed to delete GatewayClass %s because: %s", d.Id(), err)
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		_, err := conn.GatewayClasses().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		e := fmt.Errorf("GatewayClass (%s) still exists", d.Id())
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] GatewayClass %s deleted", name)
	d.SetId("")

	return nil
}

func resourceKubernetesGatewayClassV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return false, err
	}

	name := d.Id()

	log.Printf("[INFO] Checking if GatewayClass %s exists", name)
	_, err = conn.GatewayClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
