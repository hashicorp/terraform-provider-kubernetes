// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Schema translates the SDKv2 schema defined in
// dataSourceKubernetesAllNamespaces (data_source_kubernetes_all_namespaces.go)
// into the plugin framework equivalent.
//
// SDKv2 → framework attribute mapping:
//
//	"namespaces" TypeList/Computed/Elem TypeString → schema.ListAttribute{ElementType: types.StringType, Computed: true}
//	implicit id (d.SetId)               → schema.StringAttribute{Computed: true} — must be declared explicitly in framework
func (d *AllNamespacesDataSource) Schema(
	_ context.Context,
	_ datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "This data source provides a mechanism for listing the names of all available namespaces in a Kubernetes cluster. " +
			"It can be used to check for existence of a specific namespace or to apply another resource to all or a subset of existing " +
			"namespaces in a cluster. In Kubernetes, namespaces provide a scope for names and are intended as a way to divide cluster " +
			"resources between multiple users.",
		Attributes: map[string]schema.Attribute{
			// id is a SHA-256 fingerprint of all namespace names, computed in Read.
			// SDKv2 equivalent: d.SetId(fmt.Sprintf("%x", idsum.Sum(nil)))
			"id": schema.StringAttribute{
				Computed: true,
			},
			// namespaces is the list of all namespace names returned by the API.
			// SDKv2 equivalent: schema.TypeList / Computed / Elem TypeString
			"namespaces": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "List of all namespaces in a cluster.",
				Computed:    true,
			},
		},
	}
}
