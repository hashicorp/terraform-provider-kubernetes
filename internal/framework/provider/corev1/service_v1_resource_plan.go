// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type serviceCollectionPlanningRule struct {
	name      string
	nullValue attr.Value
}

func (r *ServiceV1) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || !req.Plan.Raw.IsKnown() || !req.Config.Raw.IsKnown() {
		return
	}
	rules := map[string][]serviceCollectionPlanningRule{
		"metadata": {
			{name: "annotations", nullValue: types.MapNull(types.StringType)},
			{name: "labels", nullValue: types.MapNull(types.StringType)},
		},
		"spec": {
			{name: "external_ips", nullValue: types.SetNull(types.StringType)},
			{name: "load_balancer_source_ranges", nullValue: types.SetNull(types.StringType)},
			{name: "selector", nullValue: types.MapNull(types.StringType)},
		},
	}
	configured := make(map[string]map[string]attr.Value)
	clearing := false
	for _, parent := range []string{"metadata", "spec"} {
		p := path.Root(parent)
		var config, plan types.List
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, p, &config)...)
		resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, p, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
		configObject, configKnown := servicePlanningObject(config)
		_, planKnown := servicePlanningObject(plan)
		if !configKnown || !planKnown {
			continue
		}
		configured[parent] = configObject
		var priorObject map[string]attr.Value
		if !req.State.Raw.IsNull() {
			var state types.List
			resp.Diagnostics.Append(req.State.GetAttribute(ctx, p, &state)...)
			if resp.Diagnostics.HasError() {
				return
			}
			priorObject, _ = servicePlanningObject(state)
		}
		for _, rule := range rules[parent] {
			if !configObject[rule.name].IsNull() {
				continue
			}
			prior := rule.nullValue
			if value, exists := priorObject[rule.name]; exists {
				prior = value
			}
			if prior.IsUnknown() {
				continue
			}
			desired := prior
			// SDKv2 stores both null and empty for an omitted collection.
			// Keep either representation, but do not make removal of owned
			// nonempty collections sticky through Optional+Computed.
			if serviceCollectionHasElements(prior) {
				desired = rule.nullValue
				clearing = true
			}
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, p.AtListIndex(0).AtName(rule.name), desired)...)
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}
	if clearing {
		// A clear introduced here can be the only change. Its API results
		// must not retain the pre-PATCH resource version or status.
		serviceMarkClearOutputsUnknown(ctx, configured, resp)
	}
	servicePreserveUnchangedSpec(ctx, req, resp)
}

func servicePlanningObject(list types.List) (map[string]attr.Value, bool) {
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) != 1 {
		return nil, false
	}
	element := list.Elements()[0]
	if element.IsNull() || element.IsUnknown() {
		return nil, false
	}
	object, ok := element.(types.Object)
	if !ok {
		return nil, false
	}
	return object.Attributes(), true
}

func serviceCollectionHasElements(value attr.Value) bool {
	if value.IsNull() || value.IsUnknown() {
		return false
	}
	switch value := value.(type) {
	case types.Map:
		return len(value.Elements()) > 0
	case types.Set:
		return len(value.Elements()) > 0
	default:
		return false
	}
}

