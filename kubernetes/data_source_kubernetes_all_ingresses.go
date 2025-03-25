// Copyright (c) HashiCorp, Inc.
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

func dataSourceKubernetesAllIngresses() *schema.Resource {
	return &schema.Resource{
		Description: "This data source provides a mechanism for listing all ingresses in a Kubernetes cluster. It can be used to discover ingress resources across all namespaces.",
		ReadContext: dataSourceKubernetesAllIngressesRead,
		Schema: map[string]*schema.Schema{
			"label_selector": {
				Type:        schema.TypeString,
				Description: "A selector to restrict the list of returned ingresses by their labels",
				Optional:    true,
			},
			"field_selector": {
				Type:        schema.TypeString,
				Description: "A selector to restrict the list of returned ingresses by their fields",
				Optional:    true,
			},
			"ingresses": {
				Type:        schema.TypeList,
				Description: "List of all ingresses in the cluster matching the selectors.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "Name of the ingress",
							Computed:    true,
						},
						"namespace": {
							Type:        schema.TypeString,
							Description: "Namespace of the ingress",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func dataSourceKubernetesAllIngressesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Listing ingresses")
	
	listOptions := metav1.ListOptions{}
	if v, ok := d.GetOk("label_selector"); ok {
		listOptions.LabelSelector = v.(string)
	}
	if v, ok := d.GetOk("field_selector"); ok {
		listOptions.FieldSelector = v.(string)
	}

	ingList, err := conn.NetworkingV1().Ingresses("").List(ctx, listOptions)
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}

	ingresses := make([]map[string]interface{}, len(ingList.Items))
	for i, ing := range ingList.Items {
		ingresses[i] = map[string]interface{}{
			"name":      ing.Name,
			"namespace": ing.Namespace,
		}
	}

	d.Set("ingresses", ingresses)

	// Generate stable ID
	idsum := sha256.New()
	for _, ing := range ingresses {
		_, err := idsum.Write([]byte(ing["namespace"].(string) + "/" + ing["name"].(string)))
		if err != nil {
			return diag.FromErr(err)
		}
	}
	id := fmt.Sprintf("%x", idsum.Sum(nil))
	d.SetId(id)

	return nil
}
