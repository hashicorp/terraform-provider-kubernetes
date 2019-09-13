package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	api "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

// Use generated swagger docs from kubernetes' client-go to avoid copy/pasting them here
var (
	networkPolicySpecDoc                  = api.NetworkPolicy{}.SwaggerDoc()["spec"]
	networkPolicySpecIngressDoc           = api.NetworkPolicySpec{}.SwaggerDoc()["ingress"]
	networkPolicyIngressRulePortsDoc      = api.NetworkPolicyIngressRule{}.SwaggerDoc()["ports"]
	networkPolicyIngressRuleFromDoc       = api.NetworkPolicyIngressRule{}.SwaggerDoc()["from"]
	networkPolicySpecEgressDoc            = api.NetworkPolicySpec{}.SwaggerDoc()["egress"]
	networkPolicyEgressRulePortsDoc       = api.NetworkPolicyEgressRule{}.SwaggerDoc()["ports"]
	networkPolicyEgressRuleToDoc          = api.NetworkPolicyEgressRule{}.SwaggerDoc()["to"]
	networkPolicyPortPortDoc              = api.NetworkPolicyPort{}.SwaggerDoc()["port"]
	networkPolicyPortProtocolDoc          = api.NetworkPolicyPort{}.SwaggerDoc()["protocol"]
	networkPolicyPeerIpBlockDoc           = api.NetworkPolicyPeer{}.SwaggerDoc()["ipBlock"]
	ipBlockCidrDoc                        = api.IPBlock{}.SwaggerDoc()["cidr"]
	ipBlockExceptDoc                      = api.IPBlock{}.SwaggerDoc()["except"]
	networkPolicyPeerNamespaceSelectorDoc = api.NetworkPolicyPeer{}.SwaggerDoc()["namespaceSelector"]
	networkPolicyPeerPodSelectorDoc       = api.NetworkPolicyPeer{}.SwaggerDoc()["podSelector"]
	networkPolicySpecPodSelectorDoc       = api.NetworkPolicySpec{}.SwaggerDoc()["podSelector"]
	networkPolicySpecPolicyTypesDoc       = api.NetworkPolicySpec{}.SwaggerDoc()["policyTypes"]
)

func resourceKubernetesNetworkPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesNetworkPolicyCreate,
		Read:   resourceKubernetesNetworkPolicyRead,
		Exists: resourceKubernetesNetworkPolicyExists,
		Update: resourceKubernetesNetworkPolicyUpdate,
		Delete: resourceKubernetesNetworkPolicyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("network policy", true),
			"spec": {
				Type:        schema.TypeList,
				Description: networkPolicySpecDoc,
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ingress": {
							Type:        schema.TypeList,
							Description: networkPolicySpecIngressDoc,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ports": {
										Type:        schema.TypeList,
										Description: networkPolicyIngressRulePortsDoc,
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"port": {
													Type:        schema.TypeString,
													Description: networkPolicyPortPortDoc,
													Optional:    true,
												},
												"protocol": {
													Type:        schema.TypeString,
													Description: networkPolicyPortProtocolDoc,
													Optional:    true,
													Default:     "TCP",
												},
											},
										},
									},
									"from": {
										Type:        schema.TypeList,
										Description: networkPolicyIngressRuleFromDoc,
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"ip_block": {
													Type:        schema.TypeList,
													Description: networkPolicyPeerIpBlockDoc,
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
													Description: networkPolicyPeerNamespaceSelectorDoc,
													Optional:    true,
													MaxItems:    1,
													Elem: &schema.Resource{
														Schema: labelSelectorFields(true),
													},
												},
												"pod_selector": {
													Type:        schema.TypeList,
													Description: networkPolicyPeerPodSelectorDoc,
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
							Description: networkPolicySpecEgressDoc,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ports": {
										Type:        schema.TypeList,
										Description: networkPolicyEgressRulePortsDoc,
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"port": {
													Type:        schema.TypeString,
													Description: networkPolicyPortPortDoc,
													Optional:    true,
												},
												"protocol": {
													Type:        schema.TypeString,
													Description: networkPolicyPortProtocolDoc,
													Optional:    true,
													Default:     "TCP",
												},
											},
										},
									},
									"to": {
										Type:        schema.TypeList,
										Description: networkPolicyEgressRuleToDoc,
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"ip_block": {
													Type:        schema.TypeList,
													Description: networkPolicyPeerIpBlockDoc,
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
													Description: networkPolicyPeerNamespaceSelectorDoc,
													Optional:    true,
													MaxItems:    1,
													Elem: &schema.Resource{
														Schema: labelSelectorFields(true),
													},
												},
												"pod_selector": {
													Type:        schema.TypeList,
													Description: networkPolicyPeerPodSelectorDoc,
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
							Description: networkPolicySpecPodSelectorDoc,
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
							Description: networkPolicySpecPolicyTypesDoc,
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

func resourceKubernetesNetworkPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*KubeClientsets).MainClientset

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandNetworkPolicySpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}

	svc := api.NetworkPolicy{
		ObjectMeta: metadata,
		Spec:       *spec,
	}
	log.Printf("[INFO] Creating new network policy: %#v", svc)
	out, err := conn.NetworkingV1().NetworkPolicies(metadata.Namespace).Create(&svc)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Submitted new network policy: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesNetworkPolicyRead(d, meta)
}

func resourceKubernetesNetworkPolicyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*KubeClientsets).MainClientset

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}
	log.Printf("[INFO] Reading network policy %s", name)
	svc, err := conn.NetworkingV1().NetworkPolicies(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received network policy: %#v", svc)
	err = d.Set("metadata", flattenMetadata(svc.ObjectMeta, d))
	if err != nil {
		return err
	}

	flattened := flattenNetworkPolicySpec(svc.Spec)
	log.Printf("[DEBUG] Flattened network policy spec: %#v", flattened)
	err = d.Set("spec", flattened)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesNetworkPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*KubeClientsets).MainClientset

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("spec") {
		diffOps, err := patchNetworkPolicySpec("spec.0.", "/spec", d)
		if err != nil {
			return err
		}
		ops = append(ops, *diffOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating network policy %q: %v", name, string(data))
	out, err := conn.NetworkingV1().NetworkPolicies(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update network policy: %s", err)
	}
	log.Printf("[INFO] Submitted updated network policy: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesNetworkPolicyRead(d, meta)
}

func resourceKubernetesNetworkPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*KubeClientsets).MainClientset

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}
	log.Printf("[INFO] Deleting network policy: %#v", name)
	err = conn.NetworkingV1().NetworkPolicies(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Network Policy %s deleted", name)

	return nil
}

func resourceKubernetesNetworkPolicyExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*KubeClientsets).MainClientset

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking network policy %s", name)
	_, err = conn.NetworkingV1().NetworkPolicies(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
