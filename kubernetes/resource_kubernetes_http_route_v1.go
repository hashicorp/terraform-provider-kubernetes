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

func resourceKubernetesHTTPRouteV1() *schema.Resource {
	return &schema.Resource{
		Description:        "HTTPRoute provides a way to route HTTP requests.",
		CreateContext:      resourceKubernetesHTTPRouteV1Create,
		ReadContext:        resourceKubernetesHTTPRouteV1Read,
		UpdateContext:      resourceKubernetesHTTPRouteV1Update,
		DeleteContext:      resourceKubernetesHTTPRouteV1Delete,
		DeprecationMessage: "",
		Schema:             resourceKubernetesHTTPRouteV1Schema(),
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

func resourceKubernetesHTTPRouteV1Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"metadata": namespacedMetadataSchema("httproute_v1", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Spec defines the desired state of HTTPRoute.",
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
						Description: "Hostnames defines a set of hostnames that should match against the HTTP Host header.",
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
						Description: "Rules are a list of HTTP matchers, filters and actions.",
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
									Description: "Matches define conditions used for matching the rule against incoming HTTP requests.",
									Optional:    true,
									Computed:    true,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"path": {
												Type:        schema.TypeList,
												Description: "Path specifies the HTTP request path match.",
												Optional:    true,
												Computed:    true,
												MaxItems:    1,
												Elem: &schema.Resource{
													Schema: map[string]*schema.Schema{
														"type": {
															Type:        schema.TypeString,
															Description: "Type defines the type of path match.",
															Optional:    true,
															Default:     "PathPrefix",
														},
														"value": {
															Type:        schema.TypeString,
															Description: "Value is the path value.",
															Optional:    true,
															Default:     "/",
														},
													},
												},
											},
											"headers": {
												Type:        schema.TypeList,
												Description: "Headers specifies the HTTP request header match.",
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
											"query_params": {
												Type:        schema.TypeList,
												Description: "QueryParams specifies the HTTP query parameter match.",
												Optional:    true,
												Elem: &schema.Resource{
													Schema: map[string]*schema.Schema{
														"name": {
															Type:        schema.TypeString,
															Description: "Name is the query parameter name.",
															Required:    true,
														},
														"value": {
															Type:        schema.TypeString,
															Description: "Value is the query parameter value.",
															Required:    true,
														},
														"type": {
															Type:        schema.TypeString,
															Description: "Type defines the type of query param match.",
															Optional:    true,
															Default:     "Exact",
														},
													},
												},
											},
											"method": {
												Type:        schema.TypeString,
												Description: "Method specifies the HTTP method match.",
												Optional:    true,
												ValidateFunc: validation.StringInSlice([]string{
													"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "CONNECT", "TRACE",
												}, false),
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
											"request_header_modifier": {
												Type:        schema.TypeList,
												Description: "RequestHeaderModifier modifies request headers.",
												Optional:    true,
												MaxItems:    1,
												Elem: &schema.Resource{
													Schema: map[string]*schema.Schema{
														"set": {
															Type:        schema.TypeList,
															Description: "Set overwrites the request headers.",
															Optional:    true,
															Elem: &schema.Resource{
																Schema: httpHeaderSchema(),
															},
														},
														"add": {
															Type:        schema.TypeList,
															Description: "Add adds request headers.",
															Optional:    true,
															Elem: &schema.Resource{
																Schema: httpHeaderSchema(),
															},
														},
														"remove": {
															Type:        schema.TypeList,
															Description: "Remove removes request headers.",
															Optional:    true,
															Elem:        &schema.Schema{Type: schema.TypeString},
														},
													},
												},
											},
											"response_header_modifier": {
												Type:        schema.TypeList,
												Description: "ResponseHeaderModifier modifies response headers.",
												Optional:    true,
												MaxItems:    1,
												Elem: &schema.Resource{
													Schema: map[string]*schema.Schema{
														"set": {
															Type:        schema.TypeList,
															Description: "Set overwrites the response headers.",
															Optional:    true,
															Elem: &schema.Resource{
																Schema: httpHeaderSchema(),
															},
														},
														"add": {
															Type:        schema.TypeList,
															Description: "Add adds response headers.",
															Optional:    true,
															Elem: &schema.Resource{
																Schema: httpHeaderSchema(),
															},
														},
														"remove": {
															Type:        schema.TypeList,
															Description: "Remove removes response headers.",
															Optional:    true,
															Elem:        &schema.Schema{Type: schema.TypeString},
														},
													},
												},
											},
											"request_redirect": {
												Type:        schema.TypeList,
												Description: "RequestRedirect redirects the request.",
												Optional:    true,
												MaxItems:    1,
												Elem: &schema.Resource{
													Schema: map[string]*schema.Schema{
														"scheme": {
															Type:        schema.TypeString,
															Description: "Scheme is the scheme to use for the redirect.",
															Optional:    true,
														},
														"hostname": {
															Type:        schema.TypeString,
															Description: "Hostname is the hostname to use for the redirect.",
															Optional:    true,
														},
														"path": {
															Type:        schema.TypeList,
															Description: "Path specifies the path modification for the redirect.",
															Optional:    true,
															MaxItems:    1,
															Elem: &schema.Resource{
																Schema: map[string]*schema.Schema{
																	"type": {
																		Type:        schema.TypeString,
																		Description: "Type is the type of path modification.",
																		Optional:    true,
																	},
																	"replace_full_path": {
																		Type:        schema.TypeString,
																		Description: "ReplaceFullPath is the full path to use for the redirect.",
																		Optional:    true,
																	},
																	"replace_prefix_match": {
																		Type:        schema.TypeString,
																		Description: "ReplacePrefixMatch is the prefix to replace for the redirect.",
																		Optional:    true,
																	},
																},
															},
														},
														"port": {
															Type:         schema.TypeInt,
															Description:  "Port is the port to use for the redirect.",
															Optional:     true,
															ValidateFunc: validation.IsPortNumber,
														},
														"status_code": {
															Type:        schema.TypeInt,
															Description: "StatusCode is the HTTP status code to use for the redirect.",
															Optional:    true,
															Default:     302,
														},
													},
												},
											},
											"url_rewrite": {
												Type:        schema.TypeList,
												Description: "URLRewrite rewrites the URL of the request.",
												Optional:    true,
												MaxItems:    1,
												Elem: &schema.Resource{
													Schema: map[string]*schema.Schema{
														"hostname": {
															Type:        schema.TypeString,
															Description: "Hostname is the hostname to use for the rewrite.",
															Optional:    true,
														},
														"path": {
															Type:        schema.TypeList,
															Description: "Path specifies the path modification for the rewrite.",
															Optional:    true,
															MaxItems:    1,
															Elem: &schema.Resource{
																Schema: map[string]*schema.Schema{
																	"type": {
																		Type:        schema.TypeString,
																		Description: "Type is the type of path modification.",
																		Optional:    true,
																	},
																	"replace_full_path": {
																		Type:        schema.TypeString,
																		Description: "ReplaceFullPath is the full path to use for the rewrite.",
																		Optional:    true,
																	},
																	"replace_prefix_match": {
																		Type:        schema.TypeString,
																		Description: "ReplacePrefixMatch is the prefix to replace for the rewrite.",
																		Optional:    true,
																	},
																},
															},
														},
													},
												},
											},
											"request_mirror": {
												Type:        schema.TypeList,
												Description: "RequestMirror mirrors requests to another backend.",
												Optional:    true,
												MaxItems:    1,
												Elem: &schema.Resource{
													Schema: map[string]*schema.Schema{
														"backend_ref": {
															Type:        schema.TypeList,
															Description: "BackendRef is the reference to the backend.",
															Required:    true,
															MaxItems:    1,
															Elem: &schema.Resource{
																Schema: backendObjectReferenceSchema(),
															},
														},
														"percent": {
															Type:        schema.TypeInt,
															Description: "Percent is the percentage of requests to mirror.",
															Optional:    true,
														},
													},
												},
											},
											"cors": {
												Type:        schema.TypeList,
												Description: "CORS filter configuration.",
												Optional:    true,
												MaxItems:    1,
												Elem: &schema.Resource{
													Schema: map[string]*schema.Schema{
														"allow_origins": {
															Type:        schema.TypeList,
															Description: "AllowOrigins specifies the origins that are allowed.",
															Optional:    true,
															Elem:        &schema.Schema{Type: schema.TypeString},
														},
														"allow_credentials": {
															Type:        schema.TypeBool,
															Description: "AllowCredentials indicates whether credentials are allowed.",
															Optional:    true,
														},
														"allow_methods": {
															Type:        schema.TypeList,
															Description: "AllowMethods specifies the HTTP methods that are allowed.",
															Optional:    true,
															Elem:        &schema.Schema{Type: schema.TypeString},
														},
														"allow_headers": {
															Type:        schema.TypeList,
															Description: "AllowHeaders specifies the HTTP headers that are allowed.",
															Optional:    true,
															Elem:        &schema.Schema{Type: schema.TypeString},
														},
														"expose_headers": {
															Type:        schema.TypeList,
															Description: "ExposeHeaders specifies the HTTP headers that are exposed.",
															Optional:    true,
															Elem:        &schema.Schema{Type: schema.TypeString},
														},
														"max_age": {
															Type:        schema.TypeInt,
															Description: "MaxAge is the maximum age in seconds for preflight cache.",
															Optional:    true,
														},
													},
												},
											},
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
												Description: "Weight specifies the proportion of requests forwarded to this backend.",
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
														"request_header_modifier":  headerModifierFilterSchema(),
														"response_header_modifier": headerModifierFilterSchema(),
														"request_redirect":         requestRedirectFilterSchema(),
														"url_rewrite":              urlRewriteFilterSchema(),
														"request_mirror":           requestMirrorFilterSchema(),
														"cors":                     corsFilterSchema(),
														"extension_ref":            extensionRefFilterSchema(),
													},
												},
											},
										},
									},
								},
								"timeouts": {
									Type:        schema.TypeList,
									Description: "Timeouts defines the timeouts that can be configured for an HTTP request.",
									Optional:    true,
									MaxItems:    1,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"request": {
												Type:        schema.TypeString,
												Description: "Request specifies the maximum duration for a gateway to respond to an HTTP request.",
												Optional:    true,
											},
											"backend_request": {
												Type:        schema.TypeString,
												Description: "BackendRequest specifies a timeout for an individual request from the gateway to a backend.",
												Optional:    true,
											},
										},
									},
								},
								"retry": {
									Type:        schema.TypeList,
									Description: "Retry defines the configuration for when to retry an HTTP request.",
									Optional:    true,
									MaxItems:    1,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"codes": {
												Type:        schema.TypeList,
												Description: "Codes defines the HTTP response status codes for which a backend request should be retried.",
												Optional:    true,
												Elem:        &schema.Schema{Type: schema.TypeInt},
											},
											"attempts": {
												Type:        schema.TypeInt,
												Description: "Attempts specifies the maximum number of times an individual request should be retried.",
												Optional:    true,
											},
											"backoff": {
												Type:        schema.TypeString,
												Description: "Backoff specifies the duration between retries.",
												Optional:    true,
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
											"type": {
												Type:         schema.TypeString,
												Description:  "Type defines the type of session persistence. Supported values: Cookie, Header.",
												Optional:     true,
												ValidateFunc: validation.StringInSlice([]string{"Cookie", "Header"}, false),
											},
											"cookie_config": {
												Type:        schema.TypeList,
												Description: "CookieConfig provides configuration for the cookie lifetime in cookie-based session persistence.",
												Optional:    true,
												MaxItems:    1,
												Elem: &schema.Resource{
													Schema: map[string]*schema.Schema{
														"lifetime_type": {
															Type:         schema.TypeString,
															Description:  "LifetimeType specifies whether the cookie has a permanent or session-based lifetime. Supported values: Permanent, Session.",
															Optional:     true,
															ValidateFunc: validation.StringInSlice([]string{"Permanent", "Session"}, false),
														},
													},
												},
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
	}
}

func httpHeaderSchema() map[string]*schema.Schema {
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

func backendObjectReferenceSchema() map[string]*schema.Schema {
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

func headerModifierFilterSchema() *schema.Schema {
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
						Schema: httpHeaderSchema(),
					},
				},
				"add": {
					Type:        schema.TypeList,
					Description: "Add adds headers.",
					Optional:    true,
					Elem: &schema.Resource{
						Schema: httpHeaderSchema(),
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

func requestRedirectFilterSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Description: "RequestRedirect redirects the request.",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"scheme": {
					Type:        schema.TypeString,
					Description: "Scheme is the scheme to use for the redirect.",
					Optional:    true,
				},
				"hostname": {
					Type:        schema.TypeString,
					Description: "Hostname is the hostname to use for the redirect.",
					Optional:    true,
				},
				"path": {
					Type:        schema.TypeList,
					Description: "Path specifies the path modification for the redirect.",
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"type": {
								Type:        schema.TypeString,
								Description: "Type is the type of path modification.",
								Optional:    true,
							},
							"replace_full_path": {
								Type:        schema.TypeString,
								Description: "ReplaceFullPath is the full path to use for the redirect.",
								Optional:    true,
							},
							"replace_prefix_match": {
								Type:        schema.TypeString,
								Description: "ReplacePrefixMatch is the prefix to replace for the redirect.",
								Optional:    true,
							},
						},
					},
				},
				"port": {
					Type:        schema.TypeInt,
					Description: "Port is the port to use for the redirect.",
					Optional:    true,
				},
				"status_code": {
					Type:        schema.TypeInt,
					Description: "StatusCode is the HTTP status code to use for the redirect.",
					Optional:    true,
					Default:     302,
				},
			},
		},
	}
}

func urlRewriteFilterSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Description: "URLRewrite rewrites the URL of the request.",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"hostname": {
					Type:        schema.TypeString,
					Description: "Hostname is the hostname to use for the rewrite.",
					Optional:    true,
				},
				"path": {
					Type:        schema.TypeList,
					Description: "Path specifies the path modification for the rewrite.",
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"type": {
								Type:        schema.TypeString,
								Description: "Type is the type of path modification.",
								Optional:    true,
							},
							"replace_full_path": {
								Type:        schema.TypeString,
								Description: "ReplaceFullPath is the full path to use for the rewrite.",
								Optional:    true,
							},
							"replace_prefix_match": {
								Type:        schema.TypeString,
								Description: "ReplacePrefixMatch is the prefix to replace for the rewrite.",
								Optional:    true,
							},
						},
					},
				},
			},
		},
	}
}

func requestMirrorFilterSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Description: "RequestMirror mirrors requests to another backend.",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"backend_ref": {
					Type:        schema.TypeList,
					Description: "BackendRef is the reference to the backend.",
					Required:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: backendObjectReferenceSchema(),
					},
				},
				"percent": {
					Type:        schema.TypeInt,
					Description: "Percent is the percentage of requests to mirror.",
					Optional:    true,
				},
			},
		},
	}
}

func corsFilterSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Description: "CORS filter configuration.",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"allow_origins": {
					Type:        schema.TypeList,
					Description: "AllowOrigins specifies the origins that are allowed.",
					Optional:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
				"allow_credentials": {
					Type:        schema.TypeBool,
					Description: "AllowCredentials indicates whether credentials are allowed.",
					Optional:    true,
				},
				"allow_methods": {
					Type:        schema.TypeList,
					Description: "AllowMethods specifies the HTTP methods that are allowed.",
					Optional:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
				"allow_headers": {
					Type:        schema.TypeList,
					Description: "AllowHeaders specifies the HTTP headers that are allowed.",
					Optional:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
				"expose_headers": {
					Type:        schema.TypeList,
					Description: "ExposeHeaders specifies the HTTP headers that are exposed.",
					Optional:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
				"max_age": {
					Type:        schema.TypeInt,
					Description: "MaxAge is the maximum age in seconds for preflight cache.",
					Optional:    true,
				},
			},
		},
	}
}

// parentReferenceSchema returns the shared schema for ParentReference used in route specs.
func parentReferenceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"group": {
			Type:        schema.TypeString,
			Description: "Group is the group of the referent. Defaults to gateway.networking.k8s.io.",
			Optional:    true,
			Default:     "gateway.networking.k8s.io",
		},
		"kind": {
			Type:        schema.TypeString,
			Description: "Kind is the kind of the referent. Defaults to Gateway.",
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
			Description: "SectionName is the name of a section within the target resource (e.g., a listener name).",
			Optional:    true,
		},
		"port": {
			Type:         schema.TypeInt,
			Description:  "Port is the network port this Route targets.",
			Optional:     true,
			ValidateFunc: validation.IsPortNumber,
		},
	}
}

