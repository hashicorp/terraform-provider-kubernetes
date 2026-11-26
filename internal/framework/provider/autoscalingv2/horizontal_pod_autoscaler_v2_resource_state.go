// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package autoscalingv2

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

const (
	providerAddressSuffix  = "/hashicorp/kubernetes"
	v2beta2HPATypeName     = "kubernetes_horizontal_pod_autoscaler_v2beta2"
	unversionedHPATypeName = "kubernetes_horizontal_pod_autoscaler"
	upgradeErrorSummary    = "Unable to upgrade kubernetes_horizontal_pod_autoscaler_v2 state"
)

var metricSourceKeys = []string{"container_resource", "external", "object", "pods", "resource"}

// UpgradeState converts SDKv2 state, which stored every nested object as a list block.
func (r *HorizontalPodAutoscalerV2) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				value, _, _, err := decodeSDKv2State(ctx, req.RawState)
				if err != nil {
					resp.Diagnostics.AddError(upgradeErrorSummary, err.Error())
					return
				}
				resp.State.Raw = value
			},
		},
	}
}

// MoveState accepts state from kubernetes_horizontal_pod_autoscaler_v2beta2 and from
// kubernetes_horizontal_pod_autoscaler. Both are SDKv2 resources that share the v2 schema.
func (r *HorizontalPodAutoscalerV2) MoveState(_ context.Context) []resource.StateMover {
	return []resource.StateMover{
		{StateMover: moveFrom(v2beta2HPATypeName, nil)},
		{StateMover: moveFrom(unversionedHPATypeName, requireV2Fields)},
	}
}

func moveFrom(typeName string, check func(map[string]any) error) func(context.Context, resource.MoveStateRequest, *resource.MoveStateResponse) {
	summary := fmt.Sprintf("Unable to move %s state", typeName)
	return func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
		if req.SourceTypeName != typeName || req.SourceSchemaVersion != 0 ||
			!strings.HasSuffix(req.SourceProviderAddress, providerAddressSuffix) {
			tflog.Debug(ctx, "MoveState: source does not match, skipping", map[string]any{
				"expected_type_name":      typeName,
				"source_type_name":        req.SourceTypeName,
				"source_schema_version":   req.SourceSchemaVersion,
				"source_provider_address": req.SourceProviderAddress,
			})
			return
		}
		value, namespace, name, err := decodeSDKv2StateChecked(ctx, req.SourceRawState, check)
		if err != nil {
			resp.Diagnostics.AddError(summary, err.Error())
			return
		}
		resp.TargetState.Raw = value
		if resp.TargetIdentity != nil {
			resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, hpaIdentity(namespace, name))...)
		}
	}
}

// requireV2Fields rejects kubernetes_horizontal_pod_autoscaler state that used
// autoscaling/v1, which that resource did unless metric or behavior was set.
func requireV2Fields(values map[string]any) error {
	specs, _ := values["spec"].([]any)
	if len(specs) == 1 {
		if spec, ok := specs[0].(map[string]any); ok {
			metric, _ := spec["metric"].([]any)
			behavior, _ := spec["behavior"].([]any)
			if len(metric) > 0 || len(behavior) > 0 {
				return nil
			}
		}
	}
	return fmt.Errorf("the source state sets neither spec.metric nor spec.behavior. " +
		"Move it to kubernetes_horizontal_pod_autoscaler_v1 instead")
}

func decodeSDKv2State(ctx context.Context, raw *tfprotov6.RawState) (tftypes.Value, string, string, error) {
	return decodeSDKv2StateChecked(ctx, raw, nil)
}

func decodeSDKv2StateChecked(ctx context.Context, raw *tfprotov6.RawState, check func(map[string]any) error) (tftypes.Value, string, string, error) {
	var namespace, name string
	value, err := common.DecodeLegacyState(ctx, raw, hpaSchema(), func(values map[string]any) error {
		if check != nil {
			if err := check(values); err != nil {
				return err
			}
		}
		var err error
		namespace, name, err = common.LegacyStateName(values)
		if err != nil {
			return err
		}
		return rewriteSDKv2Spec(values)
	})
	return value, namespace, name, err
}

