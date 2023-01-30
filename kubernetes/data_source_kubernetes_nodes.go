package kubernetes

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func dataSourceKubernetesNodes() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesNodesRead,
		Schema: map[string]*schema.Schema{
			"metadata": {
				Type:        schema.TypeList,
				Description: "Metadata fields to narrow node selection.",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"labels": {
							Type:         schema.TypeMap,
							Description:  "Select nodes with these labels. More info: http://kubernetes.io/docs/user-guide/labels",
							Required:     true,
							Elem:         &schema.Schema{Type: schema.TypeString},
							ValidateFunc: validateLabels,
						},
					},
				},
			},
			"nodes": {
				Type:        schema.TypeList,
				Description: "List of nodes in a cluster.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"metadata": metadataSchema("node", false),
						"spec": {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: nodeSpecFields(),
							},
						},
						"status": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: nodeStatusFields(),
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceKubernetesNodesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	listOptions := metav1.ListOptions{}

	m := d.Get("metadata").([]interface{})
	if len(m) > 0 {
		metadata := expandMetadata(m)
		labelMap, err := metav1.LabelSelectorAsMap(&metav1.LabelSelector{MatchLabels: metadata.Labels})
		if err != nil {
			return diag.FromErr(err)
		}
		labelSelector := labels.SelectorFromSet(labelMap).String()
		log.Printf("[DEBUG] using labelSelector: %s", labelSelector)
		listOptions.LabelSelector = labelSelector
	}

	log.Printf("[INFO] Listing nodes")
	nodesRaw, err := conn.CoreV1().Nodes().List(ctx, listOptions)
	if err != nil {
		return diag.FromErr(err)
	}
	nodes := make([]interface{}, len(nodesRaw.Items))
	for i, v := range nodesRaw.Items {
		log.Printf("[INFO] Received node: %s", v.Name)
		prefix := fmt.Sprintf("nodes.%d.", i)
		n := map[string]interface{}{
			"metadata": flattenMetadata(v.ObjectMeta, d, meta, prefix),
			"spec":     flattenNodeSpec(v.Spec),
			"status":   flattenNodeStatus(v.Status),
		}
		nodes[i] = n
	}
	if err := d.Set("nodes", nodes); err != nil {
		return diag.FromErr(err)
	}
	idsum := sha256.New()
	for _, v := range nodes {
		if _, err := idsum.Write([]byte(fmt.Sprintf("%#v", v))); err != nil {
			return diag.FromErr(err)
		}
	}
	id := fmt.Sprintf("%x", idsum.Sum(nil))
	d.SetId(id)
	return nil
}
