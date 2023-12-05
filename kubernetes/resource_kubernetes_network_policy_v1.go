// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

// Use generated swagger docs from kubernetes' client-go to avoid copy/pasting them here
var (
	networkPolicyV1SpecDoc                  = networking.NetworkPolicy{}.SwaggerDoc()["spec"]
	networkPolicyV1SpecIngressDoc           = networking.NetworkPolicySpec{}.SwaggerDoc()["ingress"]
	networkPolicyV1IngressRulePortsDoc      = networking.NetworkPolicyIngressRule{}.SwaggerDoc()["ports"]
	networkPolicyV1IngressRuleFromDoc       = networking.NetworkPolicyIngressRule{}.SwaggerDoc()["from"]
	networkPolicyV1SpecEgressDoc            = networking.NetworkPolicySpec{}.SwaggerDoc()["egress"]
	networkPolicyV1EgressRulePortsDoc       = networking.NetworkPolicyEgressRule{}.SwaggerDoc()["ports"]
	networkPolicyV1EgressRuleToDoc          = networking.NetworkPolicyEgressRule{}.SwaggerDoc()["to"]
	networkPolicyV1PortPortDoc              = networking.NetworkPolicyPort{}.SwaggerDoc()["port"]
	networkPolicyV1PortProtocolDoc          = networking.NetworkPolicyPort{}.SwaggerDoc()["protocol"]
	networkPolicyV1PeerIpBlockDoc           = networking.NetworkPolicyPeer{}.SwaggerDoc()["ipBlock"]
	ipBlockCidrDoc                          = networking.IPBlock{}.SwaggerDoc()["cidr"]
	ipBlockExceptDoc                        = networking.IPBlock{}.SwaggerDoc()["except"]
	networkPolicyV1PeerNamespaceSelectorDoc = networking.NetworkPolicyPeer{}.SwaggerDoc()["namespaceSelector"]
	networkPolicyV1PeerPodSelectorDoc       = networking.NetworkPolicyPeer{}.SwaggerDoc()["podSelector"]
	networkPolicyV1SpecPodSelectorDoc       = networking.NetworkPolicySpec{}.SwaggerDoc()["podSelector"]
	networkPolicyV1SpecPolicyTypesDoc       = networking.NetworkPolicySpec{}.SwaggerDoc()["policyTypes"]
)

