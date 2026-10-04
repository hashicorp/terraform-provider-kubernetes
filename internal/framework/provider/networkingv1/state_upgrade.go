// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	_ resource.ResourceWithUpgradeState = (*IngressV1)(nil)
	_ resource.ResourceWithUpgradeState = (*IngressClassV1)(nil)
	_ resource.ResourceWithUpgradeState = (*NetworkPolicyV1)(nil)
)

func (r *IngressV1) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return networkingStateUpgrader("ingress")
}
func (r *IngressClassV1) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return networkingStateUpgrader("ingress_class")
}
func (r *NetworkPolicyV1) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return networkingStateUpgrader("network_policy")
}

func networkingStateUpgrader(kind string) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{0: {StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
		state, _, _, err := decodeNetworkingState(ctx, req.RawState, resp.State.Schema.Type().TerraformType(ctx), kind)
		if err != nil {
			resp.Diagnostics.AddError("Unable to upgrade networking resource state", err.Error())
			return
		}
		resp.State.Raw = state
	}}}
}
func (r *IngressClassV1) MoveState(context.Context) []resource.StateMover {
	return networkingStateMover("ingress_class")
}
func (r *NetworkPolicyV1) MoveState(context.Context) []resource.StateMover {
	return networkingStateMover("network_policy")
}

func networkingStateMover(kind string) []resource.StateMover {
	return []resource.StateMover{{StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
		if req.SourceTypeName != "kubernetes_"+kind || req.SourceSchemaVersion != 0 || !strings.HasSuffix(req.SourceProviderAddress, "/hashicorp/kubernetes") {
			return
		}
		state, namespace, name, err := decodeNetworkingState(ctx, req.SourceRawState, resp.TargetState.Schema.Type().TerraformType(ctx), kind)
		if err != nil {
			resp.Diagnostics.AddError("Unable to move kubernetes_"+kind+" state", err.Error())
			return
		}
		resp.TargetState.Raw = state
		if resp.TargetIdentity != nil {
			if kind == "ingress_class" {
				resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, ingressClassIdentity(name))...)
			} else {
				resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, networkPolicyIdentity(namespace, name))...)
			}
		}
	}}}
}

// Both upgrade routes use the same conversion. No Kubernetes request or API
// defaulting is needed to change a stored singleton list into an object.
func decodeNetworkingState(ctx context.Context, raw *tfprotov6.RawState, target tftypes.Type, kind string) (tftypes.Value, string, string, error) {
	if raw == nil || len(raw.JSON) == 0 {
		return tftypes.Value{}, "", "", fmt.Errorf("the source state has no JSON data")
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw.JSON))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return tftypes.Value{}, "", "", err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return tftypes.Value{}, "", "", fmt.Errorf("the source state contains unexpected trailing JSON data")
	}
	namespace, name, err := networkingStateName(object, kind == "ingress_class")
	if err != nil {
		return tftypes.Value{}, "", "", err
	}
	if err := upgradeNetworkingSingletons(object, kind); err != nil {
		return tftypes.Value{}, "", "", err
	}
	encoded, err := json.Marshal(object)
	if err != nil {
		return tftypes.Value{}, "", "", err
	}
	state, err := (&tfprotov6.RawState{JSON: encoded}).Unmarshal(target)
	return state, namespace, name, err
}

func networkingStateName(object map[string]any, clusterScoped bool) (string, string, error) {
	id, _ := object["id"].(string)
	namespace, name := "", id
	if !clusterScoped {
		var found bool
		namespace, name, found = strings.Cut(id, "/")
		if !found || namespace == "" {
			return "", "", fmt.Errorf("the source state ID %q must be namespace/name", id)
		}
	}
	if name == "" || strings.Contains(name, "/") {
		return "", "", fmt.Errorf("the source state ID %q is invalid", id)
	}
	blocks, ok := object["metadata"].([]any)
	if !ok || len(blocks) != 1 {
		return "", "", fmt.Errorf("the source state must contain exactly one metadata block")
	}
	metadata, ok := blocks[0].(map[string]any)
	if !ok || metadata == nil {
		return "", "", fmt.Errorf("metadata[0] must be an object")
	}
	for field, want := range map[string]string{"name": name, "namespace": namespace} {
		if field == "namespace" && clusterScoped {
			continue
		}
		switch metadata[field] {
		case nil, "":
			metadata[field] = want
		case want:
		default:
			return "", "", fmt.Errorf("source ID %q does not match metadata.%s", id, field)
		}
	}
	return namespace, name, nil
}

