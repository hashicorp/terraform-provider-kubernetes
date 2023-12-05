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

func dataSourceKubernetesServiceAccountV1() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesServiceAccountV1Read,

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("service account", false),
			"image_pull_secret": {
				Type:        schema.TypeList,
				Description: "A list of references to secrets in the same namespace to use for pulling any images in pods that reference this Service Account. More info: https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
							Computed:    true,
						},
					},
				},
			},
			"secret": {
				Type:        schema.TypeList,
				Description: "A list of secrets allowed to be used by pods running using this Service Account. More info: https://kubernetes.io/docs/concepts/configuration/secret",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
							Computed:    true,
						},
					},
				},
			},
			"automount_service_account_token": {
				Type:        schema.TypeBool,
				Description: "True to enable automatic mounting of the service account token",
				Computed:    true,
			},
			"default_secret_name": {
				Type:       schema.TypeString,
				Computed:   true,
				Deprecated: "Starting from version 1.24.0 Kubernetes does not automatically generate a token for service accounts, in this case, `default_secret_name` will be empty",
			},
		},
	}
}

func dataSourceKubernetesServiceAccountV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	sa, err := conn.CoreV1().ServiceAccounts(metadata.Namespace).Get(ctx, metadata.Name, metav1.GetOptions{})
	if err != nil {
		return diag.Errorf("Unable to fetch service account from Kubernetes: %s", err)
	}

	defaultSecret, diagMsg := findDefaultServiceAccountV1(ctx, sa, conn)

	err = d.Set("default_secret_name", defaultSecret)
	if err != nil {
		return diag.Errorf("Unable to set default_secret_name: %s", err)
	}

	d.SetId(buildId(sa.ObjectMeta))

	log.Printf("[INFO] Reading service account %s", metadata.Name)
	svcAcc, err := conn.CoreV1().ServiceAccounts(metadata.Namespace).Get(ctx, metadata.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		diagMsg = append(diagMsg, diag.FromErr(err)...)
		return diagMsg
	}
	log.Printf("[INFO] Received service account: %#v", svcAcc)

	err = d.Set("metadata", flattenMetadataFields(svcAcc.ObjectMeta))
	if err != nil {
		diagMsg = append(diagMsg, diag.FromErr(err)...)
		return diagMsg
	}

	if svcAcc.AutomountServiceAccountToken == nil {
		err = d.Set("automount_service_account_token", false)
		if err != nil {
			diagMsg = append(diagMsg, diag.FromErr(err)...)
			return diagMsg
		}
	} else {
		err = d.Set("automount_service_account_token", *svcAcc.AutomountServiceAccountToken)
		if err != nil {
			diagMsg = append(diagMsg, diag.FromErr(err)...)
			return diagMsg
		}
	}

	err = d.Set("image_pull_secret", flattenLocalObjectReferenceArray(svcAcc.ImagePullSecrets))
	if err != nil {
		diagMsg = append(diagMsg, diag.FromErr(err)...)
		return diagMsg
	}

	defaultSecretName := d.Get("default_secret_name").(string)
	log.Printf("[DEBUG] Default secret name is %q", defaultSecretName)
	secrets := flattenServiceAccountSecrets(svcAcc.Secrets, defaultSecretName)
	log.Printf("[DEBUG] Flattened secrets: %#v", secrets)
	err = d.Set("secret", secrets)
	if err != nil {
		diagMsg = append(diagMsg, diag.FromErr(err)...)
		return diagMsg
	}

	return nil
}
