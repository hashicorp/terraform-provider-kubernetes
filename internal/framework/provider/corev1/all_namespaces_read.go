// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Read is the framework equivalent of dataSourceKubernetesAllNamespacesRead in
// data_source_kubernetes_all_namespaces.go. The Kubernetes API call and the
// SHA-256 fingerprint logic are identical; only state management changes.
func (d *AllNamespacesDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	// 1. Read existing config into the typed model.
	//    The data source takes no inputs, so this is essentially a no-op,
	//    but it is required by the framework interface.
	var model AllNamespacesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// 2. Obtain the Kubernetes client via the SDKv2 meta function.
	//    SDKv2 equivalent: conn, err := meta.(KubeClientsets).MainClientset()
	conn, err := d.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	// 3. List all namespaces — identical API call to the SDKv2 implementation.
	nsRaw, err := conn.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error listing namespaces",
			fmt.Sprintf("Failed to list Namespaces: %s", err.Error()),
		)
		return
	}

	// 4. Extract names from the API response.
	//    SDKv2 equivalent:
	//        namespaces := make([]string, len(nsRaw.Items))
	//        for i, v := range nsRaw.Items { namespaces[i] = v.Name }
	namespaces := make([]types.String, len(nsRaw.Items))
	for i, ns := range nsRaw.Items {
		namespaces[i] = types.StringValue(ns.Name)
	}
	model.Namespaces = namespaces

	// 5. Compute the SHA-256 fingerprint — byte-for-byte identical logic to
	//    the SDKv2 implementation.
	//    SDKv2 equivalent:
	//        idsum := sha256.New()
	//        for _, v := range namespaces { idsum.Write([]byte(v)) }
	//        d.SetId(fmt.Sprintf("%x", idsum.Sum(nil)))
	idsum := sha256.New()
	for _, ns := range nsRaw.Items {
		_, err := idsum.Write([]byte(ns.Name))
		if err != nil {
			resp.Diagnostics.AddError("id computation error", err.Error())
			return
		}
	}
	model.ID = types.StringValue(fmt.Sprintf("%x", idsum.Sum(nil)))

	// 6. Write the populated model into Terraform state.
	//    SDKv2 equivalent: d.Set(...) calls + d.SetId(...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
