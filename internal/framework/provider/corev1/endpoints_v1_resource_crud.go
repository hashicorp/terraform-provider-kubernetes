// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

// kubernetes_endpoints_v1 manages the core/v1 Endpoints API, which Kubernetes
// deprecated in v1.33 in favour of EndpointSlice; SA1019 is expected here.
//
//nolint:staticcheck
package corev1

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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

	// The schema already requires exactly one metadata block. This turns a plan that
	// bypassed validation into a diagnostic instead of an index-out-of-range panic.
	if len(plan.Metadata) != 1 {
		resp.Diagnostics.AddError("Invalid metadata",
			fmt.Sprintf("Expected exactly one metadata block in the plan, got %d.", len(plan.Metadata)))
		return
	}

	// Like SDKv2, the planned namespace is sent as-is: the schema default covers an omitted
	// namespace, and an explicit "" is left for the API to reject rather than redirected.
	metadata, diags := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ep := api.Endpoints{
		ObjectMeta: metadata,
		Subsets:    expandEndpointsSubsets(plan.Subset),
	}

	tflog.Info(ctx, "Creating endpoints", map[string]any{"name": metadata.Name, "namespace": metadata.Namespace})
	out, err := conn.CoreV1().Endpoints(metadata.Namespace).Create(ctx, &ep, metav1.CreateOptions{})
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
	resp.Diagnostics.Append(setEndpointsFrameworkStateMarker(ctx, resp.Private)...)

	resp.Diagnostics.Append(resp.Identity.Set(ctx, endpointsIdentity(out.Namespace, out.Name))...)
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
			// SDKv2 never stored an identity for this resource, and Terraform before 1.12
			// does not send one, but the framework still requires it when Read removes a
			// missing resource.
			if resp.Identity != nil && resp.Identity.Raw.IsNull() {
				resp.Diagnostics.Append(resp.Identity.Set(ctx, endpointsIdentity(namespace, name))...)
				if resp.Diagnostics.HasError() {
					return
				}
			}
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

	hadEmptySubset := slices.ContainsFunc(state.Subset, isEmptyEndpointsSubset)
	if len(ep.Subsets) > 0 {
		flattened := flattenEndpointsSubsets(ep.Subsets)
		frameworkState, markerDiags := hasEndpointsFrameworkStateMarker(ctx, req.Private)
		resp.Diagnostics.Append(markerDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if frameworkState {
			preserveExplicitEmptyStrings(state.Subset, flattened)
		}
		state.Subset = flattened
	} else if len(state.Subset) > 0 {
		state.Subset = []EndpointsSubsetModel{}
	} else {
		state.Subset = nil
	}
	// An empty subset {} is never sent to Kubernetes, so keep it from prior state.
	if hadEmptySubset {
		state.Subset = append(state.Subset, EndpointsSubsetModel{})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(setEndpointsFrameworkStateMarker(ctx, resp.Private)...)

	resp.Diagnostics.Append(resp.Identity.Set(ctx, endpointsIdentity(ep.Namespace, ep.Name))...)
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

	// As in Create, this supplements the schema validators rather than replacing them.
	if len(plan.Metadata) != 1 || len(state.Metadata) != 1 {
		resp.Diagnostics.AddError("Invalid metadata",
			fmt.Sprintf("Expected exactly one metadata block in plan and state, got %d and %d.", len(plan.Metadata), len(state.Metadata)))
		return
	}

	// Compare as sets, mirroring SDKv2 d.HasChange("subset"): element order is not
	// significant, a null and an empty set both mean no subsets, and empty subset {}
	// blocks are ignored because they are never sent to Kubernetes.
	var planSubset, stateSubset types.Set
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("subset"), &planSubset)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("subset"), &stateSubset)...)
	if resp.Diagnostics.HasError() {
		return
	}
	subsetsUnchanged := nonEmptyEndpointsSubsets(ctx, planSubset).Equal(nonEmptyEndpointsSubsets(ctx, stateSubset))

	ops := common.MetadataPatchOps("/metadata/", state.Metadata[0].MetadataModel, plan.Metadata[0].MetadataModel)
	if !subsetsUnchanged {
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
	resp.Diagnostics.Append(setEndpointsFrameworkStateMarker(ctx, resp.Private)...)

	resp.Diagnostics.Append(resp.Identity.Set(ctx, endpointsIdentity(out.Namespace, out.Name))...)
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

func endpointsIdentity(namespace, name string) common.NamespacedResourceIdentity {
	return common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(endpointsAPIVersion),
			Kind:       types.StringValue(endpointsKind),
			Name:       types.StringValue(name),
		},
		Namespace: types.StringValue(namespace),
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

// isEmptyEndpointsSubset reports whether a subset {} block has no addresses and no ports.
// Like SDKv2, such a block is not sent to Kubernetes, which would reject it.
func isEmptyEndpointsSubset(s EndpointsSubsetModel) bool {
	return len(s.Address) == 0 && len(s.NotReadyAddress) == 0 && len(s.Port) == 0
}

// nonEmptyEndpointsSubsets returns the subset set without empty subset {} elements.
func nonEmptyEndpointsSubsets(ctx context.Context, in types.Set) types.Set {
	kept := []attr.Value{}
	for _, element := range in.Elements() {
		object, ok := element.(types.Object)
		if !ok {
			kept = append(kept, element)
			continue
		}
		for _, name := range []string{"address", "not_ready_address", "port"} {
			if blocks, ok := object.Attributes()[name].(types.Set); ok && (blocks.IsUnknown() || len(blocks.Elements()) > 0) {
				kept = append(kept, element)
				break
			}
		}
	}
	return types.SetValueMust(in.ElementType(ctx), kept)
}

func expandEndpointsSubsets(in []EndpointsSubsetModel) []api.EndpointSubset {
	subsets := make([]api.EndpointSubset, 0, len(in))
	for _, subset := range in {
		if isEmptyEndpointsSubset(subset) {
			continue
		}
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
		subsets = append(subsets, r)
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
			Protocol: types.StringValue(string(n.Protocol)),
		}
		if n.Name != "" {
			m.Name = types.StringValue(n.Name)
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

// endpointsFrameworkStateKey marks state written by this implementation. SDKv2 stored
// "" for both unset and explicitly empty strings, so its state cannot tell them apart.
// Without the marker Read normalizes "" to null; with it, a "" in prior state came
// from configuration and is preserved so explicit empty strings converge. Always
// preserving a prior "" would instead show an update on most upgraded Endpoints.
const endpointsFrameworkStateKey = "endpoints_framework_state"

type privateStateReader interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
}

type privateStateWriter interface {
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

func hasEndpointsFrameworkStateMarker(ctx context.Context, private privateStateReader) (bool, diag.Diagnostics) {
	value, diags := private.GetKey(ctx, endpointsFrameworkStateKey)
	return len(value) > 0, diags
}

func setEndpointsFrameworkStateMarker(ctx context.Context, private privateStateWriter) diag.Diagnostics {
	// The framework always initializes Private; only direct calls in unit tests leave it nil.
	if private == nil || reflect.ValueOf(private).IsNil() {
		return nil
	}
	return private.SetKey(ctx, endpointsFrameworkStateKey, []byte(`true`))
}

type endpointsAddressKey struct {
	subset   string
	notReady bool
	ip       string
}

type endpointsPortKey struct {
	subset   string
	port     int64
	protocol string
}

// endpointsSubsetPortsKey identifies a subset by its port numbers and protocols, so the
// same IP or port in two subsets is matched only within its own subset.
func endpointsSubsetPortsKey(ports []EndpointsPortModel) string {
	keys := make([]string, len(ports))
	for i, p := range ports {
		keys[i] = fmt.Sprintf("%d/%s", p.Port.ValueInt64(), p.Protocol.ValueString())
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

// preserveExplicitEmptyStrings restores "" for hostname, node_name and port name where
// the API omitted the field and prior state held "". Subsets are matched by their ports,
// addresses within them by readiness and IP, and ports by number and protocol.
func preserveExplicitEmptyStrings(prior, current []EndpointsSubsetModel) {
	emptyHostname := map[endpointsAddressKey]bool{}
	emptyNodeName := map[endpointsAddressKey]bool{}
	emptyPortName := map[endpointsPortKey]bool{}
	isEmpty := func(v types.String) bool { return !v.IsNull() && !v.IsUnknown() && v.ValueString() == "" }

	for _, subset := range prior {
		subsetKey := endpointsSubsetPortsKey(subset.Port)
		for notReady, addresses := range map[bool][]EndpointsAddressModel{false: subset.Address, true: subset.NotReadyAddress} {
			for _, a := range addresses {
				key := endpointsAddressKey{subset: subsetKey, notReady: notReady, ip: a.IP.ValueString()}
				emptyHostname[key] = emptyHostname[key] || isEmpty(a.Hostname)
				emptyNodeName[key] = emptyNodeName[key] || isEmpty(a.NodeName)
			}
		}
		for _, p := range subset.Port {
			key := endpointsPortKey{subset: subsetKey, port: p.Port.ValueInt64(), protocol: p.Protocol.ValueString()}
			emptyPortName[key] = emptyPortName[key] || isEmpty(p.Name)
		}
	}

	for i := range current {
		subsetKey := endpointsSubsetPortsKey(current[i].Port)
		for notReady, addresses := range map[bool][]EndpointsAddressModel{false: current[i].Address, true: current[i].NotReadyAddress} {
			for j := range addresses {
				key := endpointsAddressKey{subset: subsetKey, notReady: notReady, ip: addresses[j].IP.ValueString()}
				if addresses[j].Hostname.IsNull() && emptyHostname[key] {
					addresses[j].Hostname = types.StringValue("")
				}
				if addresses[j].NodeName.IsNull() && emptyNodeName[key] {
					addresses[j].NodeName = types.StringValue("")
				}
			}
		}
		for j := range current[i].Port {
			p := &current[i].Port[j]
			key := endpointsPortKey{subset: subsetKey, port: p.Port.ValueInt64(), protocol: p.Protocol.ValueString()}
			if p.Name.IsNull() && emptyPortName[key] {
				p.Name = types.StringValue("")
			}
		}
	}
}