// rewriteSDKv2Spec converts SDKv2 list blocks under spec to objects, and the zero values
// SDKv2 stored for unset Optional attributes to null.
func rewriteSDKv2Spec(values map[string]any) error {
	specs, _ := values["spec"].([]any)
	for _, s := range specs {
		spec, ok := s.(map[string]any)
		if !ok {
			continue
		}
		nullIfZero(spec, "target_cpu_utilization_percentage")
		refs, _ := spec["scale_target_ref"].([]any)
		for _, ref := range refs {
			if m, ok := ref.(map[string]any); ok {
				nullIfZero(m, "api_version")
			}
		}

		metrics, _ := spec["metric"].([]any)
		for i, m := range metrics {
			metric, ok := m.(map[string]any)
			if !ok {
				continue
			}
			if err := rewriteMetric(metric); err != nil {
				return fmt.Errorf("spec.metric[%d]: %w", i, err)
			}
		}

		behavior, err := singleton(spec, "behavior")
		if err != nil {
			return err
		}
		if behavior != nil {
			for _, key := range []string{"scale_up", "scale_down"} {
				rules, err := singleton(behavior, key)
				if err != nil {
					return fmt.Errorf("behavior: %w", err)
				}
				if rules != nil {
					nullIfZero(rules, "select_policy")
				}
			}
		}
	}
	return nil
}

func rewriteMetric(metric map[string]any) error {
	for _, key := range metricSourceKeys {
		source, err := singleton(metric, key)
		if err != nil {
			return err
		}
		if source == nil {
			continue
		}
		target, err := singleton(source, "target")
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		if target != nil {
			nullIfZero(target, "average_utilization")
			nullIfZero(target, "average_value")
			nullIfZero(target, "value")
		}
		if _, err := singleton(source, "described_object"); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		identifier, err := singleton(source, "metric")
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		if identifier == nil {
			continue
		}
		selector, err := singleton(identifier, "selector")
		if err != nil {
			return fmt.Errorf("%s.metric: %w", key, err)
		}
		if selector != nil {
			rewriteSelector(selector)
		}
	}
	return nil
}

func rewriteSelector(selector map[string]any) {
	if labels, ok := selector["match_labels"].(map[string]any); ok && len(labels) == 0 {
		selector["match_labels"] = nil
	}
	expressions, _ := selector["match_expressions"].([]any)
	if len(expressions) == 0 {
		selector["match_expressions"] = nil
		return
	}
	for _, e := range expressions {
		expression, ok := e.(map[string]any)
		if !ok {
			continue
		}
		nullIfZero(expression, "key")
		nullIfZero(expression, "operator")
		if values, ok := expression["values"].([]any); ok && len(values) == 0 {
			expression["values"] = nil
		}
	}
}

// singleton replaces a list of at most one object with that object, or null when empty.
func singleton(parent map[string]any, key string) (map[string]any, error) {
	switch v := parent[key].(type) {
	case []any:
		switch len(v) {
		case 0:
			parent[key] = nil
			return nil, nil
		case 1:
			obj, ok := v[0].(map[string]any)
			if !ok {
				parent[key] = nil
				return nil, nil
			}
			parent[key] = obj
			return obj, nil
		default:
			return nil, fmt.Errorf("%s has %d elements, expected at most 1", key, len(v))
		}
	case map[string]any:
		return v, nil
	default:
		return nil, nil
	}
}

// nullIfZero replaces SDKv2's stored zero value for an unset Optional attribute.
func nullIfZero(m map[string]any, key string) {
	switch v := m[key].(type) {
	case string:
		if v == "" {
			m[key] = nil
		}
	case json.Number:
		if v.String() == "0" {
			m[key] = nil
		}
	}
}