func serviceMarkClearOutputsUnknown(ctx context.Context, configured map[string]map[string]attr.Value, resp *resource.ModifyPlanResponse) {
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("status"), types.ListUnknown(serviceStatusType))...)
	// Match Framework's computed-without-default marking. Stable identifiers
	// (id, metadata name/namespace/uid), static defaults, and the five managed
	// collections are deliberately excluded.
	rules := map[string]map[string]attr.Value{
		"metadata": {
			"generation":       types.Int64Unknown(),
			"resource_version": types.StringUnknown(),
		},
		"spec": {
			"cluster_ip":              types.StringUnknown(),
			"cluster_ips":             types.ListUnknown(types.StringType),
			"external_traffic_policy": types.StringUnknown(),
			"internal_traffic_policy": types.StringUnknown(),
			"ip_families":             types.ListUnknown(types.StringType),
			"ip_family_policy":        types.StringUnknown(),
			"health_check_node_port":  types.Int64Unknown(),
			"session_affinity_config": types.ObjectUnknown(serviceSessionAffinityConfigType.AttrTypes),
		},
	}
	for _, parent := range []string{"metadata", "spec"} {
		config, known := configured[parent]
		if !known {
			continue
		}
		for name, unknown := range rules[parent] {
			if config[name].IsNull() {
				resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root(parent).AtListIndex(0).AtName(name), unknown)...)
			}
		}
	}
	spec, known := configured["spec"]
	if !known {
		return
	}
	serviceMarkAffinityClearOutputsUnknown(ctx, spec["session_affinity_config"].(types.Object), resp)
	configuredPorts := spec["port"].(types.List)
	if configuredPorts.IsNull() || configuredPorts.IsUnknown() {
		return
	}
	var plannedPorts types.List
	p := path.Root("spec").AtListIndex(0).AtName("port")
	resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, p, &plannedPorts)...)
	if resp.Diagnostics.HasError() || plannedPorts.IsNull() || plannedPorts.IsUnknown() || len(configuredPorts.Elements()) != len(plannedPorts.Elements()) {
		return
	}
	for i, element := range configuredPorts.Elements() {
		if element.IsNull() || element.IsUnknown() || plannedPorts.Elements()[i].IsUnknown() || plannedPorts.Elements()[i].IsNull() {
			continue
		}
		port := element.(types.Object).Attributes()
		if port["node_port"].IsNull() {
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, p.AtListIndex(i).AtName("node_port"), types.Int64Unknown())...)
		}
		if port["target_port"].IsNull() {
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, p.AtListIndex(i).AtName("target_port"), types.StringUnknown())...)
		}
	}
}

func serviceMarkAffinityClearOutputsUnknown(ctx context.Context, config types.Object, resp *resource.ModifyPlanResponse) {
	if config.IsNull() || config.IsUnknown() {
		return
	}
	p := path.Root("spec").AtListIndex(0).AtName("session_affinity_config")
	var plan types.Object
	resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, p, &plan)...)
	if resp.Diagnostics.HasError() || plan.IsNull() || plan.IsUnknown() {
		return
	}
	client := config.Attributes()["client_ip"].(types.Object)
	clientPath := p.AtName("client_ip")
	if client.IsNull() {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, clientPath, types.ObjectUnknown(serviceClientIPType.AttrTypes))...)
		return
	}
	plannedClient := plan.Attributes()["client_ip"].(types.Object)
	if !client.IsUnknown() && !plannedClient.IsNull() && !plannedClient.IsUnknown() && client.Attributes()["timeout_seconds"].IsNull() {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, clientPath.AtName("timeout_seconds"), types.Int64Unknown())...)
	}
}

// Metadata-only edits preserve Service allocations. Spec edits, unknown inputs,
// and replacements must allow Kubernetes to choose new values.
func servicePreserveUnchangedSpec(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || resp.Diagnostics.HasError() {
		return
	}
	for _, name := range []string{"name", "namespace", "generate_name"} {
		p := path.Root("metadata").AtListIndex(0).AtName(name)
		var prior, planned types.String
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, p, &prior)...)
		resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, p, &planned)...)
		if resp.Diagnostics.HasError() || planned.IsUnknown() || !planned.Equal(prior) {
			return
		}
	}
	p := path.Root("spec")
	var config, prior, planned types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, p, &config)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, p, &prior)...)
	resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, p, &planned)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, known := servicePlanningObject(planned); !known {
		return
	}
	configuredValue, err := config.ToTerraformValue(ctx)
	if err != nil || !configuredValue.IsFullyKnown() {
		return
	}
	priorValue, err := prior.ToTerraformValue(ctx)
	if err != nil || !priorValue.IsFullyKnown() {
		return
	}
	plannedValue, err := planned.ToTerraformValue(ctx)
	if err != nil {
		return
	}
	candidate, err := tftypes.Transform(plannedValue, func(at *tftypes.AttributePath, value tftypes.Value) (tftypes.Value, error) {
		if value.IsKnown() {
			return value, nil
		}
		old, _, err := tftypes.WalkAttributePath(priorValue, at)
		if err != nil {
			return value, nil
		}
		return old.(tftypes.Value), nil
	})
	// Restore only when every known spec value, including port order and
	// collection removals, still matches. Leave status and metadata computed.
	if err == nil && candidate.Equal(priorValue) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, p, prior)...)
	}
}
