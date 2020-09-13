package kubernetes

import (
	"crypto/sha256"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesAllPods() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKubernetesAllPodsRead,
		Schema: map[string]*schema.Schema{
			"namespace": {
				Type:        schema.TypeString,
				Description: "Namespace",
				Optional:    true,
				Default:     "default",
			},
			"pods": {
				Type:        schema.TypeList,
				Description: "List of all pods in a cluster.",
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceKubernetesAllPodsRead(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	log.Printf("[INFO] Listing pods")
	nsRaw, err := conn.CoreV1().Pods(d.Get("namespace").(string)).List(metav1.ListOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	pods := make([]string, len(nsRaw.Items))
	for i, v := range nsRaw.Items {
		pods[i] = string(v.Name)
	}
	log.Printf("[INFO] Received pods: %#v", pods)
	err = d.Set("pods", pods)
	if err != nil {
		return err
	}
	idsum := sha256.New()
	for _, v := range pods {
		_, err := idsum.Write([]byte(v))
		if err != nil {
			return err
		}
	}
	id := fmt.Sprintf("%x", idsum.Sum(nil))
	d.SetId(id)
	return nil
}
