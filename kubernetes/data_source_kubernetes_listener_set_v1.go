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

func dataSourceKubernetesListenerSetV1() *schema.Resource {
	return &schema.Resource{
		Description: "Get information about a ListenerSet resource.",
		ReadContext: dataSourceKubernetesListenerSetV1Read,
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("listenerset_v1", false),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the desired state of ListenerSet.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"parent_ref": {
							Type:        schema.TypeList,
							Description: "ParentRef references the Gateway that the listeners are attached to.",
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
								},
							},
						},
						"listeners": {
							Type:        schema.TypeList,
							Description: "Listeners associated with this ListenerSet.",
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
										Description: "Hostname specifies the virtual hostname to match.",
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
														Schema: listenerSetSecretObjectReferenceSchemaComputed(),
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
														Schema: listenerSetRouteNamespacesSchemaComputed(),
													},
												},
												"kinds": {
													Type:        schema.TypeList,
													Description: "Kinds specifies the groups and kinds of Routes.",
													Computed:    true,
													Elem: &schema.Resource{
														Schema: listenerSetRouteGroupKindSchemaComputed(),
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
											Schema: listenerSetRouteGroupKindSchemaComputed(),
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
		},
	}
}

func listenerSetSecretObjectReferenceSchemaComputed() map[string]*schema.Schema {
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
	}
}

func listenerSetRouteNamespacesSchemaComputed() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"from": {
			Type:        schema.TypeString,
			Description: "From indicates where Routes will be selected from.",
			Computed:    true,
		},
		"selector": {
			Type:        schema.TypeList,
			Description: "Selector labels Routes in the selected namespaces.",
			Computed:    true,
			Elem: &schema.Resource{
				Schema: listenerSetLabelSelectorSchemaComputed(),
			},
		},
	}
}

func listenerSetLabelSelectorSchemaComputed() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"match_labels": {
			Type:        schema.TypeMap,
			Description: "MatchLabels is a map of {key,value} pairs.",
			Computed:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"match_expressions": {
			Type:        schema.TypeList,
			Description: "MatchExpressions is a list of label selector requirements.",
			Computed:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"key": {
						Type:        schema.TypeString,
						Description: "Key is the label key that the selector applies to.",
						Computed:    true,
					},
					"operator": {
						Type:        schema.TypeString,
						Description: "Operator represents a key's relationship to a set of values.",
						Computed:    true,
					},
					"values": {
						Type:        schema.TypeList,
						Description: "Values is an array of string values.",
						Computed:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
					},
				},
			},
		},
	}
}

func listenerSetRouteGroupKindSchemaComputed() map[string]*schema.Schema {
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
	}
}

func dataSourceKubernetesListenerSetV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	name := metadata.Name
	namespace := metadata.Namespace

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

	return nil
}
