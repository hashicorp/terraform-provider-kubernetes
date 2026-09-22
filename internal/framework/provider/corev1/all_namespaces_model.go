// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import "github.com/hashicorp/terraform-plugin-framework/types"

// AllNamespacesModel is the top-level state model for the
// kubernetes_all_namespaces data source. Every field maps 1:1 to a schema
// attribute via the tfsdk struct tag.
//
// SDKv2 equivalent:
//
//	map[string]*schema.Schema{
//	    "namespaces": {Type: schema.TypeList, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}},
//	}
//
// id is the SHA-256 fingerprint computed in Read, stored as the resource
// identifier (d.SetId in SDKv2).
type AllNamespacesModel struct {
	ID         types.String   `tfsdk:"id"`
	Namespaces []types.String `tfsdk:"namespaces"`
}
