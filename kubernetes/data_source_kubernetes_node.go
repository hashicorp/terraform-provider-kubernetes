package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesNode() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesNodeRead,
		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("node", false),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the behavior of the Node.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"podCIDR": {
							Type:        schema.TypeString,
							Description: "PodCIDR represents the pod IP range assigned to the node.",
							Computed:    true,
						},
						"podCIDRs": {
							Type:        schema.TypeSet,
							Description: "podCIDRs represents the IP ranges assigned to the node for usage by Pods on that node.",
							Computed:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Set: schema.HashString,
						},
						"providerID": {
							Type:        schema.TypeString,
							Description: "ID of the node assigned by the cloud provider in the format: <ProviderName>://<ProviderSpecificNodeID>.",
							Computed:    true,
						},
						"unschedulable": {
							Type:        schema.TypeBool,
							Description: "Unschedulable controls node schedulability of new pods.",
							Computed:    true,
						},
						"taints": {
							Type:        schema.TypeList,
							Description: "If specified, the node's taints.",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:        schema.TypeString,
										Description: "The taint key to be applied to a node.",
										Computed:    true,
									},
									"value": {
										Type:        schema.TypeString,
										Description: "The taint value corresponding to the taint key.",
										Computed:    true,
									},
									"effect": {
										Type:        schema.TypeString,
										Description: "The effect of the taint on pods that do not tolerate the taint",
										Computed:    true,
									},
								},
							},
						},
					},
				},
			},
			"status": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allocatable": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"cpu": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"memory": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"pods": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"ephemeral-storage": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"hugepages-1Gi": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"hugepages-2Gi": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"capacity": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"cpu": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"memory": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"pods": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"ephemeral-storage": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"hugepages-1Gi": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"hugepages-2Gi": {
										Type:     schema.TypeString,
										Optional: true,
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

func dataSourceKubernetesNodeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	om := meta_v1.ObjectMeta{
		Namespace: metadata.Name,
		Name:      metadata.Name,
	}
	d.SetId(buildId(om))

	return resourceKubernetesNodeRead(ctx, d, meta)
}

func resourceKubernetesNodeExists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	_, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking node %s", name)

	_, err = conn.CoreV1().Nodes().Get(ctx, name, meta_v1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func resourceKubernetesNodeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesNodeExists(ctx, d, meta)
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

	_, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading node %s", name)
	node, err := conn.CoreV1().Nodes().Get(ctx, name, meta_v1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.Errorf("Failed to read Node '%s' because: %s", buildId(node.ObjectMeta), err)
	}
	log.Printf("[INFO] Received node: %#v", node)
	err = d.Set("metadata", flattenMetadata(node.ObjectMeta, d))
	if err != nil {
		return diag.FromErr(err)
	}

	flattened := flattenNodeSpec(node.Spec)
	log.Printf("[DEBUG] Flattened node spec: %#v", flattened)
	err = d.Set("spec", flattened)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("status", []interface{}{
		map[string]interface{}{
			"allocatable": flattenResourceList(node.Status.Allocatable),
			"capacity ":   flattenResourceList(node.Status.Capacity),
		},
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
