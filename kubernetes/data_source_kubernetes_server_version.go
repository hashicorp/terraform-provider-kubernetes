package kubernetes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gversion "github.com/hashicorp/go-version"
)

func dataSourceKubernetesServerVersion() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesServerVersionRead,
		Schema: map[string]*schema.Schema{
			"version": {
				Type:        schema.TypeString,
				Description: "Kubernetes cluster version.",
				Computed:    true,
			},
		},
	}
}

func dataSourceKubernetesServerVersionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}
	sv, err := conn.ServerVersion()
	if err != nil {
		return diag.FromErr(err)
	}
	gv, err := gversion.NewVersion(sv.String())
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(gv.String())
	d.Set("version", gv.String())
	return nil
}
