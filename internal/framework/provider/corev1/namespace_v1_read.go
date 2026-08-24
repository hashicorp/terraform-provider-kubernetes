// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Read is the framework equivalent of dataSourceKubernetesNamespaceV1Read in
// data_source_kubernetes_namespace_v1.go. The Kubernetes API call is identical;
// only the way we read config and write state changes.
func (d *NamespaceV1DataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	// 1. Read the config the user wrote in their .tf file into our typed model.
	//    In SDKv2 this was: metadata := expandMetadata(d.Get("metadata").([]interface{}))
	var model NamespaceV1DataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// 2. Get the Kubernetes client.
	//    In SDKv2 this was: conn, err := meta.(KubeClientsets).MainClientset()
	meta := d.SDKv2Meta().(kubernetes.KubeClientsets)
	conn, err := meta.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	// 3. Extract the namespace name from the metadata block.
	//    Guard against an empty slice — the schema validator enforces exactly
	//    one block, but Read can be called before validation completes.
	if len(model.Metadata) == 0 {
		resp.Diagnostics.AddError(
			"missing metadata block",
			"exactly one metadata block with a name is required",
		)
		return
	}
	name := model.Metadata[0].Name.ValueString()

	// 4. Call the Kubernetes API — identical to the SDKv2 implementation.
	ns, err := conn.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Preserve the SDKv2 behaviour: silently return without error when
			// the namespace is not found, leaving state empty.
			return
		}
		resp.Diagnostics.AddError(
			"error reading Namespace",
			fmt.Sprintf("Failed to read Namespace %q: %s", name, err.Error()),
		)
		return
	}

	// 5. Map the API response onto our model.
	//    In SDKv2 this was done via d.Set("metadata", flattenMetadataFields(...))
	//    and d.Set("spec", flattenNamespaceV1Spec(...)).

	// Set the synthetic id — equivalent to d.SetId(metadata.Name) in SDKv2.
	model.ID = types.StringValue(ns.Name)

	// Populate metadata fields from the live ObjectMeta.
	annotations := make(map[string]types.String, len(ns.Annotations))
	for k, v := range ns.Annotations {
		annotations[k] = types.StringValue(v)
	}
	labels := make(map[string]types.String, len(ns.Labels))
	for k, v := range ns.Labels {
		labels[k] = types.StringValue(v)
	}
	model.Metadata = []MetadataModel{
		{
			Name:            types.StringValue(ns.Name),
			UID:             types.StringValue(string(ns.UID)),
			ResourceVersion: types.StringValue(ns.ResourceVersion),
			Generation:      types.Int64Value(ns.Generation),
			Labels:          labels,
			Annotations:     annotations,
		},
	}

	// Populate the spec block — equivalent to flattenNamespaceV1Spec in SDKv2.
	if len(ns.Spec.Finalizers) > 0 {
		finalizers := make([]types.String, len(ns.Spec.Finalizers))
		for i, f := range ns.Spec.Finalizers {
			finalizers[i] = types.StringValue(string(f))
		}
		model.Spec = []NamespaceSpecModel{{Finalizers: finalizers}}
	} else {
		model.Spec = []NamespaceSpecModel{}
	}

	// 6. Write the populated model into state.
	//    In SDKv2 this happened implicitly through d.Set calls.
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
