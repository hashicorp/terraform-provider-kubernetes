// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import "github.com/hashicorp/terraform-plugin-framework/types"

type AllNamespacesModel struct {
	// id is the SHA-256 fingerprint computed in Read, stored as the resource
	// identifier (d.SetId in SDKv2).
	ID         types.String   `tfsdk:"id"`
	Namespaces []types.String `tfsdk:"namespaces"`
}