func upgradeNetworkingSingletons(object map[string]any, kind string) error {
	specs, ok := object["spec"].([]any)
	if !ok || len(specs) != 1 {
		return fmt.Errorf("the source state must contain exactly one spec block")
	}
	spec, ok := specs[0].(map[string]any)
	if !ok || spec == nil {
		return fmt.Errorf("spec[0] must be an object")
	}
	switch kind {
	case "ingress_class":
		_, err := networkingSingleton(spec, "parameters", "spec[0]", false)
		return err
	case "ingress":
		if err := upgradeIngressBackend(spec, "default_backend", "spec[0]"); err != nil {
			return err
		}
		return networkingStateList(spec, "rule", "spec[0]", func(rule map[string]any, at string) error {
			http, err := networkingSingleton(rule, "http", at, false)
			if err != nil || http == nil {
				return err
			}
			return networkingStateList(http, "path", at+".http", func(path map[string]any, at string) error { return upgradeIngressBackend(path, "backend", at) })
		})
	case "network_policy":
		if _, err := networkingSingleton(spec, "pod_selector", "spec[0]", true); err != nil {
			return err
		}
		for _, direction := range []struct{ rules, peers string }{{"ingress", "from"}, {"egress", "to"}} {
			err := networkingStateList(spec, direction.rules, "spec[0]", func(rule map[string]any, at string) error {
				return networkingStateList(rule, direction.peers, at, func(peer map[string]any, at string) error {
					for _, field := range []string{"ip_block", "namespace_selector", "pod_selector"} {
						if _, err := networkingSingleton(peer, field, at, false); err != nil {
							return err
						}
					}
					return nil
				})
			})
			if err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported networking state kind %q", kind)
	}
}

func upgradeIngressBackend(parent map[string]any, field, at string) error {
	backend, err := networkingSingleton(parent, field, at, false)
	if err != nil || backend == nil {
		return err
	}
	at += "." + field
	if _, err = networkingSingleton(backend, "resource", at, false); err != nil {
		return err
	}
	service, err := networkingSingleton(backend, "service", at, false)
	if err != nil || service == nil {
		return err
	}
	_, err = networkingSingleton(service, "port", at+".service", true)
	return err
}

// Empty optional lists mean absence. A present empty selector stays an object:
// changing it to null changes which pods and namespaces a NetworkPolicy selects.
func networkingSingleton(parent map[string]any, field, at string, required bool) (map[string]any, error) {
	value := parent[field]
	at += "." + field
	if list, ok := value.([]any); ok {
		switch len(list) {
		case 0:
			value = nil
		case 1:
			value = list[0]
			if value == nil {
				return nil, fmt.Errorf("%s[0] must be an object", at)
			}
		default:
			return nil, fmt.Errorf("%s contains %d objects; expected at most one. Consolidate the configuration and refresh state with the previous provider before upgrading", at, len(list))
		}
	}
	if value == nil {
		if required {
			return nil, fmt.Errorf("%s requires a present object", at)
		}
		parent[field] = nil
		return nil, nil
	}
	object, ok := value.(map[string]any)
	if !ok || object == nil {
		return nil, fmt.Errorf("%s must be an object or a singleton object list", at)
	}
	parent[field] = object
	return object, nil
}

func networkingStateList(parent map[string]any, field, at string, visit func(map[string]any, string) error) error {
	value := parent[field]
	if value == nil {
		return nil
	}
	list, ok := value.([]any)
	if !ok {
		return fmt.Errorf("%s.%s must be a list", at, field)
	}
	for i, value := range list {
		location := fmt.Sprintf("%s.%s[%d]", at, field, i)
		object, ok := value.(map[string]any)
		if !ok || object == nil {
			return fmt.Errorf("%s must be an object", location)
		}
		if err := visit(object, location); err != nil {
			return err
		}
	}
	return nil
}
