// Copyright IBM Corp. 2017, 2025
// SPDX-License-Identifier: MPL-2.0

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

func dataSourceKubernetesAllNamespaces() *schema.Resource {
	return &schema.Resource{
		Description: "This data source provides a mechanism for listing the names of all available namespaces in a Kubernetes cluster. It can be used to check for existence of a specific namespaces or to apply another resource to all or a subset of existing namespaces in a cluster.In Kubernetes, namespaces provide a scope for names and are intended as a way to divide cluster resources between multiple users.",
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
	conn, err := meta.(KubeClientsets).MainClientset()
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
