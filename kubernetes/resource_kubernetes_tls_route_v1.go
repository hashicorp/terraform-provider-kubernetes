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

func resourceKubernetesTLSRouteV1() *schema.Resource {
	return &schema.Resource{
		Description:        "TLSRoute provides a way to route TLS requests.",
		CreateContext:      resourceKubernetesTLSRouteV1Create,
		ReadContext:        resourceKubernetesTLSRouteV1Read,
		UpdateContext:      resourceKubernetesTLSRouteV1Update,
		DeleteContext:      resourceKubernetesTLSRouteV1Delete,
		DeprecationMessage: "",
		Schema:             resourceKubernetesTLSRouteV1Schema(),
		Importer: &schema.ResourceImporter{
			StateContext: resourceIdentityImportNamespaced,
		},
		Identity: resourceIdentitySchemaNamespaced(),
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
	}
}

func resourceKubernetesTLSRouteV1Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"metadata": namespacedMetadataSchema("tlsroute_v1", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Spec defines the desired state of TLSRoute.",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"parent_refs": {
						Type:        schema.TypeList,
						Description: "ParentRefs identifies an API object (usually a Gateway) that routes should reference to attach to it.",
						Optional:    true,
						Elem: &schema.Resource{
							Schema: parentReferenceSchema(),
						},
					},
					"hostnames": {
						Type:        schema.TypeList,
						Description: "Hostnames defines a set of SNI hostnames that should match against the SNI attribute.",
						Optional:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
					},
					"use_default_gateways": {
						Type:        schema.TypeString,
						Description: "UseDefaultGateways indicates the default Gateway scope to use for this Route.",
						Optional:    true,
					},
					"rules": {
						Type:        schema.TypeList,
						Description: "Rules are a list of TLS matchers and actions.",
						Optional:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"name": {
									Type:        schema.TypeString,
									Description: "Name is the name of the route rule.",
									Optional:    true,
								},
								"backend_refs": {
									Type:        schema.TypeList,
									Description: "BackendRefs defines the backend(s) where matching requests should be sent.",
									Optional:    true,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"group": {
												Type:        schema.TypeString,
												Description: "Group is the group of the referent.",
												Optional:    true,
											},
											"kind": {
												Type:        schema.TypeString,
												Description: "Kind is the kind of the referent.",
												Optional:    true,
												Default:     "Service",
											},
											"name": {
												Type:        schema.TypeString,
												Description: "Name is the name of the referent.",
												Required:    true,
											},
											"namespace": {
												Type:        schema.TypeString,
												Description: "Namespace is the namespace of the referent.",
												Optional:    true,
											},
											"port": {
												Type:         schema.TypeInt,
												Description:  "Port is the port number of the referent.",
												Required:     true,
												ValidateFunc: validation.IsPortNumber,
											},
											"weight": {
												Type:        schema.TypeInt,
												Description: "Weight specifies the proportion of requests.",
												Optional:    true,
												Default:     1,
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
		"status": {
			Type:        schema.TypeList,
			Description: "Status defines the current state of TLSRoute.",
			Computed:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"parents": {
						Type:        schema.TypeList,
						Description: "Parents is a list of parent resources that this route is attached to.",
						Computed:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"parent_ref": tlsParentRefSchemaComputed(),
								"controller_name": {
									Type:        schema.TypeString,
									Description: "ControllerName is the controller that manages this route.",
									Computed:    true,
								},
								"conditions": {
									Type:        schema.TypeList,
									Description: "Conditions is the current state of the route.",
									Computed:    true,
									Elem: &schema.Resource{
										Schema: tlsConditionsSchemaComputed(),
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

func tlsBackendObjectReferenceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"group": {
			Type:        schema.TypeString,
			Description: "Group is the group of the referent.",
			Optional:    true,
		},
		"kind": {
			Type:        schema.TypeString,
			Description: "Kind is the kind of the referent.",
			Optional:    true,
			Default:     "Service",
		},
		"name": {
			Type:        schema.TypeString,
			Description: "Name is the name of the referent.",
			Required:    true,
		},
		"namespace": {
			Type:        schema.TypeString,
			Description: "Namespace is the namespace of the referent.",
			Optional:    true,
		},
		"port": {
			Type:        schema.TypeInt,
			Description: "Port is the port number of the referent.",
			Required:    true,
		},
	}
}

func tlsParentRefSchemaComputed() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Description: "ParentRef is a reference to the parent resource.",
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
					Description: "Kind is the kind of the referent.",
					Computed:    true,
				},
				"namespace": {
					Type:        schema.TypeString,
					Description: "Namespace is the namespace of the referent.",
					Computed:    true,
				},
				"name": {
					Type:        schema.TypeString,
					Description: "Name is the name of the referent.",
					Computed:    true,
				},
				"section_name": {
					Type:        schema.TypeString,
					Description: "SectionName is the section name of the referent.",
					Computed:    true,
				},
				"port": {
					Type:        schema.TypeInt,
					Description: "Port is the port of the referent.",
					Computed:    true,
				},
			},
		},
	}
}

