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

func dataSourceKubernetesGRPCRouteV1() *schema.Resource {
	return &schema.Resource{
		Description: "Get information about a GRPCRoute resource.",
		ReadContext: dataSourceKubernetesGRPCRouteV1Read,
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("grpcroute_v1", false),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the desired state of GRPCRoute.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"parent_refs": {
							Type:        schema.TypeList,
							Description: "ParentRefs identifies an API object (usually a Gateway) that the route should attach to.",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"group":        {Type: schema.TypeString, Computed: true},
									"kind":         {Type: schema.TypeString, Computed: true},
									"namespace":    {Type: schema.TypeString, Computed: true},
									"name":         {Type: schema.TypeString, Computed: true},
									"section_name": {Type: schema.TypeString, Computed: true},
									"port":         {Type: schema.TypeInt, Computed: true},
								},
							},
						},
						"use_default_gateways": {
							Type:        schema.TypeString,
							Description: "UseDefaultGateways indicates the default Gateway scope.",
							Computed:    true,
						},
						"hostnames": {
							Type:        schema.TypeList,
							Description: "Hostnames defines a set of hostnames that should match against the GRPC Host header.",
							Computed:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"rules": {
							Type:        schema.TypeList,
							Description: "Rules are a list of GRPC matchers, filters and actions.",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Description: "Name is the name of the route rule.",
										Computed:    true,
									},
									"matches": {
										Type:        schema.TypeList,
										Description: "Matches define conditions used for matching the rule against incoming gRPC requests.",
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"method": {
													Type:        schema.TypeList,
													Description: "Method specifies a gRPC request service/method matcher.",
													Computed:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"type": {
																Type:        schema.TypeString,
																Description: "Type specifies how to match against the service and/or method.",
																Computed:    true,
															},
															"service": {
																Type:        schema.TypeString,
																Description: "Service is the gRPC service name.",
																Computed:    true,
															},
															"method": {
																Type:        schema.TypeString,
																Description: "Method is the gRPC method name.",
																Computed:    true,
															},
														},
													},
												},
												"headers": {
													Type:        schema.TypeList,
													Description: "Headers specifies gRPC request header matchers.",
													Computed:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"name": {
																Type:        schema.TypeString,
																Description: "Name is the header name.",
																Computed:    true,
															},
															"value": {
																Type:        schema.TypeString,
																Description: "Value is the header value.",
																Computed:    true,
															},
															"type": {
																Type:        schema.TypeString,
																Description: "Type defines the type of header match.",
																Computed:    true,
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
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"type": {
													Type:        schema.TypeString,
													Description: "Type is the type of filter.",
													Computed:    true,
												},
												"request_header_modifier":  grpcHeaderModifierFilterSchemaComputed(),
												"response_header_modifier": grpcHeaderModifierFilterSchemaComputed(),
												"request_mirror":           requestMirrorFilterSchemaComputed(),
												"extension_ref":            grpcExtensionRefFilterSchemaComputed(),
											},
										},
									},
									"backend_refs": {
										Type:        schema.TypeList,
										Description: "BackendRefs defines the backend(s) where matching requests should be sent.",
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"group":     {Type: schema.TypeString, Computed: true},
												"kind":      {Type: schema.TypeString, Computed: true},
												"name":      {Type: schema.TypeString, Computed: true},
												"namespace": {Type: schema.TypeString, Computed: true},
												"port":      {Type: schema.TypeInt, Computed: true},
												"weight":    {Type: schema.TypeInt, Computed: true},
												"filters": {
													Type:        schema.TypeList,
													Description: "Filters defined at this level are applied before any rule-level filters.",
													Computed:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"type": {
																Type:        schema.TypeString,
																Description: "Type is the type of filter.",
																Computed:    true,
															},
															"request_header_modifier":  grpcHeaderModifierFilterSchemaComputed(),
															"response_header_modifier": grpcHeaderModifierFilterSchemaComputed(),
															"request_mirror":           requestMirrorFilterSchemaComputed(),
															"extension_ref":            grpcExtensionRefFilterSchemaComputed(),
														},
													},
												},
											},
										},
									},
									"session_persistence": {
										Type:        schema.TypeList,
										Description: "SessionPersistence defines and configures session persistence for the route rule.",
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"session_name": {
													Type:        schema.TypeString,
													Description: "SessionName is the name of the session.",
													Computed:    true,
												},
												"absolute_timeout": {
													Type:        schema.TypeString,
													Description: "AbsoluteTimeout specifies the maximum duration for the session.",
													Computed:    true,
												},
												"idle_timeout": {
													Type:        schema.TypeString,
													Description: "IdleTimeout specifies the duration after which an idle session should be expired.",
													Computed:    true,
												},
												"type": {
													Type:        schema.TypeString,
													Description: "Type defines the type of session persistence.",
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
		},
	}
}

func grpcBackendObjectReferenceSchemaComputed() map[string]*schema.Schema {
	return map[string]*schema.Schema{
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
		"port": {
			Type:        schema.TypeInt,
			Description: "Port is the port number of the referent.",
			Computed:    true,
		},
	}
}

func grpcHeaderModifierFilterSchemaComputed() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Description: "HeaderModifier modifies request or response headers.",
		Computed:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"set": {
					Type:        schema.TypeList,
					Description: "Set overwrites the headers.",
					Computed:    true,
					Elem: &schema.Resource{
						Schema: grpcHTTPHeaderSchemaComputed(),
					},
				},
				"add": {
					Type:        schema.TypeList,
					Description: "Add adds headers.",
					Computed:    true,
					Elem: &schema.Resource{
						Schema: grpcHTTPHeaderSchemaComputed(),
					},
				},
				"remove": {
					Type:        schema.TypeList,
					Description: "Remove removes headers.",
					Computed:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
			},
		},
	}
}

func grpcHTTPHeaderSchemaComputed() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": {
			Type:        schema.TypeString,
			Description: "Name is the header name.",
			Computed:    true,
		},
		"value": {
			Type:        schema.TypeString,
			Description: "Value is the header value.",
			Computed:    true,
		},
	}
}

func grpcExtensionRefFilterSchemaComputed() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Description: "ExtensionRef is a reference to a custom extension filter.",
		Computed:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": {
					Type:        schema.TypeString,
					Description: "Name is the name of the extension.",
					Computed:    true,
				},
			},
		},
	}
}

func dataSourceKubernetesGRPCRouteV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	name := metadata.Name
	namespace := metadata.Namespace

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

	return nil
}