func extensionRefFilterSchema() *schema.Schema {
	return &schema.Schema{
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
	}
}

func resourceKubernetesHTTPRouteV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec := expandHTTPRouteSpec(d.Get("spec").([]interface{}))

	log.Printf("[INFO] Creating new HTTPRoute: %#v", spec)
	out := &gatewayv1.HTTPRoute{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	result, err := conn.HTTPRoutes(metadata.Namespace).Create(ctx, out, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new HTTPRoute: %#v", result)
	d.SetId(buildId(result.ObjectMeta))

	return resourceKubernetesHTTPRouteV1Read(ctx, d, meta)
}

func resourceKubernetesHTTPRouteV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading HTTPRoute %s", name)
	route, err := conn.HTTPRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
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

	err = setResourceIdentityNamespaced(d, "gateway.networking.k8s.io/v1", "HTTPRoute", route.Namespace, route.Name)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesHTTPRouteV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	existing, err := conn.HTTPRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	existing.Labels = metadata.Labels
	existing.Annotations = metadata.Annotations
	existing.Spec = expandHTTPRouteSpec(d.Get("spec").([]interface{}))

	log.Printf("[INFO] Updating HTTPRoute: %#v", existing)
	result, err := conn.HTTPRoutes(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return diag.Errorf("Failed to update HTTPRoute '%s' because: %s", name, err)
	}

	log.Printf("[INFO] Submitted updated HTTPRoute: %#v", result)

	return resourceKubernetesHTTPRouteV1Read(ctx, d, meta)
}

func resourceKubernetesHTTPRouteV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	name := metadata.Name
	namespace := metadata.Namespace

	log.Printf("[INFO] Deleting HTTPRoute %s", name)
	err = conn.HTTPRoutes(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			log.Printf("[DEBUG] HTTPRoute %s not found, removing from state", name)
			d.SetId("")
			return nil
		}
		return diag.Errorf("Failed to delete HTTPRoute '%s' because: %s", name, err)
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		_, err := conn.HTTPRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		e := fmt.Errorf("HTTPRoute (%s) still exists", d.Id())
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] HTTPRoute %s deleted", name)
	d.SetId("")

	return nil
}
