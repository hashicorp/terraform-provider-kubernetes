// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func resourceKubernetesTokenRequestV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesTokenRequestV1Create,
		ReadContext:   resourceKubernetesTokenRequestV1Read,
		UpdateContext: resourceKubernetesTokenRequestV1Update,
		DeleteContext: resourceKubernetesTokenRequestV1Delete,

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("token request", true),
			"spec": {
				Type:        schema.TypeList,
				Description: authv1.TokenRequest{}.Spec.SwaggerDoc()["spec"],
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: tokenRequestV1SpecFields(),
				},
			},
			"token": {
				Type:        schema.TypeString,
				Description: "Token is the opaque bearer token.",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func resourceKubernetesTokenRequestV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec := expandTokenRequestV1Spec(d.Get("spec").([]interface{}))
	saName := d.Get("metadata.0.name").(string)

	request := authv1.TokenRequest{
		ObjectMeta: metadata,
		Spec:       *spec,
	}

	log.Printf("[INFO] Creating new TokenRequest: %#v", request)
	out, err := conn.CoreV1().ServiceAccounts(metadata.Namespace).CreateToken(ctx, saName, &request, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("token", out.Status.Token)
	s, err := flattenTokenRequestV1Spec(out.Spec, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("spec", s)

	log.Printf("[INFO] Submitted new TokenRequest: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesTokenRequestV1Read(ctx, d, meta)
}

func resourceKubernetesTokenRequestV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceKubernetesTokenRequestV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceKubernetesTokenRequestV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}
