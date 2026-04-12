// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func resourceKubernetesBackendTLSPolicyV1() *schema.Resource {
	return &schema.Resource{
		Description:   "BackendTLSPolicy configures TLS settings for backend services.",
		CreateContext: resourceKubernetesBackendTLSPolicyV1Create,
		ReadContext:   resourceKubernetesBackendTLSPolicyV1Read,
		UpdateContext: resourceKubernetesBackendTLSPolicyV1Update,
		DeleteContext: resourceKubernetesBackendTLSPolicyV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceIdentityImportNamespaced,
		},
		Identity: resourceIdentitySchemaNamespaced(),
		Schema:   resourceKubernetesBackendTLSPolicyV1Schema(),
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
	}
}

func resourceKubernetesBackendTLSPolicyV1Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"metadata": namespacedMetadataSchema("backend_tls_policy_v1", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Spec defines the desired state of BackendTLSPolicy.",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"target_refs": {
						Type:        schema.TypeList,
						Description: "TargetRefs identifies an API object to apply the policy to.",
						Required:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"group": {
									Type:        schema.TypeString,
									Description: "Group is the group of the target resource.",
									Required:    true,
								},
								"kind": {
									Type:        schema.TypeString,
									Description: "Kind is kind of the target resource.",
									Required:    true,
								},
								"name": {
									Type:        schema.TypeString,
									Description: "Name is the name of the target resource.",
									Required:    true,
								},
								"section_name": {
									Type:        schema.TypeString,
									Description: "SectionName is the name of a section within the target resource.",
									Optional:    true,
								},
							},
						},
					},
					"validation": {
						Type:        schema.TypeList,
						Description: "Validation contains backend TLS validation configuration.",
						Required:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"ca_certificate_refs": {
									Type:        schema.TypeList,
									Description: "CACertificateRefs contains references to Kubernetes objects that contain a PEM-encoded TLS CA certificate bundle.",
									Optional:    true,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"group": {
												Type:        schema.TypeString,
												Description: "Group is the group of the referent.",
												Required:    true,
											},
											"kind": {
												Type:        schema.TypeString,
												Description: "Kind is the kind of the referent.",
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
								"well_known_ca_certificates": {
									Type:        schema.TypeString,
									Description: "WellKnownCACertificates specifies whether a well-known set of CA certificates may be used.",
									Optional:    true,
								},
								"hostname": {
									Type:        schema.TypeString,
									Description: "Hostname is used for two purposes in the connection between Gateways and backends.",
									Optional:    true,
								},
								"subject_alt_names": {
									Type:        schema.TypeList,
									Description: "SubjectAltNames contains one or more Subject Alternative Names.",
									Optional:    true,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"type": {
												Type:        schema.TypeString,
												Description: "Type determines the format of the Subject Alternative Name.",
												Required:    true,
											},
											"hostname": {
												Type:        schema.TypeString,
												Description: "Hostname contains Subject Alternative Name specified in DNS name format.",
												Optional:    true,
											},
											"uri": {
												Type:        schema.TypeString,
												Description: "URI contains Subject Alternative Name specified in a full URI format.",
												Optional:    true,
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
						Optional:    true,
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
									Description: "AncestorRef corresponds with a ParentRef in the spec that this PolicyAncestorStatus describes.",
									Computed:    true,
									Elem:        backendTLSPolicyParentRefSchemaComputed(),
								},
								"controller_name": {
									Type:        schema.TypeString,
									Description: "ControllerName is a domain/path string that indicates the name of the controller that wrote this status.",
									Computed:    true,
								},
								"conditions": {
									Type:        schema.TypeList,
									Description: "Conditions describes the status of the Policy with respect to the given Ancestor.",
									Computed:    true,
									Elem:        backendTLSPolicyConditionsSchemaComputed(),
								},
							},
						},
					},
				},
			},
		},
	}
}

func backendTLSPolicyParentRefSchemaComputed() *schema.Resource {
	return &schema.Resource{
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
	}
}

func backendTLSPolicyConditionsSchemaComputed() *schema.Resource {
	return &schema.Resource{
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
				Description: "Message is a human message describing the condition.",
				Computed:    true,
			},
			"reason": {
				Type:        schema.TypeString,
				Description: "Reason is a machine-readable description of why this is the case.",
				Computed:    true,
			},
			"last_transition_time": {
				Type:        schema.TypeString,
				Description: "LastTransitionTime is the last time this condition changed.",
				Computed:    true,
			},
			"observed_generation": {
				Type:        schema.TypeInt,
				Description: "ObservedGeneration is the 'Generation' of the object that was last processed by the controller.",
				Computed:    true,
			},
		},
	}
}

func resourceKubernetesBackendTLSPolicyV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	obj := gatewayv1.BackendTLSPolicy{
		ObjectMeta: metadata,
		Spec:       expandBackendTLSPolicySpec(d.Get("spec").([]interface{})),
	}

	log.Printf("[INFO] Creating BackendTLSPolicy: %#v", obj)
	out, err := conn.BackendTLSPolicies(metadata.Namespace).Create(ctx, &obj, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Created BackendTLSPolicy: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesBackendTLSPolicyV1Read(ctx, d, meta)
}

func resourceKubernetesBackendTLSPolicyV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading BackendTLSPolicy %s", name)
	obj, err := conn.BackendTLSPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("[DEBUG] BackendTLSPolicy %s not found, removing from state", name)
			d.SetId("")
			return diag.Diagnostics{}
		}
		return diag.FromErr(err)
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

	err = setResourceIdentityNamespaced(d, "gateway.networking.k8s.io/v1", "BackendTLSPolicy", obj.Namespace, obj.Name)
	if err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func resourceKubernetesBackendTLSPolicyV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	name := metadata.Name
	namespace := metadata.Namespace

	log.Printf("[INFO] Updating BackendTLSPolicy: %s", name)

	obj, err := conn.BackendTLSPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	obj.Labels = metadata.Labels
	obj.Annotations = metadata.Annotations
	obj.Spec = expandBackendTLSPolicySpec(d.Get("spec").([]interface{}))

	out, err := conn.BackendTLSPolicies(namespace).Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Updated BackendTLSPolicy: %#v", out)

	return resourceKubernetesBackendTLSPolicyV1Read(ctx, d, meta)
}

func resourceKubernetesBackendTLSPolicyV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting BackendTLSPolicy: %s", name)
	err = conn.BackendTLSPolicies(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			log.Printf("[DEBUG] BackendTLSPolicy %s not found, removing from state", name)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		_, err := conn.BackendTLSPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}
		e := fmt.Errorf("BackendTLSPolicy (%s) still exists", d.Id())
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] BackendTLSPolicy %s deleted", name)
	d.SetId("")
	return diag.Diagnostics{}
}
