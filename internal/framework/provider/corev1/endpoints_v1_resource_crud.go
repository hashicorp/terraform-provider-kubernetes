// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	api "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

// Create implements [resource.Resource].
func (e *EndpointsV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EndpointsV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, _, metaDiags := e.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	if len(plan.Metadata) == 0 {
		resp.Diagnostics.AddError("Missing metadata", "At least one metadata block is required.")
		return
	}

	metadata, diags := common.ExpandMetadata(ctx, []common.MetadataModel{plan.Metadata[0].MetadataModel})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	namespace := plan.Metadata[0].Namespace.ValueString()
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}
	metadata.Namespace = namespace

	ep := api.Endpoints{
		ObjectMeta: metadata,
		Subsets:    expandEndpointsSubsets(plan.Subset),
	}

	tflog.Info(ctx, "Creating endpoints", map[string]any{"name": metadata.Name, "namespace": namespace})
	out, err := conn.CoreV1().Endpoints(namespace).Create(ctx, &ep, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create endpoints", err.Error())
		return
	}

	plan.ID = types.StringValue(kubernetes.BuildId(out.ObjectMeta))
	plan.Metadata[0].Name = types.StringValue(out.Name)
	plan.Metadata[0].Namespace = types.StringValue(out.Namespace)
	plan.Metadata[0].UID = types.StringValue(string(out.UID))
	plan.Metadata[0].ResourceVersion = types.StringValue(out.ResourceVersion)
	plan.Metadata[0].Generation = types.Int64Value(out.Generation)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Identity.Set(ctx, common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(endpointsAPIVersion),
			Kind:       types.StringValue(endpointsKind),
			Name:       types.StringValue(out.Name),
		},
		Namespace: types.StringValue(out.Namespace),
	})...)
}

// Read implements [resource.Resource].
func (e *EndpointsV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EndpointsV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, metadataFilters, metaDiags := e.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	id := state.ID.ValueString()
	namespace, name, err := kubernetes.IdParts(id)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to read endpoints %q", id), err.Error())
		return
	}

	ep, err := conn.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			tflog.Info(ctx, "Endpoints no longer exists, removing from state", map[string]any{"id": id})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to read endpoints %q", id), err.Error())
		return
	}

	var priorMetadata []common.MetadataModel
	if len(state.Metadata) > 0 {
		priorMetadata = []common.MetadataModel{state.Metadata[0].MetadataModel}
	}
	flattenedMeta, diags := common.FlattenMetadata(ctx, ep.ObjectMeta, priorMetadata,
		metadataFilters.GetIgnoreAnnotations(), metadataFilters.GetIgnoreLabels())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue(kubernetes.BuildId(ep.ObjectMeta))
	state.Metadata = []common.NamespacedMetadataModel{{
		MetadataModel: flattenedMeta[0],
		Namespace:     types.StringValue(ep.Namespace),
	}}

	if len(ep.Subsets) > 0 {
		state.Subset = flattenEndpointsSubsets(ep.Subsets)
	} else if len(state.Subset) > 0 {
		state.Subset = []EndpointsSubsetModel{}
	} else {
		state.Subset = nil
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Identity.Set(ctx, common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(endpointsAPIVersion),
			Kind:       types.StringValue(endpointsKind),
			Name:       types.StringValue(ep.Name),
		},
		Namespace: types.StringValue(ep.Namespace),
	})...)
}

// Update implements [resource.Resource].
func (e *EndpointsV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EndpointsV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state EndpointsV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, _, metaDiags := e.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	id := state.ID.ValueString()
	namespace, name, err := kubernetes.IdParts(id)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update endpoints %q", id), err.Error())
		return
	}

	ops := common.MetadataPatchOps("/metadata/", state.Metadata[0].MetadataModel, plan.Metadata[0].MetadataModel)
	if !reflect.DeepEqual(state.Subset, plan.Subset) {
		subsets := expandEndpointsSubsets(plan.Subset)
		ops = append(ops, &kubernetes.ReplaceOperation{
			Path:  "/subsets",
			Value: subsets,
		})
	}

	var out *api.Endpoints
	if len(ops) == 0 {
		out, err = conn.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Failed to read endpoints during update", err.Error())
			return
		}
	} else {
		data, err := ops.MarshalJSON()
		if err != nil {
			resp.Diagnostics.AddError("Failed to marshal update json patch", err.Error())
			return
		}
		tflog.Info(ctx, "Updating endpoints", map[string]any{"name": name, "namespace": namespace, "patch": string(data)})
		out, err = conn.CoreV1().Endpoints(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Failed to update endpoints", err.Error())
			return
		}
	}

	plan.ID = types.StringValue(kubernetes.BuildId(out.ObjectMeta))
	plan.Metadata[0].Name = types.StringValue(out.Name)
	plan.Metadata[0].Namespace = types.StringValue(out.Namespace)
	plan.Metadata[0].UID = types.StringValue(string(out.UID))
	plan.Metadata[0].ResourceVersion = types.StringValue(out.ResourceVersion)
	plan.Metadata[0].Generation = types.Int64Value(out.Generation)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Identity.Set(ctx, common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(endpointsAPIVersion),
			Kind:       types.StringValue(endpointsKind),
			Name:       types.StringValue(out.Name),
		},
		Namespace: types.StringValue(out.Namespace),
	})...)
}

