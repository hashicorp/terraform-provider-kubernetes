package kubernetes

import (
	"crypto/sha256"
	"fmt"
	"k8s.io/apimachinery/pkg/labels"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesAllNamespaces() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKubernetesAllNamespacesRead,
		Schema: map[string]*schema.Schema{
			"namespaces": {
				Type:        schema.TypeList,
				Description: "List of all namespaces in a cluster.",
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"metadata": metadataSchema("namespace", true),
		},
	}
}

func dataSourceKubernetesAllNamespacesRead(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	lo := metadata.GetLabels()

	log.Printf("[INFO] Listing namespaces")
	nsRaw, err := conn.CoreV1().Namespaces().List(metav1.ListOptions{LabelSelector: labels.SelectorFromSet(lo).String()})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	namespaces := make([]string, len(nsRaw.Items))
	for i, v := range nsRaw.Items {
		namespaces[i] = string(v.Name)
	}
	log.Printf("[INFO] Received namespaces: %#v", namespaces)
	err = d.Set("namespaces", namespaces)
	if err != nil {
		return err
	}
	idsum := sha256.New()
	for _, v := range namespaces {
		_, err := idsum.Write([]byte(v))
		if err != nil {
			return err
		}
	}
	id := fmt.Sprintf("%x", idsum.Sum(nil))
	d.SetId(id)
	return nil
}
