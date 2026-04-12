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

func resourceKubernetesListenerSetV1() *schema.Resource {
	return &schema.Resource{
		Description:        "ListenerSet defines a set of additional listeners to attach to an existing Gateway.",
		CreateContext:      resourceKubernetesListenerSetV1Create,
		ReadContext:        resourceKubernetesListenerSetV1Read,
		UpdateContext:      resourceKubernetesListenerSetV1Update,
		DeleteContext:      resourceKubernetesListenerSetV1Delete,
		DeprecationMessage: "",
		Schema:             resourceKubernetesListenerSetV1Schema(),
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

func resourceKubernetesListenerSetV1Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"metadata": namespacedMetadataSchema("listenerset_v1", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Spec defines the desired state of ListenerSet.",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"parent_ref": {
						Type:        schema.TypeList,
						Description: "ParentRef references the Gateway that the listeners are attached to.",
						Required:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"group": {
									Type:        schema.TypeString,
									Description: "Group is the group of the referent.",
									Optional:    true,
									Default:     "gateway.networking.k8s.io",
								},
								"kind": {
									Type:        schema.TypeString,
									Description: "Kind is the kind of the referent.",
									Optional:    true,
									Default:     "Gateway",
								},
								"namespace": {
									Type:        schema.TypeString,
									Description: "Namespace is the namespace of the referent.",
									Optional:    true,
								},
								"name": {
									Type:        schema.TypeString,
									Description: "Name is the name of the referent.",
									Required:    true,
								},
								"section_name": {
									Type:        schema.TypeString,
									Description: "SectionName is the section name of the referent.",
									Optional:    true,
								},
							},
						},
					},
					"listeners": {
						Type:        schema.TypeList,
						Description: "Listeners associated with this ListenerSet.",
						Required:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"name": {
									Type:        schema.TypeString,
									Description: "Name is the name of the Listener.",
									Required:    true,
								},
								"hostname": {
									Type:        schema.TypeString,
									Description: "Hostname specifies the virtual hostname to match.",
									Optional:    true,
								},
								"port": {
									Type:         schema.TypeInt,
									Description:  "Port is the network port.",
									Required:     true,
									ValidateFunc: validation.IsPortNumber,
								},
								"protocol": {
									Type:        schema.TypeString,
									Description: "Protocol specifies the network protocol.",
									Required:    true,
								},
								"tls": {
									Type:        schema.TypeList,
									Description: "TLS is the TLS configuration for the Listener.",
									Optional:    true,
									MaxItems:    1,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"mode": {
												Type:        schema.TypeString,
												Description: "Mode defines the TLS behavior.",
												Optional:    true,
												Default:     "Terminate",
											},
											"certificate_refs": {
												Type:        schema.TypeList,
												Description: "CertificateRefs contains references to TLS certificates.",
												Optional:    true,
												Elem: &schema.Resource{
													Schema: listenerSetSecretObjectReferenceSchema(),
												},
											},
											"options": {
												Type:        schema.TypeMap,
												Description: "Options are a list of key/value pairs for TLS configuration.",
												Optional:    true,
												Elem:        &schema.Schema{Type: schema.TypeString},
											},
										},
									},
								},
								"allowed_routes": {
									Type:        schema.TypeList,
									Description: "AllowedRoutes defines the types of routes that MAY be attached.",
									Optional:    true,
									Computed:    true,
									MaxItems:    1,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"namespaces": {
												Type:        schema.TypeList,
												Description: "Namespaces indicates namespaces from which Routes may be attached.",
												Optional:    true,
												Computed:    true,
												MaxItems:    1,
												Elem: &schema.Resource{
													Schema: listenerSetRouteNamespacesSchema(),
												},
											},
											"kinds": {
												Type:        schema.TypeList,
												Description: "Kinds specifies the groups and kinds of Routes.",
												Optional:    true,
												Elem: &schema.Resource{
													Schema: listenerSetRouteGroupKindSchema(),
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
		"status": {
			Type:        schema.TypeList,
			Description: "Status defines the current state of ListenerSet.",
			Computed:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"conditions": {
						Type:        schema.TypeList,
						Description: "Conditions describe the current state of the ListenerSet.",
						Computed:    true,
						Elem: &schema.Resource{
							Schema: listenersetConditionsSchemaComputed(),
						},
					},
					"listeners": {
						Type:        schema.TypeList,
						Description: "Listeners provide status for each unique listener port defined in the Spec.",
						Computed:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"name": {
									Type:        schema.TypeString,
									Description: "Name is the name of the Listener.",
									Computed:    true,
								},
								"supported_kinds": {
									Type:        schema.TypeList,
									Description: "SupportedKinds is the list indicating the Kinds supported by this listener.",
									Computed:    true,
									Elem: &schema.Resource{
										Schema: listenerSetRouteGroupKindSchema(),
									},
								},
								"attached_routes": {
									Type:        schema.TypeInt,
									Description: "AttachedRoutes represents the total number of Routes that have been attached to this Listener.",
									Computed:    true,
								},
								"conditions": {
									Type:        schema.TypeList,
									Description: "Conditions is the current state of the Listener.",
									Computed:    true,
									Elem: &schema.Resource{
										Schema: listenersetConditionsSchemaComputed(),
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

func listenerSetSecretObjectReferenceSchema() map[string]*schema.Schema {
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
	}
}

func listenerSetRouteNamespacesSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"from": {
			Type:        schema.TypeString,
			Description: "From indicates where Routes will be selected from.",
			Optional:    true,
			Default:     "Same",
		},
		"selector": {
			Type:        schema.TypeList,
			Description: "Selector labels Routes in the selected namespaces.",
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: listenerSetLabelSelectorSchema(),
			},
		},
	}
}

func listenerSetLabelSelectorSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"match_labels": {
			Type:        schema.TypeMap,
			Description: "MatchLabels is a map of {key,value} pairs.",
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"match_expressions": {
			Type:        schema.TypeList,
			Description: "MatchExpressions is a list of label selector requirements.",
			Optional:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"key": {
						Type:        schema.TypeString,
						Description: "Key is the label key that the selector applies to.",
						Optional:    true,
					},
					"operator": {
						Type:        schema.TypeString,
						Description: "Operator represents a key's relationship to a set of values.",
						Optional:    true,
					},
					"values": {
						Type:        schema.TypeList,
						Description: "Values is an array of string values.",
						Optional:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
					},
				},
			},
		},
	}
}

func listenerSetRouteGroupKindSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"group": {
			Type:        schema.TypeString,
			Description: "Group is the group of the referent.",
			Optional:    true,
		},
		"kind": {
			Type:        schema.TypeString,
			Description: "Kind is the kind of the referent.",
			Required:    true,
		},
	}
}

func listenersetConditionsSchemaComputed() map[string]*schema.Schema {
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

func resourceKubernetesListenerSetV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec := expandListenerSetSpec(d.Get("spec").([]interface{}))

	log.Printf("[INFO] Creating new ListenerSet: %#v", spec)
	out := &gatewayv1.ListenerSet{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	result, err := conn.ListenerSets(metadata.Namespace).Create(ctx, out, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new ListenerSet: %#v", result)
	d.SetId(buildId(result.ObjectMeta))

	return resourceKubernetesListenerSetV1Read(ctx, d, meta)
}

func resourceKubernetesListenerSetV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading ListenerSet %s", name)
	lset, err := conn.ListenerSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("[DEBUG] ListenerSet %s not found, removing from state", name)
			d.SetId("")
			return diag.Diagnostics{}
		}
		return diag.Errorf("Failed to read ListenerSet '%s' because: %s", name, err)
	}
	log.Printf("[INFO] Received ListenerSet: %#v", lset)

	err = d.Set("metadata", flattenMetadata(lset.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedSpec := flattenListenerSetSpec(lset.Spec)
	log.Printf("[DEBUG] Flattened ListenerSet spec: %#v", flattenedSpec)
	err = d.Set("spec", flattenedSpec)
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedStatus := flattenListenerSetStatus(lset.Status)
	err = d.Set("status", flattenedStatus)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildId(lset.ObjectMeta))

	err = setResourceIdentityNamespaced(d, "gateway.networking.k8s.io/v1", "ListenerSet", lset.Namespace, lset.Name)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesListenerSetV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	existing, err := conn.ListenerSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	existing.Labels = metadata.Labels
	existing.Annotations = metadata.Annotations
	existing.Spec = expandListenerSetSpec(d.Get("spec").([]interface{}))

	log.Printf("[INFO] Updating ListenerSet: %s/%s", namespace, name)
	result, err := conn.ListenerSets(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return diag.Errorf("Failed to update ListenerSet '%s/%s' because: %s", namespace, name, err)
	}

	log.Printf("[INFO] Submitted updated ListenerSet: %#v", result)

	return resourceKubernetesListenerSetV1Read(ctx, d, meta)
}

func resourceKubernetesListenerSetV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	name := metadata.Name
	namespace := metadata.Namespace

	log.Printf("[INFO] Deleting ListenerSet %s", name)
	err = conn.ListenerSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			log.Printf("[DEBUG] ListenerSet %s not found, removing from state", name)
			d.SetId("")
			return nil
		}
		return diag.Errorf("Failed to delete ListenerSet '%s' because: %s", name, err)
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		_, err := conn.ListenerSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		e := fmt.Errorf("ListenerSet (%s) still exists", d.Id())
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] ListenerSet %s deleted", name)
	d.SetId("")

	return nil
}
