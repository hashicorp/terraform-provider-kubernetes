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

func dataSourceKubernetesHTTPRouteV1() *schema.Resource {
	return &schema.Resource{
		Description: "Get information about an HTTPRoute resource.",
		ReadContext: dataSourceKubernetesHTTPRouteV1Read,
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("httproute_v1", false),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the desired state of HTTPRoute.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"parent_refs": {
							Type:        schema.TypeList,
							Description: "ParentRefs identifies an API object (usually a Gateway) that routes should reference to attach to it.",
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
										Description: "SectionName is the name of a section within the target resource.",
										Computed:    true,
									},
									"port": {
										Type:        schema.TypeInt,
										Description: "Port is the network port this Route targets.",
										Computed:    true,
									},
								},
							},
						},
						"hostnames": {
							Type:        schema.TypeList,
							Description: "Hostnames defines a set of hostnames that should match against the HTTP Host header.",
							Computed:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"use_default_gateways": {
							Type:        schema.TypeString,
							Description: "UseDefaultGateways indicates the default Gateway scope.",
							Computed:    true,
						},
						"rules": {
							Type:        schema.TypeList,
							Description: "Rules are a list of HTTP matchers, filters and actions.",
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
										Description: "Matches define conditions used for matching the rule against incoming HTTP requests.",
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"path": {
													Type:        schema.TypeList,
													Description: "Path specifies the HTTP request path match.",
													Computed:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"type": {
																Type:        schema.TypeString,
																Description: "Type defines the type of path match.",
																Computed:    true,
															},
															"value": {
																Type:        schema.TypeString,
																Description: "Value is the path value.",
																Computed:    true,
															},
														},
													},
												},
												"headers": {
													Type:        schema.TypeList,
													Description: "Headers specifies the HTTP request header match.",
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
												"query_params": {
													Type:        schema.TypeList,
													Description: "QueryParams specifies the HTTP query parameter match.",
													Computed:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"name": {
																Type:        schema.TypeString,
																Description: "Name is the query parameter name.",
																Computed:    true,
															},
															"value": {
																Type:        schema.TypeString,
																Description: "Value is the query parameter value.",
																Computed:    true,
															},
															"type": {
																Type:        schema.TypeString,
																Description: "Type defines the type of query param match.",
																Computed:    true,
															},
														},
													},
												},
												"method": {
													Type:        schema.TypeString,
													Description: "Method specifies the HTTP method match.",
													Computed:    true,
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
												"request_header_modifier":  headerModifierFilterSchemaComputed(),
												"response_header_modifier": headerModifierFilterSchemaComputed(),
												"request_redirect":         requestRedirectFilterSchemaComputed(),
												"url_rewrite":              urlRewriteFilterSchemaComputed(),
												"request_mirror":           requestMirrorFilterSchemaComputed(),
												"cors":                     corsFilterSchemaComputed(),
												"extension_ref":            extensionRefFilterSchemaComputed(),
											},
										},
									},
									"backend_refs": {
										Type:        schema.TypeList,
										Description: "BackendRefs defines the backend(s) where matching requests should be sent.",
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
												"weight": {
													Type:        schema.TypeInt,
													Description: "Weight specifies the proportion of requests forwarded to this backend.",
													Computed:    true,
												},
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
															"request_header_modifier":  headerModifierFilterSchemaComputed(),
															"response_header_modifier": headerModifierFilterSchemaComputed(),
															"request_redirect":         requestRedirectFilterSchemaComputed(),
															"url_rewrite":              urlRewriteFilterSchemaComputed(),
															"request_mirror":           requestMirrorFilterSchemaComputed(),
															"cors":                     corsFilterSchemaComputed(),
															"extension_ref":            extensionRefFilterSchemaComputed(),
														},
													},
												},
											},
										},
									},
									"timeouts": {
										Type:        schema.TypeList,
										Description: "Timeouts defines the timeouts that can be configured for an HTTP request.",
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"request": {
													Type:        schema.TypeString,
													Description: "Request specifies the maximum duration for a gateway to respond to an HTTP request.",
													Computed:    true,
												},
												"backend_request": {
													Type:        schema.TypeString,
													Description: "BackendRequest specifies a timeout for an individual request from the gateway to a backend.",
													Computed:    true,
												},
											},
										},
									},
									"retry": {
										Type:        schema.TypeList,
										Description: "Retry defines the configuration for when to retry an HTTP request.",
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"codes": {
													Type:        schema.TypeList,
													Description: "Codes defines the HTTP response status codes for which a backend request should be retried.",
													Computed:    true,
													Elem:        &schema.Schema{Type: schema.TypeInt},
												},
												"attempts": {
													Type:        schema.TypeInt,
													Description: "Attempts specifies the maximum number of times an individual request should be retried.",
													Computed:    true,
												},
												"backoff": {
													Type:        schema.TypeString,
													Description: "Backoff specifies the duration between retries.",
													Computed:    true,
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
												"cookie_config": {
													Type:        schema.TypeList,
													Description: "CookieConfig specifies the configuration for session persistence via cookies.",
													Computed:    true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"lifetime_type": {
																Type:        schema.TypeString,
																Description: "LifetimeType specifies the cookie lifetime type.",
																Computed:    true,
															},
														},
													},
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
				Description: "Status defines the current state of HTTPRoute.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"parents": {
							Type:        schema.TypeList,
							Description: "Parents is a list of parent resources that this route is attached to.",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"parent_ref": {
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
									},
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
											Schema: map[string]*schema.Schema{
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

func backendObjectReferenceSchemaComputed() map[string]*schema.Schema {
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

func headerModifierFilterSchemaComputed() *schema.Schema {
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
						Schema: httpHeaderSchemaComputed(),
					},
				},
				"add": {
					Type:        schema.TypeList,
					Description: "Add adds headers.",
					Computed:    true,
					Elem: &schema.Resource{
						Schema: httpHeaderSchemaComputed(),
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

func httpHeaderSchemaComputed() map[string]*schema.Schema {
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

func requestRedirectFilterSchemaComputed() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Description: "RequestRedirect redirects the request.",
		Computed:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"scheme": {
					Type:        schema.TypeString,
					Description: "Scheme is the scheme to use for the redirect.",
					Computed:    true,
				},
				"hostname": {
					Type:        schema.TypeString,
					Description: "Hostname is the hostname to use for the redirect.",
					Computed:    true,
				},
				"path": {
					Type:        schema.TypeList,
					Description: "Path specifies the path modification for the redirect.",
					Computed:    true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"type": {
								Type:        schema.TypeString,
								Description: "Type is the type of path modification.",
								Computed:    true,
							},
							"replace_full_path": {
								Type:        schema.TypeString,
								Description: "ReplaceFullPath is the full path to use for the redirect.",
								Computed:    true,
							},
							"replace_prefix_match": {
								Type:        schema.TypeString,
								Description: "ReplacePrefixMatch is the prefix to replace for the redirect.",
								Computed:    true,
							},
						},
					},
				},
				"port": {
					Type:        schema.TypeInt,
					Description: "Port is the port to use for the redirect.",
					Computed:    true,
				},
				"status_code": {
					Type:        schema.TypeInt,
					Description: "StatusCode is the HTTP status code to use for the redirect.",
					Computed:    true,
				},
			},
		},
	}
}

