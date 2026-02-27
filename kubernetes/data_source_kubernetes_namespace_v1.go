// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesNamespaceV1(deprecationMessage string) *schema.Resource {
	return &schema.Resource{
		Description:        "This data source provides a mechanism to query attributes of any specific namespace within a Kubernetes cluster. In Kubernetes, namespaces provide a scope for names and are intended as a way to divide cluster resources between multiple users.",
		ReadContext:        dataSourceKubernetesNamespaceV1Read,
		DeprecationMessage: deprecationMessage,

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("namespace", false),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the behavior of the Namespace.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"finalizers": {
							Type:        schema.TypeList,
							Description: "Finalizers is an opaque list of values that must be empty to permanently remove object from storage.",
							Optional:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceKubernetesNamespaceV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	d.SetId(metadata.Name)

	namespace, err := conn.CoreV1().Namespaces().Get(ctx, metadata.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received namespace: %#v", namespace)

	err = d.Set("metadata", flattenMetadataFields(namespace.ObjectMeta))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("spec", flattenNamespaceV1Spec(&namespace.Spec))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func flattenNamespaceV1Spec(in *corev1.NamespaceSpec) []interface{} {
	if in == nil || len(in.Finalizers) == 0 {
		return []interface{}{}
	}
	spec := make(map[string]interface{})
	fin := make([]string, len(in.Finalizers))
	for i, f := range in.Finalizers {
		fin[i] = string(f)
	}
	spec["finalizers"] = fin

	return []interface{}{spec}
}
