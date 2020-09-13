package kubernetes

import (
	"crypto/sha256"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesAllNodes() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKubernetesAllNodesRead,
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

func dataSourceKubernetesAllNodesRead(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	log.Printf("[INFO] Listing nodes")
	nsRaw, err := conn.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	nodes := make([]string, len(nsRaw.Items))
	for i, v := range nsRaw.Items {
		nodes[i] = string(v.Name)
	}
	log.Printf("[INFO] Received nodes: %#v", nodes)
	err = d.Set("nodes", nodes)
	if err != nil {
		return err
	}
	idsum := sha256.New()
	for _, v := range nodes {
		_, err := idsum.Write([]byte(v))
		if err != nil {
			return err
		}
	}
	id := fmt.Sprintf("%x", idsum.Sum(nil))
	d.SetId(id)
	return nil
}