func urlRewriteFilterSchemaComputed() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Description: "URLRewrite rewrites the URL of the request.",
		Computed:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"hostname": {
					Type:        schema.TypeString,
					Description: "Hostname is the hostname to use for the rewrite.",
					Computed:    true,
				},
				"path": {
					Type:        schema.TypeList,
					Description: "Path specifies the path modification for the rewrite.",
					Computed:    true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"type": {
								Type:        schema.TypeString,
								Description: "Type is the type of path modification.",
								Computed:    true,
							},
							"replace_full_path": {
								Type:        schema.TypeString,
								Description: "ReplaceFullPath is the full path to use for the rewrite.",
								Computed:    true,
							},
							"replace_prefix_match": {
								Type:        schema.TypeString,
								Description: "ReplacePrefixMatch is the prefix to replace for the rewrite.",
								Computed:    true,
							},
						},
					},
				},
			},
		},
	}
}

func requestMirrorFilterSchemaComputed() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Description: "RequestMirror mirrors requests to another backend.",
		Computed:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"backend_ref": {
					Type:        schema.TypeList,
					Description: "BackendRef is the reference to the backend.",
					Computed:    true,
					Elem: &schema.Resource{
						Schema: backendObjectReferenceSchemaComputed(),
					},
				},
				"percent": {
					Type:        schema.TypeInt,
					Description: "Percent is the percentage of requests to mirror.",
					Computed:    true,
				},
			},
		},
	}
}

func corsFilterSchemaComputed() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Description: "CORS filter configuration.",
		Computed:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"allow_origins": {
					Type:        schema.TypeList,
					Description: "AllowOrigins specifies the origins that are allowed.",
					Computed:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
				"allow_credentials": {
					Type:        schema.TypeBool,
					Description: "AllowCredentials indicates whether credentials are allowed.",
					Computed:    true,
				},
				"allow_methods": {
					Type:        schema.TypeList,
					Description: "AllowMethods specifies the HTTP methods that are allowed.",
					Computed:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
				"allow_headers": {
					Type:        schema.TypeList,
					Description: "AllowHeaders specifies the HTTP headers that are allowed.",
					Computed:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
				"expose_headers": {
					Type:        schema.TypeList,
					Description: "ExposeHeaders specifies the HTTP headers that are exposed.",
					Computed:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
				"max_age": {
					Type:        schema.TypeInt,
					Description: "MaxAge is the maximum age in seconds for preflight cache.",
					Computed:    true,
				},
			},
		},
	}
}

func extensionRefFilterSchemaComputed() *schema.Schema {
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

func dataSourceKubernetesHTTPRouteV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	name := metadata.Name
	namespace := metadata.Namespace

	log.Printf("[INFO] Reading HTTPRoute %s", name)
	route, err := conn.HTTPRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("[DEBUG] HTTPRoute %s not found, removing from state", name)
			d.SetId("")
			return diag.Diagnostics{}
		}
		return diag.Errorf("Failed to read HTTPRoute '%s' because: %s", name, err)
	}
	log.Printf("[INFO] Received HTTPRoute: %#v", route)

	err = d.Set("metadata", flattenMetadata(route.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedSpec := flattenHTTPRouteSpec(route.Spec)
	log.Printf("[DEBUG] Flattened HTTPRoute spec: %#v", flattenedSpec)
	err = d.Set("spec", flattenedSpec)
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedStatus := flattenHTTPRouteStatus(route.Status)
	err = d.Set("status", flattenedStatus)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildId(route.ObjectMeta))

	return nil
}
