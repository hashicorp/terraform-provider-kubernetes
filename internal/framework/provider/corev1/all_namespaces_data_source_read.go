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

func (d *AllNamespacesDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {

	var model AllNamespacesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := d.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	nsRaw, err := conn.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error listing namespaces",
			fmt.Sprintf("Failed to list Namespaces: %s", err.Error()),
		)
		return
	}

	namespaces := make([]types.String, len(nsRaw.Items))
	for i, ns := range nsRaw.Items {
		namespaces[i] = types.StringValue(ns.Name)
	}
	model.Namespaces = namespaces

	idsum := sha256.New()
	for _, ns := range nsRaw.Items {
		_, err := idsum.Write([]byte(ns.Name))
		if err != nil {
			resp.Diagnostics.AddError("id computation error", err.Error())
			return
		}
	}
	model.ID = types.StringValue(fmt.Sprintf("%x", idsum.Sum(nil)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
