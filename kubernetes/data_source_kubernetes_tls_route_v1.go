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

func dataSourceKubernetesTLSRouteV1() *schema.Resource {
	return &schema.Resource{
		Description: "Get information about a TLSRoute resource.",
		ReadContext: dataSourceKubernetesTLSRouteV1Read,
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("tlsroute_v1", false),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the desired state of TLSRoute.",
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
							Description: "Hostnames defines a set of SNI hostnames that should match against the SNI attribute.",
							Computed:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"rules": {
							Type:        schema.TypeList,
							Description: "Rules are a list of TLS matchers and actions.",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Description: "Name is the name of the route rule.",
										Computed:    true,
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
		},
	}
}

func tlsBackendObjectReferenceSchemaComputed() map[string]*schema.Schema {
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

func dataSourceKubernetesTLSRouteV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	name := metadata.Name
	namespace := metadata.Namespace

	log.Printf("[INFO] Reading TLSRoute %s", name)
	route, err := conn.TLSRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
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

	return nil
}
