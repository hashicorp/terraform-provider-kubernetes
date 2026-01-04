// Copyright IBM Corp. 2017, 2025
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"

	gversion "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceKubernetesServerVersion() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesServerVersionRead,
		Description: "This data source reads the versioning information of the server and makes specific attributes available to Terraform. Read more at [version info reference](https://pkg.go.dev/k8s.io/apimachinery/pkg/version#Info)",
		Schema: map[string]*schema.Schema{
			"version": {
				Type:        schema.TypeString,
				Description: "Composite Kubernetes server version",
				Computed:    true,
			},
			"build_date": {
				Type:        schema.TypeString,
				Description: "Kubernetes server build date",
				Computed:    true,
			},
			"compiler": {
				Type:        schema.TypeString,
				Description: "Compiler used to build Kubernetes",
				Computed:    true,
			},
			"git_commit": {
				Type:        schema.TypeString,
				Description: "Git commit SHA",
				Computed:    true,
			},
			"git_tree_state": {
				Type:        schema.TypeString,
				Description: "Git commit tree state",
				Computed:    true,
			},
			"git_version": {
				Type:        schema.TypeString,
				Description: "Composite version and git commit sha",
				Computed:    true,
			},
			"major": {
				Type:        schema.TypeString,
				Description: "Major Kubernetes version",
				Computed:    true,
			},
			"minor": {
				Type:        schema.TypeString,
				Description: "Minor Kubernetes version",
				Computed:    true,
			},
			"platform": {
				Type:        schema.TypeString,
				Description: "Platform",
				Computed:    true,
			},
			"go_version": {
				Type:        schema.TypeString,
				Description: "Go compiler version",
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
	d.Set("build_date", sv.BuildDate)
	d.Set("compiler", sv.Compiler)
	d.Set("git_commit", sv.GitCommit)
	d.Set("git_tree_state", sv.GitTreeState)
	d.Set("git_version", sv.GitVersion)
	d.Set("go_version", sv.GoVersion)
	d.Set("major", sv.Major)
	d.Set("minor", sv.Minor)
	d.Set("platform", sv.Platform)
	return nil
}
