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

func resourceKubernetesGRPCRouteV1() *schema.Resource {
	return &schema.Resource{
		Description:        "GRPCRoute provides a way to route gRPC requests.",
		CreateContext:      resourceKubernetesGRPCRouteV1Create,
		ReadContext:        resourceKubernetesGRPCRouteV1Read,
		UpdateContext:      resourceKubernetesGRPCRouteV1Update,
		DeleteContext:      resourceKubernetesGRPCRouteV1Delete,
		DeprecationMessage: "",
		Schema:             resourceKubernetesGRPCRouteV1Schema(),
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

func resourceKubernetesGRPCRouteV1Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"metadata": namespacedMetadataSchema("grpcroute_v1", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Spec defines the desired state of GRPCRoute.",
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
						Description: "Hostnames defines a set of hostnames that should match against the GRPC Host header.",
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
						Description: "Rules are a list of GRPC matchers, filters and actions.",
						Optional:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"name": {
									Type:        schema.TypeString,
									Description: "Name is the name of the route rule.",
									Optional:    true,
								},
								"matches": {
									Type:        schema.TypeList,
									Description: "Matches define conditions used for matching the rule against incoming gRPC requests.",
									Optional:    true,
									Computed:    true,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"method": {
												Type:        schema.TypeList,
												Description: "Method specifies a gRPC request service/method matcher.",
												Optional:    true,
												MaxItems:    1,
												Elem: &schema.Resource{
													Schema: map[string]*schema.Schema{
														"type": {
															Type:        schema.TypeString,
															Description: "Type specifies how to match against the service and/or method.",
															Optional:    true,
															Default:     "Exact",
														},
														"service": {
															Type:        schema.TypeString,
															Description: "Service is the gRPC service name.",
															Optional:    true,
														},
														"method": {
															Type:        schema.TypeString,
															Description: "Method is the gRPC method name.",
															Optional:    true,
														},
													},
												},
											},
											"headers": {
												Type:        schema.TypeList,
												Description: "Headers specifies gRPC request header matchers.",
												Optional:    true,
												Elem: &schema.Resource{
													Schema: map[string]*schema.Schema{
														"name": {
															Type:        schema.TypeString,
															Description: "Name is the header name.",
															Required:    true,
														},
														"value": {
															Type:        schema.TypeString,
															Description: "Value is the header value.",
															Required:    true,
														},
														"type": {
															Type:        schema.TypeString,
															Description: "Type defines the type of header match.",
															Optional:    true,
															Default:     "Exact",
														},
													},
												},
											},
										},
									},
								},
								"filters": {
									Type:        schema.TypeList,
									Description: "Filters define the filters that are applied to requests that match this rule.",
									Optional:    true,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"type": {
												Type:        schema.TypeString,
												Description: "Type is the type of filter.",
												Required:    true,
											},
											"request_header_modifier":  grpcHeaderModifierFilterSchema(),
											"response_header_modifier": grpcHeaderModifierFilterSchema(),
											"request_mirror":           requestMirrorFilterSchema(),
											"extension_ref": {
												Type:        schema.TypeList,
												Description: "ExtensionRef is a reference to a custom extension filter.",
												Optional:    true,
												MaxItems:    1,
												Elem: &schema.Resource{
													Schema: map[string]*schema.Schema{
														"name": {
															Type:        schema.TypeString,
															Description: "Name is the name of the extension.",
															Required:    true,
														},
													},
												},
											},
										},
									},
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
											"filters": {
												Type:        schema.TypeList,
												Description: "Filters defined at this level are applied before any rule-level filters.",
												Optional:    true,
												Elem: &schema.Resource{
													Schema: map[string]*schema.Schema{
														"type": {
															Type:        schema.TypeString,
															Description: "Type is the type of filter.",
															Required:    true,
														},
														"request_header_modifier":  grpcHeaderModifierFilterSchema(),
														"response_header_modifier": grpcHeaderModifierFilterSchema(),
														"request_mirror":           requestMirrorFilterSchema(),
														"extension_ref": {
															Type:        schema.TypeList,
															Description: "ExtensionRef is a reference to a custom extension filter.",
															Optional:    true,
															MaxItems:    1,
															Elem: &schema.Resource{
																Schema: map[string]*schema.Schema{
																	"name": {
																		Type:        schema.TypeString,
																		Description: "Name is the name of the extension.",
																		Required:    true,
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
								"session_persistence": {
									Type:        schema.TypeList,
									Description: "SessionPersistence defines and configures session persistence for the route rule.",
									Optional:    true,
									MaxItems:    1,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"session_name": {
												Type:        schema.TypeString,
												Description: "SessionName is the name of the session.",
												Optional:    true,
											},
											"absolute_timeout": {
												Type:        schema.TypeString,
												Description: "AbsoluteTimeout specifies the maximum duration for the session.",
												Optional:    true,
											},
											"idle_timeout": {
												Type:        schema.TypeString,
												Description: "IdleTimeout specifies the duration after which an idle session should be expired.",
												Optional:    true,
											},
											"type": {
												Type:        schema.TypeString,
												Description: "Type defines the type of session persistence.",
												Optional:    true,
												Default:     "Cookie",
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
			Description: "Status defines the current state of GRPCRoute.",
			Computed:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"parents": {
						Type:        schema.TypeList,
						Description: "Parents is a list of parent resources that this route is attached to.",
						Computed:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"parent_ref": grpcParentRefSchemaComputed(),
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
										Schema: conditionsSchemaComputed(),
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

func grpcBackendObjectReferenceSchema() map[string]*schema.Schema {
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

func grpcHeaderModifierFilterSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Description: "HeaderModifier modifies request or response headers.",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"set": {
					Type:        schema.TypeList,
					Description: "Set overwrites the headers.",
					Optional:    true,
					Elem: &schema.Resource{
						Schema: grpcHTTPHeaderSchema(),
					},
				},
				"add": {
					Type:        schema.TypeList,
					Description: "Add adds headers.",
					Optional:    true,
					Elem: &schema.Resource{
						Schema: grpcHTTPHeaderSchema(),
					},
				},
				"remove": {
					Type:        schema.TypeList,
					Description: "Remove removes headers.",
					Optional:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
			},
		},
	}
}

func grpcHTTPHeaderSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": {
			Type:        schema.TypeString,
			Description: "Name is the header name.",
			Required:    true,
		},
		"value": {
			Type:        schema.TypeString,
			Description: "Value is the header value.",
			Required:    true,
		},
	}
}

func grpcParentRefSchemaComputed() *schema.Schema {
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

func conditionsSchemaComputed() map[string]*schema.Schema {
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

func resourceKubernetesGRPCRouteV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec := expandGRPCRouteSpec(d.Get("spec").([]interface{}))

	log.Printf("[INFO] Creating new GRPCRoute: %#v", spec)
	out := &gatewayv1.GRPCRoute{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	result, err := conn.GRPCRoutes(metadata.Namespace).Create(ctx, out, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new GRPCRoute: %#v", result)
	d.SetId(buildId(result.ObjectMeta))

	return resourceKubernetesGRPCRouteV1Read(ctx, d, meta)
}

func resourceKubernetesGRPCRouteV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading GRPCRoute %s", name)
	route, err := conn.GRPCRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("[DEBUG] GRPCRoute %s not found, removing from state", name)
			d.SetId("")
			return diag.Diagnostics{}
		}
		return diag.Errorf("Failed to read GRPCRoute '%s' because: %s", name, err)
	}
	log.Printf("[INFO] Received GRPCRoute: %#v", route)

	err = d.Set("metadata", flattenMetadata(route.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedSpec := flattenGRPCRouteSpec(route.Spec)
	log.Printf("[DEBUG] Flattened GRPCRoute spec: %#v", flattenedSpec)
	err = d.Set("spec", flattenedSpec)
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedStatus := flattenGRPCRouteStatus(route.Status)
	err = d.Set("status", flattenedStatus)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildId(route.ObjectMeta))

	err = setResourceIdentityNamespaced(d, "gateway.networking.k8s.io/v1", "GRPCRoute", route.Namespace, route.Name)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesGRPCRouteV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	existing, err := conn.GRPCRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	existing.Labels = metadata.Labels
	existing.Annotations = metadata.Annotations
	existing.Spec = expandGRPCRouteSpec(d.Get("spec").([]interface{}))

	log.Printf("[INFO] Updating GRPCRoute: %#v", existing)
	result, err := conn.GRPCRoutes(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return diag.Errorf("Failed to update GRPCRoute '%s' because: %s", name, err)
	}

	log.Printf("[INFO] Submitted updated GRPCRoute: %#v", result)

	return resourceKubernetesGRPCRouteV1Read(ctx, d, meta)
}

func resourceKubernetesGRPCRouteV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	name := metadata.Name
	namespace := metadata.Namespace

	log.Printf("[INFO] Deleting GRPCRoute %s", name)
	err = conn.GRPCRoutes(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			log.Printf("[DEBUG] GRPCRoute %s not found, removing from state", name)
			d.SetId("")
			return nil
		}
		return diag.Errorf("Failed to delete GRPCRoute '%s' because: %s", name, err)
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		_, err := conn.GRPCRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		e := fmt.Errorf("GRPCRoute (%s) still exists", d.Id())
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] GRPCRoute %s deleted", name)
	d.SetId("")

	return nil
}
