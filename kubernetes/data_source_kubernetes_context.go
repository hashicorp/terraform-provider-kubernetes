package kubernetes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceKubernetesConfig() *schema.Resource {

	return &schema.Resource{
		ReadContext: dataSourceKubernetesConfigRead,
		Schema: map[string]*schema.Schema{
			"host": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"client_certificate": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"client_key": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"cluster_ca_certificate": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"username": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"password": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"token": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func dataSourceKubernetesConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clientSets := meta.(kubeClientsets)
	cfg := clientSets.config

	d.SetId(cfg.Host)

	if err := d.Set("host", cfg.Host); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("client_certificate", string(cfg.TLSClientConfig.CertData)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("client_key", string(cfg.TLSClientConfig.KeyData)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("cluster_ca_certificate", string(cfg.TLSClientConfig.CAData)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("username", cfg.Username); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("password", cfg.Password); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("token", cfg.BearerToken); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
