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

func dataSourceKubernetesBackendTLSPolicyV1() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesBackendTLSPolicyV1Read,
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("backend_tls_policy_v1", false),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the desired state of BackendTLSPolicy.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target_refs": {
							Type:        schema.TypeList,
							Description: "TargetRefs identifies an API object to apply the policy to.",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"group": {
										Type:        schema.TypeString,
										Description: "Group is the group of the target resource.",
										Computed:    true,
									},
									"kind": {
										Type:        schema.TypeString,
										Description: "Kind is kind of the target resource.",
										Computed:    true,
									},
									"name": {
										Type:        schema.TypeString,
										Description: "Name is the name of the target resource.",
										Computed:    true,
									},
									"section_name": {
										Type:        schema.TypeString,
										Description: "SectionName is the name of a section within the target resource.",
										Computed:    true,
									},
								},
							},
						},
						"validation": {
							Type:        schema.TypeList,
							Description: "Validation contains backend TLS validation configuration.",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ca_certificate_refs": {
										Type:        schema.TypeList,
										Description: "CACertificateRefs contains references to Kubernetes objects that contain a PEM-encoded TLS CA certificate bundle.",
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
											},
										},
									},
									"well_known_ca_certificates": {
										Type:        schema.TypeString,
										Description: "WellKnownCACertificates specifies whether a well-known set of CA certificates may be used.",
										Computed:    true,
									},
									"hostname": {
										Type:        schema.TypeString,
										Description: "Hostname is used for two purposes in the connection between Gateways and backends.",
										Computed:    true,
									},
									"subject_alt_names": {
										Type:        schema.TypeList,
										Description: "SubjectAltNames contains one or more Subject Alternative Names.",
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"type": {
													Type:        schema.TypeString,
													Description: "Type determines the format of the Subject Alternative Name.",
													Computed:    true,
												},
												"hostname": {
													Type:        schema.TypeString,
													Description: "Hostname contains Subject Alternative Name specified in DNS name format.",
													Computed:    true,
												},
												"uri": {
													Type:        schema.TypeString,
													Description: "URI contains Subject Alternative Name specified in a full URI format.",
													Computed:    true,
												},
											},
										},
									},
								},
							},
						},
						"options": {
							Type:        schema.TypeMap,
							Description: "Options are a list of key/value pairs to enable extended TLS configuration.",
							Computed:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"status": {
				Type:        schema.TypeList,
				Description: "Status defines the current state of BackendTLSPolicy.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ancestors": {
							Type:        schema.TypeList,
							Description: "Ancestors is a list of ancestor resources that are associated with the policy.",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ancestor_ref": {
										Type:        schema.TypeList,
										Description: "AncestorRef corresponds with a ParentRef in the spec.",
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
									"controller_name": {
										Type:        schema.TypeString,
										Description: "ControllerName is a domain/path string.",
										Computed:    true,
									},
									"conditions": {
										Type:        schema.TypeList,
										Description: "Conditions describes the status of the Policy.",
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"type":                 {Type: schema.TypeString, Computed: true},
												"status":               {Type: schema.TypeString, Computed: true},
												"message":              {Type: schema.TypeString, Computed: true},
												"reason":               {Type: schema.TypeString, Computed: true},
												"last_transition_time": {Type: schema.TypeString, Computed: true},
												"observed_generation":  {Type: schema.TypeInt, Computed: true},
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

func dataSourceKubernetesBackendTLSPolicyV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	name := metadata.Name
	namespace := metadata.Namespace

	log.Printf("[INFO] Reading BackendTLSPolicy %s", name)
	obj, err := conn.BackendTLSPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("[DEBUG] BackendTLSPolicy %s not found, removing from state", name)
			d.SetId("")
			return diag.Diagnostics{}
		}
		return diag.Errorf("Failed to read BackendTLSPolicy '%s' because: %s", name, err)
	}
	log.Printf("[INFO] Received BackendTLSPolicy: %#v", obj)

	err = d.Set("metadata", flattenMetadata(obj.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedSpec := flattenBackendTLSPolicySpec(obj.Spec)
	log.Printf("[DEBUG] Flattened BackendTLSPolicy spec: %#v", flattenedSpec)
	err = d.Set("spec", flattenedSpec)
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedStatus := flattenBackendTLSPolicyStatus(obj.Status)
	err = d.Set("status", flattenedStatus)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildId(obj.ObjectMeta))
	return diag.Diagnostics{}
}
