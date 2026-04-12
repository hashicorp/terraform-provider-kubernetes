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

func dataSourceKubernetesGatewayV1() *schema.Resource {
	return &schema.Resource{
		Description: "Get information about a Gateway resource.",
		ReadContext: dataSourceKubernetesGatewayV1Read,
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("gateway_v1", false),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the desired state of Gateway.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"gateway_class_name": {
							Type:        schema.TypeString,
							Description: "GatewayClassName used for this Gateway.",
							Computed:    true,
						},
						"listeners": {
							Type:        schema.TypeList,
							Description: "Listeners associated with this Gateway.",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Description: "Name is the name of the Listener.",
										Computed:    true,
									},
									"hostname": {
										Type:        schema.TypeString,
										Description: "Hostname specifies the virtual hostname.",
										Computed:    true,
									},
									"port": {
										Type:        schema.TypeInt,
										Description: "Port is the network port.",
										Computed:    true,
									},
									"protocol": {
										Type:        schema.TypeString,
										Description: "Protocol specifies the network protocol.",
										Computed:    true,
									},
									"tls": {
										Type:        schema.TypeList,
										Description: "TLS is the TLS configuration for the Listener.",
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"mode": {
													Type:        schema.TypeString,
													Description: "Mode defines the TLS behavior.",
													Computed:    true,
												},
												"certificate_refs": {
													Type:        schema.TypeList,
													Description: "CertificateRefs contains references to TLS certificates.",
													Computed:    true,
													Elem: &schema.Resource{
														Schema: secretObjectReferenceSchema(),
													},
												},
												"options": {
													Type:        schema.TypeMap,
													Description: "Options are a list of key/value pairs for TLS configuration.",
													Computed:    true,
													Elem:        &schema.Schema{Type: schema.TypeString},
												},
											},
										},
									},
									"allowed_routes": {
										Type:        schema.TypeList,
										Description: "AllowedRoutes defines the types of routes that MAY be attached.",
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"namespaces": {
													Type:        schema.TypeList,
													Description: "Namespaces indicates namespaces from which Routes may be attached.",
													Computed:    true,
													Elem: &schema.Resource{
														Schema: routeNamespacesSchema(),
													},
												},
												"kinds": {
													Type:        schema.TypeList,
													Description: "Kinds specifies the groups and kinds of Routes.",
													Computed:    true,
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
						"infrastructure": {
							Type:        schema.TypeList,
							Description: "Infrastructure defines infrastructure level attributes.",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"labels": {
										Type:        schema.TypeMap,
										Description: "Labels are a list of labels.",
										Computed:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"annotations": {
										Type:        schema.TypeMap,
										Description: "Annotations are a list of annotations.",
										Computed:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"parameters_ref": {
										Type:        schema.TypeList,
										Description: "ParametersRef is a reference to a resource containing controller-specific configuration.",
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
											},
										},
									},
								},
							},
						},
						"allowed_listeners": {
							Type:        schema.TypeList,
							Description: "AllowedListeners defines which ListenerSets can be attached.",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"namespaces": {
										Type:        schema.TypeList,
										Description: "Namespaces defines which namespaces ListenerSets can be attached.",
										Computed:    true,
										Elem: &schema.Resource{
											Schema: listenerNamespacesSchema(),
										},
									},
								},
							},
						},
						"tls": {
							Type:        schema.TypeList,
							Description: "TLS specifies frontend and backend tls configuration.",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"backend": {
										Type:        schema.TypeList,
										Description: "Backend describes TLS configuration for gateway.",
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"client_certificate_ref": {
													Type:        schema.TypeList,
													Description: "ClientCertificateRef references client certificate.",
													Computed:    true,
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
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"default": {
													Type:        schema.TypeList,
													Description: "Default specifies the default client certificate validation.",
													Computed:    true,
													Elem: &schema.Resource{
														Schema: tlsConfigSchema(),
													},
												},
												"per_port": {
													Type:        schema.TypeList,
													Description: "PerPort specifies tls configuration assigned per port.",
													Computed:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"port": {
																Type:        schema.TypeInt,
																Description: "Port for TLS configuration.",
																Computed:    true,
															},
															"tls": {
																Type:        schema.TypeList,
																Description: "TLS configuration for the port.",
																Computed:    true,
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
							Description: "DefaultScope configures the Gateway as a default Gateway.",
							Computed:    true,
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
							Description: "Addresses is the list of addresses bound to this Gateway.",
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
							Description: "Conditions is the current status from the controller.",
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
										Description: "LastTransitionTime is the last time.",
										Computed:    true,
									},
									"observed_generation": {
										Type:        schema.TypeInt,
										Description: "ObservedGeneration represents the generation observed.",
										Computed:    true,
									},
								},
							},
						},
						"listeners": {
							Type:        schema.TypeList,
							Description: "Listeners is the current status from the controller for listeners.",
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
										Description: "AttachedRoutes is the number of routes attached.",
										Computed:    true,
									},
									"conditions": {
										Type:        schema.TypeList,
										Description: "Conditions is the current status from the controller.",
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
													Description: "Message is a human readable message.",
													Computed:    true,
												},
												"reason": {
													Type:        schema.TypeString,
													Description: "Reason is a unique reason.",
													Computed:    true,
												},
												"last_transition_time": {
													Type:        schema.TypeString,
													Description: "LastTransitionTime is the last time.",
													Computed:    true,
												},
												"observed_generation": {
													Type:        schema.TypeInt,
													Description: "ObservedGeneration represents the generation.",
													Computed:    true,
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
	}
}

func dataSourceKubernetesGatewayV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	name := metadata.Name
	namespace := metadata.Namespace

	log.Printf("[INFO] Reading Gateway %s", name)
	gateway, err := conn.Gateways(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
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
	err = d.Set("status", flattenedStatus)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildId(gateway.ObjectMeta))

	return nil
}
