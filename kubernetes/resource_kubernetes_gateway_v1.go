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

func resourceKubernetesGatewayV1(deprecationMessage string) *schema.Resource {
	return &schema.Resource{
		Description:        "Gateway represents an instance of a service-traffic handling infrastructure by binding Listeners to a set of IP addresses.",
		CreateContext:      resourceKubernetesGatewayV1Create,
		ReadContext:        resourceKubernetesGatewayV1Read,
		UpdateContext:      resourceKubernetesGatewayV1Update,
		DeleteContext:      resourceKubernetesGatewayV1Delete,
		DeprecationMessage: deprecationMessage,
		Schema:             resourceKubernetesGatewayV1Schema(),
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

func resourceKubernetesGatewayV1Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"metadata": namespacedMetadataSchema("gateway_v1", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Spec defines the desired state of Gateway.",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"gateway_class_name": {
						Type:        schema.TypeString,
						Description: "GatewayClassName used for this Gateway. This is the name of a GatewayClass resource.",
						Required:    true,
					},
					"listeners": {
						Type:        schema.TypeList,
						Description: "Listeners associated with this Gateway. Listeners define logical endpoints that are bound on this Gateway's addresses.",
						Required:    true,
						MinItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"name": {
									Type:        schema.TypeString,
									Description: "Name is the name of the Listener. This name MUST be unique within a Gateway.",
									Required:    true,
								},
								"hostname": {
									Type:        schema.TypeString,
									Description: "Hostname specifies the virtual hostname to match for protocol types that define this concept.",
									Optional:    true,
								},
								"port": {
									Type:         schema.TypeInt,
									Description:  "Port is the network port. Multiple listeners may use the same port, subject to the Listener compatibility rules.",
									Required:     true,
									ValidateFunc: validation.IsPortNumber,
								},
								"protocol": {
									Type:        schema.TypeString,
									Description: "Protocol specifies the network protocol this listener expects to receive.",
									Required:    true,
									ValidateFunc: validation.StringInSlice([]string{
										"HTTP", "HTTPS", "TCP", "UDP", "TLS",
									}, false),
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
												Description: "Mode defines the TLS behavior for the TLS session initiated by the client.",
												Optional:    true,
												Default:     "Terminate",
											},
											"certificate_refs": {
												Type:        schema.TypeList,
												Description: "CertificateRefs contains a series of references to Kubernetes objects that contains TLS certificates and private keys.",
												Optional:    true,
												Elem: &schema.Resource{
													Schema: secretObjectReferenceSchema(),
												},
											},
											"options": {
												Type:        schema.TypeMap,
												Description: "Options are a list of key/value pairs to enable extended TLS configuration.",
												Optional:    true,
												Elem:        &schema.Schema{Type: schema.TypeString},
											},
										},
									},
								},
								"allowed_routes": {
									Type:        schema.TypeList,
									Description: "AllowedRoutes defines the types of routes that MAY be attached to a Listener and the trusted namespaces where those Route resources MAY be present.",
									Optional:    true,
									Computed:    true,
									MaxItems:    1,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"namespaces": {
												Type:        schema.TypeList,
												Description: "Namespaces indicates namespaces from which Routes may be attached to this Listener.",
												Optional:    true,
												Computed:    true,
												MaxItems:    1,
												Elem: &schema.Resource{
													Schema: routeNamespacesSchema(),
												},
											},
											"kinds": {
												Type:        schema.TypeList,
												Description: "Kinds specifies the groups and kinds of Routes that are allowed to bind to this Gateway Listener.",
												Optional:    true,
												Elem: &schema.Resource{
													Schema: routeGroupKindSchema(),
												},
											},
										},
									},
								},
							},
						},
					},
					"addresses": {
						Type:        schema.TypeList,
						Description: "Addresses requested for this Gateway.",
						Optional:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"type": {
									Type:        schema.TypeString,
									Description: "Type of the address.",
									Optional:    true,
									Default:     "IPAddress",
								},
								"value": {
									Type:        schema.TypeString,
									Description: "Value of the address.",
									Optional:    true,
								},
							},
						},
					},
					"infrastructure": {
						Type:        schema.TypeList,
						Description: "Infrastructure defines infrastructure level attributes about this Gateway instance.",
						Optional:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"labels": {
									Type:        schema.TypeMap,
									Description: "Labels are a list of labels to apply to this resource.",
									Optional:    true,
									Elem:        &schema.Schema{Type: schema.TypeString},
								},
								"annotations": {
									Type:        schema.TypeMap,
									Description: "Annotations are a list of annotations to apply to this resource.",
									Optional:    true,
									Elem:        &schema.Schema{Type: schema.TypeString},
								},
								"parameters_ref": {
									Type:        schema.TypeList,
									Description: "ParametersRef is a reference to a resource containing controller-specific configuration.",
									Optional:    true,
									MaxItems:    1,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"group": {
												Type:        schema.TypeString,
												Description: "Group is the group of the referent.",
												Optional:    true,
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
										},
									},
								},
							},
						},
					},
					"allowed_listeners": {
						Type:        schema.TypeList,
						Description: "AllowedListeners defines which ListenerSets can be attached to this Gateway.",
						Optional:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"namespaces": {
									Type:        schema.TypeList,
									Description: "Namespaces defines which namespaces ListenerSets can be attached to this Gateway.",
									Optional:    true,
									MaxItems:    1,
									Elem: &schema.Resource{
										Schema: listenerNamespacesSchema(),
									},
								},
							},
						},
					},
					"tls": {
						Type:        schema.TypeList,
						Description: "TLS specifies frontend and backend tls configuration for entire gateway.",
						Optional:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"backend": {
									Type:        schema.TypeList,
									Description: "Backend describes TLS configuration for gateway when connecting to backends.",
									Optional:    true,
									MaxItems:    1,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"client_certificate_ref": {
												Type:        schema.TypeList,
												Description: "ClientCertificateRef references an object that contains a client certificate and its associated private key.",
												Optional:    true,
												MaxItems:    1,
												Elem: &schema.Resource{
													Schema: secretObjectReferenceSchema(),
												},
											},
										},
									},
								},
								"frontend": {
									Type:        schema.TypeList,
									Description: "Frontend describes TLS config when client connects to Gateway.",
									Optional:    true,
									MaxItems:    1,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"default": {
												Type:        schema.TypeList,
												Description: "Default specifies the default client certificate validation configuration.",
												Required:    true,
												MaxItems:    1,
												Elem: &schema.Resource{
													Schema: tlsConfigSchema(),
												},
											},
											"per_port": {
												Type:        schema.TypeList,
												Description: "PerPort specifies tls configuration assigned per port.",
												Optional:    true,
												Elem: &schema.Resource{
													Schema: map[string]*schema.Schema{
														"port": {
															Type:        schema.TypeInt,
															Description: "Port for TLS configuration.",
															Required:    true,
														},
														"tls": {
															Type:        schema.TypeList,
															Description: "TLS configuration for the port.",
															Required:    true,
															MaxItems:    1,
															Elem: &schema.Resource{
																Schema: tlsConfigSchema(),
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
					"default_scope": {
						Type:        schema.TypeString,
						Description: "DefaultScope, when set, configures the Gateway as a default Gateway.",
						Optional:    true,
					},
				},
			},
		},
		"status": {
			Type:        schema.TypeList,
			Description: "Status defines the current state of Gateway.",
			Computed:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"addresses": {
						Type:        schema.TypeList,
						Description: "Addresses is the list of addresses that have been bound to this Gateway.",
						Computed:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"type": {
									Type:        schema.TypeString,
									Description: "Type of the address.",
									Computed:    true,
								},
								"value": {
									Type:        schema.TypeString,
									Description: "Value of the address.",
									Computed:    true,
								},
							},
						},
					},
					"conditions": {
						Type:        schema.TypeList,
						Description: "Conditions is the current status from the controller for this Gateway.",
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
								"observed_generation": {
									Type:        schema.TypeInt,
									Description: "ObservedGeneration represents the generation of the resource that was observed by the controller.",
									Computed:    true,
								},
							},
						},
					},
					"listeners": {
						Type:        schema.TypeList,
						Description: "Listeners is the current status from the controller for this Gateway's listeners.",
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
									Description: "SupportedKinds indicates the Kinds supported by this listener.",
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
										},
									},
								},
								"attached_routes": {
									Type:        schema.TypeInt,
									Description: "AttachedRoutes is the number of routes attached to this Listener.",
									Computed:    true,
								},
								"conditions": {
									Type:        schema.TypeList,
									Description: "Conditions is the current status from the controller for this Listener.",
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
											"observed_generation": {
												Type:        schema.TypeInt,
												Description: "ObservedGeneration represents the generation of the resource that was observed by the controller.",
												Computed:    true,
											},
										},
									},
								},
							},
						},
					},
					"attached_listener_sets": {
						Type:        schema.TypeInt,
						Description: "AttachedListenerSets represents the total number of ListenerSets that have been successfully attached to this Gateway.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func secretObjectReferenceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"group": {
			Type:        schema.TypeString,
			Description: "Group is the group of the referent.",
			Optional:    true,
		},
		"kind": {
			Type:        schema.TypeString,
			Description: "Kind is kind of the referent.",
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

func routeNamespacesSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"from": {
			Type:        schema.TypeString,
			Description: "From indicates where Routes will be selected for this Gateway. Possible values are: All, Selector, Same.",
			Optional:    true,
			Default:     "Same",
		},
		"selector": {
			Type:        schema.TypeList,
			Description: "Selector must be specified when From is set to 'Selector'.",
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: labelSelectorSchema(),
			},
		},
	}
}

func listenerNamespacesSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"from": {
			Type:        schema.TypeString,
			Description: "From indicates where ListenerSets can attach to this Gateway. Possible values are: All, Selector, Same, None.",
			Optional:    true,
			Default:     "None",
		},
		"selector": {
			Type:        schema.TypeList,
			Description: "Selector must be specified when From is set to 'Selector'.",
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: labelSelectorSchema(),
			},
		},
	}
}

func labelSelectorSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"match_labels": {
			Type:        schema.TypeMap,
			Description: "MatchLabels is a map of {key,value} pairs used to select by label.",
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
						Description: "Values is a list of string values.",
						Optional:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
					},
				},
			},
		},
	}
}

func routeGroupKindSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"group": {
			Type:        schema.TypeString,
			Description: "Group is the group of the Route.",
			Optional:    true,
			Default:     "gateway.networking.k8s.io",
		},
		"kind": {
			Type:        schema.TypeString,
			Description: "Kind is the kind of the Route.",
			Required:    true,
		},
	}
}

func tlsConfigSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"validation": {
			Type:        schema.TypeList,
			Description: "Validation holds configuration information for validating the frontend (client).",
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"ca_certificate_refs": {
						Type:        schema.TypeList,
						Description: "CACertificateRefs contains references to Kubernetes objects that contains CA certificates.",
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
									Description: "Kind is kind of the referent.",
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
							},
						},
					},
					"mode": {
						Type:        schema.TypeString,
						Description: "FrontendValidationMode defines the mode for validating the client certificate.",
						Optional:    true,
						Default:     "AllowValidOnly",
					},
				},
			},
		},
	}
}

