package kubernetes

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesAllNodes() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesAllNodesRead,
		Schema: map[string]*schema.Schema{
			"nodes": {
				Type:        schema.TypeList,
				Description: "List of all nodes in a cluster.",
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceKubernetesAllNodesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Listing nodes")
	nodesRaw, err := conn.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	nodes := make([]string, len(nodesRaw.Items))
	for i, v := range nodesRaw.Items {
		nodes[i] = v.Name
	}
	log.Printf("[INFO] Received nodes: %#v", nodes)
	if err := d.Set("nodes", nodes); err != nil {
		return diag.FromErr(err)
	}
	idsum := sha256.New()
	for _, v := range nodes {
		if _, err := idsum.Write([]byte(v)); err != nil {
			return diag.FromErr(err)
		}
	}
	id := fmt.Sprintf("%x", idsum.Sum(nil))
	d.SetId(id)
	return nil
}
