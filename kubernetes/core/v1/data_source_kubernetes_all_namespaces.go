package v1

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/provider"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func DataSourceKubernetesAllNamespaces() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesAllNamespacesRead,
		Schema: map[string]*schema.Schema{
			"namespaces": {
				Type:        schema.TypeList,
				Description: "List of all namespaces in a cluster.",
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceKubernetesAllNamespacesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Listing namespaces")
	nsRaw, err := conn.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	namespaces := make([]string, len(nsRaw.Items))
	for i, v := range nsRaw.Items {
		namespaces[i] = v.Name
	}
	log.Printf("[INFO] Received namespaces: %#v", namespaces)
	err = d.Set("namespaces", namespaces)
	if err != nil {
		return diag.FromErr(err)
	}
	idsum := sha256.New()
	for _, v := range namespaces {
		_, err := idsum.Write([]byte(v))
		if err != nil {
			return diag.FromErr(err)
		}
	}
	id := fmt.Sprintf("%x", idsum.Sum(nil))
	d.SetId(id)
	return nil
}
