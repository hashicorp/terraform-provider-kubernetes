// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceKubernetesServiceAccountIssuer() *schema.Resource {
	return &schema.Resource{
		Description: "Returns the cluster OpenID Connect issuer and jwks configuration. issuer and jwks_uri are taken from the OpenID discovery document (/.well-known/openid-configuration). The JWKS document itself is fetched from /openid/v1/jwks using the provider's configured authentication.",
		ReadContext: dataSourceKubernetesServiceAccountIssuerRead,
		Schema: map[string]*schema.Schema{
			"issuer": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "OIDC issuer as reported by the cluster OpenID configuration.",
			},
			"jwks_uri": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "jwks_uri as reported by the cluster OpenID configuration.",
			},
			"jwks_raw": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Raw JSON payload returned from /openid/v1/jwks.",
			},
			"jwks_keys": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
					Elem: &schema.Schema{Type: schema.TypeString},
				},
				Description: "List of JWKS keys as string maps.",
			},
		},
	}
}

func dataSourceKubernetesServiceAccountIssuerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	var discoveryBody []byte
	var getErr error
	paths := []string{"/.well-known/openid-configuration", "/openid-configuration"}
	for _, p := range paths {
		discoveryBody, getErr = conn.Discovery().RESTClient().Get().AbsPath(p).DoRaw(ctx)
		if getErr == nil {
			break
		}
	}
	if getErr != nil {
		return diag.FromErr(fmt.Errorf("failed to fetch OpenID configuration from known paths: %w", getErr))
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(discoveryBody, &payload); err != nil {
		return diag.FromErr(fmt.Errorf("failed to decode OpenID configuration: %w", err))
	}

	issuer := ""
	if v, ok := payload["issuer"].(string); ok {
		issuer = v
	}
	jwksURI := ""
	if v, ok := payload["jwks_uri"].(string); ok {
		jwksURI = v
	}

	if err := d.Set("issuer", issuer); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("jwks_uri", jwksURI); err != nil {
		return diag.FromErr(err)
	}

	// Fetch JWKS payload from the cluster endpoint. Use /openid/v1/jwks to keep existing login/auth behaviour.
	jwksBody, err := conn.Discovery().RESTClient().Get().AbsPath("/openid/v1/jwks").DoRaw(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("jwks_raw", string(jwksBody)); err != nil {
		return diag.FromErr(err)
	}

	keys, err := parseJWKSKeys(jwksBody)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("jwks_keys", keys); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("kubernetes_service_account_issuer")
	return nil
}
