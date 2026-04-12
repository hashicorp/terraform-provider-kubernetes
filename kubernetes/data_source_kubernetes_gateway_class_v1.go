// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesGatewayClassV1() *schema.Resource {
	return &schema.Resource{
		Description: "Get information about a GatewayClass resource.",
		ReadContext: dataSourceKubernetesGatewayClassV1Read,
		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("gateway_class_v1", false),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the desired state of GatewayClass.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"controller_name": {
							Type:        schema.TypeString,
							Description: "ControllerName is the name of the controller that is managing Gateways of this class.",
							Computed:    true,
						},
						"description": {
							Type:        schema.TypeString,
							Description: "Description helps describe a GatewayClass with more details.",
							Computed:    true,
						},
						"parameters_ref": {
							Type:        schema.TypeList,
							Description: "ParametersRef is a reference to a resource that contains the configuration parameters.",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"group": {
										Type:        schema.TypeString,
										Description: "Group is the group of the referent.",
										Computed:    true,
									},
									"kind": {
										Type:        schema.TypeString,
										Description: "Kind is kind of the referent.",
										Computed:    true,
									},
									"name": {
										Type:        schema.TypeString,
										Description: "Name is the name of the referent.",
										Computed:    true,
									},
									"namespace": {
										Type:        schema.TypeString,
										Description: "Namespace is the namespace of the referent.",
										Computed:    true,
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
		},
	}
}

func dataSourceKubernetesGatewayClassV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("metadata.0.name").(string)

	d.SetId(name)

	log.Printf("[INFO] Reading GatewayClass %s", name)
	gc, err := conn.GatewayClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			log.Printf("[DEBUG] GatewayClass %s not found", name)
			return nil
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

	return nil
}