func tlsConditionsSchemaComputed() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"type": {
			Type:        schema.TypeString,
			Description: "Type of the condition.",
			Computed:    true,
		},
		"status": {
			Type:        schema.TypeString,
			Description: "Status of the condition.",
			Computed:    true,
		},
		"message": {
			Type:        schema.TypeString,
			Description: "Message is a human readable message.",
			Computed:    true,
		},
		"reason": {
			Type:        schema.TypeString,
			Description: "Reason is a unique reason for the condition.",
			Computed:    true,
		},
		"last_transition_time": {
			Type:        schema.TypeString,
			Description: "LastTransitionTime is the last time the condition transitioned.",
			Computed:    true,
		},
		"observed_generation": {
			Type:        schema.TypeInt,
			Description: "ObservedGeneration is the observed generation.",
			Computed:    true,
		},
	}
}

func resourceKubernetesTLSRouteV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec := expandTLSRouteSpec(d.Get("spec").([]interface{}))

	log.Printf("[INFO] Creating new TLSRoute: %#v", spec)
	out := &gatewayv1.TLSRoute{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	result, err := conn.TLSRoutes(metadata.Namespace).Create(ctx, out, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new TLSRoute: %#v", result)
	d.SetId(buildId(result.ObjectMeta))

	return resourceKubernetesTLSRouteV1Read(ctx, d, meta)
}

func resourceKubernetesTLSRouteV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading TLSRoute %s", name)
	route, err := conn.TLSRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			log.Printf("[DEBUG] TLSRoute %s not found, removing from state", name)
			d.SetId("")
			return diag.Diagnostics{}
		}
		return diag.Errorf("Failed to read TLSRoute '%s' because: %s", name, err)
	}
	log.Printf("[INFO] Received TLSRoute: %#v", route)

	err = d.Set("metadata", flattenMetadata(route.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedSpec := flattenTLSRouteSpec(route.Spec)
	log.Printf("[DEBUG] Flattened TLSRoute spec: %#v", flattenedSpec)
	err = d.Set("spec", flattenedSpec)
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedStatus := flattenTLSRouteStatus(route.Status)
	err = d.Set("status", flattenedStatus)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildId(route.ObjectMeta))

	err = setResourceIdentityNamespaced(d, "gateway.networking.k8s.io/v1", "TLSRoute", route.Namespace, route.Name)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesTLSRouteV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	existing, err := conn.TLSRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	existing.Labels = metadata.Labels
	existing.Annotations = metadata.Annotations
	existing.Spec = expandTLSRouteSpec(d.Get("spec").([]interface{}))

	log.Printf("[INFO] Updating TLSRoute: %s/%s", namespace, name)
	result, err := conn.TLSRoutes(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return diag.Errorf("Failed to update TLSRoute '%s/%s' because: %s", namespace, name, err)
	}

	log.Printf("[INFO] Submitted updated TLSRoute: %#v", result)

	return resourceKubernetesTLSRouteV1Read(ctx, d, meta)
}

func resourceKubernetesTLSRouteV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	name := metadata.Name
	namespace := metadata.Namespace

	log.Printf("[INFO] Deleting TLSRoute %s", name)
	err = conn.TLSRoutes(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			log.Printf("[DEBUG] TLSRoute %s not found, removing from state", name)
			d.SetId("")
			return nil
		}
		return diag.Errorf("Failed to delete TLSRoute '%s' because: %s", name, err)
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		_, err := conn.TLSRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		e := fmt.Errorf("TLSRoute (%s) still exists", d.Id())
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] TLSRoute %s deleted", name)
	d.SetId("")

	return nil
}
