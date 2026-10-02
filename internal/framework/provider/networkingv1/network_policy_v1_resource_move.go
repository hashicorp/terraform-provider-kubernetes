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
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// These structs describe the frozen SDKv2 alias's version-0 JSON, not a future
// target schema. No API conversion is used: empty maps/sets and zero scalars are state.
type networkPolicySDKStateV0 struct {
	ID       string                       `json:"id"`
	Metadata []networkPolicySDKMetadataV0 `json:"metadata"`
	Spec     []networkPolicySDKSpecV0     `json:"spec"`
}

type networkPolicySDKMetadataV0 struct {
	Annotations     map[string]string `json:"annotations"`
	GenerateName    string            `json:"generate_name"`
	Generation      int64             `json:"generation"`
	Labels          map[string]string `json:"labels"`
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	ResourceVersion string            `json:"resource_version"`
	UID             string            `json:"uid"`
}

type networkPolicySDKSpecV0 struct {
	PodSelector []networkPolicySDKSelectorV0 `json:"pod_selector"`
	Ingress     []networkPolicySDKIngressV0  `json:"ingress"`
	Egress      []networkPolicySDKEgressV0   `json:"egress"`
	PolicyTypes []string                     `json:"policy_types"`
}

type networkPolicySDKIngressV0 struct {
	Ports []networkPolicySDKPortV0 `json:"ports"`
	From  []networkPolicySDKPeerV0 `json:"from"`
}

type networkPolicySDKEgressV0 struct {
	Ports []networkPolicySDKPortV0 `json:"ports"`
	To    []networkPolicySDKPeerV0 `json:"to"`
}

type networkPolicySDKPortV0 struct {
	Port     string `json:"port"`
	EndPort  int64  `json:"end_port"`
	Protocol string `json:"protocol"`
}

type networkPolicySDKPeerV0 struct {
	IPBlock           []networkPolicySDKIPBlockV0  `json:"ip_block"`
	NamespaceSelector []networkPolicySDKSelectorV0 `json:"namespace_selector"`
	PodSelector       []networkPolicySDKSelectorV0 `json:"pod_selector"`
}

type networkPolicySDKIPBlockV0 struct {
	CIDR   string   `json:"cidr"`
	Except []string `json:"except"`
}

type networkPolicySDKSelectorV0 struct {
	MatchLabels      map[string]string              `json:"match_labels"`
	MatchExpressions []networkPolicySDKExpressionV0 `json:"match_expressions"`
}

type networkPolicySDKExpressionV0 struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

func (r *NetworkPolicyV1) MoveState(context.Context) []resource.StateMover {
	return []resource.StateMover{{StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
		if req.SourceTypeName != "kubernetes_network_policy" ||
			req.SourceSchemaVersion != 0 ||
			!strings.HasSuffix(req.SourceProviderAddress, "/hashicorp/kubernetes") {
			return
		}
		const summary = "Unable to move kubernetes_network_policy state"
		if req.SourceRawState == nil || len(req.SourceRawState.JSON) == 0 {
			resp.Diagnostics.AddError(summary, "The source state has no JSON data.")
			return
		}
		var prior networkPolicySDKStateV0
		decoder := json.NewDecoder(bytes.NewReader(req.SourceRawState.JSON))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&prior); err != nil {
			resp.Diagnostics.AddError(summary, fmt.Sprintf("Could not decode the source state: %s", err))
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			resp.Diagnostics.AddError(summary, "The source state contains unexpected trailing JSON data.")
			return
		}
		namespace, name, err := kubernetes.IdParts(prior.ID)
		if err != nil || name == "" || len(prior.Metadata) != 1 || len(prior.Spec) != 1 ||
			len(prior.Spec[0].PodSelector) != 1 {
			resp.Diagnostics.AddError(summary, "Expected a namespace/name ID and exactly one metadata, spec, and pod_selector block.")
			return
		}
		data, err := json.Marshal(prior)
		if err != nil {
			resp.Diagnostics.AddError(summary, fmt.Sprintf("Could not encode the source state: %s", err))
			return
		}
		raw, err := tftypes.ValueFromJSON(data, resp.TargetState.Schema.Type().TerraformType(ctx))
		if err != nil {
			resp.Diagnostics.AddError(summary, fmt.Sprintf("Could not convert the source state: %s", err))
			return
		}
		converted := tfsdk.State{Schema: resp.TargetState.Schema, Raw: raw}
		var model NetworkPolicyV1Model
		resp.Diagnostics.Append(converted.Get(ctx, &model)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(resp.TargetState.Set(ctx, &model)...)
		if resp.TargetIdentity != nil {
			resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, networkPolicyIdentity(namespace, name))...)
		}
	}}}
}
