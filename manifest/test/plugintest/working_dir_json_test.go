// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package plugintest_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestJSONConfig verifies that TestStep.Config can contain JSON.
// This test also proves that when changing the HCL and JSON formats back and
// forth, the framework deletes the previous configuration file.
func TestJSONConfig(t *testing.T) {
	providerFactories := map[string]func() (*schema.Provider, error){
		"tst": func() (*schema.Provider, error) { return tstProvider(), nil }, //nolint:unparam // required signature
	}
	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: `{"resource":{"tst_t":{"r1":{"s":"x1"}}}}`,
			Check:  resource.TestCheckResourceAttr("tst_t.r1", "s", "x1"),
		}, {
			Config: `resource "tst_t" "r1" { s = "x2" }`,
			Check:  resource.TestCheckResourceAttr("tst_t.r1", "s", "x2"),
		}, {
			Config: `{"resource":{"tst_t":{"r1":{"s":"x3"}}}}`,
			Check:  resource.TestCheckResourceAttr("tst_t.r1", "s", "x3"),
		}},
	})
}

func tstProvider() *schema.Provider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"tst_t": {
				CreateContext: resourceTstTCreate,
				ReadContext:   resourceTstTRead,
				UpdateContext: resourceTstTCreate, // Update is the same as Create
				DeleteContext: resourceTstTDelete,
				Schema: map[string]*schema.Schema{
					"s": {
						Type:     schema.TypeString,
						Required: true,
					},
				},
			},
		},
	}
}

func resourceTstTCreate(ctx context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	d.SetId(d.Get("s").(string))
	return nil
}

func resourceTstTRead(ctx context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	if err := d.Set("s", d.Id()); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceTstTDelete(ctx context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	d.SetId("")
	return nil
}