func resourceKubernetesNetworkPolicyV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesNetworkPolicyV1Create,
		ReadContext:   resourceKubernetesNetworkPolicyV1Read,
		UpdateContext: resourceKubernetesNetworkPolicyV1Update,
		DeleteContext: resourceKubernetesNetworkPolicyV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("network policy", true),
			"spec": {
				Type:        schema.TypeList,
				Description: networkPolicyV1SpecDoc,
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ingress": {
							Type:        schema.TypeList,
							Description: networkPolicyV1SpecIngressDoc,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ports": {
										Type:        schema.TypeList,
										Description: networkPolicyV1IngressRulePortsDoc,
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"port": {
													Type:        schema.TypeString,
													Description: networkPolicyV1PortPortDoc,
													Optional:    true,
												},
												"protocol": {
													Type:        schema.TypeString,
													Description: networkPolicyV1PortProtocolDoc,
													Optional:    true,
													Default:     "TCP",
												},
											},
										},
									},
									"from": {
										Type:        schema.TypeList,
										Description: networkPolicyV1IngressRuleFromDoc,
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"ip_block": {
													Type:        schema.TypeList,
													Description: networkPolicyV1PeerIpBlockDoc,
													Optional:    true,
													MaxItems:    1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"cidr": {
																Type:        schema.TypeString,
																Description: ipBlockCidrDoc,
																Optional:    true,
															},
															"except": {
																Type:        schema.TypeList,
																Description: ipBlockExceptDoc,
																Optional:    true,
																Elem:        &schema.Schema{Type: schema.TypeString},
															},
														},
													},
												},
												"namespace_selector": {
													Type:        schema.TypeList,
													Description: networkPolicyV1PeerNamespaceSelectorDoc,
													Optional:    true,
													MaxItems:    1,
													Elem: &schema.Resource{
														Schema: labelSelectorFields(true),
													},
												},
												"pod_selector": {
													Type:        schema.TypeList,
													Description: networkPolicyV1PeerPodSelectorDoc,
													Optional:    true,
													MaxItems:    1,
													Elem: &schema.Resource{
														Schema: labelSelectorFields(true),
													},
												},
											},
										},
									},
								},
							},
						},
						"egress": {
							Type:        schema.TypeList,
							Description: networkPolicyV1SpecEgressDoc,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ports": {
										Type:        schema.TypeList,
										Description: networkPolicyV1EgressRulePortsDoc,
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"port": {
													Type:        schema.TypeString,
													Description: networkPolicyV1PortPortDoc,
													Optional:    true,
												},
												"protocol": {
													Type:        schema.TypeString,
													Description: networkPolicyV1PortProtocolDoc,
													Optional:    true,
													Default:     "TCP",
												},
											},
										},
									},
									"to": {
										Type:        schema.TypeList,
										Description: networkPolicyV1EgressRuleToDoc,
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"ip_block": {
													Type:        schema.TypeList,
													Description: networkPolicyV1PeerIpBlockDoc,
													Optional:    true,
													MaxItems:    1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"cidr": {
																Type:        schema.TypeString,
																Description: ipBlockCidrDoc,
																Optional:    true,
															},
															"except": {
																Type:        schema.TypeList,
																Description: ipBlockExceptDoc,
																Optional:    true,
																Elem:        &schema.Schema{Type: schema.TypeString},
															},
														},
													},
												},
												"namespace_selector": {
													Type:        schema.TypeList,
													Description: networkPolicyV1PeerNamespaceSelectorDoc,
													Optional:    true,
													MaxItems:    1,
													Elem: &schema.Resource{
														Schema: labelSelectorFields(true),
													},
												},
												"pod_selector": {
													Type:        schema.TypeList,
													Description: networkPolicyV1PeerPodSelectorDoc,
													Optional:    true,
													MaxItems:    1,
													Elem: &schema.Resource{
														Schema: labelSelectorFields(true),
													},
												},
											},
										},
									},
								},
							},
						},
						"pod_selector": {
							Type:        schema.TypeList,
							Description: networkPolicyV1SpecPodSelectorDoc,
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: labelSelectorFields(true),
							},
						},
						// The policy_types property is made required because the default value is only evaluated server side on resource creation.
						// During the initial creation, a default value is determined and stored, then PolicyTypes is no longer considered unset,
						// it will stick to that value on further updates unless explicitly overridden.
						// Leaving the policy_types property optional here would prevent further updates adding egress rules after the initial resource creation
						// without egress rules nor policy types from working as expected as PolicyTypes will stick to Ingress server side.
						"policy_types": {
							Type:        schema.TypeList,
							Description: networkPolicyV1SpecPolicyTypesDoc,
							Required:    true,
							MinItems:    1,
							MaxItems:    2,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesNetworkPolicyV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandNetworkPolicyV1Spec(d.Get("spec").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}

	svc := networking.NetworkPolicy{
		ObjectMeta: metadata,
		Spec:       *spec,
	}
	log.Printf("[INFO] Creating new network policy: %#v", svc)
	out, err := conn.NetworkingV1().NetworkPolicies(metadata.Namespace).Create(ctx, &svc, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new network policy: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesNetworkPolicyV1Read(ctx, d, meta)
}

func resourceKubernetesNetworkPolicyV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesNetworkPolicyV1Exists(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return diag.Diagnostics{}
	}
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Reading network policy %s", name)
	svc, err := conn.NetworkingV1().NetworkPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received network policy: %#v", svc)
	err = d.Set("metadata", flattenMetadata(svc.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	flattened := flattenNetworkPolicyV1Spec(svc.Spec)
	log.Printf("[DEBUG] Flattened network policy spec: %#v", flattened)
	err = d.Set("spec", flattened)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesNetworkPolicyV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("spec") {
		diffOps, err := patchNetworkPolicyV1Spec("spec.0.", "/spec", d)
		if err != nil {
			return diag.FromErr(err)
		}
		ops = append(ops, *diffOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating network policy %q: %v", name, string(data))
	out, err := conn.NetworkingV1().NetworkPolicies(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update network policy: %s", err)
	}
	log.Printf("[INFO] Submitted updated network policy: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesNetworkPolicyV1Read(ctx, d, meta)
}

func resourceKubernetesNetworkPolicyV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Deleting network policy: %#v", name)
	err = conn.NetworkingV1().NetworkPolicies(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Network Policy %s deleted", name)

	return nil
}

func resourceKubernetesNetworkPolicyV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking network policy %s", name)
	_, err = conn.NetworkingV1().NetworkPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
