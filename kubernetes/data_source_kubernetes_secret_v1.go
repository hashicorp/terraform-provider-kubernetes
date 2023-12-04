// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesSecretV1() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesSecretV1Read,

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("secret", true),
			"data": {
				Type:        schema.TypeMap,
				Description: "A map of the secret data.",
				Computed:    true,
				Sensitive:   true,
			},
			"binary_data": {
				Type:        schema.TypeMap,
				Description: "A map of the secret data with values encoded in base64 format",
				Optional:    true,
				Sensitive:   true,
			},
			"type": {
				Type:        schema.TypeString,
				Description: "Type of secret",
				Computed:    true,
			},
			"immutable": {
				Type:        schema.TypeBool,
				Description: "Ensures that data stored in the Secret cannot be updated (only object metadata can be modified).",
				Computed:    true,
			},
		},
	}
}

func dataSourceKubernetesSecretV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	om := metav1.ObjectMeta{
		Namespace: metadata.Namespace,
		Name:      metadata.Name,
	}
	d.SetId(buildId(om))

	log.Printf("[INFO] Reading secret %s", metadata.Name)
	secret, err := conn.CoreV1().Secrets(metadata.Namespace).Get(ctx, metadata.Name, metav1.GetOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received secret: %#v", secret.ObjectMeta)

	err = d.Set("metadata", flattenMetadataFields(secret.ObjectMeta))
	if err != nil {
		return diag.FromErr(err)
	}

	binaryDataKeys := []string{}
	if v, ok := d.GetOk("binary_data"); ok {
		binaryData := map[string][]byte{}
		for k := range v.(map[string]interface{}) {
			binaryData[k] = secret.Data[k]
			binaryDataKeys = append(binaryDataKeys, k)
		}
		d.Set("binary_data", base64EncodeByteMap(binaryData))
	}

	for _, k := range binaryDataKeys {
		delete(secret.Data, k)
	}
	d.Set("data", flattenByteMapToStringMap(secret.Data))
	d.Set("type", secret.Type)
	d.Set("immutable", secret.Immutable)

	return nil
}