func resourceKubernetesGatewayV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandGatewayV1Spec(d.Get("spec").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}

	gateway := &gatewayv1.Gateway{
		ObjectMeta: metadata,
		Spec:       *spec,
	}

	log.Printf("[INFO] Creating new Gateway: %#v", gateway)
	out, err := conn.Gateways(metadata.Namespace).Create(ctx, gateway, metav1.CreateOptions{})
	if err != nil {
		return diag.Errorf("Failed to create Gateway '%s' because: %s", buildId(metadata), err)
	}
	log.Printf("[INFO] Submitted new Gateway: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesGatewayV1Read(ctx, d, meta)
}

func resourceKubernetesGatewayV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading Gateway %s", name)
	gateway, err := conn.Gateways(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			log.Printf("[DEBUG] Gateway %s not found, removing from state", name)
			d.SetId("")
			return diag.Diagnostics{}
		}
		return diag.Errorf("Failed to read Gateway '%s' because: %s", name, err)
	}
	log.Printf("[INFO] Received Gateway: %#v", gateway)

	err = d.Set("metadata", flattenMetadata(gateway.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedSpec := flattenGatewayV1Spec(gateway.Spec)
	log.Printf("[DEBUG] Flattened Gateway spec: %#v", flattenedSpec)
	err = d.Set("spec", flattenedSpec)
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedStatus := flattenGatewayV1Status(gateway.Status)
	log.Printf("[DEBUG] Flattened Gateway status: %#v", flattenedStatus)
	err = d.Set("status", flattenedStatus)
	if err != nil {
		return diag.FromErr(err)
	}

	err = setResourceIdentityNamespaced(d, "gateway.networking.k8s.io/v1", "Gateway", namespace, name)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesGatewayV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	existing, err := conn.Gateways(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	existing.Labels = metadata.Labels
	existing.Annotations = metadata.Annotations

	spec, err := expandGatewayV1Spec(d.Get("spec").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}
	existing.Spec = *spec

	log.Printf("[INFO] Updating Gateway: %#v", existing)
	out, err := conn.Gateways(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return diag.Errorf("Failed to update Gateway %s because: %s", d.Id(), err)
	}
	log.Printf("[INFO] Submitted updated Gateway: %#v", out)

	return resourceKubernetesGatewayV1Read(ctx, d, meta)
}

func resourceKubernetesGatewayV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting Gateway %s", name)
	err = conn.Gateways(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			log.Printf("[DEBUG] Gateway %s not found, removing from state", name)
			d.SetId("")
			return diag.Diagnostics{}
		}
		return diag.Errorf("Failed to delete Gateway %s because: %s", d.Id(), err)
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		_, err := conn.Gateways(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		e := fmt.Errorf("Gateway (%s) still exists", d.Id())
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Gateway %s deleted", name)
	d.SetId("")

	return nil
}