// Delete implements [resource.Resource].
func (e *EndpointsV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EndpointsV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, _, metaDiags := e.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	id := state.ID.ValueString()
	namespace, name, err := kubernetes.IdParts(id)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete endpoints %q", id), err.Error())
		return
	}

	tflog.Info(ctx, "Deleting endpoints", map[string]any{"name": name, "namespace": namespace})
	err = conn.CoreV1().Endpoints(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Failed to delete endpoints", err.Error())
		return
	}
}

func expandEndpointsAddresses(in []EndpointsAddressModel) []api.EndpointAddress {
	if len(in) == 0 {
		return []api.EndpointAddress{}
	}
	addresses := make([]api.EndpointAddress, len(in))
	for i, addr := range in {
		r := api.EndpointAddress{
			IP: addr.IP.ValueString(),
		}
		if !addr.Hostname.IsNull() && !addr.Hostname.IsUnknown() {
			r.Hostname = addr.Hostname.ValueString()
		}
		if !addr.NodeName.IsNull() && !addr.NodeName.IsUnknown() && addr.NodeName.ValueString() != "" {
			r.NodeName = ptr.To(addr.NodeName.ValueString())
		}
		addresses[i] = r
	}
	return addresses
}

func expandEndpointsPorts(in []EndpointsPortModel) []api.EndpointPort {
	if len(in) == 0 {
		return []api.EndpointPort{}
	}
	ports := make([]api.EndpointPort, len(in))
	for i, port := range in {
		r := api.EndpointPort{
			Port: int32(port.Port.ValueInt64()),
		}
		if !port.Name.IsNull() && !port.Name.IsUnknown() {
			r.Name = port.Name.ValueString()
		}
		if !port.Protocol.IsNull() && !port.Protocol.IsUnknown() {
			r.Protocol = api.Protocol(port.Protocol.ValueString())
		} else {
			r.Protocol = api.ProtocolTCP
		}
		ports[i] = r
	}
	return ports
}

func expandEndpointsSubsets(in []EndpointsSubsetModel) []api.EndpointSubset {
	if len(in) == 0 {
		return []api.EndpointSubset{}
	}
	subsets := make([]api.EndpointSubset, len(in))
	for i, subset := range in {
		r := api.EndpointSubset{}
		if len(subset.Address) > 0 {
			r.Addresses = expandEndpointsAddresses(subset.Address)
		}
		if len(subset.NotReadyAddress) > 0 {
			r.NotReadyAddresses = expandEndpointsAddresses(subset.NotReadyAddress)
		}
		if len(subset.Port) > 0 {
			r.Ports = expandEndpointsPorts(subset.Port)
		}
		subsets[i] = r
	}
	return subsets
}

func flattenEndpointsAddresses(in []api.EndpointAddress) []EndpointsAddressModel {
	if len(in) == 0 {
		return nil
	}
	att := make([]EndpointsAddressModel, len(in))
	for i, n := range in {
		m := EndpointsAddressModel{
			IP:       types.StringValue(n.IP),
			Hostname: types.StringNull(),
			NodeName: types.StringNull(),
		}
		if n.Hostname != "" {
			m.Hostname = types.StringValue(n.Hostname)
		}
		if n.NodeName != nil && *n.NodeName != "" {
			m.NodeName = types.StringValue(*n.NodeName)
		}
		att[i] = m
	}
	return att
}

func flattenEndpointsPorts(in []api.EndpointPort) []EndpointsPortModel {
	if len(in) == 0 {
		return nil
	}
	att := make([]EndpointsPortModel, len(in))
	for i, n := range in {
		m := EndpointsPortModel{
			Port:     types.Int64Value(int64(n.Port)),
			Name:     types.StringNull(),
			Protocol: types.StringValue("TCP"),
		}
		if n.Name != "" {
			m.Name = types.StringValue(n.Name)
		}
		if n.Protocol != "" {
			m.Protocol = types.StringValue(string(n.Protocol))
		}
		att[i] = m
	}
	return att
}

func flattenEndpointsSubsets(in []api.EndpointSubset) []EndpointsSubsetModel {
	if len(in) == 0 {
		return nil
	}
	att := make([]EndpointsSubsetModel, len(in))
	for i, n := range in {
		m := EndpointsSubsetModel{}
		if len(n.Addresses) > 0 {
			m.Address = flattenEndpointsAddresses(n.Addresses)
		}
		if len(n.NotReadyAddresses) > 0 {
			m.NotReadyAddress = flattenEndpointsAddresses(n.NotReadyAddresses)
		}
		if len(n.Ports) > 0 {
			m.Port = flattenEndpointsPorts(n.Ports)
		}
		att[i] = m
	}
	return att
}
